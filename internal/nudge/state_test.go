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
			// NextCue starts out equal to Until — Later is the only
			// thing that ever makes them diverge.
			switch {
			case c.wantUntil && (got.NextCue == nil || !got.NextCue.Equal(*got.Until)):
				t.Fatalf("expected NextCue to equal Until, got NextCue=%v Until=%v", got.NextCue, got.Until)
			case !c.wantUntil && got.NextCue != nil:
				t.Fatalf("expected NextCue to be nil, got %v", *got.NextCue)
			}
		})
	}
}

func TestLater(t *testing.T) {
	until := now.Add(20 * time.Minute)
	delay := 5 * time.Minute

	cases := []struct {
		name        string
		start       State
		wantApplied bool
	}{
		{
			name:        "postpones a pending focus cue",
			start:       State{Phase: Focus, Since: now.Add(-5 * time.Minute), Until: &until, NextCue: &until},
			wantApplied: true,
		},
		{
			name:        "postpones a pending rest cue",
			start:       State{Phase: Rest, Since: now.Add(-1 * time.Minute), Until: &until, NextCue: &until},
			wantApplied: true,
		},
		{
			name:        "no-op while idle",
			start:       State{Phase: Idle},
			wantApplied: false,
		},
		{
			name:        "no-op for an open-ended session with no cue",
			start:       State{Phase: Focus, Since: now.Add(-5 * time.Minute)},
			wantApplied: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, applied := c.start.Later(now, delay)

			if applied != c.wantApplied {
				t.Fatalf("applied = %v, want %v", applied, c.wantApplied)
			}
			if !c.wantApplied {
				if got != c.start {
					t.Fatalf("no-op mutated state: got %+v, want unchanged %+v", got, c.start)
				}
				return
			}

			wantNextCue := now.Add(delay)
			if got.NextCue == nil || !got.NextCue.Equal(wantNextCue) {
				t.Fatalf("NextCue = %v, want %v", got.NextCue, wantNextCue)
			}
			if got.Phase != c.start.Phase {
				t.Fatalf("phase changed: got %v, want unchanged %v", got.Phase, c.start.Phase)
			}
			if !got.Since.Equal(c.start.Since) {
				t.Fatalf("since changed: got %v, want unchanged %v", got.Since, c.start.Since)
			}
			if !got.Until.Equal(*c.start.Until) {
				t.Fatalf("until changed: got %v, want unchanged %v", got.Until, c.start.Until)
			}
		})
	}
}

func TestLoopBookkeepingAcrossTransitions(t *testing.T) {
	t.Run("in and out preserve LoopActive and SessionsCompleted", func(t *testing.T) {
		start := State{Phase: Rest, Since: now.Add(-5 * time.Minute), LoopActive: true, SessionsCompleted: 2}

		focused, applied := start.In(now, 25*time.Minute)
		if !applied {
			t.Fatalf("In: applied = false, want true")
		}
		if !focused.LoopActive || focused.SessionsCompleted != 2 {
			t.Fatalf("In: LoopActive=%v SessionsCompleted=%d, want true/2", focused.LoopActive, focused.SessionsCompleted)
		}

		rested, applied := focused.Out(now, 5*time.Minute)
		if !applied {
			t.Fatalf("Out: applied = false, want true")
		}
		if !rested.LoopActive || rested.SessionsCompleted != 2 {
			t.Fatalf("Out: LoopActive=%v SessionsCompleted=%d, want true/2", rested.LoopActive, rested.SessionsCompleted)
		}
	})

	t.Run("done clears the loop entirely", func(t *testing.T) {
		start := State{Phase: Focus, Since: now.Add(-5 * time.Minute), LoopActive: true, SessionsCompleted: 3}

		idle, applied := start.Done(now)
		if !applied {
			t.Fatalf("Done: applied = false, want true")
		}
		if idle.LoopActive || idle.SessionsCompleted != 0 {
			t.Fatalf("Done: LoopActive=%v SessionsCompleted=%d, want false/0", idle.LoopActive, idle.SessionsCompleted)
		}
	})
}

func TestStartStopLoop(t *testing.T) {
	t.Run("start arms an inactive loop and resets the counter", func(t *testing.T) {
		start := State{Phase: Focus, SessionsCompleted: 7}

		got, applied := start.StartLoop()

		if !applied {
			t.Fatalf("applied = false, want true")
		}
		if !got.LoopActive || got.SessionsCompleted != 0 {
			t.Fatalf("LoopActive=%v SessionsCompleted=%d, want true/0", got.LoopActive, got.SessionsCompleted)
		}
	})

	t.Run("start while already active is a no-op", func(t *testing.T) {
		start := State{Phase: Focus, LoopActive: true, SessionsCompleted: 2}

		got, applied := start.StartLoop()

		if applied {
			t.Fatalf("applied = true, want false")
		}
		if got != start {
			t.Fatalf("no-op mutated state: got %+v, want unchanged %+v", got, start)
		}
	})

	t.Run("stop disarms an active loop without touching the session", func(t *testing.T) {
		start := State{Phase: Focus, Since: now, LoopActive: true, SessionsCompleted: 2}

		got, applied := start.StopLoop()

		if !applied {
			t.Fatalf("applied = false, want true")
		}
		if got.LoopActive {
			t.Fatalf("LoopActive = true, want false")
		}
		if got.Phase != Focus || !got.Since.Equal(now) {
			t.Fatalf("in-progress session was touched: got %+v", got)
		}
	})

	t.Run("stop while already inactive is a no-op", func(t *testing.T) {
		start := State{Phase: Idle}

		got, applied := start.StopLoop()

		if applied {
			t.Fatalf("applied = true, want false")
		}
		if got != start {
			t.Fatalf("no-op mutated state: got %+v, want unchanged %+v", got, start)
		}
	})
}

func TestCompletedFocusSession(t *testing.T) {
	t.Run("no-op when the loop isn't active", func(t *testing.T) {
		start := State{Phase: Focus, SessionsCompleted: 3}

		got, due := start.CompletedFocusSession(4)

		if due {
			t.Fatalf("due = true, want false")
		}
		if got != start {
			t.Fatalf("no-op mutated state: got %+v, want unchanged %+v", got, start)
		}
	})

	t.Run("increments and signals on every Nth session", func(t *testing.T) {
		s := State{LoopActive: true}
		var due bool

		for i := 1; i <= 4; i++ {
			s, due = s.CompletedFocusSession(4)
			if s.SessionsCompleted != i {
				t.Fatalf("after session %d: SessionsCompleted = %d, want %d", i, s.SessionsCompleted, i)
			}
			wantDue := i == 4
			if due != wantDue {
				t.Fatalf("after session %d: due = %v, want %v", i, due, wantDue)
			}
		}

		// The cycle repeats: the 8th session is due again, the 5th-7th aren't.
		for i := 5; i <= 8; i++ {
			s, due = s.CompletedFocusSession(4)
			wantDue := i == 8
			if due != wantDue {
				t.Fatalf("after session %d: due = %v, want %v", i, due, wantDue)
			}
		}
	})
}
