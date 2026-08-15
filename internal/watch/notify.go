package watch

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// fire delivers one cue firing: a terminal bell (unchanged since Phase
// 4 — the only signal that works over SSH/tmux) plus a best-effort
// native OS notification, which is what actually makes the cue visible
// regardless of which app has focus or whether a terminal is even open.
func fire(phase string) {
	fmt.Fprint(os.Stdout, "\a")
	title, message := notificationText(phase)
	sendNotification(title, message)
}

// sendNotification delivers a native notification. It's a package
// variable, not a plain function, so tests can swap it out — without
// that, running `go test` on a Mac would pop a real notification for
// every test case that exercises a firing. The real implementation
// (below) is macOS-only (osascript, no new dependency); on any other
// platform it's a silent no-op rather than attempting and failing —
// the bell already fired regardless.
var sendNotification = func(title, message string) {
	if runtime.GOOS != "darwin" {
		return
	}
	script := fmt.Sprintf(`display notification %s with title %s`, appleScriptString(message), appleScriptString(title))
	_ = exec.Command("osascript", "-e", script).Run()
}

func notificationText(phase string) (title, message string) {
	switch phase {
	case "focus":
		return "nudge", "focus session up — time for a break"
	case "rest":
		return "nudge", "break's over — back to focus when ready"
	default:
		return "nudge", "time to check in"
	}
}

// appleScriptString quotes s as an AppleScript string literal for use
// inside an `osascript -e` argument. This is AppleScript's own quoting,
// not shell quoting — exec.Command never runs args through a shell, so
// there's no shell-injection surface either way.
func appleScriptString(s string) string {
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
