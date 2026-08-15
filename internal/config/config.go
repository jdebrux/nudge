// Package config persists nudge's durable defaults — focus/break/long-break
// durations and the loop's long-break cadence. It's independent of
// internal/store: config rarely changes and lives under $XDG_CONFIG_HOME,
// while state.json is the live, frequently-rewritten rhythm state under
// $XDG_STATE_HOME. The two have genuinely different lifecycles, so this
// package doesn't share store's atomic-write helper — a small amount of
// duplication here beats coupling two things that should evolve separately.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Config holds nudge's adjustable defaults. It applies to every
// no-argument `in`/`out` regardless of whether a loop is active — loop
// only adds the long-break escalation on top, using SessionsPerLongBreak.
// RepeatInterval/MaxRepeats govern how a missed cue re-fires — see
// internal/watch.
type Config struct {
	Focus                time.Duration
	Break                time.Duration
	LongBreak            time.Duration
	SessionsPerLongBreak int
	RepeatInterval       time.Duration
	MaxRepeats           int
}

// Default returns the values nudge has always used before config existed.
func Default() Config {
	return Config{
		Focus:                25 * time.Minute,
		Break:                5 * time.Minute,
		LongBreak:            15 * time.Minute,
		SessionsPerLongBreak: 4,
		RepeatInterval:       5 * time.Minute,
		MaxRepeats:           6,
	}
}

// configJSON mirrors Config but with durations as strings ("25m0s"), so
// the persisted file reads like a duration rather than raw nanoseconds —
// same precedent as Phase's string encoding in internal/nudge/state.go.
type configJSON struct {
	Focus                string `json:"focus"`
	Break                string `json:"break"`
	LongBreak            string `json:"long_break"`
	SessionsPerLongBreak int    `json:"sessions_per_long_break"`
	RepeatInterval       string `json:"repeat_interval"`
	MaxRepeats           int    `json:"max_repeats"`
}

func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(configJSON{
		Focus:                c.Focus.String(),
		Break:                c.Break.String(),
		LongBreak:            c.LongBreak.String(),
		SessionsPerLongBreak: c.SessionsPerLongBreak,
		RepeatInterval:       c.RepeatInterval.String(),
		MaxRepeats:           c.MaxRepeats,
	})
}

func (c *Config) UnmarshalJSON(data []byte) error {
	var aux configJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	focus, err := time.ParseDuration(aux.Focus)
	if err != nil {
		return err
	}
	brk, err := time.ParseDuration(aux.Break)
	if err != nil {
		return err
	}
	longBreak, err := time.ParseDuration(aux.LongBreak)
	if err != nil {
		return err
	}
	repeatInterval, err := time.ParseDuration(aux.RepeatInterval)
	if err != nil {
		return err
	}

	c.Focus = focus
	c.Break = brk
	c.LongBreak = longBreak
	c.SessionsPerLongBreak = aux.SessionsPerLongBreak
	c.RepeatInterval = repeatInterval
	c.MaxRepeats = aux.MaxRepeats
	return nil
}

// DefaultPath returns the config file location, honouring
// $XDG_CONFIG_HOME if set and falling back to ~/.config/nudge/config.json.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "nudge", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "nudge", "config.json"), nil
}

// Load reads the config file at path. A missing file reads as Default() —
// "no config file yet" and "using the defaults" mean the same thing.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, err
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes config to path atomically: a temp file in the same
// directory, renamed into place, so a crash never leaves a half-written
// config file — same pattern as store.Save.
func Save(path string, c Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".config-*.json")
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
