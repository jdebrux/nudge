// Command nudge is a quiet CLI companion for a focus/rest rhythm. Every
// invocation reads persisted state, applies at most one transition, and
// returns immediately — there is no long-running foreground process
// (PRODUCT.md §5).
package main

import (
	"fmt"
	"os"
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
	if err := run(os.Args[1:], time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "nudge:", err)
		os.Exit(1)
	}
}

func run(args []string, now time.Time) error {
	path, err := store.DefaultPath()
	if err != nil {
		return err
	}
	current, err := store.Load(path)
	if err != nil {
		return err
	}

	cfgPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	var cmd string
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "":
		return bare(path, current, cfg, now)
	case "status":
		fmt.Println(render.Status(current, now))
		return nil
	case "in":
		duration, err := parseDuration(args[1:], cfg.Focus)
		if err != nil {
			return err
		}
		return applyIn(path, current, now, duration)
	case "out":
		duration, err := parseDuration(args[1:], cfg.Break)
		if err != nil {
			return err
		}
		return applyOut(path, current, now, duration)
	case "done":
		return applyDone(path, current, now)
	case "later":
		return applyLater(path, current, now)
	case "config":
		return runConfig(cfgPath, cfg, args[1:])
	case watch.Subcommand:
		return runWatch(path, args[1:])
	default:
		return fmt.Errorf("unknown command %q", cmd)
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
		return fmt.Errorf("usage: nudge config set <focus|break|long-break|every> <value>")
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
// deliver the terminal-bell cue for a timed phase. Not a user-facing
// command.
func runWatch(path string, args []string) error {
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
	return watch.Wait(path, args[0], since, cueAt)
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

func applyOut(path string, current nudge.State, now time.Time, duration time.Duration) error {
	next, applied := current.Out(now, duration)
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

func parseDuration(args []string, fallback time.Duration) (time.Duration, error) {
	if len(args) == 0 {
		return fallback, nil
	}
	d, err := time.ParseDuration(args[0])
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", args[0])
	}
	return d, nil
}
