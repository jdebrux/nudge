package render

import (
	"strings"
	"testing"
	"time"

	"github.com/jdebrux/nudge/internal/config"
)

func TestConfigSummary(t *testing.T) {
	c := config.Config{
		Focus: 25 * time.Minute, Break: 5 * time.Minute, LongBreak: 15 * time.Minute, SessionsPerLongBreak: 4,
		RepeatInterval: 5 * time.Minute, MaxRepeats: 6,
	}

	got := ConfigSummary(c)

	// "repeat limit" (12 chars) is the longest label, so every row pads
	// to that width + 2 — computed here rather than hand-counted, since
	// hand-typed padding is exactly the kind of thing that silently
	// drifts when a row is added.
	const width = len("repeat limit") + 2
	pad := func(label string) string { return label + strings.Repeat(" ", width-len(label)) }

	for _, want := range []string{
		pad("focus") + "25m0s",
		pad("break") + "5m0s",
		pad("long break") + "15m0s",
		pad("every") + "4 sessions",
		pad("repeat") + "5m0s",
		pad("repeat limit") + "6 times",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("ConfigSummary() = %q, want it to contain %q", got, want)
		}
	}
}

func TestConfigUpdated(t *testing.T) {
	got := ConfigUpdated("focus", "30m0s")
	want := "focus set to 30m0s"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
