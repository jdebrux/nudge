package render

import "github.com/charmbracelet/lipgloss"

// primaryColor is nudge's brand colour — used everywhere, for both
// phases alike, rather than differentiating focus/rest by hue. Fixed
// rather than adaptive: it's a specific brand blue, not a value meant
// to shift with terminal theme.
var primaryColor = lipgloss.Color("#1B4EF5")

var (
	dimColor = lipgloss.AdaptiveColor{Light: "#78716C", Dark: "#A8A29E"}

	labelStyle = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	hintStyle  = lipgloss.NewStyle().Foreground(dimColor)
)

const barWidth = 22

// bayerRow is a 4x4 ordered-dither (Bayer) matrix, flattened row-major.
// A one-row terminal bar has no second dimension to dither across, so
// this is the natural reduction of nudge's dithered art style down to
// one dimension: a fixed, repeating spatial threshold pattern rather
// than a flat fill.
var bayerRow = [16]int{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}

// ditherThreshold sets the stipple density of the bar's unfilled
// portion (out of 16) — how many of the 16 Bayer cells render a dot.
const ditherThreshold = 6

// bar renders a static progress bar for the given fraction (0..1) — a
// one-shot render, not an animated component, since nudge never occupies
// the terminal (PRODUCT.md §7/§5: every command returns immediately).
// The elapsed portion is solid; the remaining portion is dithered
// rather than a flat fill, echoing the reference art's ordered-dither
// texture — solid where a session has committed its time, stippled
// where it hasn't yet.
func bar(fraction float64) string {
	switch {
	case fraction < 0:
		fraction = 0
	case fraction > 1:
		fraction = 1
	}
	filled := int(fraction * float64(barWidth))

	runes := make([]rune, barWidth)
	for i := range runes {
		switch {
		case i < filled:
			runes[i] = '█'
		case bayerRow[i%len(bayerRow)] < ditherThreshold:
			runes[i] = '░'
		default:
			runes[i] = ' '
		}
	}
	return lipgloss.NewStyle().Foreground(primaryColor).Render(string(runes))
}
