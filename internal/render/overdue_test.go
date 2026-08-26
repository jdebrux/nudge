package render

import (
	"testing"

	"github.com/jdebrux/nudge/internal/nudge"
)

func TestOverduePromptCopy(t *testing.T) {
	cases := []struct {
		name        string
		phase       nudge.Phase
		wantTitle   string
		wantPrimary string
	}{
		{"focus", nudge.Focus, "focus session done — what next?", "take a break"},
		{"rest", nudge.Rest, "break's over — what next?", "back to focus"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := OverduePromptTitle(c.phase); got != c.wantTitle {
				t.Fatalf("OverduePromptTitle(%v) = %q, want %q", c.phase, got, c.wantTitle)
			}
			if got := OverduePromptPrimaryLabel(c.phase); got != c.wantPrimary {
				t.Fatalf("OverduePromptPrimaryLabel(%v) = %q, want %q", c.phase, got, c.wantPrimary)
			}
		})
	}

	if got, want := StopForNowLabel(), "stop for now"; got != want {
		t.Fatalf("StopForNowLabel() = %q, want %q", got, want)
	}
}
