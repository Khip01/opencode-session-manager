package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

const defaultWatchInterval = 3 * time.Second

type watchTickMsg time.Time

func watchTick() tea.Cmd {
	return tea.Tick(defaultWatchInterval, func(t time.Time) tea.Msg {
		return watchTickMsg(t)
	})
}

func (m *model) watchOn() {
	if m.watching {
		return
	}
	m.watching = true
	m.status = m.styles.statusOK.Render(
		"watching opencode.db, auto-refresh every " + defaultWatchInterval.String(),
	)
}

func (m *model) watchOff() {
	if !m.watching {
		return
	}
	m.watching = false
	m.status = m.styles.subtle.Render("watch paused, press w to resume")
}

func (m *model) toggleWatch() tea.Cmd {
	if m.watching {
		m.watchOff()
		return nil
	}
	m.watchOn()
	return watchTick()
}

func (m *model) handleWatchTick() tea.Cmd {
	if !m.watching {
		return nil
	}
	ctx := context.Background()
	orphans, active, err := m.loader.Load(ctx)
	if err != nil {
		m.status = m.styles.statusErr.Render("watch: reload failed: " + err.Error())
		return watchTick()
	}

	changed := len(orphans) != len(m.orphans) || len(active) != len(m.active)
	m.orphans = orphans
	m.active = active
	m.populateList()
	m.refreshDetail()
	if changed {
		m.status = m.styles.statusOK.Render(
			"watch: refreshed, " + statusSummary(len(orphans), len(active)),
		)
	}
	return watchTick()
}
