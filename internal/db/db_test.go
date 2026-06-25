package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePath_WithOverride(t *testing.T) {
	got, err := ResolvePath("/tmp/custom.db")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/custom.db", got)
}

func TestResolvePath_WithRelativeOverride(t *testing.T) {
	got, err := ResolvePath("./relative/path.db")
	require.NoError(t, err)
	assert.Equal(t, "relative/path.db", got)
}

func TestResolvePath_DefaultUsesHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	expected := filepath.Join(home, defaultRelativePath)

	got, err := ResolvePath("")
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestExists_MissingFile(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, Exists(filepath.Join(dir, "does-not-exist.db")))
}

func TestExists_RealFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.db")
	require.NoError(t, os.WriteFile(path, []byte("sqlite"), DefaultMode()))

	assert.True(t, Exists(path))
}

func TestExists_DirectoryIsNotFile(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, Exists(dir))
}

func TestAccessible_MissingFile(t *testing.T) {
	dir := t.TempDir()
	err := Accessible(filepath.Join(dir, "missing.db"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDBNotFound)
}

func TestAccessible_RealFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.db")
	require.NoError(t, os.WriteFile(path, []byte("sqlite"), DefaultMode()))

	assert.NoError(t, Accessible(path))
}

func TestOpen_RejectsMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Open(filepath.Join(dir, "missing.db"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDBNotFound)
}
