// Package relinker implements the core logic for recovering orphaned OpenCode
// sessions and relocating active sessions across project directories.
//
// The Phase 1 (project_id matching) algorithm is conceptually adapted from
// bbl21/opencode-session-recovery (MIT License). See NOTICE for full attribution.
package relinker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/Khip01/opencode-session-manager/internal/db"
)

var (
	ErrNoMatches    = errors.New("no matches found")
	ErrInvalidMatch = errors.New("invalid match: empty session id or directory")
)

const (
	StrategyProjectID = "project_id"
	StrategyManual    = "manual"
)

type Match struct {
	SessionID    string
	OldDirectory string
	NewDirectory string
	Strategy     string
}

type Relinker struct {
	dbPath string
}

func New(dbPath string) *Relinker {
	return &Relinker{dbPath: dbPath}
}

func (r *Relinker) DBPath() string {
	return r.dbPath
}

func (r *Relinker) open() (*sql.DB, error) {
	return db.Open(r.dbPath)
}

func (r *Relinker) FindStaleSessions(ctx context.Context) ([]db.Session, error) {
	handle, err := r.open()
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	sessions, err := db.ListSessions(ctx, handle)
	if err != nil {
		return nil, err
	}

	stale := make([]db.Session, 0, len(sessions))
	for _, s := range sessions {
		if !dirExists(s.Directory) {
			stale = append(stale, s)
		}
	}
	return stale, nil
}

func (r *Relinker) FindPhase1Matches(ctx context.Context) ([]Match, error) {
	handle, err := r.open()
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	worktrees, err := db.BuildWorktreeIndex(ctx, handle)
	if err != nil {
		return nil, fmt.Errorf("build worktree index: %w", err)
	}

	stale, err := r.FindStaleSessions(ctx)
	if err != nil {
		return nil, err
	}

	matches := make([]Match, 0, len(stale))
	for _, s := range stale {
		if s.ProjectID == "" {
			continue
		}
		worktree, ok := worktrees[s.ProjectID]
		if !ok {
			continue
		}
		matches = append(matches, Match{
			SessionID:    s.ID,
			OldDirectory: s.Directory,
			NewDirectory: worktree,
			Strategy:     StrategyProjectID,
		})
	}
	return matches, nil
}

func (r *Relinker) ApplyAll(ctx context.Context, matches []Match) error {
	if len(matches) == 0 {
		return nil
	}
	for _, m := range matches {
		if err := validateMatch(m); err != nil {
			return err
		}
	}

	if _, err := Backup(r.dbPath); err != nil {
		return fmt.Errorf("backup before write: %w", err)
	}

	handle, err := r.open()
	if err != nil {
		return err
	}
	defer handle.Close()

	tx, err := handle.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `UPDATE session SET directory = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("prepare update: %w", err)
	}
	defer stmt.Close()

	for _, m := range matches {
		if _, err := stmt.ExecContext(ctx, m.NewDirectory, m.SessionID); err != nil {
			return fmt.Errorf("update %s: %w", m.SessionID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func validateMatch(m Match) error {
	if m.SessionID == "" || m.NewDirectory == "" {
		return fmt.Errorf("%w: %+v", ErrInvalidMatch, m)
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
