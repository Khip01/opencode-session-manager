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

	list         list.Model
	chatViewport viewport.Model
	picker       filepicker.Model
	help         help.Model

	width    int
	height   int
	ready    bool
	loading  bool
	quitting bool
	err      error
	status   string

	mode  modeID
	modal modalState

	mouseX int
	mouseY int

	chatMessages        []db.Message
	chatLoadedSessionID string

	leftTopH, leftBotH   int
	rightTopH, rightBotH int
}

func newModel(opts Options, handle *sql.DB) *model {
	items := make([]list.Item, 0)
	l := list.New(items, list.NewDefaultDelegate(), 40, 20)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetShowFilter(false)

	cv := viewport.New(viewport.WithWidth(40), viewport.WithHeight(10))
	cv.SetContent("")

	fp := filepicker.New()
	fp.AllowedTypes = []string{}
	fp.ShowHidden = false
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.SetHeight(16)

	h := help.New()
	palette := opencodePalette()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(palette.accent)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(palette.fg)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(palette.muted)
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(palette.accent)
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(palette.fg)
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(palette.muted)

	return &model{
		options:      opts,
		styles:       defaultStyles(),
		keys:         defaultKeyMap(),
		db:           handle,
		loader:       newDataLoader(handle),
		relinker:     relinker.New(opts.DBPath),
		tab:          tabOrphans,
		list:         l,
		chatViewport: cv,
		picker:       fp,
		help:         h,
		loading:      true,
		status:       "loading...",
		mode:         modeNone,
		modal:        newModalState(),
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

func tuiView(s string) tea.View {
	v := tea.NewView(s)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func makePanel(w, h int, borderColor string, _ string, content string) string {
	clipped := lipgloss.NewStyle().MaxHeight(h - 2).Render(content)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Width(w).
		Height(h).
		Render(clipped)
}

func makePanelChat(w, h int, borderColor string, _ string, content string) string {
	wrapped := lipgloss.NewStyle().Width(w - 4).Render(content)
	clipped := lipgloss.NewStyle().MaxHeight(h - 4).Render(wrapped)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		PaddingTop(1).
		PaddingBottom(1).
		PaddingLeft(2).
		Width(w).
		Height(h).
		Render(clipped)
}

func makePanelPad(w, h int, borderColor string, _ string, content string) string {
	wrapped := lipgloss.NewStyle().Width(w - 6).Render(content)
	clipped := lipgloss.NewStyle().MaxHeight(h - 4).Render(wrapped)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2).
		Width(w).
		Height(h).
		Render(clipped)
}

func (m model) renderBody() string {
	if m.width <= 0 || !m.ready {
		return m.styles.subtle.Render("waiting...")
	}
	availH := m.height - 1
	contentW := m.width / 2
	bc := "#484848"

	leftBotH := 7
	leftTopH := availH - leftBotH
	if leftTopH < 6 {
		leftTopH = 6
		leftBotH = availH - leftTopH
	}

	rightTopH := int(float64(availH) * 0.40)
	rightBotH := availH - rightTopH
	if rightTopH < 8 {
		rightTopH = 8
		rightBotH = availH - rightTopH
	}
	if rightBotH < 6 {
		rightBotH = 6
		rightTopH = availH - rightBotH
	}

	bg := "#0a0a0a"
	leftTop := makePanelPad(contentW, leftTopH, bc, bg, m.list.View())
	leftBot := makePanelPad(contentW, leftBotH, bc, bg,
		m.help.FullHelpView(m.keys.listFullHelp()))

	si, _ := selectedItem(m.list)
	rightTop := makePanelPad(contentW, rightTopH, bc, bg,
		renderMetadataPanel(si, m.styles))

	chatVp := m.chatViewport.View()
	scrollbar := renderScrollbar(m.chatViewport.Height(), m.chatViewport.ScrollPercent())
	chatScrollV := lipgloss.JoinHorizontal(lipgloss.Top, chatVp, scrollbar)
	chatTitle := m.styles.detailHeader.Render("Chat Preview")
	chatBody := chatTitle + "\n" + chatScrollV
	rightBot := makePanelChat(contentW, rightBotH, bc, bg, chatBody)

	leftCol := lipgloss.JoinVertical(lipgloss.Top, leftTop, leftBot)
	rightCol := lipgloss.JoinVertical(lipgloss.Top, rightTop, rightBot)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
}

func (m model) renderHeader() string {
	left := m.styles.header.Render(" opencode-sm " + m.options.Version + " ")
	right := m.styles.subtle.Render("DB: " + m.options.DBPath)
	if m.width <= 0 {
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	rightPlaced := lipgloss.PlaceHorizontal(
		m.width-lipgloss.Width(left)-2,
		lipgloss.Right,
		right,
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, rightPlaced)
}

func (m *model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	if !m.ready {
		return tuiView(m.styles.subtle.Render("starting..."))
	}
	availH := m.height - 1
	leftTopH := availH - 7
	if leftTopH < 6 {
		leftTopH = 6
	}
	lw := m.width / 2
	m.list.SetWidth(lw - 4)
	m.list.SetHeight(leftTopH - 4)

	full := lipgloss.JoinVertical(lipgloss.Top, m.renderHeader(), m.renderBody())
	switch m.mode {
	case modeRelinkChoice:
		return tuiView(m.overlayModal(full, m.modal.renderChoice()))
	case modeConfirm:
		return tuiView(m.overlayModal(full, m.modal.renderConfirm()))
	case modeRunningWarn:
		return tuiView(m.overlayModal(full, m.modal.renderRunningWarn()))
	case modeResult:
		return tuiView(m.overlayModal(full, m.modal.renderResult()))
	case modeFilepicker:
		return tuiView(m.overlayModal(full, m.picker.View()))
	}
	return tuiView(full)
}

func (m *model) recomputeLayout() {
	if m.width > 0 && m.height > 1 {
		m.ready = true
	}
}

func (m model) overlayModal(base, content string) string {
	modalWidth := m.width * 2 / 3
	if modalWidth < 50 {
		modalWidth = 50
	}
	if modalWidth > m.width-4 {
		modalWidth = m.width - 4
	}
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		m.styles.modalBox.Width(modalWidth).Render(content)) +
		"\n" + m.styles.dimmed.Render(base)
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


