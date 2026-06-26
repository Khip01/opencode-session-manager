package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Khip01/opencode-session-manager/internal/db"
)

// Chat preview display limits. Tuned for typical 120x40 terminals
// where the right column is ~60 cols and the chat panel is ~25 rows.
// 10 most recent messages total (mix of user and AI), 3 lines each.
// When content overflows the viewport, user scrolls with mouse
// wheel or arrow keys.
const (
	chatPreviewMaxMessages       = 10
	chatPreviewMaxLinesPerMessage = 3
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
	b.WriteString(styles.detailHeader.Render("Chat Preview"))
	b.WriteString("\n\n")

	for i, m := range preview {
		if i > 0 {
			b.WriteString("\n\n")
		}
		renderOneChatMessage(&b, m, styles, width)
	}

	total := len(previewable)
	if total > chatPreviewMaxMessages {
		b.WriteString("\n\n")
		b.WriteString(styles.subtle.Render(
			fmt.Sprintf("... and %d earlier messages", total-chatPreviewMaxMessages)))
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
	var roleLabel string
	switch m.Role {
	case "assistant":
		roleLabel = "ASSISTANT"
	case "user":
		roleLabel = "USER"
	default:
		roleLabel = strings.ToUpper(m.Role)
	}

	var header string
	switch m.Role {
	case "assistant":
		header = styles.detailOK.Render(roleLabel)
	case "user":
		header = styles.detailWarn.Render(roleLabel)
	default:
		header = styles.detailLabel.Render(roleLabel)
	}
	b.WriteString(header)
	b.WriteString("\n")

	textWidth := width - 4
	if textWidth < 20 {
		textWidth = 20
	}
	wrapStyle := styles.detailLabel.Width(textWidth)

	indent := "  "
	for _, p := range m.Parts {
		if p.Type != "text" || p.Text == "" {
			continue
		}
		lines := strings.Split(p.Text, "\n")
		max := chatPreviewMaxLinesPerMessage
		if len(lines) > max {
			lines = lines[:max]
		}
		for _, ln := range lines {
			b.WriteString(indent)
			b.WriteString(wrapStyle.Render(ln))
			b.WriteString("\n")
		}
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
