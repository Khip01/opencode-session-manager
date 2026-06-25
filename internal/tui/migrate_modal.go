package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khip01/opencode-session-manager/internal/relinker"
)

const defaultMigrateCount = 5

func (m *model) openMigrateFlow() {
	si, ok := selectedItem(m.list)
	if !ok {
		return
	}
	m.modal.pending = &pendingRelink{
		sessionID:    si.session.ID,
		sessionTitle: si.session.Title,
		sessionKind:  si.kind,
		oldDirectory: si.session.Directory,
		strategy:     strategyMigrate,
		migrateCount: defaultMigrateCount,
	}
	m.modal.prompt = fmt.Sprintf(
		"Cross-project migrate\n\nPick the target project directory.\n\nSource: %s\nCount:  %d most recent sessions",
		truncateModal(si.session.Directory, 60),
		defaultMigrateCount,
	)
	m.mode = modeFilepicker
	m.picker = newDirPicker()
	if si.session.Directory != "" {
		m.picker.CurrentDirectory = si.session.Directory
	}
}

func (m *model) confirmMigrate() {
	p := m.modal.pending
	if p == nil || p.newDirectory == "" {
		m.mode = modeNone
		m.modal.clear()
		return
	}

	ctx := context.Background()
	preview, err := m.relinker.PreviewMigrate(ctx, p.oldDirectory, p.migrateCount)
	if err != nil {
		m.showResult("Migrate preview failed: "+err.Error(), resultErr)
		return
	}
	m.modal.migratePreview = preview

	if len(preview) == 0 {
		m.showResult("No sessions to migrate.", resultWarn)
		return
	}

	var list strings.Builder
	for i, s := range preview {
		fmt.Fprintf(&list, "  %d. %s  %s\n", i+1, truncateModal(s.ID, 14), truncateModal(s.Title, 50))
	}

	m.modal.prompt = fmt.Sprintf(
		"Apply migrate?\n\nSource:   %s\nTarget:   %s\nCount:    %d most recent\n\nSessions:\n%s\nA backup will be created before any write.",
		truncateModal(p.oldDirectory, 60),
		truncateModal(p.newDirectory, 60),
		p.migrateCount,
		list.String(),
	)
	m.mode = modeConfirm
}

func (m *model) doMigrate() {
	p := m.modal.pending
	if p == nil {
		return
	}

	ctx := context.Background()
	matches, err := m.relinker.Migrate(ctx, p.oldDirectory, p.newDirectory, p.migrateCount)
	if err != nil {
		m.showResult("Migrate failed: "+err.Error(), resultErr)
		return
	}

	if len(matches) == 0 {
		m.showResult(
			"All recent sessions already belong to the target project — nothing to migrate.",
			resultWarn,
		)
		m.refreshSessions()
		return
	}

	backupAt, _ := relinker.Backup(m.options.DBPath)
	msg := fmt.Sprintf(
		"Migrated %d session(s).\nBackup: %s",
		len(matches),
		backupAt,
	)
	m.showResult(msg, resultOK)
	m.refreshSessions()
}
