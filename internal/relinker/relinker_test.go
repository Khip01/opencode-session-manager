package relinker

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Khip01/opencode-session-manager/internal/db"

	_ "modernc.org/sqlite"
)

func writeFixtureDB(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "opencode.db")

	conn, err := sql.Open("sqlite", src)
	require.NoError(t, err)

	_, err = conn.Exec(`
		CREATE TABLE project (
			id TEXT PRIMARY KEY,
			worktree TEXT NOT NULL,
			vcs TEXT,
			name TEXT,
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
	require.NoError(t, conn.Close())
	return src
}

func seedStaleAndActive(t *testing.T, dbPath string, staleDir, activeDir string) {
	t.Helper()

	conn, err := db.Open(dbPath)
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx,
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_git", activeDir, 1700000000000,
	)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx,
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_other", "/some/other/path", 1700000000000,
	)
	require.NoError(t, err)

	sessions := []db.Session{
		{ID: "ses_stale_match", ProjectID: "proj_git", Directory: staleDir, Title: "stale but matchable", TimeCreated: 1000},
		{ID: "ses_stale_no_match", ProjectID: "proj_ghost", Directory: staleDir, Title: "stale project gone", TimeCreated: 2000},
		{ID: "ses_active_git", ProjectID: "proj_git", Directory: activeDir, Title: "active in git project", TimeCreated: 3000},
		{ID: "ses_active_no_pid", ProjectID: "", Directory: activeDir, Title: "active, no project_id", TimeCreated: 4000},
	}
	for _, s := range sessions {
		var pid any
		if s.ProjectID != "" {
			pid = s.ProjectID
		}
		_, err = conn.ExecContext(ctx,
			`INSERT INTO session (id, project_id, directory, title, time_created, time_updated)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			s.ID, pid, s.Directory, s.Title, s.TimeCreated, s.TimeCreated,
		)
		require.NoError(t, err)
	}
}

func seedOnlyActiveSessions(t *testing.T, dbPath, activeDir string) {
	t.Helper()

	conn, err := db.Open(dbPath)
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx,
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_git", activeDir, 1700000000000,
	)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx,
		`INSERT INTO session (id, project_id, directory, title, time_created, time_updated)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"ses_active_git", "proj_git", activeDir, "active", 3000, 3000,
	)
	require.NoError(t, err)
}

func TestNew_StoresDBPath(t *testing.T) {
	r := New("/tmp/opencode.db")
	assert.Equal(t, "/tmp/opencode.db", r.DBPath())
}

func TestFindStaleSessions_FiltersByExistence(t *testing.T) {
	dbPath := writeFixtureDB(t)
	tmp := t.TempDir()
	stale := filepath.Join(tmp, "deleted-folder")
	active := filepath.Join(tmp, "alive-folder")
	require.NoError(t, os.MkdirAll(active, 0o755))

	seedStaleAndActive(t, dbPath, stale, active)

	r := New(dbPath)
	staleSessions, err := r.FindStaleSessions(context.Background())
	require.NoError(t, err)

	ids := make([]string, 0, len(staleSessions))
	for _, s := range staleSessions {
		ids = append(ids, s.ID)
	}
	assert.ElementsMatch(t, []string{"ses_stale_match", "ses_stale_no_match"}, ids)
}

func TestFindPhase1Matches_OnlyStaleWithValidProject(t *testing.T) {
	dbPath := writeFixtureDB(t)
	tmp := t.TempDir()
	stale := filepath.Join(tmp, "deleted-folder")
	active := filepath.Join(tmp, "alive-folder")
	require.NoError(t, os.MkdirAll(active, 0o755))

	seedStaleAndActive(t, dbPath, stale, active)

	r := New(dbPath)
	matches, err := r.FindPhase1Matches(context.Background())
	require.NoError(t, err)
	require.Len(t, matches, 1)

	assert.Equal(t, "ses_stale_match", matches[0].SessionID)
	assert.Equal(t, stale, matches[0].OldDirectory)
	assert.Equal(t, active, matches[0].NewDirectory)
	assert.Equal(t, StrategyProjectID, matches[0].Strategy)
}

func TestFindPhase1Matches_EmptyWhenNoStale(t *testing.T) {
	dbPath := writeFixtureDB(t)
	tmp := t.TempDir()
	active := filepath.Join(tmp, "alive-folder")
	require.NoError(t, os.MkdirAll(active, 0o755))

	seedOnlyActiveSessions(t, dbPath, active)

	r := New(dbPath)
	matches, err := r.FindPhase1Matches(context.Background())
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestApplyAll_UpdatesDirectoriesAndCreatesBackup(t *testing.T) {
	dbPath := writeFixtureDB(t)
	tmp := t.TempDir()
	stale := filepath.Join(tmp, "deleted-folder")
	active := filepath.Join(tmp, "alive-folder")
	require.NoError(t, os.MkdirAll(active, 0o755))

	seedStaleAndActive(t, dbPath, stale, active)

	r := New(dbPath)
	matches, err := r.FindPhase1Matches(context.Background())
	require.NoError(t, err)
	require.Len(t, matches, 1)

	originalDir := mustReadSessionDir(t, dbPath, "ses_stale_match")
	assert.Equal(t, stale, originalDir)

	err = r.ApplyAll(context.Background(), matches)
	require.NoError(t, err)

	if _, err := os.Stat(BackupPath(dbPath)); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}

	updated := mustReadSessionDir(t, dbPath, "ses_stale_match")
	assert.Equal(t, active, updated)

	backupDir := mustReadSessionDir(t, BackupPath(dbPath), "ses_stale_match")
	assert.Equal(t, stale, backupDir)
}

func TestApplyAll_EmptyMatchesNoOp(t *testing.T) {
	r := New("/tmp/does-not-matter.db")
	err := r.ApplyAll(context.Background(), nil)
	assert.NoError(t, err)
}

func TestApplyAll_RejectsInvalidMatch(t *testing.T) {
	r := New("/tmp/does-not-matter.db")
	err := r.ApplyAll(context.Background(), []Match{
		{SessionID: "", OldDirectory: "/old", NewDirectory: "/new", Strategy: "manual"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMatch)
}

func mustReadSessionDir(t *testing.T, dbPath, sessionID string) string {
	t.Helper()
	conn, err := db.Open(dbPath)
	require.NoError(t, err)
	defer conn.Close()

	var dir string
	row := conn.QueryRowContext(context.Background(), `SELECT directory FROM session WHERE id = ?`, sessionID)
	require.NoError(t, row.Scan(&dir))
	return dir
}
