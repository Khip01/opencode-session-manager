package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Khip01/opencode-session-manager/internal/db"
	"charm.land/lipgloss/v2"
)

// Chat preview display limits. Tuned for typical 120x40 terminals
// where the right column is ~60 cols and the chat panel is ~25 rows.
// 10 most recent messages total (mix of user and AI), 20 lines each
// (markdown rendering may expand content).
// When content overflows the viewport, user scrolls with mouse
// wheel or arrow keys.
const (
	chatPreviewMaxMessages       = 10
	chatPreviewMaxLinesPerMessage = 20
)



// renderMetadataPanel returns the metadata text for the top-right
// panel. No panel border is rendered here; the caller wraps with
// the detailPanel style and sets Height.
func renderMetadataPanel(s sessionItem, styles styles) string {
	if (s == sessionItem{}) {
		return styles.detailEmpty.Render("Select a session to see details.")
	}

	var b strings.Builder
	b.WriteString(styles.detailHeader.Render(s.session.Title))
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("ID:          "))
	b.WriteString(s.session.ID)
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("Directory:   "))
	b.WriteString(s.session.Directory)
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("Status:      "))
	if s.kind == itemKindOrphan {
		b.WriteString(styles.detailWarn.Render("ORPHAN, directory does not exist on disk"))
	} else {
		b.WriteString(styles.detailOK.Render("active, directory exists"))
	}
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("Agent:       "))
	b.WriteString(orDash(s.session.Agent))
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("Project ID:  "))
	b.WriteString(orDash(s.session.ProjectID))
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("Subagent:    "))
	b.WriteString(boolText(s.session.IsSubagent()))
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("Archived:    "))
	b.WriteString(boolText(s.session.IsArchived()))
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("Created:     "))
	b.WriteString(formatTime(s.session.TimeCreated))
	b.WriteString("\n")

	b.WriteString(styles.detailLabel.Render("Updated:     "))
	b.WriteString(formatTime(s.session.TimeUpdated))

	return b.String()
}

// renderChatViewportContent builds the text shown in the chat
// preview viewport. It filters to messages with at least one text
// part, shows the last N (most recent) with role labels, and
// truncates each message body to a fixed number of lines so the
// preview stays scannable. The returned string is meant to be fed
// to viewport.SetContent; the caller wraps the viewport in a panel.
func renderChatViewportContent(messages []db.Message, styles styles, width int) string {
	if len(messages) == 0 {
		return styles.detailEmpty.Render("No chat messages in this session.")
	}

	previewable := make([]db.Message, 0, len(messages))
	for _, m := range messages {
		if messageHasText(m) {
			previewable = append(previewable, m)
		}
	}
	if len(previewable) == 0 {
		return styles.subtle.Render("No text content in this session (only tool calls or reasoning).")
	}

	limit := chatPreviewMaxMessages
	if len(previewable) < limit {
		limit = len(previewable)
	}
	preview := previewable[len(previewable)-limit:]

	var b strings.Builder
	total := len(previewable)
	if total > chatPreviewMaxMessages {
		b.WriteString(styles.subtle.Render(
			fmt.Sprintf("... and %d earlier messages\n", total-chatPreviewMaxMessages)))
		b.WriteString(strings.Repeat("─", width-6))
		b.WriteString("\n\n")
	}

	for i, m := range preview {
		if i > 0 {
			b.WriteString("\n\n")
		}
		renderOneChatMessage(&b, m, styles, width)
	}
	return b.String()
}

func messageHasText(m db.Message) bool {
	for _, p := range m.Parts {
		if p.Type == "text" && p.Text != "" {
			return true
		}
	}
	return false
}



func renderOneChatMessage(b *strings.Builder, m db.Message, styles styles, width int) {
	var mdText string
	for _, p := range m.Parts {
		if p.Type == "text" && p.Text != "" {
			mdText += p.Text + " "
		}
	}
	if mdText == "" {
		return
	}

	mdText = strings.ReplaceAll(mdText, "\\n", "\n")
	mdText = strings.ReplaceAll(mdText, "\\\"", "\"")
	mdText = strings.ReplaceAll(mdText, "\\\\", "\\")
	mdText = strings.TrimSpace(mdText)

	renderWidth := width - 8
	if renderWidth < 20 {
		renderWidth = 20
	}

	rendered := renderMarkdown(mdText, renderWidth)
	wrapped := lipgloss.Wrap(rendered, renderWidth, "")

	switch m.Role {
	case "user":
		fw := width - 3
		if fw < 10 {
			fw = 10
		}
		userStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#5c9cf5")).
			Background(lipgloss.Color("#1e1e1e")).
			Width(fw).
			Padding(1, 2, 1, 1)
		b.WriteString(userStyle.Render(wrapped))
		b.WriteString("\n")
	default:
		b.WriteString(wrapped)
		b.WriteString("\n")
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func boolText(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func formatTime(ms int64) string {
	if ms == 0 {
		return "-"
	}
	t := time.UnixMilli(ms)
	return t.Format("2006-01-02 15:04:05 MST")
}
