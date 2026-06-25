package tui

import (
	"testing"

	"github.com/Khip01/opencode-session-manager/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestSessionItem_TitleAndDescription(t *testing.T) {
	s := sessionItem{
		kind: itemKindOrphan,
		session: db.Session{
			ID:        "ses_abc123def456",
			Title:     "Test title",
			Directory: "/very/long/path/that/exceeds/the/limit",
		},
	}

	title := s.Title()
	assert.Contains(t, title, "[!]")
	assert.Contains(t, title, "ses_abc123def")
	assert.Contains(t, title, "Test title")

	desc := s.Description()
	assert.Contains(t, desc, "[missing]")
	assert.Contains(t, desc, "/very/long/path")
}

func TestSessionItem_Title_ActiveKind(t *testing.T) {
	s := sessionItem{
		kind: itemKindActive,
		session: db.Session{
			ID:        "ses_active",
			Title:     "Active",
			Directory: "/home/user/proj",
		},
	}
	assert.Contains(t, s.Title(), "[ ]")
	assert.NotContains(t, s.Description(), "[missing]")
}

func TestSessionItem_FilterValue(t *testing.T) {
	s := sessionItem{
		session: db.Session{
			ID:        "ses_filter",
			Title:     "My filter test",
			Directory: "/some/path",
		},
	}
	fv := s.FilterValue()
	assert.Contains(t, fv, "ses_filter")
	assert.Contains(t, fv, "My filter test")
	assert.Contains(t, fv, "/some/path")
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate("abcdef", 3))
	assert.Equal(t, "abc…", truncate("abcdefgh", 4))
	assert.Equal(t, "short", truncate("short", 10))
	assert.Equal(t, "", truncate("anything", 0))
	assert.Equal(t, "", truncate("", 5))
	assert.Equal(t, "abc", truncate("abc", 3))
}

func TestItemKind_IconAndLabel(t *testing.T) {
	assert.Equal(t, "[!]", itemKindOrphan.Icon())
	assert.Equal(t, "[ ]", itemKindActive.Icon())
	assert.Equal(t, "Orphans", itemKindOrphan.Label())
	assert.Equal(t, "Active", itemKindActive.Label())
}

func TestDirectoryExists(t *testing.T) {
	assert.False(t, directoryExists(""))
	assert.False(t, directoryExists("/this/does/not/exist/anywhere/xyz"))
	assert.True(t, directoryExists(t.TempDir()))
}

func TestKeyMap_Defaults(t *testing.T) {
	km := defaultKeyMap()
	assert.NotEmpty(t, km.Up.Keys())
	assert.NotEmpty(t, km.Down.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.ForceQuit.Keys())
	assert.NotEmpty(t, km.Relink.Keys())
	assert.NotEmpty(t, km.Manual.Keys())
	assert.NotEmpty(t, km.Cancel.Keys())
	assert.NotEmpty(t, km.Enter.Keys())
	assert.NotEmpty(t, km.Yes.Keys())
	assert.NotEmpty(t, km.No.Keys())
}

func TestKeyMap_ShortAndFullHelp(t *testing.T) {
	km := defaultKeyMap()
	assert.NotEmpty(t, km.listShortHelp())
	assert.NotEmpty(t, km.listFullHelp())
	assert.GreaterOrEqual(t, len(km.listFullHelp()), 3)
	assert.NotEmpty(t, km.modalShortHelp())
}
