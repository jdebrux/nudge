package render

import (
	"fmt"
	"strings"

	"github.com/jdebrux/nudge/internal/config"
)

// ConfigSummary renders the current defaults, as shown by bare
// `nudge config`.
func ConfigSummary(c config.Config) string {
	rows := []string{
		"",
		fmt.Sprintf("  focus       %s", c.Focus),
		fmt.Sprintf("  break       %s", c.Break),
		fmt.Sprintf("  long break  %s", c.LongBreak),
		fmt.Sprintf("  every       %d sessions", c.SessionsPerLongBreak),
	}
	return strings.Join(rows, "\n")
}

// ConfigUpdated is the echo for a successful `nudge config set`.
func ConfigUpdated(key, value string) string {
	return fmt.Sprintf("%s set to %s", key, value)
}
