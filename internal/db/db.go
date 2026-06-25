package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const (
	defaultRelativePath = ".local/share/opencode/opencode.db"
	defaultFileMode      = os.FileMode(0o644)
)

var (
	ErrDBNotFound = errors.New("opencode database not found")
	ErrDBNoAccess = errors.New("no read permission for opencode database")
)

func ResolvePath(override string) (string, error) {
	if override != "" {
		return filepath.Clean(override), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, defaultRelativePath), nil
}

func Exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func Accessible(path string) error {
	if !Exists(path) {
		return fmt.Errorf("%w: %s", ErrDBNotFound, path)
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("%w: %s", ErrDBNoAccess, path)
		}
		return fmt.Errorf("open database file: %w", err)
	}
	_ = f.Close()
	return nil
}

func Open(path string) (*sql.DB, error) {
	if err := Accessible(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite handle: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

func DefaultMode() os.FileMode {
	return defaultFileMode
}
