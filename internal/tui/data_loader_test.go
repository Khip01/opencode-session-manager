package tui

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

const tuiTestSchema = `
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

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	_, err = conn.Exec(tuiTestSchema)
	require.NoError(t, err)
	return conn
}

func TestDataLoader_SeparatesOrphansAndActive(t *testing.T) {
	conn := newTestDB(t)
	defer conn.Close()

	tmp := t.TempDir()
	aliveDir := filepath.Join(tmp, "alive")
	require.NoError(t, os.MkdirAll(aliveDir, 0o755))

	_, err := conn.Exec(
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		"proj_git", aliveDir, 1700000000000,
	)
	require.NoError(t, err)

	inserts := []struct {
		id, pid, dir, title string
		archived           int64
		parent             string
	}{
		{"ses_active1", "proj_git", aliveDir, "active 1", 0, ""},
		{"ses_active2", "", aliveDir, "active 2 no pid", 0, ""},
		{"ses_orphan1", "proj_git", "/does/not/exist", "orphan with valid pid", 0, ""},
		{"ses_orphan2", "", "/also/missing", "orphan no pid", 0, ""},
		{"ses_sub", "proj_git", aliveDir, "subagent", 0, "ses_active1"},
		{"ses_arch", "proj_git", aliveDir, "archived", 1700000000001, ""},
	}
	for _, s := range inserts {
		var pid, par any
		if s.pid != "" {
			pid = s.pid
		}
		if s.parent != "" {
			par = s.parent
		}
		_, err := conn.Exec(
			`INSERT INTO session (id, project_id, parent_id, directory, title, time_created, time_updated, time_archived)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			s.id, pid, par, s.dir, s.title, 1700000000000, 1700000000000, s.archived,
		)
		require.NoError(t, err)
	}

	d := newDataLoader(conn)
	orphans, active, err := d.Load(t.Context())
	require.NoError(t, err)

	orphanIDs := ids(orphans)
	activeIDs := ids(active)

	assert.ElementsMatch(t, []string{"ses_orphan1", "ses_orphan2"}, orphanIDs)
	assert.ElementsMatch(t, []string{"ses_active1", "ses_active2"}, activeIDs)

	for _, o := range orphans {
		assert.Equal(t, itemKindOrphan, o.kind)
	}
	for _, a := range active {
		assert.Equal(t, itemKindActive, a.kind)
	}
}

func TestDataLoader_EmptyDatabase(t *testing.T) {
	conn := newTestDB(t)
	defer conn.Close()

	d := newDataLoader(conn)
	orphans, active, err := d.Load(t.Context())
	require.NoError(t, err)
	assert.Empty(t, orphans)
	assert.Empty(t, active)
}

func ids(items []sessionItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.session.ID
	}
	return out
}
