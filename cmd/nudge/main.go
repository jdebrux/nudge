// Command nudge is a quiet CLI companion for a focus/rest rhythm. Every
// invocation reads persisted state, applies at most one transition, and
// returns immediately — there is no long-running foreground process
// (PRODUCT.md §5).
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/jdebrux/nudge/internal/config"
	"github.com/jdebrux/nudge/internal/nudge"
	"github.com/jdebrux/nudge/internal/render"
	"github.com/jdebrux/nudge/internal/store"
	"github.com/jdebrux/nudge/internal/watch"
)

const defaultLaterDelay = 5 * time.Minute

func main() {
	code, err := run(os.Args[1:], time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, "nudge:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

// run returns an exit code alongside the usual error. Every command
// except `await` always returns 0 here — await is the one deliberate
// exception to "every command returns immediately" (PRODUCT.md's
// deployment scenario, §5): it runs a child command itself and mirrors
// that child's exit code back out.
func run(args []string, now time.Time) (int, error) {
	if len(args) > 0 && isHelp(args[0]) {
		fmt.Println(render.Help())
		return 0, nil
	}

	path, err := store.DefaultPath()
	if err != nil {
		return 0, err
	}
	current, err := store.Load(path)
	if err != nil {
		return 0, err
	}

	cfgPath, err := config.DefaultPath()
	if err != nil {
		return 0, err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return 0, err
	}

	var cmd string
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "":
		return 0, bare(path, current, cfg, now)
	case "status":
		fmt.Println(render.Status(current, now))
		return 0, nil
	case "in":
		explicit, err := parseExplicitDuration(args[1:])
		if err != nil {
			return 0, err
		}
		duration := cfg.Focus
		if explicit != nil {
			duration = *explicit
		}
		return 0, applyIn(path, current, now, duration)
	case "out":
		explicit, err := parseExplicitDuration(args[1:])
		if err != nil {
			return 0, err
		}
		return 0, applyOut(path, current, cfg, now, explicit)
	case "done":
		return 0, applyDone(path, current, now)
	case "later":
		return 0, applyLater(path, current, now)
	case "config":
		return 0, runConfig(cfgPath, cfg, args[1:])
	case "loop":
		return 0, runLoop(path, current, args[1:])
	case "await":
		return applyAwait(path, current, cfg, now, args[1:])
	case watch.Subcommand:
		return 0, runWatch(path, cfgPath, args[1:])
	default:
		return 0, fmt.Errorf("unknown command %q — try `nudge help`", cmd)
	}
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

// applyAwait runs cmdArgs itself, blocking until it exits. It behaves
// like an open-ended `out` for the duration — no fixed cue, since the
// real signal here is the command finishing, not a timer — and
// auto-returns to FOCUS the instant it exits, regardless of the child's
// exit code. Only valid from FOCUS, same as `out`; the guard is checked
// before the command ever runs, so a no-op truly has zero side effects.
func applyAwait(path string, current nudge.State, cfg config.Config, now time.Time, args []string) (int, error) {
	if len(args) == 0 || args[0] != "--" || len(args) == 1 {
		return 0, fmt.Errorf("usage: nudge await -- <command>")
	}
	cmdArgs := args[1:]

	if current.Phase != nudge.Focus {
		if current.Phase == nudge.Idle {
			fmt.Println(render.NotFocused())
		} else {
			fmt.Println(render.AlreadyOnABreak(current, now))
		}
		return 0, nil
	}

	child := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	if err := child.Start(); err != nil {
		return 0, err
	}

	resting, applied := current.Out(time.Now(), 0)
	if applied {
		if err := store.Save(path, resting); err != nil {
			return 0, err
		}
		fmt.Println(render.Status(resting, time.Now()))
	}

	waitErr := child.Wait()

	returned := time.Now()
	focused, applied := resting.In(returned, cfg.Focus)
	if applied {
		if err := persist(path, focused); err != nil {
			return 0, err
		}
		fmt.Println(render.Status(focused, returned))
	}

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, waitErr
	}
	return 0, nil
}

// runLoop handles `nudge loop start` and `nudge loop stop`.
func runLoop(path string, current nudge.State, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: nudge loop <start|stop>")
	}

	switch args[0] {
	case "start":
		next, applied := current.StartLoop()
		if !applied {
			fmt.Println(render.LoopAlreadyActive())
			return nil
		}
		if err := store.Save(path, next); err != nil {
			return err
		}
		fmt.Println(render.LoopStarted())
		return nil
	case "stop":
		next, applied := current.StopLoop()
		if !applied {
			fmt.Println(render.LoopAlreadyInactive())
			return nil
		}
		if err := store.Save(path, next); err != nil {
			return err
		}
		fmt.Println(render.LoopStopped())
		return nil
	default:
		return fmt.Errorf("unknown loop command %q", args[0])
	}
}

