package render

import "strings"

// helpRows is nudge's whole command surface — deliberately short and
// flat (PRODUCT.md §4: "keep this small vocabulary set closed").
var helpRows = []struct{ cmd, desc string }{
	{"nudge", "show status, or start the default rhythm if idle"},
	{"nudge in [duration]", "begin a focus period (idle or returning from a break)"},
	{"nudge out [duration]", "begin a break"},
	{"nudge later", "postpone the next cue"},
	{"nudge done", "stop tracking entirely"},
	{"nudge status", "what state am I in, time remaining"},
	{"nudge prompt", "one-line status for embedding in your shell prompt"},
	{"nudge loop start|stop", "manage a repeating rhythm"},
	{"nudge config", "show current defaults"},
	{"nudge config set <key> <value>", "set a default (focus, break, long-break, every, repeat, repeat-limit)"},
	{"nudge await -- <cmd>", "run a command, return to focus when it exits"},
	{"nudge help", "show this"},
}

// Help lists every command, as shown by `nudge help`/`-h`/`--help`.
func Help() string {
	width := 0
	for _, r := range helpRows {
		if len(r.cmd) > width {
			width = len(r.cmd)
		}
	}

	lines := []string{""}
	for _, r := range helpRows {
		cmd := valueStyle.Render(r.cmd + strings.Repeat(" ", width-len(r.cmd)+2))
		lines = append(lines, "  "+cmd+hintStyle.Render(r.desc))
	}
	return strings.Join(lines, "\n")
}
