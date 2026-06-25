package tui

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Khip01/opencode-session-manager/internal/db"
	"github.com/Khip01/opencode-session-manager/internal/relinker"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
)

type Options struct {
	DBPath  string
	Version string
}

type sessionsLoadedMsg struct {
	orphans []sessionItem
	active  []sessionItem
	err     error
}

type tabID int

const (
	tabOrphans tabID = iota
	tabActive
)

func (t tabID) Label() string {
	switch t {
	case tabOrphans:
		return "Orphans"
	case tabActive:
		return "Active"
	default:
		return "Unknown"
	}
}

type model struct {
	options  Options
	styles   styles
	keys     keyMap
	db       *sql.DB
	loader   *dataLoader
	relinker *relinker.Relinker

	tab tabID

	orphans []sessionItem
	active  []sessionItem

	list   list.Model
	detail viewport.Model
	picker filepicker.Model
	help   help.Model

	width    int
	height   int
	ready    bool
	loading  bool
	quitting bool
	err      error
	status   string

	mode  modeID
	modal modalState
}

func newModel(opts Options, handle *sql.DB) model {
	km := defaultKeyMap()
	l := newList(nil, 40, 20)
	d := newDetail(40, 20)
	fp := newDirPicker()
	h := help.New()

	return model{
		options:  opts,
		styles:   defaultStyles(),
		keys:     km,
		db:       handle,
		loader:   newDataLoader(handle),
		relinker: relinker.New(opts.DBPath),
		tab:      tabOrphans,
		list:     l,
		detail:   d,
		picker:   fp,
		help:     h,
		loading:  true,
		status:   "loading…",
		mode:     modeNone,
		modal:    newModalState(),
	}
}

func (m model) Init() tea.Cmd {
	return loadSessionsCmd(m.loader)
}

func loadSessionsCmd(loader *dataLoader) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		orphans, active, err := loader.Load(ctx)
		return sessionsLoadedMsg{orphans: orphans, active: active, err: err}
	}
}

func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("Bye!")
	}

	base := m.renderBase()

	switch m.mode {
	case modeRelinkChoice:
		return tea.NewView(m.overlayModal(base, m.modal.renderChoice()))
	case modeConfirm:
		return tea.NewView(m.overlayModal(base, m.modal.renderConfirm()))
	case modeRunningWarn:
		return tea.NewView(m.overlayModal(base, m.modal.renderRunningWarn()))
	case modeResult:
		return tea.NewView(m.overlayModal(base, m.modal.renderResult()))
	case modeFilepicker:
		return tea.NewView(m.overlayModal(base, m.picker.View()))
	}

	v := tea.NewView(base)
	v.AltScreen = true
	return v
}

func (m model) renderBase() string {
	if !m.ready {
		return m.styles.subtle.Render("starting…")
	}
	header := m.renderHeader()
	tabs := m.renderTabs()
	body := m.renderBody()
	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, body, footer)
}

func (m model) overlayModal(base, content string) string {
	modalWidth := m.width * 2 / 3
	if modalWidth < 50 {
		modalWidth = 50
	}
	if modalWidth > m.width-4 {
		modalWidth = m.width - 4
	}

	modalStyle := m.styles.modalBox.Width(modalWidth)
	modal := modalStyle.Render(content)
	dimmed := m.styles.dimmed.Render(base)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal) + "\n" + dimmed
}

func (m model) renderHeader() string {
	left := m.styles.header.Render(" opencode-sm " + m.options.Version + " ")
	right := m.styles.subtle.Render("DB: " + m.options.DBPath)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m model) renderTabs() string {
	tabs := []tabID{tabOrphans, tabActive}
	parts := make([]string, len(tabs))
	for i, t := range tabs {
		label := t.Label()
		switch t {
		case tabOrphans:
			label += " (" + itoa(len(m.orphans)) + ")"
		case tabActive:
			label += " (" + itoa(len(m.active)) + ")"
		}
		if t == m.tab {
			parts[i] = m.styles.tabActive.Render(" " + label + " ")
		} else {
			parts[i] = m.styles.tabInactive.Render(" " + label + " ")
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m model) renderBody() string {
	listWidth := m.width / 3
	if listWidth < 28 {
		listWidth = 28
	}
	detailWidth := m.width - listWidth - 4
	if detailWidth < 28 {
		detailWidth = 28
	}
	bodyHeight := m.height - 6
	if bodyHeight < 6 {
		bodyHeight = 6
	}

	m.list.SetWidth(listWidth)
	m.list.SetHeight(bodyHeight)
	m.detail.SetWidth(detailWidth)
	m.detail.SetHeight(bodyHeight)

	left := m.styles.listPanel.
		Width(listWidth).
		Height(bodyHeight).
		Render(m.list.View())

	right := m.styles.detailPanel.
		Width(detailWidth).
		Height(bodyHeight).
		Render(m.detail.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m model) renderFooter() string {
	status := m.status
	if status == "" {
		status = m.styles.subtle.Render("ready — r: relink, m: manual remap, tab: switch, q: quit")
	}
	helpView := m.help.ShortHelpView(m.keys.listShortHelp())
	bar := m.styles.helpBar.Width(m.width).Render(helpView)
	return status + "\n" + bar
}

func (m *model) recomputeLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	m.ready = true
}

func Run(opts Options) error {
	handle, err := db.Open(opts.DBPath)
	if err != nil {
		return err
	}
	defer handle.Close()

	p := tea.NewProgram(newModel(opts, handle))
	_, err = p.Run()
	if err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return err
	}
	return nil
}
