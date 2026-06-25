package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrSessionNotFound = errors.New("session not found")

const sessionColumns = `
	id, project_id, parent_id, directory, title, agent,
	time_created, time_updated, time_archived
`

func ListSessions(ctx context.Context, database *sql.DB) ([]Session, error) {
	rows, err := database.QueryContext(ctx, `SELECT `+sessionColumns+` FROM session ORDER BY time_created DESC`)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	return scanSessions(rows)
}

func ListActiveSessions(ctx context.Context, database *sql.DB) ([]Session, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT `+sessionColumns+`
		FROM session
		WHERE parent_id IS NULL AND time_archived = 0
		ORDER BY time_created DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query active sessions: %w", err)
	}
	defer rows.Close()

	return scanSessions(rows)
}

func GetSession(ctx context.Context, database *sql.DB, id string) (*Session, error) {
	row := database.QueryRowContext(ctx, `SELECT `+sessionColumns+` FROM session WHERE id = ?`, id)
	s, err := scanSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, id)
		}
		return nil, err
	}
	return &s, nil
}

func UpdateSessionDirectory(ctx context.Context, database *sql.DB, id, newDirectory string) error {
	res, err := database.ExecContext(ctx,
		`UPDATE session SET directory = ? WHERE id = ?`,
		newDirectory, id,
	)
	if err != nil {
		return fmt.Errorf("update session directory: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, id)
	}
	return nil
}

func scanSessions(rows *sql.Rows) ([]Session, error) {
	var sessions []Session
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return sessions, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSession(s scanner) (Session, error) {
	return scanSessionRow(s)
}

func scanSessionRow(s scanner) (Session, error) {
	var (
		sess        Session
		parentID    sql.NullString
		projectID   sql.NullString
		agent       sql.NullString
		timeArch    sql.NullInt64
	)
	err := s.Scan(
		&sess.ID,
		&projectID,
		&parentID,
		&sess.Directory,
		&sess.Title,
		&agent,
		&sess.TimeCreated,
		&sess.TimeUpdated,
		&timeArch,
	)
	if err != nil {
		return Session{}, err
	}
	if projectID.Valid {
		sess.ProjectID = projectID.String
	}
	if parentID.Valid {
		sess.ParentID = parentID.String
	}
	if agent.Valid {
		sess.Agent = agent.String
	}
	if timeArch.Valid {
		sess.TimeArchived = timeArch.Int64
	}
	return sess, nil
}
