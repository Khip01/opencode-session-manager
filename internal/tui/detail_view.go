package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"

	"github.com/Khip01/opencode-session-manager/internal/db"
)

const (
	chatPreviewMaxMessages = 6
	chatPreviewTextMaxLines = 6
)

func newDetail(width, height int) viewport.Model {
	return viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
}

func renderDetail(s sessionItem, messages []db.Message, styles styles) string {
	if (s == sessionItem{}) {
		return styles.detailEmpty.Render("Select a session to see details.")
	}

	meta := renderMetadata(s, styles)
	chat := renderChatPreview(messages, styles)

	if chat == "" {
		return meta
	}
	return meta + "\n\n" + chat
}

func renderMetadata(s sessionItem, styles styles) string {
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
	b.WriteString("\n")

	return b.String()
}

func renderChatPreview(messages []db.Message, styles styles) string {
	if len(messages) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(styles.detailHeader.Render("Chat Preview"))
	b.WriteString("\n")

	limit := chatPreviewMaxMessages
	if len(messages) < limit {
		limit = len(messages)
	}
	preview := messages[len(messages)-limit:]

	for i, m := range preview {
		if i > 0 {
			b.WriteString("\n")
		}
		renderOneMessage(&b, m, styles)
	}

	if len(messages) > chatPreviewMaxMessages {
		b.WriteString("\n")
		b.WriteString(styles.subtle.Render(
			fmt.Sprintf("... and %d earlier messages (use database tools to view all)",
				len(messages)-chatPreviewMaxMessages)))
	}

	return b.String()
}

func renderOneMessage(b *strings.Builder, m db.Message, styles styles) {
	roleLabel := strings.ToUpper(m.Role)
	if m.Role == "assistant" {
		roleLabel = "ASSISTANT"
	} else if m.Role == "user" {
		roleLabel = "USER"
	}

	var header string
	if m.Role == "assistant" {
		header = styles.detailOK.Render(roleLabel)
	} else if m.Role == "user" {
		header = styles.detailWarn.Render(roleLabel)
	} else {
		header = styles.detailLabel.Render(roleLabel)
	}
	b.WriteString(header)

	for _, p := range m.Parts {
		if p.Type != "text" || p.Text == "" {
			continue
		}
		b.WriteString("\n")
		lines := strings.Split(p.Text, "\n")
		max := chatPreviewTextMaxLines
		if len(lines) > max {
			lines = lines[:max]
			lines = append(lines, fmt.Sprintf("... (%d more lines)", 0))
		}
		for _, ln := range lines {
			b.WriteString("  ")
			b.WriteString(ln)
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
