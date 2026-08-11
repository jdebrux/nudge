package nudge

import (
	"testing"
	"time"
)

var now = time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC)

func TestTransitions(t *testing.T) {
	cases := []struct {
		name        string
		start       State
		call        func(State) (State, bool)
		wantPhase   Phase
		wantApplied bool
		wantUntil   bool // expect Until to be set on the resulting state
	}{
		{
			name:        "in from idle starts focus",
			start:       State{Phase: Idle},
			call:        func(s State) (State, bool) { return s.In(now, 25*time.Minute) },
			wantPhase:   Focus,
			wantApplied: true,
			wantUntil:   true,
		},
		{
			name:        "in from rest returns to focus",
			start:       State{Phase: Rest, Since: now.Add(-2 * time.Minute)},
			call:        func(s State) (State, bool) { return s.In(now, 25*time.Minute) },
			wantPhase:   Focus,
			wantApplied: true,
			wantUntil:   true,
		},
		{
			name:        "in with no duration is open-ended",
			start:       State{Phase: Idle},
			call:        func(s State) (State, bool) { return s.In(now, 0) },
			wantPhase:   Focus,
			wantApplied: true,
			wantUntil:   false,
		},
		{
			name:        "in while already focus is a no-op",
			start:       State{Phase: Focus, Since: now.Add(-5 * time.Minute)},
			call:        func(s State) (State, bool) { return s.In(now, 25*time.Minute) },
			wantApplied: false,
		},
		{
			name:        "out from focus starts rest",
			start:       State{Phase: Focus, Since: now.Add(-25 * time.Minute)},
			call:        func(s State) (State, bool) { return s.Out(now, 5*time.Minute) },
			wantPhase:   Rest,
			wantApplied: true,
			wantUntil:   true,
		},
		{
			name:        "out while idle is a no-op",
			start:       State{Phase: Idle},
			call:        func(s State) (State, bool) { return s.Out(now, 5*time.Minute) },
			wantApplied: false,
		},
		{
			name:        "out while already resting is a no-op",
			start:       State{Phase: Rest, Since: now.Add(-1 * time.Minute)},
			call:        func(s State) (State, bool) { return s.Out(now, 5*time.Minute) },
			wantApplied: false,
		},
		{
			name:        "done from focus returns to idle",
			start:       State{Phase: Focus, Since: now.Add(-10 * time.Minute)},
			call:        func(s State) (State, bool) { return s.Done(now) },
			wantPhase:   Idle,
			wantApplied: true,
		},
		{
			name:        "done from rest returns to idle",
			start:       State{Phase: Rest, Since: now.Add(-3 * time.Minute)},
			call:        func(s State) (State, bool) { return s.Done(now) },
			wantPhase:   Idle,
			wantApplied: true,
		},
		{
			name:        "done while idle is a no-op",
			start:       State{Phase: Idle},
			call:        func(s State) (State, bool) { return s.Done(now) },
			wantApplied: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, applied := c.call(c.start)

			if applied != c.wantApplied {
				t.Fatalf("applied = %v, want %v", applied, c.wantApplied)
			}
			if !c.wantApplied {
				if got != c.start {
					t.Fatalf("no-op mutated state: got %+v, want unchanged %+v", got, c.start)
				}
				return
			}
			if got.Phase != c.wantPhase {
				t.Fatalf("phase = %v, want %v", got.Phase, c.wantPhase)
			}
			if !got.Since.Equal(now) {
				t.Fatalf("since = %v, want %v", got.Since, now)
			}
			switch {
			case c.wantUntil && got.Until == nil:
				t.Fatalf("expected Until to be set, got nil")
			case !c.wantUntil && got.Until != nil:
				t.Fatalf("expected Until to be nil, got %v", *got.Until)
			}
		})
	}
}
