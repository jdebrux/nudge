// Package watch delivers the proactive cue for a timed phase: a terminal
// bell when it ends. nudge never occupies the foreground (every command
// returns immediately), so the cue is delivered by a small detached
// process spawned alongside the state transition that started the timed
// phase — nothing runs while IDLE, and nothing runs at all for
// open-ended sessions.
package watch

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/jdebrux/nudge/internal/nudge"
	"github.com/jdebrux/nudge/internal/store"
)

// Subcommand is the hidden re-exec entry point cmd/nudge dispatches to.
const Subcommand = "__watch"

// Spawn starts a detached process that waits until s.Until and then
// rings the terminal bell — unless the state has moved on by then. It
// is a no-op for open-ended states (s.Until == nil).
func Spawn(s nudge.State) error {
	if s.Until == nil {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, Subcommand, s.Phase.String(), s.Since.Format(time.RFC3339Nano), s.Until.Format(time.RFC3339Nano))
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

// Wait blocks until until, then rings the terminal bell — but only if
// the state persisted at path still matches what this cue was scheduled
// for. An early `done` or a new `in`/`out` before then makes the cue
// stale, and Wait exits quietly instead.
func Wait(path, phase string, since, until time.Time) error {
	time.Sleep(time.Until(until))

	current, err := store.Load(path)
	if err != nil {
		return err
	}
	if !stillCurrent(current, phase, since, until) {
		return nil
	}

	fmt.Fprint(os.Stdout, "\a")
	return nil
}

func stillCurrent(s nudge.State, phase string, since, until time.Time) bool {
	if s.Phase.String() != phase || !s.Since.Equal(since) {
		return false
	}
	return s.Until != nil && s.Until.Equal(until)
}
