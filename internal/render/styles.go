package render

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// primaryColor is nudge's brand colour — used everywhere, for both
// phases alike, rather than differentiating focus/rest by hue. Fixed
// rather than adaptive: it's a specific brand blue, not a value meant
// to shift with terminal theme.
var primaryColor = lipgloss.Color("#1B4EF5")

var (
	dimColor = lipgloss.AdaptiveColor{Light: "#78716C", Dark: "#A8A29E"}

	labelStyle = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	valueStyle = lipgloss.NewStyle().Foreground(primaryColor)
	hintStyle  = lipgloss.NewStyle().Foreground(dimColor)
)

const barWidth = 22

// bar renders a static progress bar for the given fraction (0..1) — a
// one-shot render, not an animated component, since nudge never occupies
// the terminal (PRODUCT.md §7/§5: every command returns immediately).
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
		if i < filled {
			runes[i] = '█'
		} else {
			runes[i] = '░'
		}
	}
	return lipgloss.NewStyle().Foreground(primaryColor).Render(string(runes)) + fmt.Sprintf(" %d%%", int(fraction*100))
}
