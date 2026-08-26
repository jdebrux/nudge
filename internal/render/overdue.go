package render

import "github.com/jdebrux/nudge/internal/nudge"

// OverduePromptTitle is the picker title shown by bare `nudge` when a
// timed session has run past its end (see nudge.State.Overdue).
func OverduePromptTitle(phase nudge.Phase) string {
	if phase == nudge.Rest {
		return "break's over — what next?"
	}
	return "focus session done — what next?"
}

// OverduePromptPrimaryLabel is the phase-appropriate first option in
// that picker — the natural next step the state machine actually
// supports (out from FOCUS, in from REST).
func OverduePromptPrimaryLabel(phase nudge.Phase) string {
	if phase == nudge.Rest {
		return "back to focus"
	}
	return "take a break"
}

// StopForNowLabel is the picker's other option, valid from either
// phase.
func StopForNowLabel() string {
	return "stop for now"
}
