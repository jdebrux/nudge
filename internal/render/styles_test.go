package render

import (
	"strings"
	"testing"
)

// visibleLen counts runes, not bytes — the bar is styled (ANSI escape
// codes may wrap it), so len() on the raw string isn't reliable.
func visibleLen(s string) int {
	return len([]rune(stripANSI(s)))
}

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

func TestBarWidthIsConstant(t *testing.T) {
	for _, f := range []float64{-1, 0, 0.01, 0.5, 0.99, 1, 2} {
		got := stripANSI(bar(f))
		if n := visibleLen(got); n != barWidth {
			t.Fatalf("bar(%v) width = %d, want %d (content: %q)", f, n, barWidth, got)
		}
	}
}

func TestBarFullyElapsedIsAllSolid(t *testing.T) {
	got := stripANSI(bar(1))
	if strings.ContainsAny(got, "░ ") {
		t.Fatalf("bar(1) = %q, want no dithered or blank cells", got)
	}
	if strings.Count(got, "█") != barWidth {
		t.Fatalf("bar(1) = %q, want all %d cells solid", got, barWidth)
	}
}

func TestBarUnelapsedIsDitheredNotSolid(t *testing.T) {
	got := stripANSI(bar(0))
	if strings.Contains(got, "█") {
		t.Fatalf("bar(0) = %q, want no solid cells", got)
	}
	if !strings.Contains(got, "░") {
		t.Fatalf("bar(0) = %q, want a dithered stipple, not a flat fill", got)
	}
}

func TestBarPartialHasAllThreeCellKinds(t *testing.T) {
	// A wide-enough partial fraction should show solid (elapsed),
	// dithered (stippled remainder), and blank (remainder gaps) cells —
	// confirms the bar isn't just a two-tone hard split.
	got := stripANSI(bar(0.4))
	if !strings.Contains(got, "█") {
		t.Fatalf("bar(0.4) = %q, want solid cells", got)
	}
	if !strings.Contains(got, "░") {
		t.Fatalf("bar(0.4) = %q, want dithered cells", got)
	}
	if !strings.Contains(got, " ") {
		t.Fatalf("bar(0.4) = %q, want blank cells", got)
	}
}
