package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func renderScrollbar(height int, percent float64) string {
	thumbSize := height / 4
	if thumbSize < 1 {
		thumbSize = 1
	}
	if thumbSize > height-1 {
		thumbSize = height - 1
	}
	thumbStart := int(float64(height-thumbSize) * percent)

	var b strings.Builder
	blue := lipgloss.NewStyle().Foreground(lipgloss.Color("#5c9cf5"))
	gray := lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))

	for i := range height {
		if i >= thumbStart && i < thumbStart+thumbSize {
			b.WriteString(blue.Render("▐"))
		} else {
			b.WriteString(gray.Render("│"))
		}
		if i < height-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
