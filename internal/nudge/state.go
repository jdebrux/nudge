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
	Phase             Phase      `json:"phase"`
	Since             time.Time  `json:"since"`
	Until             *time.Time `json:"until,omitempty"`    // nil means open-ended
	NextCue           *time.Time `json:"next_cue,omitempty"` // when the next bell fires; starts equal to Until, moved forward by Later
	LoopActive        bool       `json:"loop_active,omitempty"`
	SessionsCompleted int        `json:"sessions_completed,omitempty"` // completed focus sessions since the loop started
}

// Overdue reports whether a timed phase has run past its end without
// anything acting on it yet. IDLE and open-ended sessions (Until == nil)
// are never overdue — there's no deadline to have missed.
func (s State) Overdue(now time.Time) bool {
	return s.Until != nil && now.After(*s.Until)
}

// In begins a focus period, from IDLE or REST — one verb covers both,
// since the state already determines which applies. Calling In while
// already in FOCUS is a no-op.
func (s State) In(now time.Time, duration time.Duration) (State, bool) {
	if s.Phase == Focus {
		return s, false
	}
	return s.transitionTo(Focus, now, duration), true
}

// Out begins a break. Only valid from FOCUS; calling Out from IDLE or REST
// is a no-op.
func (s State) Out(now time.Time, duration time.Duration) (State, bool) {
	if s.Phase != Focus {
		return s, false
	}
	return s.transitionTo(Rest, now, duration), true
}

// Done stops tracking entirely, returning to IDLE from either FOCUS or
// REST — including disarming an active loop, per PRODUCT.md §4: "nudge
// done stops tracking altogether for the day... regardless of whether a
// loop was running." Calling Done while already IDLE is a no-op.
func (s State) Done(now time.Time) (State, bool) {
	if s.Phase == Idle {
		return s, false
	}
	next := s.transitionTo(Idle, now, 0)
	next.LoopActive = false
	next.SessionsCompleted = 0
	return next, true
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

// StartLoop arms the repeating rhythm: subsequent focus sessions are
// counted, and CompletedFocusSession signals when a long break is due.
// It never touches the current phase/session — mirrors StopLoop's same
// guarantee (PRODUCT.md §4: "nudge loop stop ends a repeating rhythm but
// nudge may still be tracking a single in-progress focus/rest period").
// Calling StartLoop while already active is a no-op.
func (s State) StartLoop() (State, bool) {
	if s.LoopActive {
		return s, false
	}
	s.LoopActive = true
	s.SessionsCompleted = 0
	return s, true
}

// StopLoop disarms the loop without touching the current phase/session.
// Calling StopLoop while not active is a no-op.
func (s State) StopLoop() (State, bool) {
	if !s.LoopActive {
		return s, false
	}
	s.LoopActive = false
	return s, true
}

// CompletedFocusSession records that a focus session just ended and
// reports whether this was the Nth one (per sessionsPerLongBreak) — the
// signal to use a long break instead of a short one. A no-op (false,
// counter untouched) when the loop isn't active.
func (s State) CompletedFocusSession(sessionsPerLongBreak int) (State, bool) {
	if !s.LoopActive {
		return s, false
	}
	s.SessionsCompleted++
	dueForLongBreak := s.SessionsCompleted%sessionsPerLongBreak == 0
	return s, dueForLongBreak
}

// transitionTo builds the state for entering phase at since, carrying
// loop bookkeeping forward from s (only Done clears it — see above) and
// setting Until/NextCue for a timed session (duration > 0).
func (s State) transitionTo(phase Phase, since time.Time, duration time.Duration) State {
	next := State{
		Phase:             phase,
		Since:             since,
		LoopActive:        s.LoopActive,
		SessionsCompleted: s.SessionsCompleted,
	}
	if duration > 0 {
		until := since.Add(duration)
		next.Until = &until
		nextCue := until
		next.NextCue = &nextCue
	}
	return next
}
