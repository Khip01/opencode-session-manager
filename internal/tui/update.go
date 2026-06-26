package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.handleWindowSize(msg)
		return m, nil

	case sessionsLoadedMsg:
		m = m.applyLoaded(msg)
		if m.watching {
			return m, watchTick()
		}
		return m, nil

	case watchTickMsg:
		cmd := m.handleWatchTick()
		return m, cmd

	case tea.MouseMsg:
		// Track cursor position and forward the event through the
		// normal component pipeline. forwardToComponents routes wheel
		// events based on cursor position; other mouse events go to
		// both viewports so click and motion keep working.
		m = m.updateMouseFromMsg(msg)
		return m.forwardToComponents(msg)

	case tea.KeyPressMsg:
		if m.quitting {
			return m, nil
		}
		return m.dispatchKey(msg)
	}

	return m.forwardToComponents(msg)
}

func (m *model) updateMouseFromMsg(msg tea.MouseMsg) *model {
	ev := msg.Mouse()
	m.mouseX = ev.X
	m.mouseY = ev.Y
	return m
}

// routeTarget describes which panel under the cursor should receive
// scroll events.
type routeTarget int

const (
	routeNone      routeTarget = iota
	routeList                // left column, list session area
	routeChat                // right column, chat preview area
)

// scrollTarget inspects the current mouse position and returns
// which viewport should receive a scroll event, or routeNone if
// the cursor is over a non-scrollable region (header, hints panel,
// meta panel, or outside the terminal).
//
// Panel layout (row indices from the top of the terminal):
//
//	row 0                          header
//	rows 1 .. 1+listHeight         left list panel
//	rows 1+listHeight .. bodyEnd    left hints panel
//	rows 1 .. 1+metaHeight         right meta panel
//	rows 1+metaHeight .. bodyEnd    right chat panel
//
// Column split is at m.width / 2.
func (m model) scrollTarget() routeTarget {
	if m.height <= 1 || m.width <= 0 {
		return routeNone
	}
	bodyHeight := m.height - 1
	leftWidth := m.width / 2
	listHeight := int(float64(bodyHeight) * 0.80)
	metaHeight := int(float64(bodyHeight) * 0.30)
	if listHeight < 4 {
		listHeight = 4
	}
	if metaHeight < 5 {
		metaHeight = 5
	}

	y := m.mouseY
	x := m.mouseX
	if y <= 0 || y >= m.height {
		return routeNone
	}

	if x < leftWidth {
		// Left column.
		if y <= 1+listHeight {
			return routeList
		}
		return routeNone // hints panel is not scrollable
	}

	// Right column.
	if y <= 1+metaHeight {
		return routeNone // meta panel is not scrollable
	}
	return routeChat
}

func (m *model) forwardToComponents(msg tea.Msg) (*model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// For wheel events, route to the panel under the cursor instead
	// of forwarding to both viewports. The list.Model and viewport
	// otherwise both react to wheel and it feels like everything is
	// moving when the user only meant to scroll the chat preview.
	if _, isWheel := msg.(tea.MouseWheelMsg); isWheel {
		target := m.scrollTarget()
		switch target {
		case routeList:
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
		case routeChat:
			m.chatViewport, cmd = m.chatViewport.Update(msg)
			cmds = append(cmds, cmd)
		default:
			// Cursor over a non-scrollable region. Drop the event.
		}
		return m, tea.Batch(cmds...)
	}

	// Non-wheel events: forward to both. The list captures clicks
	// and arrow keys; the viewport scrolls with arrow keys when it
	// has focus. Chat preview content is re-rendered on every tick.
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	m.chatViewport, cmd = m.chatViewport.Update(msg)
	cmds = append(cmds, cmd)
	m.refreshDetail()
	return m, tea.Batch(cmds...)
}

func (m *model) dispatchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeNone:
		return m.handleListKey(msg)
	case modeRelinkChoice:
		return m.handleChoiceKey(msg)
	case modeFilepicker:
		return m.handlePickerKey(msg)
	case modeConfirm:
		return m.handleConfirmKey(msg)
	case modeRunningWarn:
		return m.handleRunningKey(msg)
	case modeResult:
		return m.handleResultKey(msg)
	}
	return m, nil
}

func (m *model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys.ForceQuit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys.ChatUp):
		m.chatViewport.HalfPageUp()
		return m, nil
	case key.Matches(msg, m.keys.ChatDown):
		m.chatViewport.HalfPageDown()
		return m, nil
	case key.Matches(msg, m.keys.ChatTop):
		m.chatViewport.GotoTop()
		return m, nil
	case key.Matches(msg, m.keys.ChatBottom):
		m.chatViewport.GotoBottom()
		return m, nil
	case key.Matches(msg, m.keys.NextTab):
		m.switchTab(+1)
		return m, nil
	case key.Matches(msg, m.keys.PrevTab):
		m.switchTab(-1)
		return m, nil
	case key.Matches(msg, m.keys.Relink):
		if _, ok := selectedItem(m.list); ok {
			m.openRelinkChoice()
			return m, nil
		}
	case key.Matches(msg, m.keys.Manual):
		if si, ok := selectedItem(m.list); ok {
			m.modal.pending = &pendingRelink{
				sessionID:    si.session.ID,
				sessionTitle: si.session.Title,
				sessionKind:  si.kind,
				oldDirectory: si.session.Directory,
				strategy:     strategyManual,
			}
			m.startManualPath()
			return m, nil
		}
	case key.Matches(msg, m.keys.Migrate):
		if _, ok := selectedItem(m.list); ok {
			m.openMigrateFlow()
			return m, nil
		}
	case key.Matches(msg, m.keys.Watch):
		return m, m.toggleWatch()
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.refreshDetail()
	return m, cmd
}

