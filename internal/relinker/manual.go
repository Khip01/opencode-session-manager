package relinker

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Khip01/opencode-session-manager/internal/db"
)

func (r *Relinker) RelinkByPath(ctx context.Context, oldPath, newPath string) ([]Match, error) {
	if oldPath == "" || newPath == "" {
		return nil, fmt.Errorf("%w: old and new paths are required", ErrInvalidMatch)
	}

	oldNorm := filepath.Clean(oldPath)
	newNorm := filepath.Clean(newPath)
	if oldNorm == newNorm {
		return nil, fmt.Errorf("%w: old and new paths are identical (%s)", ErrInvalidMatch, oldNorm)
	}

	if _, err := Backup(r.dbPath); err != nil {
		return nil, fmt.Errorf("backup before write: %w", err)
	}

	handle, err := r.open(ctx)
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	rows, err := handle.QueryContext(ctx, `SELECT id, directory FROM session`)
	if err != nil {
		return nil, fmt.Errorf("query sessions for manual remap: %w", err)
	}
	defer rows.Close()

	type pending struct {
		id  string
		old string
	}
	var toUpdate []pending
	for rows.Next() {
		var id, dir string
		if err := rows.Scan(&id, &dir); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		if filepath.Clean(dir) == oldNorm {
			toUpdate = append(toUpdate, pending{id: id, old: dir})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	if len(toUpdate) == 0 {
		return nil, fmt.Errorf("%w: no sessions match directory %q", ErrNoMatches, oldPath)
	}

	tx, err := handle.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE session SET directory = ? WHERE id = ?`)
	if err != nil {
		return nil, fmt.Errorf("prepare update: %w", err)
	}
	defer stmt.Close()

	matches := make([]Match, 0, len(toUpdate))
	for _, p := range toUpdate {
		if _, err := stmt.ExecContext(ctx, newNorm, p.id); err != nil {
			return nil, fmt.Errorf("update %s: %w", p.id, err)
		}
		matches = append(matches, Match{
			SessionID:    p.id,
			OldDirectory: p.old,
			NewDirectory: newNorm,
			Strategy:     StrategyManual,
		})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return matches, nil
}

func (r *Relinker) PreviewByPath(ctx context.Context, oldPath string) ([]db.Session, error) {
	handle, err := r.open(ctx)
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	oldNorm := filepath.Clean(oldPath)
	rows, err := handle.QueryContext(ctx, `SELECT id, project_id, parent_id, directory, title, agent, time_created, time_updated, time_archived FROM session`)
	if err != nil {
		return nil, fmt.Errorf("query sessions for preview: %w", err)
	}
	defer rows.Close()

	var out []db.Session
	for rows.Next() {
		var s db.Session
		var projectID, parentID, agent *string
		var timeArch int64
		if err := rows.Scan(&s.ID, &projectID, &parentID, &s.Directory, &s.Title, &agent, &s.TimeCreated, &s.TimeUpdated, &timeArch); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		if filepath.Clean(s.Directory) != oldNorm {
			continue
		}
		if projectID != nil {
			s.ProjectID = *projectID
		}
		if parentID != nil {
			s.ParentID = *parentID
		}
		if agent != nil {
			s.Agent = *agent
		}
		s.TimeArchived = timeArch
		out = append(out, s)
	}
	return out, rows.Err()
}