// runConfig handles bare `nudge config` (show current defaults) and
// `nudge config set <key> <value>`.
func runConfig(cfgPath string, cfg config.Config, args []string) error {
	if len(args) == 0 {
		fmt.Println(render.ConfigSummary(cfg))
		return nil
	}
	if args[0] != "set" {
		return fmt.Errorf("unknown config command %q", args[0])
	}
	if len(args) != 3 {
		return fmt.Errorf("usage: nudge config set <focus|break|long-break|every|repeat|repeat-limit> <value>")
	}

	key, value := args[1], args[2]
	switch key {
	case "focus":
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration %q", value)
		}
		cfg.Focus = d
	case "break":
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration %q", value)
		}
		cfg.Break = d
	case "long-break":
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration %q", value)
		}
		cfg.LongBreak = d
	case "every":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return fmt.Errorf("invalid session count %q", value)
		}
		cfg.SessionsPerLongBreak = n
	case "repeat":
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration %q", value)
		}
		cfg.RepeatInterval = d
	case "repeat-limit":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid repeat limit %q", value)
		}
		cfg.MaxRepeats = n
	default:
		return fmt.Errorf("unknown config key %q", key)
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}
	fmt.Println(render.ConfigUpdated(key, value))
	return nil
}

// runWatch is the hidden re-exec entry point spawned by save() to
// deliver the cue for a timed phase. Not a user-facing command.
func runWatch(path, cfgPath string, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("%s: expected phase, since, and until", watch.Subcommand)
	}
	since, err := time.Parse(time.RFC3339Nano, args[1])
	if err != nil {
		return err
	}
	cueAt, err := time.Parse(time.RFC3339Nano, args[2])
	if err != nil {
		return err
	}
	return watch.Wait(path, cfgPath, args[0], since, cueAt)
}

// bare is what a plain `nudge` runs: show status, or start the default
// rhythm if nothing is running — never a surprise action either way
// (PRODUCT.md §5).
func bare(path string, current nudge.State, cfg config.Config, now time.Time) error {
	if current.Phase == nudge.Idle {
		return applyIn(path, current, now, cfg.Focus)
	}
	fmt.Println(render.Status(current, now))
	return nil
}

func applyIn(path string, current nudge.State, now time.Time, duration time.Duration) error {
	next, applied := current.In(now, duration)
	if !applied {
		fmt.Println(render.AlreadyFocused(current, now))
		return nil
	}
	return save(path, next, now)
}

// applyOut ends a focus session. With no explicit duration, an active
// loop gets a say: it counts the completed focus session and escalates
// to cfg.LongBreak every Nth one (per cfg.SessionsPerLongBreak) instead
// of the plain cfg.Break. An explicit duration bypasses loop bookkeeping
// entirely — the user's call always wins.
func applyOut(path string, current nudge.State, cfg config.Config, now time.Time, explicit *time.Duration) error {
	working := current
	duration := cfg.Break

	switch {
	case explicit != nil:
		duration = *explicit
	case current.Phase == nudge.Focus && current.LoopActive:
		var dueForLongBreak bool
		working, dueForLongBreak = current.CompletedFocusSession(cfg.SessionsPerLongBreak)
		if dueForLongBreak {
			duration = cfg.LongBreak
		}
	}

	next, applied := working.Out(now, duration)
	if !applied {
		if current.Phase == nudge.Idle {
			fmt.Println(render.NotFocused())
		} else {
			fmt.Println(render.AlreadyOnABreak(current, now))
		}
		return nil
	}
	return save(path, next, now)
}

func applyDone(path string, current nudge.State, now time.Time) error {
	next, applied := current.Done(now)
	if !applied {
		fmt.Println(render.AlreadyIdle())
		return nil
	}
	return save(path, next, now)
}

// applyLater postpones the next cue without changing the current phase —
// Since/Until stay put, only NextCue moves. It reuses persist() so the
// rescheduled cue gets its own watcher, but prints a different one-line
// echo than a real transition since nothing about the session itself
// changed.
func applyLater(path string, current nudge.State, now time.Time) error {
	next, applied := current.Later(now, defaultLaterDelay)
	if !applied {
		fmt.Println(render.NothingToPostpone())
		return nil
	}
	if err := persist(path, next); err != nil {
		return err
	}
	fmt.Println(render.Postponed(next, now))
	return nil
}

func save(path string, next nudge.State, now time.Time) error {
	if err := persist(path, next); err != nil {
		return err
	}
	fmt.Println(render.Status(next, now))
	return nil
}

// persist saves state and, if it left a timed cue pending, spawns a
// watcher for it. A failure to schedule the cue doesn't fail the
// command — the state transition itself already succeeded.
func persist(path string, next nudge.State) error {
	if err := store.Save(path, next); err != nil {
		return err
	}
	if err := watch.Spawn(next); err != nil {
		fmt.Fprintln(os.Stderr, "nudge: couldn't schedule a cue:", err)
	}
	return nil
}

// parseExplicitDuration parses an optional duration argument, returning
// nil (not an error) when none was given — callers need to tell "nothing
// specified, use the default" apart from "specified explicitly", since
// applyOut only lets loop's long-break escalation apply in the former
// case.
func parseExplicitDuration(args []string) (*time.Duration, error) {
	if len(args) == 0 {
		return nil, nil
	}
	d, err := time.ParseDuration(args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q", args[0])
	}
	return &d, nil
}
