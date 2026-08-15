package render

import (
	"strings"
	"testing"
)

func TestHelpListsEveryCommand(t *testing.T) {
	got := Help()

	for _, want := range []string{
		"nudge", "nudge in", "nudge out", "nudge later", "nudge done",
		"nudge status", "nudge prompt", "nudge loop start|stop", "nudge config",
		"nudge config set", "nudge await -- <cmd>", "nudge help",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Help() = %q, want it to contain %q", got, want)
		}
	}
}

func TestHelpColumnsAreAligned(t *testing.T) {
	got := Help()

	for _, line := range strings.Split(strings.TrimSpace(got), "\n") {
		if !strings.Contains(line, "  ") {
			t.Fatalf("line %q has no column gap", line)
		}
	}
}
