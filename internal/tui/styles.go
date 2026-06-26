package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type styles struct {
	header       lipgloss.Style
	subtle       lipgloss.Style
	footer       lipgloss.Style
	tabActive    lipgloss.Style
	tabInactive  lipgloss.Style
	statusOK     lipgloss.Style
	statusWarn   lipgloss.Style
	statusErr    lipgloss.Style
	detailLabel  lipgloss.Style
	detailHeader lipgloss.Style
	detailHint   lipgloss.Style
	detailEmpty  lipgloss.Style
	detailOK     lipgloss.Style
	detailWarn   lipgloss.Style
	listPanel    lipgloss.Style
	detailPanel  lipgloss.Style
	helpBar      lipgloss.Style
	modalBox     lipgloss.Style
	dimmed       lipgloss.Style
}

type palette struct {
	bg        color.Color
	fg        color.Color
	muted     color.Color
	accent    color.Color
	secondary color.Color
	warm      color.Color
	green     color.Color
	red       color.Color
	orange    color.Color
	cyan      color.Color
	panel     color.Color
	border    color.Color
}

func opencodePalette() palette {
	colors := map[string]string{
		"bg":        "#0a0a0a",
		"panel":     "#141414",
		"element":   "#1e1e1e",
		"fg":        "#eeeeee",
		"muted":     "#808080",
		"border":    "#484848",
		"accent":    "#5c9cf5",
		"secondary": "#9d7cd8",
		"warm":      "#fab283",
		"green":     "#7fd88f",
		"red":       "#e06c75",
		"orange":    "#f5a742",
		"cyan":      "#56b6c2",
	}
	return palette{
		bg:        lipgloss.Color(colors["bg"]),
		fg:        lipgloss.Color(colors["fg"]),
		muted:     lipgloss.Color(colors["muted"]),
		accent:    lipgloss.Color(colors["accent"]),
		secondary: lipgloss.Color(colors["secondary"]),
		warm:      lipgloss.Color(colors["warm"]),
		green:     lipgloss.Color(colors["green"]),
		red:       lipgloss.Color(colors["red"]),
		orange:    lipgloss.Color(colors["orange"]),
		cyan:      lipgloss.Color(colors["cyan"]),
		panel:     lipgloss.Color(colors["panel"]),
		border:    lipgloss.Color(colors["border"]),
	}
}

func defaultStyles() styles {
	p := opencodePalette()

	return styles{
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.fg).
			Background(p.accent).
			Padding(0, 2),

		subtle: lipgloss.NewStyle().
			Foreground(p.muted).
			Italic(true),

		footer: lipgloss.NewStyle().
			Foreground(p.muted).
			MarginTop(1),

		tabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.fg).
			Background(p.accent).
			Padding(0, 2),

		tabInactive: lipgloss.NewStyle().
			Foreground(p.muted).
			Background(p.panel).
			Padding(0, 2),

		statusOK: lipgloss.NewStyle().
			Foreground(p.green),

		statusWarn: lipgloss.NewStyle().
			Foreground(p.orange),

		detailLabel: lipgloss.NewStyle().
			Foreground(p.muted).
			Width(14),

		detailHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent).
			MarginBottom(1),

		detailHint: lipgloss.NewStyle().
			Foreground(p.orange).
			Italic(true),

		detailEmpty: lipgloss.NewStyle().
			Foreground(p.muted).
			Italic(true),

		detailOK: lipgloss.NewStyle().
			Foreground(p.green).
			Bold(true),

		detailWarn: lipgloss.NewStyle().
			Foreground(p.red).
			Bold(true),

		listPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.border),

		detailPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.border).
			Padding(1, 2),

		helpBar: lipgloss.NewStyle().
			Foreground(p.fg).
			Bold(true).
			Padding(0, 1),

		modalBox: lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(p.accent).
			Background(p.panel).
			Foreground(p.fg).
			Padding(1, 2),

		dimmed: lipgloss.NewStyle().
			Foreground(p.muted),
	}
}
