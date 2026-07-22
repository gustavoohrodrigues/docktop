package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

func spark(values []float64, width int, color lipgloss.Color) string {
	if width < 1 {
		return ""
	}
	start := max(0, len(values)-width)
	values = values[start:]
	maxV := 1.0
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	var b strings.Builder
	for range width - len(values) {
		b.WriteRune('·')
	}
	for _, v := range values {
		i := int(math.Round(v / maxV * float64(len(sparkRunes)-1)))
		i = min(max(i, 0), len(sparkRunes)-1)
		b.WriteRune(sparkRunes[i])
	}
	return lipgloss.NewStyle().Foreground(color).Render(b.String())
}
func meter(value float64, width int, color lipgloss.Color) string {
	value = max(0, min(value, 100))
	filled := int(value / 100 * float64(width))
	return lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("━", filled)) + lipgloss.NewStyle().Foreground(lipgloss.Color("#263247")).Render(strings.Repeat("─", width-filled))
}
