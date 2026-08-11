package render

import "github.com/charmbracelet/lipgloss"

var (
	focusColor = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#F59E0B"} // warm amber
	restColor  = lipgloss.AdaptiveColor{Light: "#0369A1", Dark: "#38BDF8"} // cool blue
	dimColor   = lipgloss.AdaptiveColor{Light: "#78716C", Dark: "#A8A29E"}

	focusLabelStyle = lipgloss.NewStyle().Foreground(focusColor).Bold(true)
	restLabelStyle  = lipgloss.NewStyle().Foreground(restColor).Bold(true)
	hintStyle       = lipgloss.NewStyle().Foreground(dimColor)
)

const barWidth = 22

// bar renders a static progress bar for the given fraction (0..1) — a
// one-shot render, not an animated component, since nudge never occupies
// the terminal (PRODUCT.md §7/§5: every command returns immediately).
func bar(fraction float64, color lipgloss.AdaptiveColor) string {
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
	return lipgloss.NewStyle().Foreground(color).Render(string(runes))
}
