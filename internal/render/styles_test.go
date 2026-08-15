package render

import (
	"fmt"
	"strings"
	"testing"
)

// stripANSI removes lipgloss's SGR escape sequences so bar content can
// be inspected directly. Good enough for test assertions: it only needs
// to strip what lipgloss actually emits (CSI ... 'm').
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestBarShowsFractionFilledAndPercentage(t *testing.T) {
	cases := []struct {
		fraction float64
		filled   int
		percent  int
	}{
		{-1, 0, 0},
		{0, 0, 0},
		{0.5, 11, 50},
		{1, barWidth, 100},
		{2, barWidth, 100},
	}

	for _, c := range cases {
		want := strings.Repeat("█", c.filled) + strings.Repeat("░", barWidth-c.filled) + fmt.Sprintf(" %d%%", c.percent)
		got := stripANSI(bar(c.fraction))
		if got != want {
			t.Fatalf("bar(%v) = %q, want %q", c.fraction, got, want)
		}
	}
}
