package render

import (
	"fmt"
	"strings"

	"github.com/jdebrux/nudge/internal/config"
)

// ConfigSummary renders the current defaults, as shown by bare
// `nudge config`. Labels are dim, values carry the brand colour — the
// same label/value hierarchy as the help listing.
func ConfigSummary(c config.Config) string {
	rows := []struct{ label, value string }{
		{"focus", c.Focus.String()},
		{"break", c.Break.String()},
		{"long break", c.LongBreak.String()},
		{"every", fmt.Sprintf("%d sessions", c.SessionsPerLongBreak)},
	}

	width := 0
	for _, r := range rows {
		if len(r.label) > width {
			width = len(r.label)
		}
	}

	lines := []string{""}
	for _, r := range rows {
		label := hintStyle.Render(r.label + strings.Repeat(" ", width-len(r.label)+2))
		lines = append(lines, "  "+label+valueStyle.Render(r.value))
	}
	return strings.Join(lines, "\n")
}

// ConfigUpdated is the echo for a successful `nudge config set`.
func ConfigUpdated(key, value string) string {
	return hintStyle.Render(fmt.Sprintf("%s set to %s", key, value))
}
