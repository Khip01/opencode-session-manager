package tui

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/Khip01/opencode-session-manager/internal/db"
	"github.com/Khip01/opencode-session-manager/internal/relinker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func setupMigrateModalFixture(t *testing.T) (dbPath, srcDir, dstDir string, m model) {
	t.Helper()
	tmp := t.TempDir()
	dbPath = filepath.Join(tmp, "opencode.db")
	srcDir = filepath.Join(tmp, "src-project")
	dstDir = filepath.Join(tmp, "dst-project")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.MkdirAll(dstDir, 0o755))

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(modalTestSchema)
	require.NoError(t, err)

	_, err = conn.Exec(
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_src", srcDir, 1700000000000,
	)
	require.NoError(t, err)
	_, err = conn.Exec(
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_dst", dstDir, 1700000000000,
	)
	require.NoError(t, err)

	_, err = conn.Exec(
		`INSERT INTO session (id, project_id, directory, title, time_created, time_updated)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"ses_m1", "proj_src", srcDir, "migrate 1", 1000, 1000,
	)
	require.NoError(t, err)
	_, err = conn.Exec(
		`INSERT INTO session (id, project_id, directory, title, time_created, time_updated)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"ses_m2", "proj_src", srcDir, "migrate 2", 2000, 2000,
	)
	require.NoError(t, err)

	handle, err := db.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { handle.Close() })

	opts := Options{DBPath: dbPath, Version: "test"}
	m = newModel(opts, handle)
	m.relinker = relinker.New(dbPath)
	return dbPath, srcDir, dstDir, m
}

func TestOpenMigrateFlow_SetsPendingAndPicker(t *testing.T) {
	_, srcDir, _, m := setupMigrateModalFixture(t)

	_, _, err := m.loader.Load(t.Context())
	require.NoError(t, err)
	m.orphans, m.active, _ = m.loader.Load(t.Context())
	m.tab = tabActive
	m.populateList()
	require.NotEmpty(t, m.active, "fixture must produce active sessions")
	m.list.Select(0)

	m.openMigrateFlow()

	assert.Equal(t, modeFilepicker, m.mode)
	require.NotNil(t, m.modal.pending)
	assert.Equal(t, strategyMigrate, m.modal.pending.strategy)
	assert.Equal(t, defaultMigrateCount, m.modal.pending.migrateCount)
	assert.Equal(t, srcDir, m.picker.CurrentDirectory)
}

func TestConfirmMigrate_PopulatesPreview(t *testing.T) {
	_, srcDir, dstDir, m := setupMigrateModalFixture(t)

	m.modal.pending = &pendingRelink{
		sessionID:    "ses_m1",
		oldDirectory: srcDir,
		newDirectory: dstDir,
		strategy:     strategyMigrate,
		migrateCount: defaultMigrateCount,
	}

	m.confirmMigrate()

	assert.Equal(t, modeConfirm, m.mode)
	require.NotEmpty(t, m.modal.migratePreview)
	assert.Contains(t, m.modal.prompt, "Apply migrate?")
	assert.Contains(t, m.modal.prompt, "dst-")
	assert.Contains(t, m.modal.prompt, "ses_m1")
}

func TestConfirmMigrate_NoSessions_ShowsWarning(t *testing.T) {
	dbPath := setupEmptyDB(t)

	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "empty-src")
	dstDir := filepath.Join(tmp, "dst")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.MkdirAll(dstDir, 0o755))

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Exec(
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_src", srcDir, 1700000000000,
	)
	require.NoError(t, err)
	_, err = conn.Exec(
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_dst", dstDir, 1700000000000,
	)
	require.NoError(t, err)

	handle, err := db.Open(dbPath)
	require.NoError(t, err)
	defer handle.Close()

	opts := Options{DBPath: dbPath, Version: "test"}
	m := newModel(opts, handle)
	m.relinker = relinker.New(dbPath)

	m.modal.pending = &pendingRelink{
		sessionID:    "ses_x",
		oldDirectory: srcDir,
		newDirectory: dstDir,
		strategy:     strategyMigrate,
		migrateCount: defaultMigrateCount,
	}

	m.confirmMigrate()

	assert.Equal(t, modeResult, m.mode)
	assert.Equal(t, resultWarn, m.modal.resultKind)
}

func TestDoMigrate_UpdatesDirectoryAndProjectID(t *testing.T) {
	dbPath, srcDir, dstDir, m := setupMigrateModalFixture(t)

	m.modal.pending = &pendingRelink{
		sessionID:    "ses_m1",
		oldDirectory: srcDir,
		newDirectory: dstDir,
		strategy:     strategyMigrate,
		migrateCount: defaultMigrateCount,
	}
	m.confirmMigrate()
	require.Equal(t, modeConfirm, m.mode)

	m.doMigrate()

	assert.Equal(t, modeResult, m.mode)
	assert.Equal(t, resultOK, m.modal.resultKind)

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()

	rows, err := conn.Query(`SELECT id, directory, project_id FROM session ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	results := map[string]struct {
		dir       string
		projectID string
	}{}
	for rows.Next() {
		var id, dir, pid string
		require.NoError(t, rows.Scan(&id, &dir, &pid))
		results[id] = struct {
			dir       string
			projectID string
		}{dir, pid}
	}

	assert.Equal(t, dstDir, results["ses_m1"].dir)
	assert.Equal(t, "proj_dst", results["ses_m1"].projectID)
	assert.Equal(t, dstDir, results["ses_m2"].dir)
	assert.Equal(t, "proj_dst", results["ses_m2"].projectID)
}

func TestDoMigrate_AllAlreadyInTarget_ShowsWarn(t *testing.T) {
	dbPath, srcDir, dstDir, m := setupMigrateModalFixture(t)

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = conn.Exec(
		`UPDATE session SET project_id = 'proj_dst' WHERE project_id = 'proj_src'`,
	)
	require.NoError(t, err)
	conn.Close()

	m.modal.pending = &pendingRelink{
		sessionID:    "ses_m1",
		oldDirectory: srcDir,
		newDirectory: dstDir,
		strategy:     strategyMigrate,
		migrateCount: defaultMigrateCount,
	}

	m.doMigrate()

	assert.Equal(t, modeResult, m.mode)
	assert.Equal(t, resultWarn, m.modal.resultKind)
	assert.Contains(t, m.modal.resultMsg, "already belong")
}

func TestProceedToConfirmAfterPicker_MigrateTriggersConfirmMigrate(t *testing.T) {
	_, srcDir, dstDir, m := setupMigrateModalFixture(t)

	m.modal.pending = &pendingRelink{
		sessionID:    "ses_m1",
		oldDirectory: srcDir,
		newDirectory: dstDir,
		strategy:     strategyMigrate,
		migrateCount: defaultMigrateCount,
	}

	m.proceedToConfirmAfterPicker()

	assert.Equal(t, modeConfirm, m.mode)
	assert.NotEmpty(t, m.modal.migratePreview)
}

func setupEmptyDB(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "opencode.db")
	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Exec(modalTestSchema)
	require.NoError(t, err)
	return dbPath
}