func (m *model) handleChoiceKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = modeNone
		m.modal.clear()
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.modal.choiceIdx > 0 {
			m.modal.choiceIdx--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.modal.choiceIdx < len(m.modal.choiceOptions)-1 {
			m.modal.choiceIdx++
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		m.confirmChoice()
		return m, nil
	}
	return m, nil
}

func (m *model) handlePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)

	if selected, path := m.picker.DidSelectFile(msg); selected {
		if m.modal.pending != nil {
			m.modal.pending.newDirectory = path
		}
		m.proceedToConfirmAfterPicker()
		return m, cmd
	}

	if key.Matches(msg, m.keys.Cancel) {
		m.mode = modeNone
		m.modal.clear()
		return m, cmd
	}
	return m, cmd
}

func (m *model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Yes), key.Matches(msg, m.keys.Enter):
		if m.modal.pending != nil && m.modal.pending.strategy == strategyMigrate {
			m.doMigrate()
		} else {
			m.doApply()
		}
		return m, nil
	case key.Matches(msg, m.keys.No), key.Matches(msg, m.keys.Cancel):
		m.mode = modeNone
		m.modal.clear()
		return m, nil
	}
	return m, nil
}

func (m *model) handleRunningKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Yes), key.Matches(msg, m.keys.Enter):
		m.killRunningAndProceed()
		return m, nil
	case key.Matches(msg, m.keys.No), key.Matches(msg, m.keys.Cancel):
		m.mode = modeNone
		m.modal.clear()
		return m, nil
	}
	return m, nil
}

func (m *model) handleResultKey(_ tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.mode = modeNone
	m.modal.clear()
	return m, nil
}

func (m *model) handleWindowSize(msg tea.WindowSizeMsg) *model {
	m.width = msg.Width
	m.height = msg.Height
	m.recomputeLayout()
	m.syncViewportContent()
	return m
}

// syncViewportContent resizes the chat preview viewport and refreshes
// its content from m.chatMessages. This is the single source of truth
// for chat viewport state, called from Update path. View() must NOT
// mutate the viewport because View runs with a value-receiver copy
// in Bubble Tea v2's render flow.
func (m *model) syncViewportContent() {
	if m.width <= 0 {
		return
	}
	bodyHeight := m.height - 1
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

	metaHeight := int(float64(bodyHeight) * 0.30)
	chatHeight := bodyHeight - metaHeight
	if metaHeight < 5 {
		metaHeight = 5
		chatHeight = bodyHeight - metaHeight
	}

	m.chatViewport.SetWidth(rightWidth - 2)
	m.chatViewport.SetHeight(chatHeight - 2)
	chatContent := renderChatViewportContent(m.chatMessages, m.styles, rightWidth-4)
	m.chatViewport.SetContent(chatContent)
}

func (m *model) switchTab(delta int) {
	tabs := []tabID{tabOrphans, tabActive}
	idx := 0
	for i, t := range tabs {
		if t == m.tab {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(tabs)) % len(tabs)
	m.tab = tabs[idx]
	m.populateList()
	m.refreshDetail()
}

func (m *model) populateList() {
	var items []sessionItem
	switch m.tab {
	case tabOrphans:
		items = m.orphans
	case tabActive:
		items = m.active
	}
	m.list = refreshList(m.list, items)
	m.list = setListTabTitle(m.list, m.tab, len(m.orphans), len(m.active))
	if len(items) == 0 {
		m.list.Select(0)
	}
}

func (m *model) refreshDetail() {
	si, ok := selectedItem(m.list)
	if !ok {
		m.chatMessages = nil
		m.chatLoadedSessionID = ""
		return
	}

	// Load chat messages if we have not loaded them for this
	// session yet. Sync is fine here because typical preview
	// payloads are small (a few dozen messages at most).
	if m.chatLoadedSessionID != si.session.ID {
		ctx := context.Background()
		msgs, err := m.loader.LoadMessages(ctx, si.session.ID, 0)
		if err != nil {
			msgs = nil
		}
		m.chatMessages = msgs
		m.chatLoadedSessionID = si.session.ID
	}
	// Push fresh chat content into the viewport so the panel shows
	// the newly selected session. View() must not mutate the viewport,
	// so all sizing and content writes happen here in Update path.
	m.syncViewportContent()
}

func (m *model) applyLoaded(msg sessionsLoadedMsg) *model {
	if msg.err != nil {
		m.err = msg.err
		m.status = m.styles.statusErr.Render("load error: " + msg.err.Error())
		return m
	}
	m.orphans = msg.orphans
	m.active = msg.active
	m.status = m.styles.statusOK.Render(
		statusSummary(len(msg.orphans), len(msg.active)),
	)
	m.populateList()
	m.refreshDetail()
	return m
}
