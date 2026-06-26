package tui

import (
	"path/filepath"
	"testing"

	"github.com/Khip01/opencode-session-manager/internal/db"
	"github.com/Khip01/opencode-session-manager/internal/relinker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"database/sql"

	_ "modernc.org/sqlite"
)

const modalTestSchema = `
CREATE TABLE project (
    id TEXT PRIMARY KEY,
    worktree TEXT NOT NULL,
    time_created INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE session (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    parent_id TEXT,
    directory TEXT NOT NULL,
    title TEXT NOT NULL,
    agent TEXT,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL,
    time_archived INTEGER NOT NULL DEFAULT 0
);
`

func newApplyFixture(t *testing.T) (dbPath, aliveDir string) {
	t.Helper()
	tmp := t.TempDir()
	dbPath = filepath.Join(tmp, "opencode.db")
	aliveDir = filepath.Join(tmp, "alive")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(modalTestSchema)
	require.NoError(t, err)

	_, err = conn.Exec(
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_git", aliveDir, 1700000000000,
	)
	require.NoError(t, err)
	return dbPath, aliveDir
}

func insertSession(t *testing.T, dbPath, id, pid, dir, title string) {
	t.Helper()
	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()
	var pidArg any
	if pid != "" {
		pidArg = pid
	}
	_, err = conn.Exec(
		`INSERT INTO session (id, project_id, directory, title, time_created, time_updated)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, pidArg, dir, title, 1, 1,
	)
	require.NoError(t, err)
}

func openTestModel(t *testing.T, dbPath string) *model {
	t.Helper()
	handle, err := db.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { handle.Close() })

	opts := Options{DBPath: dbPath, Version: "test"}
	m := newModel(opts, handle)
	m.relinker = relinker.New(dbPath)
	return m
}

func TestApply_Pending_Phase1UpdatesAndCreatesBackup(t *testing.T) {
	dbPath, aliveDir := newApplyFixture(t)
	insertSession(t, dbPath, "ses_x", "proj_git", "/deleted/path", "orphan")

	m := openTestModel(t, dbPath)
	m.modal.pending = &pendingRelink{
		sessionID:    "ses_x",
		sessionTitle: "orphan",
		oldDirectory: "/deleted/path",
		strategy:     strategyPhase1,
		phase1Match: &relinkerMatch{
			sessionID: "ses_x",
			oldDir:    "/deleted/path",
			newDir:    aliveDir,
		},
	}

	m.doApply()

	assert.Equal(t, modeResult, m.mode)

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()

	var newDir string
	row := conn.QueryRow(`SELECT directory FROM session WHERE id = ?`, "ses_x")
	require.NoError(t, row.Scan(&newDir))
	assert.Equal(t, aliveDir, newDir)

	_, err = sql.Open("sqlite", dbPath+relinker.BackupSuffix)
	require.NoError(t, err)
}

func TestApply_Pending_ManualPath(t *testing.T) {
	dbPath, _ := newApplyFixture(t)

	tmp := t.TempDir()
	newDir := filepath.Join(tmp, "newpath")
	insertSession(t, dbPath, "ses_y", "proj_git", "/old/path", "old one")

	m := openTestModel(t, dbPath)
	m.modal.pending = &pendingRelink{
		sessionID:    "ses_y",
		oldDirectory: "/old/path",
		newDirectory: newDir,
		strategy:     strategyManual,
	}

	m.doApply()
	assert.Equal(t, modeResult, m.mode)

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()

	var got string
	row := conn.QueryRow(`SELECT directory FROM session WHERE id = ?`, "ses_y")
	require.NoError(t, row.Scan(&got))
	assert.Equal(t, newDir, got)
}

func TestModalState_Lifecycle(t *testing.T) {
	ms := newModalState()
	assert.False(t, ms.hasPending())

	ms.pending = &pendingRelink{sessionID: "abc"}
	assert.True(t, ms.hasPending())

	ms.clear()
	assert.False(t, ms.hasPending())
	assert.Empty(t, ms.choiceOptions)
}

func TestRefreshSessions_ReloadsFromDB(t *testing.T) {
	dbPath, _ := newApplyFixture(t)
	insertSession(t, dbPath, "ses_r", "", "/deleted/refreshtest", "refresh me")

	m := openTestModel(t, dbPath)
	m.refreshSessions()
	assert.NotEmpty(t, m.orphans)
}

func TestOpenRelinkChoice_SetsPendingAndMode(t *testing.T) {
	dbPath, _ := newApplyFixture(t)
	insertSession(t, dbPath, "ses_sel", "", "/deleted", "selected")

	m := openTestModel(t, dbPath)
	_, _, err := m.loader.Load(t.Context())
	require.NoError(t, err)
	m.orphans, m.active, _ = m.loader.Load(t.Context())
	m.populateList()
	m.list.Select(0)

	m.openRelinkChoice()

	assert.Equal(t, modeRelinkChoice, m.mode)
	require.NotNil(t, m.modal.pending)
	assert.Equal(t, "ses_sel", m.modal.pending.sessionID)
	assert.Len(t, m.modal.choiceOptions, 3)
}

func TestShowResult_SetsModeAndMessage(t *testing.T) {
	dbPath, _ := newApplyFixture(t)
	m := openTestModel(t, dbPath)
	m.showResult("done", resultOK)
	assert.Equal(t, modeResult, m.mode)
	assert.Equal(t, "done", m.modal.resultMsg)
	assert.Equal(t, resultOK, m.modal.resultKind)
}

func TestConfirmChoice_ManualOpensFilepicker(t *testing.T) {
	dbPath, _ := newApplyFixture(t)
	insertSession(t, dbPath, "ses_c", "proj_git", "/deleted/c", "c")

	m := openTestModel(t, dbPath)
	m.modal.prompt = "test"
	m.modal.choiceOptions = []string{"Phase 1", "Manual", "Cancel"}
	m.modal.choiceIdx = 1
	m.modal.pending = &pendingRelink{
		sessionID:    "ses_c",
		sessionTitle: "c",
		oldDirectory: "/deleted/c",
	}

	m.confirmChoice()

	assert.Equal(t, modeFilepicker, m.mode)
	assert.Equal(t, strategyManual, m.modal.pending.strategy)
}

func TestConfirmChoice_CancelClearsModal(t *testing.T) {
	dbPath, _ := newApplyFixture(t)
	m := openTestModel(t, dbPath)
	m.modal.choiceOptions = []string{"Phase 1", "Manual", "Cancel"}
	m.modal.choiceIdx = 2
	m.modal.pending = &pendingRelink{sessionID: "x"}

	m.confirmChoice()

	assert.Equal(t, modeNone, m.mode)
	assert.False(t, m.modal.hasPending())
}

func TestRelinkStrategy_Labels(t *testing.T) {
	assert.Equal(t, "Phase 1 (project_id)", strategyPhase1.Label())
	assert.Equal(t, "Manual path remap", strategyManual.Label())
}

func TestModeLabels(t *testing.T) {
	assert.Equal(t, "list", modeNone.Label())
	assert.Equal(t, "relink-choice", modeRelinkChoice.Label())
	assert.Equal(t, "filepicker", modeFilepicker.Label())
	assert.Equal(t, "confirm", modeConfirm.Label())
	assert.Equal(t, "running-warn", modeRunningWarn.Label())
	assert.Equal(t, "result", modeResult.Label())
}
