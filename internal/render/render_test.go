package render

import (
	"strings"
	"testing"
	"time"

	"github.com/jdebrux/nudge/internal/nudge"
)

var now = time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC)

func TestStatusIdle(t *testing.T) {
	got := Status(nudge.State{Phase: nudge.Idle}, now)
	if !strings.Contains(got, "idle") {
		t.Fatalf("Status(idle) = %q, want it to contain %q", got, "idle")
	}
}

func TestStatusFocusShowsRemainingAndBar(t *testing.T) {
	until := now.Add(18*time.Minute + 42*time.Second)
	s := nudge.State{Phase: nudge.Focus, Since: now.Add(-6 * time.Minute), Until: &until}

	got := Status(s, now)

	if !strings.Contains(got, "focus · 18:42") {
		t.Fatalf("Status(focus) = %q, want it to contain %q", got, "focus · 18:42")
	}
	if !strings.Contains(got, "█") || !strings.Contains(got, "░") {
		t.Fatalf("Status(focus) = %q, want a partially-filled progress bar", got)
	}
}

func TestStatusRestShowsRemainingAndStepAwayMessage(t *testing.T) {
	until := now.Add(4*time.Minute + 51*time.Second)
	s := nudge.State{Phase: nudge.Rest, Since: now.Add(-10 * time.Minute), Until: &until}

	got := Status(s, now)

	if !strings.Contains(got, "break · 04:51") {
		t.Fatalf("Status(rest) = %q, want it to contain %q", got, "break · 04:51")
	}
	if !strings.Contains(got, "time to step away.") {
		t.Fatalf("Status(rest) = %q, want the step-away message", got)
	}
	if strings.Contains(got, "█") {
		t.Fatalf("Status(rest) = %q, want no progress bar for rest", got)
	}
}

func TestStatusFocusClampsPastDeadlineToZero(t *testing.T) {
	until := now.Add(-1 * time.Minute) // already elapsed
	s := nudge.State{Phase: nudge.Focus, Since: now.Add(-26 * time.Minute), Until: &until}

	got := Status(s, now)

	if !strings.Contains(got, "focus · 00:00") {
		t.Fatalf("Status(focus, overdue) = %q, want clamped to 00:00", got)
	}
}

func TestNoOpMessages(t *testing.T) {
	until := now.Add(4*time.Minute + 51*time.Second)
	resting := nudge.State{Phase: nudge.Rest, Since: now, Until: &until}

	focusedUntil := now.Add(18*time.Minute + 42*time.Second)
	focused := nudge.State{Phase: nudge.Focus, Since: now, Until: &focusedUntil}

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"already focused", AlreadyFocused(focused, now), "already focused, 18:42 left"},
		{"not focused", NotFocused(), "not focused right now"},
		{"already on a break", AlreadyOnABreak(resting, now), "already on a break, 04:51 left"},
		{"already idle", AlreadyIdle(), "already idle"},
		{"nothing to postpone", NothingToPostpone(), "nothing to postpone right now"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.got != c.want {
				t.Fatalf("got %q, want %q", c.got, c.want)
			}
		})
	}
}

func TestPostponed(t *testing.T) {
	nextCue := now.Add(5 * time.Minute)
	s := nudge.State{Phase: nudge.Focus, Since: now.Add(-20 * time.Minute), NextCue: &nextCue}

	got := Postponed(s, now)

	if got != "nudging again in 05:00" {
		t.Fatalf("got %q, want %q", got, "nudging again in 05:00")
	}
}
