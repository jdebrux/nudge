package render

// LoopStarted is the echo for a successful `nudge loop start`.
func LoopStarted() string {
	return "loop started"
}

// LoopAlreadyActive is the no-op echo for `nudge loop start` while a
// loop is already running.
func LoopAlreadyActive() string {
	return "loop already running"
}

// LoopStopped is the echo for a successful `nudge loop stop`.
func LoopStopped() string {
	return "loop stopped"
}

// LoopAlreadyInactive is the no-op echo for `nudge loop stop` while no
// loop is running.
func LoopAlreadyInactive() string {
	return "loop wasn't running"
}
