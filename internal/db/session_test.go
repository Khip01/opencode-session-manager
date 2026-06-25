package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSessions_Empty(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	sessions, err := ListSessions(context.Background(), f.DB)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestListSessions_ReturnsAllSortedByCreated(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	f.InsertSession(t, Session{ID: "ses_1", Directory: "/a", Title: "old", TimeCreated: 1000})
	f.InsertSession(t, Session{ID: "ses_2", Directory: "/b", Title: "new", TimeCreated: 2000})
	f.InsertSession(t, Session{ID: "ses_3", Directory: "/c", Title: "mid", TimeCreated: 1500})

	sessions, err := ListSessions(context.Background(), f.DB)
	require.NoError(t, err)
	require.Len(t, sessions, 3)
	assert.Equal(t, "ses_2", sessions[0].ID)
	assert.Equal(t, "ses_3", sessions[1].ID)
	assert.Equal(t, "ses_1", sessions[2].ID)
}

func TestListSessions_NullableFields(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	f.InsertSession(t, Session{
		ID:        "ses_x",
		Directory: "/x",
		Title:     "no project, no parent",
	})

	sessions, err := ListSessions(context.Background(), f.DB)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "", sessions[0].ProjectID)
	assert.Equal(t, "", sessions[0].ParentID)
	assert.Equal(t, "", sessions[0].Agent)
	assert.False(t, sessions[0].IsSubagent())
	assert.False(t, sessions[0].IsArchived())
}

func TestListActiveSessions_ExcludesSubagentsAndArchived(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	f.InsertSession(t, Session{ID: "ses_main1", Directory: "/m1", Title: "main 1"})
	f.InsertSession(t, Session{ID: "ses_main2", Directory: "/m2", Title: "main 2"})
	f.InsertSession(t, Session{ID: "ses_sub1", Directory: "/s1", Title: "sub", ParentID: "ses_main1"})
	f.InsertSession(t, Session{ID: "ses_arch", Directory: "/a1", Title: "archived", TimeArchived: 1700000000001})

	active, err := ListActiveSessions(context.Background(), f.DB)
	require.NoError(t, err)
	require.Len(t, active, 2)

	ids := []string{active[0].ID, active[1].ID}
	assert.Contains(t, ids, "ses_main1")
	assert.Contains(t, ids, "ses_main2")
}

func TestGetSession_Existing(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	f.InsertSession(t, Session{ID: "ses_target", Directory: "/target", Title: "found", ProjectID: "proj_1", Agent: "build"})

	got, err := GetSession(context.Background(), f.DB, "ses_target")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ses_target", got.ID)
	assert.Equal(t, "found", got.Title)
	assert.Equal(t, "/target", got.Directory)
	assert.Equal(t, "proj_1", got.ProjectID)
	assert.Equal(t, "build", got.Agent)
}

func TestGetSession_NotFound(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	_, err := GetSession(context.Background(), f.DB, "ses_missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestUpdateSessionDirectory_Success(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	f.InsertSession(t, Session{ID: "ses_upd", Directory: "/old", Title: "update me"})

	err := UpdateSessionDirectory(context.Background(), f.DB, "ses_upd", "/new")
	require.NoError(t, err)

	got, err := GetSession(context.Background(), f.DB, "ses_upd")
	require.NoError(t, err)
	assert.Equal(t, "/new", got.Directory)
}

func TestUpdateSessionDirectory_MissingSession(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	err := UpdateSessionDirectory(context.Background(), f.DB, "ses_ghost", "/anywhere")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}
