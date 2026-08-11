// Package store persists nudge.State to disk between invocations. It is
// the only part of nudge that touches the filesystem for state; the
// domain rules live in package nudge and know nothing about files.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/jdebrux/nudge/internal/nudge"
)

// DefaultPath returns the state file location, honouring $XDG_STATE_HOME
// if set and falling back to ~/.local/state/nudge/state.json.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "nudge", "state.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "nudge", "state.json"), nil
}

// Load reads the state file at path. A missing file is not an error — it
// reads as a fresh IDLE state, since "no state file yet" and "nothing
// tracked" mean the same thing.
func Load(path string) (nudge.State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nudge.State{Phase: nudge.Idle}, nil
	}
	if err != nil {
		return nudge.State{}, err
	}

	var s nudge.State
	if err := json.Unmarshal(data, &s); err != nil {
		return nudge.State{}, err
	}
	return s, nil
}

// Save writes state to path atomically: it writes to a temp file in the
// same directory and renames it into place, so a crash or a concurrent
// read never observes a half-written state file.
func Save(path string, s nudge.State) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".state-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once the rename below succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}
