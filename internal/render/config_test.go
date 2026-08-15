package render

import (
	"strings"
	"testing"
	"time"

	"github.com/jdebrux/nudge/internal/config"
)

func TestConfigSummary(t *testing.T) {
	c := config.Config{Focus: 25 * time.Minute, Break: 5 * time.Minute, LongBreak: 15 * time.Minute, SessionsPerLongBreak: 4}

	got := ConfigSummary(c)

	for _, want := range []string{"focus       25m0s", "break       5m0s", "long break  15m0s", "every       4 sessions"} {
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
