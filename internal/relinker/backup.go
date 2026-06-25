package relinker

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const BackupSuffix = ".opencode-sm-backup"

var ErrBackupSourceMissing = errors.New("backup source file not found")

func BackupPath(dbPath string) string {
	return dbPath + BackupSuffix
}

func Backup(dbPath string) (string, error) {
	src, err := os.Open(dbPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrBackupSourceMissing, dbPath)
		}
		return "", fmt.Errorf("open source db: %w", err)
	}
	defer src.Close()

	dstPath := BackupPath(dbPath)
	if err := os.Remove(dstPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("remove old backup: %w", err)
	}

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("copy db to backup: %w", err)
	}
	if err := dst.Sync(); err != nil {
		return "", fmt.Errorf("sync backup: %w", err)
	}
	return dstPath, nil
}
