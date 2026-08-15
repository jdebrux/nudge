package render

// LoopStarted is the echo for a successful `nudge loop start`.
func LoopStarted() string {
	return hintStyle.Render("loop started")
}

// LoopAlreadyActive is the no-op echo for `nudge loop start` while a
// loop is already running.
func LoopAlreadyActive() string {
	return hintStyle.Render("loop already running")
}

// LoopStopped is the echo for a successful `nudge loop stop`.
func LoopStopped() string {
	return hintStyle.Render("loop stopped")
}

// LoopAlreadyInactive is the no-op echo for `nudge loop stop` while no
// loop is running.
func LoopAlreadyInactive() string {
	return hintStyle.Render("loop wasn't running")
}
