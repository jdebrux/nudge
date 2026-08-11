// Package nudge implements the IDLE/FOCUS/REST rhythm state machine at the
// core of nudge. It is pure domain logic: no file I/O, no CLI, no rendering.
package nudge

import "time"

// Phase is one of the three states nudge tracks.
type Phase int

const (
	Idle Phase = iota
	Focus
	Rest
)

// State is the current rhythm state, persisted between invocations.
type State struct {
	Phase Phase
	Since time.Time
	Until *time.Time // nil means open-ended
}

// In begins a focus period, from IDLE or REST — one verb covers both,
// since the state already determines which applies. Calling In while
// already in FOCUS is a no-op.
func (s State) In(now time.Time, duration time.Duration) (State, bool) {
	if s.Phase == Focus {
		return s, false
	}
	return newState(Focus, now, duration), true
}

// Out begins a break. Only valid from FOCUS; calling Out from IDLE or REST
// is a no-op.
func (s State) Out(now time.Time, duration time.Duration) (State, bool) {
	if s.Phase != Focus {
		return s, false
	}
	return newState(Rest, now, duration), true
}

// Done stops tracking entirely, returning to IDLE from either FOCUS or
// REST. Calling Done while already IDLE is a no-op.
func (s State) Done(now time.Time) (State, bool) {
	if s.Phase == Idle {
		return s, false
	}
	return newState(Idle, now, 0), true
}

func newState(phase Phase, since time.Time, duration time.Duration) State {
	s := State{Phase: phase, Since: since}
	if duration > 0 {
		until := since.Add(duration)
		s.Until = &until
	}
	return s
}
