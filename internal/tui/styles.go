package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type styles struct {
	header  lipgloss.Style
	subtle  lipgloss.Style
	body    lipgloss.Style
	welcome lipgloss.Style
	footer  lipgloss.Style
}

func defaultStyles() styles {
	palette := opencodePalette()

	return styles{
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(palette.fg).
			Background(palette.accent).
			Padding(0, 2),

		subtle: lipgloss.NewStyle().
			Foreground(palette.muted).
			Italic(true).
			MarginTop(1),

		body: lipgloss.NewStyle().
			Foreground(palette.fg).
			Padding(2, 4),

		welcome: lipgloss.NewStyle().
			Foreground(palette.fg).
			MarginTop(2).
			Bold(true),

		footer: lipgloss.NewStyle().
			Foreground(palette.muted).
			MarginTop(2),
	}
}

type palette struct {
	bg     color.Color
	fg     color.Color
	muted  color.Color
	accent color.Color
}

func opencodePalette() palette {
	return palette{
		bg:     lipgloss.Color("#0d1117"),
		fg:     lipgloss.Color("#e6edf3"),
		muted:  lipgloss.Color("#7d8590"),
		accent: lipgloss.Color("#1f6feb"),
	}
}
