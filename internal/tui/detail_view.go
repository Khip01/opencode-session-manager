package tui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
)

func newDetail(width, height int) viewport.Model {
	return viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
}

func renderDetail(s sessionItem, styles styles) string {
	if (s == sessionItem{}) {
		return styles.detailEmpty.Render("Select a session to see details.")
	}

	var b strings.Builder
	b.WriteString(styles.detailHeader.Render(s.session.Title))
	b.WriteString("\n\n")

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

	if s.kind == itemKindOrphan {
		b.WriteString("\n")
		b.WriteString(styles.detailHint.Render(
			"Tip: use Phase 1 (project_id match) or manual remap to restore this session.",
		))
	}

	return b.String()
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
