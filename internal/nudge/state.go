// Package nudge implements the IDLE/FOCUS/REST rhythm state machine at the
// core of nudge. It is pure domain logic: no file I/O, no CLI, no rendering.
package nudge

import (
	"encoding/json"
	"fmt"
	"time"
)

// Phase is one of the three states nudge tracks.
type Phase int

const (
	Idle Phase = iota
	Focus
	Rest
)

// String returns the lowercase name used both for display and as the
// persisted JSON representation, so a state file reads as "focus" rather
// than an opaque integer.
func (p Phase) String() string {
	switch p {
	case Focus:
		return "focus"
	case Rest:
		return "rest"
	default:
		return "idle"
	}
}

// MarshalJSON encodes Phase as its lowercase name.
func (p Phase) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// UnmarshalJSON decodes Phase from its lowercase name.
func (p *Phase) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "idle":
		*p = Idle
	case "focus":
		*p = Focus
	case "rest":
		*p = Rest
	default:
		return fmt.Errorf("nudge: unknown phase %q", s)
	}
	return nil
}

// State is the current rhythm state, persisted between invocations.
type State struct {
	Phase   Phase      `json:"phase"`
	Since   time.Time  `json:"since"`
	Until   *time.Time `json:"until,omitempty"`    // nil means open-ended
	NextCue *time.Time `json:"next_cue,omitempty"` // when the next bell fires; starts equal to Until, moved forward by Later
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

// Later postpones the next cue by delay, without changing the current
// phase — Since and Until are untouched, only NextCue moves. This is a
// "not yet, ask again shortly" reaction to a cue, not a state
// transition. Calling Later with no timed cue pending (IDLE, or an
// open-ended session) is a no-op.
func (s State) Later(now time.Time, delay time.Duration) (State, bool) {
	if s.Phase == Idle || s.NextCue == nil {
		return s, false
	}
	next := now.Add(delay)
	s.NextCue = &next
	return s, true
}

func newState(phase Phase, since time.Time, duration time.Duration) State {
	s := State{Phase: phase, Since: since}
	if duration > 0 {
		until := since.Add(duration)
		s.Until = &until
		nextCue := until
		s.NextCue = &nextCue
	}
	return s
}
