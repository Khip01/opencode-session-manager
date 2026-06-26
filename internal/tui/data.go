package tui

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/Khip01/opencode-session-manager/internal/db"
)

type itemKind int

const (
	itemKindOrphan itemKind = iota
	itemKindActive
)

func (k itemKind) Icon() string {
	switch k {
	case itemKindOrphan:
		return "[!]"
	case itemKindActive:
		return "[ ]"
	default:
		return "[?]"
	}
}

func (k itemKind) Label() string {
	switch k {
	case itemKindOrphan:
		return "Orphans"
	case itemKindActive:
		return "Active"
	default:
		return "Unknown"
	}
}

type sessionItem struct {
	kind    itemKind
	session db.Session
}

func (s sessionItem) Title() string {
	icon := s.kind.Icon()
	return fmt.Sprintf("%s %s  %s", icon, truncate(s.session.ID, 14), truncate(s.session.Title, 36))
}

func (s sessionItem) Description() string {
	dir := s.session.Directory
	if s.kind == itemKindOrphan {
		dir = "[missing] " + dir
	}
	return truncate(dir, 64)
}

func (s sessionItem) FilterValue() string {
	return s.session.Title + " " + s.session.ID + " " + s.session.Directory
}

type dataLoader struct {
	handle *sql.DB
}

func newDataLoader(handle *sql.DB) *dataLoader {
	return &dataLoader{handle: handle}
}

func (d *dataLoader) Load(ctx context.Context) (orphans, active []sessionItem, err error) {
	all, err := db.ListSessions(ctx, d.handle)
	if err != nil {
		return nil, nil, fmt.Errorf("load sessions: %w", err)
	}
	_, err = db.BuildWorktreeIndex(ctx, d.handle)
	if err != nil {
		return nil, nil, fmt.Errorf("load projects: %w", err)
	}

	for _, s := range all {
		if s.IsSubagent() || s.IsArchived() {
			continue
		}
		if directoryExists(s.Directory) {
			active = append(active, sessionItem{kind: itemKindActive, session: s})
		} else {
			orphans = append(orphans, sessionItem{kind: itemKindOrphan, session: s})
		}
	}
	return orphans, active, nil
}

func directoryExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// LoadMessages returns up to `limit` recent messages for a session,
// with their parts attached, suitable for the chat preview pane.
func (d *dataLoader) LoadMessages(ctx context.Context, sessionID string, limit int) ([]db.Message, error) {
	return db.ListMessages(ctx, d.handle, sessionID, limit)
}
