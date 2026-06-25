package relinker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Khip01/opencode-session-manager/internal/db"
)

const (
	StrategyMigrate = "migrate"
	DefaultMigrateCount = 5
)

var ErrTargetProjectMissing = errors.New("target project not found in database")

func (r *Relinker) Migrate(ctx context.Context, oldPath, newPath string, count int) ([]Match, error) {
	if oldPath == "" || newPath == "" {
		return nil, fmt.Errorf("%w: old and new paths are required", ErrInvalidMatch)
	}
	if count <= 0 {
		count = DefaultMigrateCount
	}
	oldNorm := filepath.Clean(oldPath)
	newNorm := filepath.Clean(newPath)
	if oldNorm == newNorm {
		return nil, fmt.Errorf("%w: old and new paths are identical (%s)", ErrInvalidMatch, oldNorm)
	}

	handle, err := r.open(ctx)
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	var targetProjectID string
	err = handle.QueryRowContext(ctx, `SELECT id FROM project WHERE worktree = ?`, newNorm).Scan(&targetProjectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: no project with worktree %q", ErrTargetProjectMissing, newPath)
		}
		return nil, fmt.Errorf("lookup target project: %w", err)
	}

	rows, err := handle.QueryContext(ctx, `
		SELECT id, title, project_id
		FROM session
		WHERE directory = ?
		ORDER BY time_created DESC
		LIMIT ?
	`, oldNorm, count)
	if err != nil {
		return nil, fmt.Errorf("query sessions to migrate: %w", err)
	}
	defer rows.Close()

	type pending struct {
		id           string
		title        string
		oldProjectID string
	}
	var all []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.title, &p.oldProjectID); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		all = append(all, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("%w: no sessions match directory %q", ErrNoMatches, oldPath)
	}

	toUpdate := make([]pending, 0, len(all))
	for _, p := range all {
		if p.oldProjectID == targetProjectID {
			continue
		}
		toUpdate = append(toUpdate, p)
	}

	if len(toUpdate) == 0 {
		return nil, nil
	}

	if _, err := Backup(r.dbPath); err != nil {
		return nil, fmt.Errorf("backup before write: %w", err)
	}

	tx, err := handle.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE session SET directory = ?, project_id = ? WHERE id = ?`)
	if err != nil {
		return nil, fmt.Errorf("prepare update: %w", err)
	}
	defer stmt.Close()

	matches := make([]Match, 0, len(toUpdate))
	for _, p := range toUpdate {
		if _, err := stmt.ExecContext(ctx, newNorm, targetProjectID, p.id); err != nil {
			return nil, fmt.Errorf("update %s: %w", p.id, err)
		}
		matches = append(matches, Match{
			SessionID:    p.id,
			OldDirectory: oldNorm,
			NewDirectory: newNorm,
			Strategy:     StrategyMigrate,
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return matches, nil
}

func (r *Relinker) PreviewMigrate(ctx context.Context, oldPath string, count int) ([]db.Session, error) {
	if count <= 0 {
		count = DefaultMigrateCount
	}
	oldNorm := filepath.Clean(oldPath)
	handle, err := r.open(ctx)
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	rows, err := handle.QueryContext(ctx, `
		SELECT id, project_id, parent_id, directory, title, agent,
		       time_created, time_updated, time_archived
		FROM session
		WHERE directory = ?
		ORDER BY time_created DESC
		LIMIT ?
	`, oldNorm, count)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var out []db.Session
	for rows.Next() {
		var s db.Session
		var projectID, parentID, agent sql.NullString
		var timeArch sql.NullInt64
		if err := rows.Scan(
			&s.ID, &projectID, &parentID, &s.Directory, &s.Title, &agent,
			&s.TimeCreated, &s.TimeUpdated, &timeArch,
		); err != nil {
			return nil, err
		}
		if projectID.Valid {
			s.ProjectID = projectID.String
		}
		if parentID.Valid {
			s.ParentID = parentID.String
		}
		if agent.Valid {
			s.Agent = agent.String
		}
		if timeArch.Valid {
			s.TimeArchived = timeArch.Int64
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
