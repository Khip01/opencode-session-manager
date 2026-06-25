package tui

import (
	"context"
	"fmt"

	"github.com/Khip01/opencode-session-manager/internal/relinker"
)

type applyResult struct {
	updated   int
	backupAt  string
	err       error
}

func (m *model) applyPending() applyResult {
	if m.modal.pending == nil {
		return applyResult{err: fmt.Errorf("no pending relink")}
	}

	p := m.modal.pending
	ctx := context.Background()
	r := relinker.New(m.options.DBPath)

	switch p.strategy {
	case strategyPhase1:
		if p.phase1Match == nil {
			return applyResult{err: fmt.Errorf("phase1 match missing")}
		}
		matches := []relinker.Match{{
			SessionID:    p.phase1Match.sessionID,
			OldDirectory: p.phase1Match.oldDir,
			NewDirectory: p.phase1Match.newDir,
			Strategy:     relinker.StrategyProjectID,
		}}
		if err := r.ApplyAll(ctx, matches); err != nil {
			return applyResult{err: err}
		}
		backupAt, _ := relinker.Backup(m.options.DBPath)
		return applyResult{updated: len(matches), backupAt: backupAt}

	case strategyManual:
		matches, err := r.RelinkByPath(ctx, p.oldDirectory, p.newDirectory)
		if err != nil {
			return applyResult{err: err}
		}
		backupAt, _ := relinker.Backup(m.options.DBPath)
		return applyResult{updated: len(matches), backupAt: backupAt}

	default:
		return applyResult{err: fmt.Errorf("unknown strategy")}
	}
}

func (m *model) doApply() {
	result := m.applyPending()
	if result.err != nil {
		m.showResult("Relink failed: "+result.err.Error(), resultErr)
		return
	}

	msg := fmt.Sprintf(
		"Updated %d session(s).\nBackup: %s",
		result.updated,
		result.backupAt,
	)
	m.showResult(msg, resultOK)
	m.refreshSessions()
}

func (m *model) refreshSessions() {
	ctx := context.Background()
	orphans, active, err := m.loader.Load(ctx)
	if err != nil {
		m.status = m.styles.statusErr.Render("refresh failed: " + err.Error())
		return
	}
	m.orphans = orphans
	m.active = active
	m.populateList()
	m.refreshDetail()
}

func (m *model) showResult(msg string, kind resultKind) {
	m.modal.resultMsg = msg
	m.modal.resultKind = kind
	m.mode = modeResult
}
