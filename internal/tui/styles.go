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
	bg     color.Color
	fg     color.Color
	muted  color.Color
	accent color.Color
	green  color.Color
	red    color.Color
	yellow color.Color
	panel  color.Color
	border color.Color
}

func opencodePalette() palette {
	return palette{
		bg:     lipgloss.Color("#0d1117"),
		fg:     lipgloss.Color("#e6edf3"),
		muted:  lipgloss.Color("#7d8590"),
		accent: lipgloss.Color("#1f6feb"),
		green:  lipgloss.Color("#3fb950"),
		red:    lipgloss.Color("#f85149"),
		yellow: lipgloss.Color("#d29922"),
		panel:  lipgloss.Color("#161b22"),
		border: lipgloss.Color("#30363d"),
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
			Foreground(p.yellow),

		statusErr: lipgloss.NewStyle().
			Foreground(p.red).
			Bold(true),

		detailLabel: lipgloss.NewStyle().
			Foreground(p.muted).
			Width(14),

		detailHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent).
			MarginBottom(1),

		detailHint: lipgloss.NewStyle().
			Foreground(p.yellow).
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
			Foreground(p.muted).
			Background(p.panel).
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
