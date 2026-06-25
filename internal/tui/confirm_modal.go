package tui

import "strings"

func (m modalState) renderConfirm() string {
	var b strings.Builder
	b.WriteString(m.prompt)
	if len(m.warnProcs) > 0 {
		b.WriteString("\n\n")
		b.WriteString("Running opencode instances detected:\n")
		b.WriteString("  ")
		b.WriteString(formatRunningList(m.warnProcs))
		b.WriteString("\n")
		b.WriteString("\nContinuing may risk SQLite lock conflicts.")
	}
	b.WriteString("\n\n[y/enter to confirm, n/esc to cancel]")
	return b.String()
}

func (m modalState) renderResult() string {
	var b strings.Builder
	prefix := "✓ "
	switch m.resultKind {
	case resultWarn:
		prefix = "⚠ "
	case resultErr:
		prefix = "✗ "
	}
	b.WriteString(prefix)
	b.WriteString(m.resultMsg)
	b.WriteString("\n\n[any key to dismiss]")
	return b.String()
}

func (m modalState) renderRunningWarn() string {
	var b strings.Builder
	b.WriteString("OpenCode is currently running.\n\n")
	b.WriteString("Detected processes:\n")
	b.WriteString("  ")
	b.WriteString(formatRunningList(m.warnProcs))
	b.WriteString("\n\n")
	b.WriteString(m.prompt)
	b.WriteString("\n\n[y to kill and continue, n to cancel]")
	return b.String()
}

func truncateModal(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

var _ = strings.TrimSpace
