package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrProjectNotFound = errors.New("project not found")

const projectColumns = `id, worktree, name, vcs`

func ListProjects(ctx context.Context, database *sql.DB) ([]Project, error) {
	rows, err := database.QueryContext(ctx, `SELECT `+projectColumns+` FROM project ORDER BY time_created DESC`)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		var name, vcs sql.NullString
		if err := rows.Scan(&p.ID, &p.Worktree, &name, &vcs); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		if name.Valid {
			p.Name = name.String
		}
		if vcs.Valid {
			p.VCS = vcs.String
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

func GetProject(ctx context.Context, database *sql.DB, id string) (*Project, error) {
	var (
		p    Project
		name sql.NullString
		vcs  sql.NullString
	)
	err := database.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM project WHERE id = ?`, id).
		Scan(&p.ID, &p.Worktree, &name, &vcs)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrProjectNotFound, id)
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	if name.Valid {
		p.Name = name.String
	}
	if vcs.Valid {
		p.VCS = vcs.String
	}
	return &p, nil
}

func BuildWorktreeIndex(ctx context.Context, database *sql.DB) (map[string]string, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT id, worktree FROM project
		WHERE worktree IS NOT NULL AND worktree != '' AND worktree != '/'
	`)
	if err != nil {
		return nil, fmt.Errorf("query worktree index: %w", err)
	}
	defer rows.Close()

	idx := make(map[string]string)
	for rows.Next() {
		var id, worktree string
		if err := rows.Scan(&id, &worktree); err != nil {
			return nil, fmt.Errorf("scan worktree: %w", err)
		}
		idx[id] = worktree
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate worktrees: %w", err)
	}
	return idx, nil
}
