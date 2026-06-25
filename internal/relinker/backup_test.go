package relinker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupPath_AppendsSuffix(t *testing.T) {
	got := BackupPath("/tmp/opencode.db")
	assert.Equal(t, "/tmp/opencode.db.opencode-sm-backup", got)
}

func TestBackup_SourceMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := Backup(filepath.Join(dir, "missing.db"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBackupSourceMissing)
}

func TestBackup_CreatesIdenticalCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "opencode.db")
	original := []byte("SQLite-format-content-here")
	require.NoError(t, os.WriteFile(src, original, 0o644))

	backupPath, err := Backup(src)
	require.NoError(t, err)
	assert.Equal(t, src+BackupSuffix, backupPath)

	got, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, original, got)
}

func TestBackup_OverwritesExistingBackup(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "opencode.db")
	backupPath := src + BackupSuffix

	require.NoError(t, os.WriteFile(src, []byte("new content"), 0o644))
	require.NoError(t, os.WriteFile(backupPath, []byte("stale backup"), 0o644))

	_, err := Backup(src)
	require.NoError(t, err)

	got, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("new content"), got)
}

func TestBackup_PreservesFileContent_AfterDBWrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "opencode.db")
	require.NoError(t, os.WriteFile(src, []byte("before-write"), 0o644))

	backupPath, err := Backup(src)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(src, []byte("after-write"), 0o644))

	got, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("before-write"), got)
}
