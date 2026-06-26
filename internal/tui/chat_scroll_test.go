package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Khip01/opencode-session-manager/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pgDownMsg constructs a KeyPressMsg from the rune constant for
// PageDown. tea.Key is a struct; the rune is assigned to its Code
// field.
func pgDownMsg() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown})
}

func pgUpMsg() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp})
}

func homeMsg() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyHome})
}

func endMsg() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd})
}

// TestViewportKeyStrings verifies what String() returns for the
// standard Key codes so we know what string to bind.
func TestViewportKeyStrings(t *testing.T) {
	tests := []struct {
		code     rune
		expected string
	}{
		{tea.KeyPgDown, "pgdown"},
		{tea.KeyPgUp, "pgup"},
		{tea.KeyHome, "home"},
		{tea.KeyEnd, "end"},
	}
	for _, tt := range tests {
		got := tea.KeyPressMsg(tea.Key{Code: tt.code}).String()
		assert.Equal(t, tt.expected, got, "Key{Code:%d}.String()", tt.code)
	}
}

// TestChatScrollKeyboardBindings verifies the four chat scroll
// bindings (PageUp/Down, Home, End) actually scroll the chat
// viewport when received by handleListKey.
func TestChatScrollKeyboardBindings(t *testing.T) {
	m := newModel(Options{Version: "test"}, nil)
	m.quitting = false
	m.mode = modeNone
	m.chatViewport.SetWidth(40)
	m.chatViewport.SetHeight(5)
	m.chatViewport.SetContent(makeContent(40))

	keym := m.keys

	// The Bubble Tea v2 renderer renders Key codes as "pgdown",
	// "pgup", "home", "end" etc., so we bind on those exact strings.
	require.True(t, key.Matches(pgDownMsg(), keym.ChatDown), "PgDn should match ChatDown")
	require.True(t, key.Matches(pgUpMsg(), keym.ChatUp), "PgUp should match ChatUp")
	require.True(t, key.Matches(homeMsg(), keym.ChatTop), "Home should match ChatTop")
	require.True(t, key.Matches(endMsg(), keym.ChatBottom), "End should match ChatBottom")

	// Initial: top
	require.Equal(t, 0, m.chatViewport.YOffset())
	require.False(t, m.chatViewport.AtBottom())

	// PageDown via handleListKey
	m.handleListKey(pgDownMsg())
	assert.Greater(t, m.chatViewport.YOffset(), 0, "PgDn should scroll chat down")

	// PageUp via handleListKey
	offsetAfterPD := m.chatViewport.YOffset()
	m.handleListKey(pgUpMsg())
	assert.Less(t, m.chatViewport.YOffset(), offsetAfterPD, "PgUp should scroll chat up")

	// Home via handleListKey
	m.handleListKey(homeMsg())
	assert.Equal(t, 0, m.chatViewport.YOffset(), "Home should reset to top")

	// End via handleListKey
	m.handleListKey(endMsg())
	assert.True(t, m.chatViewport.AtBottom(), "End should reach bottom")
}

// TestChatScrollShortContentNoop verifies scrolling is a no-op
// when content fits entirely.
func TestChatScrollShortContentNoop(t *testing.T) {
	m := newModel(Options{Version: "test"}, nil)
	m.chatViewport.SetWidth(40)
	m.chatViewport.SetHeight(20)
	m.chatViewport.SetContent("short message")

	m.chatViewport.HalfPageDown()
	assert.Equal(t, 0, m.chatViewport.YOffset(), "no-op when content fits")
	assert.True(t, m.chatViewport.AtBottom())
}

func makeContent(n int) string {
	out := "line"
	for i := 1; i < n; i++ {
		out += "\nline"
	}
	return out
}

// _ unused reference to satisfy linter for the db import
var _ = db.Session{}
