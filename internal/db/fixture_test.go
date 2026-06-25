package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

const testSchema = `
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
`

type Fixture struct {
	DB        *sql.DB
	TmpDir    string
	Projects  []Project
	Sessions  []Session
	CloseFunc func()
}

func NewFixture(t *testing.T) *Fixture {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}

	if _, err := db.Exec(testSchema); err != nil {
		_ = db.Close()
		t.Fatalf("apply test schema: %v", err)
	}

	tmpDir := t.TempDir()

	return &Fixture{
		DB:     db,
		TmpDir: tmpDir,
		CloseFunc: func() {
			_ = db.Close()
		},
	}
}

func (f *Fixture) Close() {
	if f.CloseFunc != nil {
		f.CloseFunc()
	}
}

func (f *Fixture) MakeProjectDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(f.TmpDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	return dir
}

func (f *Fixture) InsertProject(t *testing.T, id, worktree string) {
	t.Helper()
	_, err := f.DB.ExecContext(context.Background(),
		`INSERT INTO project (id, worktree, time_created) VALUES (?, ?, ?)`,
		id, worktree, 1700000000000,
	)
	if err != nil {
		t.Fatalf("insert project %s: %v", id, err)
	}
}

func (f *Fixture) InsertSession(t *testing.T, s Session) {
	t.Helper()
	var projectID, parentID, agent any
	if s.ProjectID != "" {
		projectID = s.ProjectID
	}
	if s.ParentID != "" {
		parentID = s.ParentID
	}
	if s.Agent != "" {
		agent = s.Agent
	}
	if s.TimeCreated == 0 {
		s.TimeCreated = 1700000000000
	}
	if s.TimeUpdated == 0 {
		s.TimeUpdated = 1700000000000
	}
	_, err := f.DB.ExecContext(context.Background(),
		`INSERT INTO session (id, project_id, parent_id, directory, title, agent, time_created, time_updated, time_archived)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, projectID, parentID, s.Directory, s.Title, agent, s.TimeCreated, s.TimeUpdated, s.TimeArchived,
	)
	if err != nil {
		t.Fatalf("insert session %s: %v", s.ID, err)
	}
}
