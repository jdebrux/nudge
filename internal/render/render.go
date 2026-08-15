// Package render turns nudge.State into the small, restrained text nudge
// prints — never a dashboard (PRODUCT.md §7). Every function here is pure:
// given a state and a clock reading, it returns a string. No I/O.
package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/jdebrux/nudge/internal/nudge"
)

// Status renders the current rhythm state, as shown by bare `nudge` and
// by `nudge status`.
func Status(s nudge.State, now time.Time) string {
	switch s.Phase {
	case nudge.Focus:
		return block(focusLabelStyle.Render(fmt.Sprintf("focus · %s", remaining(s, now))), bar(elapsedFraction(s, now), focusColor))
	case nudge.Rest:
		return block(restLabelStyle.Render(fmt.Sprintf("break · %s", remaining(s, now))), "time to step away.")
	default:
		return block("idle", hintStyle.Render("nudge in to start"))
	}
}

// AlreadyFocused is the no-op echo for `nudge in` while already in FOCUS.
func AlreadyFocused(s nudge.State, now time.Time) string {
	return "already focused" + remainingSuffix(s, now)
}

// NotFocused is the no-op echo for `nudge out` while IDLE — you can't take
// a break from work you haven't started.
func NotFocused() string {
	return "not focused right now"
}

// AlreadyOnABreak is the no-op echo for `nudge out` while already in REST.
func AlreadyOnABreak(s nudge.State, now time.Time) string {
	return "already on a break" + remainingSuffix(s, now)
}

// AlreadyIdle is the no-op echo for `nudge done` while already IDLE.
func AlreadyIdle() string {
	return "already idle"
}

// Postponed is the echo for a successful `nudge later`.
func Postponed(s nudge.State, now time.Time) string {
	return fmt.Sprintf("nudging again in %s", clock(*s.NextCue, now))
}

// NothingToPostpone is the no-op echo for `nudge later` when there's no
// timed cue pending (IDLE, or an open-ended session).
func NothingToPostpone() string {
	return "nothing to postpone right now"
}

func block(label, detail string) string {
	return strings.Join([]string{"", "  " + label, "", "  " + detail}, "\n")
}

func remainingSuffix(s nudge.State, now time.Time) string {
	if s.Until == nil {
		return ""
	}
	return fmt.Sprintf(", %s left", clock(*s.Until, now))
}

func remaining(s nudge.State, now time.Time) string {
	if s.Until == nil {
		return clock(now, s.Since) // open-ended: count up from the start instead
	}
	return clock(*s.Until, now)
}

// clock formats the (clamped, non-negative) gap between two points in
// time as MM:SS.
func clock(a, b time.Time) string {
	d := a.Sub(b)
	if d < 0 {
		d = 0
	}
	total := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

func elapsedFraction(s nudge.State, now time.Time) float64 {
	if s.Until == nil {
		return 0
	}
	total := s.Until.Sub(s.Since)
	if total <= 0 {
		return 1
	}
	elapsed := now.Sub(s.Since)
	f := float64(elapsed) / float64(total)
	switch {
	case f < 0:
		return 0
	case f > 1:
		return 1
	default:
		return f
	}
}
