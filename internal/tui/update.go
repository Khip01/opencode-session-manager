package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case tea.KeyPressMsg:
		if m.quitting {
			return m, nil
		}
		return m.dispatchKey(msg)
	}

	return m, m.forwardToComponents(msg)
}

func (m model) forwardToComponents(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	m.detail, cmd = m.detail.Update(msg)
	cmds = append(cmds, cmd)
	m.refreshDetail()
	return tea.Batch(cmds...)
}

func (m model) dispatchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys.ForceQuit):
		m.quitting = true
		return m, tea.Quit
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
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.refreshDetail()
	return m, cmd
}

func (m model) handleChoiceKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m model) handlePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m model) handleRunningKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m model) handleResultKey(_ tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.mode = modeNone
	m.modal.clear()
	return m, nil
}

func (m model) handleWindowSize(msg tea.WindowSizeMsg) model {
	m.width = msg.Width
	m.height = msg.Height
	m.recomputeLayout()
	return m
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
	if len(items) == 0 {
		m.list.Select(0)
	}
}

func (m *model) refreshDetail() {
	si, ok := selectedItem(m.list)
	if !ok {
		m.detail.SetContent(m.styles.detailEmpty.Render("Select a session to see details."))
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

	m.detail.SetContent(renderDetail(si, m.chatMessages, m.styles))
}

func (m model) applyLoaded(msg sessionsLoadedMsg) model {
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
