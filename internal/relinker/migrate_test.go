package relinker

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

const migrateTestSchema = `
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

func newMigrateFixture(t *testing.T) (dbPath, srcDir, dstDir string) {
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
	_, err = conn.Exec(migrateTestSchema)
	require.NoError(t, err)
	return dbPath, srcDir, dstDir
}

func seedProject(t *testing.T, dbPath, id, worktree string) {
	t.Helper()
	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.Exec(
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		id, worktree, 1700000000000,
	)
	require.NoError(t, err)
}

func seedMigrateSession(t *testing.T, dbPath, id, pid, dir, title string, created int64) {
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
		id, pidArg, dir, title, created, created,
	)
	require.NoError(t, err)
}

func TestMigrate_UpdatesDirectoryAndProject(t *testing.T) {
	dbPath, srcDir, dstDir := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedProject(t, dbPath, "proj_dst", dstDir)
	seedMigrateSession(t, dbPath, "ses_1", "proj_src", srcDir, "first", 1000)
	seedMigrateSession(t, dbPath, "ses_2", "proj_src", srcDir, "second", 2000)
	seedMigrateSession(t, dbPath, "ses_3", "proj_src", srcDir, "third", 3000)

	r := New(dbPath)
	matches, err := r.Migrate(context.Background(), srcDir, dstDir, 5)
	require.NoError(t, err)
	assert.Len(t, matches, 3)

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
	for _, id := range []string{"ses_1", "ses_2", "ses_3"} {
		assert.Equal(t, dstDir, results[id].dir)
		assert.Equal(t, "proj_dst", results[id].projectID)
	}
}

func TestMigrate_RespectsCount(t *testing.T) {
	dbPath, srcDir, dstDir := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedProject(t, dbPath, "proj_dst", dstDir)
	seedMigrateSession(t, dbPath, "ses_1", "proj_src", srcDir, "first", 1000)
	seedMigrateSession(t, dbPath, "ses_2", "proj_src", srcDir, "second", 2000)
	seedMigrateSession(t, dbPath, "ses_3", "proj_src", srcDir, "third", 3000)

	r := New(dbPath)
	matches, err := r.Migrate(context.Background(), srcDir, dstDir, 2)
	require.NoError(t, err)
	assert.Len(t, matches, 2)

	for _, m := range matches {
		assert.Equal(t, srcDir, m.OldDirectory)
		assert.Equal(t, dstDir, m.NewDirectory)
		assert.Equal(t, StrategyMigrate, m.Strategy)
	}

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer conn.Close()

	var migratedCount int
	require.NoError(t, conn.QueryRow(
		`SELECT COUNT(*) FROM session WHERE directory = ?`, dstDir,
	).Scan(&migratedCount))
	assert.Equal(t, 2, migratedCount)
}

func TestMigrate_PicksMostRecent(t *testing.T) {
	dbPath, srcDir, dstDir := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedProject(t, dbPath, "proj_dst", dstDir)
	seedMigrateSession(t, dbPath, "ses_old", "proj_src", srcDir, "old", 100)
	seedMigrateSession(t, dbPath, "ses_mid", "proj_src", srcDir, "mid", 500)
	seedMigrateSession(t, dbPath, "ses_new", "proj_src", srcDir, "new", 1000)

	r := New(dbPath)
	matches, err := r.Migrate(context.Background(), srcDir, dstDir, 2)
	require.NoError(t, err)
	assert.Len(t, matches, 2)

	ids := []string{matches[0].SessionID, matches[1].SessionID}
	assert.ElementsMatch(t, []string{"ses_mid", "ses_new"}, ids)
}

func TestMigrate_SkipsSessionsAlreadyInTarget(t *testing.T) {
	dbPath, srcDir, dstDir := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedProject(t, dbPath, "proj_dst", dstDir)
	seedMigrateSession(t, dbPath, "ses_already", "proj_dst", srcDir, "already in dst", 1000)
	seedMigrateSession(t, dbPath, "ses_migrate", "proj_src", srcDir, "needs migrate", 2000)

	r := New(dbPath)
	matches, err := r.Migrate(context.Background(), srcDir, dstDir, 5)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "ses_migrate", matches[0].SessionID)
}

func TestMigrate_TargetProjectMissing(t *testing.T) {
	dbPath, srcDir, _ := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedMigrateSession(t, dbPath, "ses_1", "proj_src", srcDir, "first", 1000)

	r := New(dbPath)
	_, err := r.Migrate(context.Background(), srcDir, "/non/existent/target", 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTargetProjectMissing)
}

func TestMigrate_NoMatchingSessions(t *testing.T) {
	dbPath, srcDir, dstDir := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_dst", dstDir)

	r := New(dbPath)
	_, err := r.Migrate(context.Background(), srcDir, dstDir, 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoMatches)
}

func TestMigrate_RejectsIdenticalPaths(t *testing.T) {
	r := New("/tmp/x.db")
	_, err := r.Migrate(context.Background(), "/same", "/same", 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMatch)
}

func TestMigrate_RequiresPaths(t *testing.T) {
	r := New("/tmp/x.db")
	_, err := r.Migrate(context.Background(), "", "/dst", 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMatch)

	_, err = r.Migrate(context.Background(), "/src", "", 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMatch)
}

func TestMigrate_DefaultCount(t *testing.T) {
	dbPath, srcDir, dstDir := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedProject(t, dbPath, "proj_dst", dstDir)
	for i := 0; i < 10; i++ {
		id := "ses_" + string(rune('a'+i))
		seedMigrateSession(t, dbPath, id, "proj_src", srcDir, id, int64(i*100))
	}

	r := New(dbPath)
	matches, err := r.Migrate(context.Background(), srcDir, dstDir, 0)
	require.NoError(t, err)
	assert.Len(t, matches, DefaultMigrateCount)
}

func TestMigrate_CreatesBackup(t *testing.T) {
	dbPath, srcDir, dstDir := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedProject(t, dbPath, "proj_dst", dstDir)
	seedMigrateSession(t, dbPath, "ses_1", "proj_src", srcDir, "first", 1000)

	r := New(dbPath)
	_, err := r.Migrate(context.Background(), srcDir, dstDir, 5)
	require.NoError(t, err)

	_, statErr := os.Stat(BackupPath(dbPath))
	assert.NoError(t, statErr)
}

func TestPreviewMigrate_ReturnsSessions(t *testing.T) {
	dbPath, srcDir, _ := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedMigrateSession(t, dbPath, "ses_a", "proj_src", srcDir, "alpha", 1000)
	seedMigrateSession(t, dbPath, "ses_b", "proj_src", srcDir, "beta", 2000)
	seedMigrateSession(t, dbPath, "ses_c", "proj_src", srcDir, "gamma", 3000)

	r := New(dbPath)
	preview, err := r.PreviewMigrate(context.Background(), srcDir, 5)
	require.NoError(t, err)
	assert.Len(t, preview, 3)

	titles := []string{preview[0].Title, preview[1].Title, preview[2].Title}
	assert.Equal(t, []string{"gamma", "beta", "alpha"}, titles)
}

func TestPreviewMigrate_RespectsCount(t *testing.T) {
	dbPath, srcDir, dstDir := newMigrateFixture(t)
	seedProject(t, dbPath, "proj_src", srcDir)
	seedProject(t, dbPath, "proj_dst", dstDir)
	seedMigrateSession(t, dbPath, "ses_a", "proj_src", srcDir, "alpha", 1000)
	seedMigrateSession(t, dbPath, "ses_b", "proj_src", srcDir, "beta", 2000)
	seedMigrateSession(t, dbPath, "ses_c", "proj_src", srcDir, "gamma", 3000)

	r := New(dbPath)
	preview, err := r.PreviewMigrate(context.Background(), srcDir, 2)
	require.NoError(t, err)
	assert.Len(t, preview, 2)
}
