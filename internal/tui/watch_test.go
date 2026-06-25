package tui

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Khip01/opencode-session-manager/internal/db"

	_ "modernc.org/sqlite"
)

func newWatchFixture(t *testing.T) (dbPath string) {
	t.Helper()
	dbPath = filepath.Join(t.TempDir(), "opencode.db")
	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(`
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
	`)
	require.NoError(t, err)

	tmp := t.TempDir()
	_, err = conn.Exec(
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_git", tmp, 1700000000000,
	)
	require.NoError(t, err)

	_, err = conn.Exec(
		`INSERT INTO session (id, project_id, directory, title, time_created, time_updated)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"ses_w1", "proj_git", tmp, "watch 1", 1, 1,
	)
	require.NoError(t, err)
	return dbPath
}

func TestWatchOn_StartsWatching(t *testing.T) {
	dbPath := newWatchFixture(t)
	handle, err := db.Open(dbPath)
	require.NoError(t, err)
	defer handle.Close()

	m := newModel(Options{DBPath: dbPath}, handle)
	m.watchOn()

	assert.True(t, m.watching)
	assert.Contains(t, m.status, "watching")
}

func TestWatchOff_StopsWatching(t *testing.T) {
	dbPath := newWatchFixture(t)
	handle, err := db.Open(dbPath)
	require.NoError(t, err)
	defer handle.Close()

	m := newModel(Options{DBPath: dbPath}, handle)
	m.watching = true
	m.watchOff()

	assert.False(t, m.watching)
	assert.Contains(t, m.status, "paused")
}

func TestToggleWatch_Toggles(t *testing.T) {
	dbPath := newWatchFixture(t)
	handle, err := db.Open(dbPath)
	require.NoError(t, err)
	defer handle.Close()

	m := newModel(Options{DBPath: dbPath}, handle)
	require.False(t, m.watching)

	cmd := m.toggleWatch()
	require.NotNil(t, cmd, "toggleWatch should start a tick when enabling")
	assert.True(t, m.watching)

	cmd2 := m.toggleWatch()
	assert.Nil(t, cmd2, "toggleWatch should not schedule a tick when disabling")
	assert.False(t, m.watching)
}

func TestHandleWatchTick_UpdatesLists(t *testing.T) {
	dbPath := newWatchFixture(t)
	handle, err := db.Open(dbPath)
	require.NoError(t, err)
	defer handle.Close()

	m := newModel(Options{DBPath: dbPath}, handle)
	m.watching = true
	_, _, _ = m.loader.Load(t.Context())
	m.orphans = nil
	m.active = nil

	cmd := m.handleWatchTick()
	require.NotNil(t, cmd)
	assert.NotNil(t, m.active)
}

func TestHandleWatchTick_SkipsWhenNotWatching(t *testing.T) {
	dbPath := newWatchFixture(t)
	handle, err := db.Open(dbPath)
	require.NoError(t, err)
	defer handle.Close()

	m := newModel(Options{DBPath: dbPath}, handle)
	m.watching = false
	m.orphans = nil

	cmd := m.handleWatchTick()
	assert.Nil(t, cmd)
	assert.Nil(t, m.orphans)
}

func TestWatchTick_ReturnsCommand(t *testing.T) {
	cmd := watchTick()
	assert.NotNil(t, cmd)
}
