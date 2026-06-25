package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Options struct {
	DBPath  string
	Version string
}

type tickMsg struct{}

type model struct {
	options Options
	styles  styles
	ready   bool
}

func newModel(opts Options) model {
	return model{
		options: opts,
		styles:  defaultStyles(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	if !m.ready {
		return tea.NewView("")
	}

	header := m.styles.header.Render(
		fmt.Sprintf("opencode-sm %s", m.options.Version),
	)
	dbInfo := m.styles.subtle.Render(fmt.Sprintf("DB: %s", m.options.DBPath))

	body := m.styles.body.Render(
		fmt.Sprintf(
			"%s\n%s\n\n%s\n\n%s",
			header,
			dbInfo,
			m.styles.welcome.Render("Welcome. Session browser is coming in Milestone 2."),
			m.styles.footer.Render("Press q to quit"),
		),
	)

	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func Run(opts Options) error {
	p := tea.NewProgram(newModel(opts))
	_, err := p.Run()
	return err
}

var _ = lipgloss.NewStyle
