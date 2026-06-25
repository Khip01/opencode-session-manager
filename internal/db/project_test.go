package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects_Empty(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	projects, err := ListProjects(context.Background(), f.DB)
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestListProjects_ReturnsAll(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	f.InsertProject(t, "proj_a", "/path/a")
	f.InsertProject(t, "proj_b", "/path/b")

	projects, err := ListProjects(context.Background(), f.DB)
	require.NoError(t, err)
	assert.Len(t, projects, 2)
}

func TestGetProject_Existing(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	f.InsertProject(t, "proj_x", "/path/x")

	got, err := GetProject(context.Background(), f.DB, "proj_x")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "proj_x", got.ID)
	assert.Equal(t, "/path/x", got.Worktree)
}

func TestGetProject_NotFound(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	_, err := GetProject(context.Background(), f.DB, "proj_missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestBuildWorktreeIndex_ExcludesRootAndEmpty(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	f.InsertProject(t, "proj_good", "/valid/path")
	f.InsertProject(t, "proj_root", "/")
	f.InsertProject(t, "proj_empty", "")

	idx, err := BuildWorktreeIndex(context.Background(), f.DB)
	require.NoError(t, err)
	assert.Len(t, idx, 1)
	assert.Equal(t, "/valid/path", idx["proj_good"])
}

func TestBuildWorktreeIndex_EmptyDB(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	idx, err := BuildWorktreeIndex(context.Background(), f.DB)
	require.NoError(t, err)
	assert.NotNil(t, idx)
	assert.Empty(t, idx)
}
