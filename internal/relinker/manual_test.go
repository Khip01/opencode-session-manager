package relinker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Khip01/opencode-session-manager/internal/db"
)

func seedForManual(t *testing.T, dbPath string) (oldDir, newDir string) {
	t.Helper()

	tmp := t.TempDir()
	oldDir = filepath.Join(tmp, "old-project")
	newDir = filepath.Join(tmp, "new-project")
	require.NoError(t, os.MkdirAll(newDir, 0o755))

	conn, err := db.Open(dbPath)
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()
	inserts := []db.Session{
		{ID: "ses_old1", ProjectID: "proj_x", Directory: oldDir, Title: "old 1", TimeCreated: 1000, TimeUpdated: 1000},
		{ID: "ses_old2", ProjectID: "proj_x", Directory: oldDir, Title: "old 2", TimeCreated: 2000, TimeUpdated: 2000},
		{ID: "ses_other", ProjectID: "proj_y", Directory: filepath.Join(tmp, "other"), Title: "other", TimeCreated: 3000, TimeUpdated: 3000},
	}
	for _, s := range inserts {
		_, err = conn.ExecContext(ctx,
			`INSERT INTO session (id, project_id, directory, title, time_created, time_updated)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			s.ID, s.ProjectID, s.Directory, s.Title, s.TimeCreated, s.TimeUpdated,
		)
		require.NoError(t, err)
	}
	return oldDir, newDir
}

func TestRelinkByPath_UpdatesMatchingSessions(t *testing.T) {
	dbPath := writeFixtureDB(t)
	oldDir, newDir := seedForManual(t, dbPath)

	r := New(dbPath)
	matches, err := r.RelinkByPath(context.Background(), oldDir, newDir)
	require.NoError(t, err)
	require.Len(t, matches, 2)

	ids := []string{matches[0].SessionID, matches[1].SessionID}
	assert.ElementsMatch(t, []string{"ses_old1", "ses_old2"}, ids)
	for _, m := range matches {
		assert.Equal(t, oldDir, m.OldDirectory)
		assert.Equal(t, newDir, m.NewDirectory)
		assert.Equal(t, StrategyManual, m.Strategy)
	}

	conn, err := db.Open(dbPath)
	require.NoError(t, err)
	defer conn.Close()

	rows, err := conn.QueryContext(context.Background(), `SELECT id, directory FROM session ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	got := map[string]string{}
	for rows.Next() {
		var id, dir string
		require.NoError(t, rows.Scan(&id, &dir))
		got[id] = dir
	}
	assert.Equal(t, newDir, got["ses_old1"])
	assert.Equal(t, newDir, got["ses_old2"])
	assert.Equal(t, filepath.Join(filepath.Dir(oldDir), "other"), got["ses_other"])
}

func TestRelinkByPath_NoMatches(t *testing.T) {
	dbPath := writeFixtureDB(t)
	_, newDir := seedForManual(t, dbPath)

	r := New(dbPath)
	_, err := r.RelinkByPath(context.Background(), "/does/not/exist/anywhere", newDir)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoMatches)
}

func TestRelinkByPath_RejectsIdenticalPaths(t *testing.T) {
	r := New("/tmp/whatever.db")
	_, err := r.RelinkByPath(context.Background(), "/same", "/same")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMatch)
}

func TestRelinkByPath_RequiresBothPaths(t *testing.T) {
	r := New("/tmp/whatever.db")
	_, err := r.RelinkByPath(context.Background(), "", "/new")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMatch)

	_, err = r.RelinkByPath(context.Background(), "/old", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMatch)
}

func TestRelinkByPath_CreatesBackup(t *testing.T) {
	dbPath := writeFixtureDB(t)
	oldDir, newDir := seedForManual(t, dbPath)

	r := New(dbPath)
	_, err := r.RelinkByPath(context.Background(), oldDir, newDir)
	require.NoError(t, err)

	_, statErr := os.Stat(BackupPath(dbPath))
	assert.NoError(t, statErr)
}

func TestPreviewByPath_ReturnsMatchingSessions(t *testing.T) {
	dbPath := writeFixtureDB(t)
	oldDir, _ := seedForManual(t, dbPath)

	r := New(dbPath)
	preview, err := r.PreviewByPath(context.Background(), oldDir)
	require.NoError(t, err)
	require.Len(t, preview, 2)

	ids := []string{preview[0].ID, preview[1].ID}
	assert.ElementsMatch(t, []string{"ses_old1", "ses_old2"}, ids)
}
