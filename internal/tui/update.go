package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.handleWindowSize(msg)
		return m, nil

	case sessionsLoadedMsg:
		m = m.applyLoaded(msg)
		return m, nil

	case tea.KeyPressMsg:
		if m.quitting {
			return m, nil
		}
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.refreshDetail()
	return m, cmd
}

func (m model) handleWindowSize(msg tea.WindowSizeMsg) model {
	m.width = msg.Width
	m.height = msg.Height
	m.recomputeLayout()
	return m
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.refreshDetail()
	return m, cmd
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
		return
	}
	m.detail.SetContent(renderDetail(si, m.styles))
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
