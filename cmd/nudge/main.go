// Command nudge is a quiet CLI companion for a focus/rest rhythm. Every
// invocation reads persisted state, applies at most one transition, and
// returns immediately — there is no long-running foreground process
// (PRODUCT.md §5).
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jdebrux/nudge/internal/nudge"
	"github.com/jdebrux/nudge/internal/render"
	"github.com/jdebrux/nudge/internal/store"
)

const (
	defaultFocusDuration = 25 * time.Minute
	defaultBreakDuration = 5 * time.Minute
)

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

	var cmd string
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "":
		return bare(path, current, now)
	case "status":
		fmt.Println(render.Status(current, now))
		return nil
	case "in":
		duration, err := parseDuration(args[1:], defaultFocusDuration)
		if err != nil {
			return err
		}
		return applyIn(path, current, now, duration)
	case "out":
		duration, err := parseDuration(args[1:], defaultBreakDuration)
		if err != nil {
			return err
		}
		return applyOut(path, current, now, duration)
	case "done":
		return applyDone(path, current, now)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// bare is what a plain `nudge` runs: show status, or start the default
// rhythm if nothing is running — never a surprise action either way
// (PRODUCT.md §5).
func bare(path string, current nudge.State, now time.Time) error {
	if current.Phase == nudge.Idle {
		return applyIn(path, current, now, defaultFocusDuration)
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

func save(path string, next nudge.State, now time.Time) error {
	if err := store.Save(path, next); err != nil {
		return err
	}
	fmt.Println(render.Status(next, now))
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
