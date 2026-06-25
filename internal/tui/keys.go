package tui

import (
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	NextTab   key.Binding
	PrevTab   key.Binding
	Quit      key.Binding
	ForceQuit key.Binding
	Help      key.Binding

	Relink key.Binding
	Manual key.Binding
	Migrate key.Binding
	Cancel key.Binding
	Enter  key.Binding
	Filter key.Binding

	UpDown     key.Binding
	LeftRight  key.Binding
	Yes        key.Binding
	No         key.Binding
	SelectFile key.Binding
	Back       key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab", "right", "l"),
			key.WithHelp("tab/→", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab", "left", "h"),
			key.WithHelp("⇧tab/←", "prev tab"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Relink: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "relink selected"),
		),
		Manual: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "manual remap"),
		),
		Migrate: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "migrate to project"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Yes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
		SelectFile: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "up dir"),
		),
	}
}

func (k keyMap) listShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Filter, k.Relink, k.Quit}
}

func (k keyMap) listFullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Filter},
		{k.NextTab, k.PrevTab},
		{k.Relink, k.Manual, k.Migrate},
		{k.Help, k.Quit, k.ForceQuit},
	}
}

func (k keyMap) modalShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Cancel}
}
