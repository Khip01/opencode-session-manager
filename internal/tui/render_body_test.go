package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"

	"charm.land/lipgloss/v2"

	"github.com/stretchr/testify/assert"
)

// newRenderTestModel builds a minimal model suitable for exercising
// renderBody without a database handle. The list is empty, the chat
// viewport has no content, and the help model is bare.
func newRenderTestModel(width, height int) *model {
	return &model{
		width:  width,
		height: height,
		ready:  true,
		styles: defaultStyles(),
		list: list.New(
			[]list.Item{},
			list.NewDefaultDelegate(),
			width/2-4,
			height-8,
		),
		chatViewport: viewport.New(
			viewport.WithWidth(width/2-4),
			viewport.WithHeight(height-16),
		),
		help: help.New(),
		keys: defaultKeyMap(),
	}
}

// TestRenderBody_DoesNotReadRightBotHField is the regression test for
// the "mendelep" bottom-right panel bug. Before the fix, renderBody
// passed the model field m.rightBotH (which is never assigned) to
// makePanelChat, so the chat panel was rendered with height 0 and
// the right column was visually shorter than the left column. The
// fix uses the locally computed rightBotH variable.
//
// To make this test deterministic we pre-set m.rightBotH to a value
// that would obviously be wrong (much taller than the terminal). If
// renderBody reads the field, the body would balloon to that height.
// With the fix, the field is ignored and the body is constrained to
// a reasonable size.
func TestRenderBody_DoesNotReadRightBotHField(t *testing.T) {
	width := 120
	height := 40
	const bogusFieldValue = 500

	m := newRenderTestModel(width, height)
	m.rightBotH = bogusFieldValue

	body := m.renderBody()

	// lipgloss.Height counts the visible terminal rows of the body.
	// If renderBody read m.rightBotH (the bug), the chat panel alone
	// would be 500 rows tall, making the body at least 500 rows.
	// With the fix the body fits the terminal with room to spare.
	assert.Less(t, lipgloss.Height(body), bogusFieldValue,
		"renderBody must not read m.rightBotH (was set to %d, body would balloon)",
		bogusFieldValue)

	// The chat panel header must still appear in the rendered output.
	plain := stripANSI(body)
	assert.Contains(t, plain, "Chat Preview",
		"chat preview header must be present in the rendered body")
}

// TestRenderBody_ChatPreviewPanelPresent is a smoke test that ensures
// the chat preview header is rendered in the body for a normal
// (120x40) terminal. Before the fix the panel was rendered with
// height 0 and the header was clipped out of the visible body.
func TestRenderBody_ChatPreviewPanelPresent(t *testing.T) {
	m := newRenderTestModel(120, 40)

	body := m.renderBody()
	plain := stripANSI(body)

	assert.Contains(t, plain, "Chat Preview",
		"chat preview header must be present in the rendered body")
}

// stripANSI removes ANSI CSI escape sequences from a string so tests
// can assert on visible text without color codes. Handles the form
// ESC [ ... letter that lipgloss v2 emits.
func stripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) {
				c := s[j]
				j++
				if c >= '@' && c <= '~' {
					break
				}
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
