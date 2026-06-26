// Package tui implements the Bubble Tea-based interactive terminal UI
// for opencode-sm.
//
// Layout (full terminal window):
//
//   +----------------------------------------------------+
//   | header: version badge + DB path                    |  1 row
//   +----------------------------------------------------+
//   |                       |                            |
//   |   LEFT COLUMN         |   RIGHT COLUMN             |
//   |   (sessions + hints)  |   (meta + chat preview)    |
//   |                       |                            |
//   |   +---------------+   |   +--------------------+   |
//   |   | sessions      |   |   | session metadata |   |  ~30% right col
//   |   | list (80%)    |   |   | (top)            |   |
//   |   |               |   |   +--------------------+   |
//   |   +---------------+   |   | chat preview      |   |  ~70% right col
//   |   | hints (20%)    |   |   | (scrollable)     |   |  (viewport)
//   |   +---------------+   |   +--------------------+   |
//   |                       |                            |
//   +----------------------------------------------------+
//
// The chat preview panel uses a viewport so long messages can be
// scrolled with mouse wheel or arrow keys. The hint panel at the
// bottom-left replaces the old global footer so keybinding hints
// are always visible without scrolling.
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
	Watch   bool
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


// Chat preview display limits are defined in detail_view.go where
// the rendering helpers live.


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

	// Mouse tracking for hover-based scroll routing. Updated on
	// mouse-motion events; consumed when mouse-wheel events arrive to
	// decide which panel should receive the scroll.
	mouseX int
	mouseY int

	watching bool

	chatMessages        []db.Message
	chatLoadedSessionID string
}

func newModel(opts Options, handle *sql.DB) *model {
	km := defaultKeyMap()
	l := newList(nil, 40, 20)
	cv := viewport.New(viewport.WithWidth(40), viewport.WithHeight(10))
	fp := newDirPicker()
	h := help.New()

	palette := opencodePalette()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(palette.accent)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(palette.fg)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(palette.muted)

	return &model{
		options:     opts,
		styles:      defaultStyles(),
		keys:        km,
		db:          handle,
		loader:      newDataLoader(handle),
		relinker:    relinker.New(opts.DBPath),
		tab:         tabOrphans,
		list:        l,
		chatViewport: cv,
		picker:      fp,
		help:        h,
		loading:     true,
		status:      "loading…",
		mode:        modeNone,
		modal:       newModalState(),
		watching:    opts.Watch,
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{loadSessionsCmd(m.loader)}
	if m.watching {
		cmds = append(cmds, watchTick())
	}
	return tea.Batch(cmds...)
}

func loadSessionsCmd(loader *dataLoader) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		orphans, active, err := loader.Load(ctx)
		return sessionsLoadedMsg{orphans: orphans, active: active, err: err}
	}
}

func (m *model) View() tea.View {
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
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m model) renderBase() string {
	if !m.ready {
		return m.styles.subtle.Render("starting…")
	}
	header := m.renderHeader()
	body := m.renderBody()
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
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
	watch := ""
	if m.watching {
		watch = m.styles.statusOK.Render(" watching")
	}
	right := m.styles.subtle.Render("DB: "+m.options.DBPath) + watch

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

func (m model) renderBody() string {
	if m.width <= 0 || m.height <= 1 {
		return ""
	}

	bodyHeight := m.height - 1 // minus header
	if bodyHeight < 6 {
		bodyHeight = 6
	}

	leftWidth := m.width / 2
	if leftWidth < 32 {
		leftWidth = 32
	}
	rightWidth := m.width - leftWidth - 1
	if rightWidth < 24 {
		rightWidth = 24
	}

	leftCol := m.renderLeftColumn(leftWidth, bodyHeight)
	rightCol := m.renderRightColumn(rightWidth, bodyHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
}

func (m model) renderLeftColumn(width, height int) string {
	listHeight := int(float64(height) * 0.80)
	hintsHeight := height - listHeight
	if listHeight < 4 {
		listHeight = height - 3
		hintsHeight = 3
	}

	m.list.SetWidth(width - 2)
	m.list.SetHeight(listHeight - 2)

	listPanel := m.styles.listPanel.
		Width(width).
		Height(listHeight).
		Render(m.list.View())

	hintsPanel := m.renderHints(width, hintsHeight)

	return lipgloss.JoinVertical(lipgloss.Left, listPanel, hintsPanel)
}

func (m model) renderRightColumn(width, height int) string {
	metaHeight := int(float64(height) * 0.30)
	chatHeight := height - metaHeight
	if metaHeight < 5 {
		metaHeight = 5
		chatHeight = height - metaHeight
	}

	si, _ := selectedItem(m.list)
	metaContent := renderMetadataPanel(si, m.styles)
	metaPanel := m.styles.detailPanel.
		Width(width).
		Height(metaHeight).
		Render(metaContent)

	chatPanel := m.styles.detailPanel.
		Width(width).
		Height(chatHeight).
		Render(m.chatViewport.View())

	return lipgloss.JoinVertical(lipgloss.Left, metaPanel, chatPanel)
}

func (m model) renderHints(width, height int) string {
	helpView := m.help.ShortHelpView(m.keys.listShortHelp())
	content := m.styles.subtle.Render(helpView)
	return m.styles.detailPanel.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
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

	m := newModel(opts, handle)
	p := tea.NewProgram(m)
	_, err = p.Run()
	if err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return err
	}
	return nil
}
