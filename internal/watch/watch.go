// Package watch delivers the proactive cue for a timed phase: a terminal
// bell plus a native notification, repeated on a bounded schedule until
// acted on. nudge never occupies the foreground (every command returns
// immediately), so the cue is delivered by a small detached process
// spawned alongside whatever state transition set (or moved) NextCue —
// nothing runs while IDLE, and nothing runs at all for open-ended
// sessions.
package watch

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/jdebrux/nudge/internal/config"
	"github.com/jdebrux/nudge/internal/nudge"
	"github.com/jdebrux/nudge/internal/store"
)

// Subcommand is the hidden re-exec entry point cmd/nudge dispatches to.
const Subcommand = "__watch"

// Spawn starts a detached process that waits until s.NextCue and then
// fires the cue — unless the state has moved on by then (an early
// `done`, a new `in`/`out`, or a `later` that pushed the cue further
// out). It is a no-op when there's no cue pending (s.NextCue == nil).
func Spawn(s nudge.State) error {
	if s.NextCue == nil {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, Subcommand, s.Phase.String(), s.Since.Format(time.RFC3339Nano), s.NextCue.Format(time.RFC3339Nano))
	if devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0); err == nil {
		cmd.Stdin = devnull
	}
	// Stdout/Stderr stay attached to the current terminal so the bell
	// (and any error output) still lands there once it fires; Setsid
	// only detaches process-group/session signals (e.g. the shell
	// exiting) so the watcher isn't killed the moment this command
	// returns.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// Wait blocks until cueAt, then fires the cue (bell + native
// notification) — but only if the state persisted at statePath still
// has NextCue equal to cueAt. An early `done`, a new `in`/`out`, or a
// `later` that rescheduled the cue makes this particular cue stale, and
// Wait skips firing instead of nagging about a session that's moved on.
//
// If the state is still current, the cue keeps firing every
// RepeatInterval (re-read from configPath before each firing, so a
// mid-wait config change takes effect) until either MaxRepeats firings
// have happened or a later check finds the state has moved on —
// checking status doesn't count as "acted on it" (status never mutates
// state), only a real transition does.
func Wait(statePath, configPath, phase string, since, cueAt time.Time) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.Default()
	}

	for attempt := 0; attempt < cfg.MaxRepeats; attempt++ {
		time.Sleep(time.Until(cueAt.Add(time.Duration(attempt) * cfg.RepeatInterval)))

		current, err := store.Load(statePath)
		if err != nil {
			return err
		}
		if !stillCurrent(current, phase, since, cueAt) {
			return nil
		}

		fire(phase)

		// Reload before the next iteration so a mid-wait config change
		// (a different interval, a different limit) takes effect on
		// the remaining repeats rather than only on the next Spawn.
		if fresh, err := config.Load(configPath); err == nil {
			cfg = fresh
		}
	}
	return nil
}

func stillCurrent(s nudge.State, phase string, since, cueAt time.Time) bool {
	if s.Phase.String() != phase || !s.Since.Equal(since) {
		return false
	}
	return s.NextCue != nil && s.NextCue.Equal(cueAt)
}
