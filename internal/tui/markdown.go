package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type mdStyle struct {
	bold      lipgloss.Style
	italic    lipgloss.Style
	code      lipgloss.Style
	codeBlock lipgloss.Style
	link      lipgloss.Style
	heading   lipgloss.Style
	bullet    lipgloss.Style
	listItem  lipgloss.Style
}

func opencodeMDStyle() mdStyle {
	p := opencodePalette()

	codeBg := lipgloss.NewStyle().
		Background(lipgloss.Color("#1e1e1e")).
		Padding(0, 1).
		Italic(false)

	return mdStyle{
		bold: lipgloss.NewStyle().
			Bold(true),
		italic: lipgloss.NewStyle().
			Italic(true),
		code: codeBg.Copy().
			Foreground(p.green),
		codeBlock: lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1e1e")).
			Foreground(p.fg).
			Padding(0, 2).
			Width(60),
		link: lipgloss.NewStyle().
			Foreground(p.cyan).
			Underline(true),
		heading: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent),
		bullet: lipgloss.NewStyle().
			Foreground(p.warm),
		listItem: lipgloss.NewStyle().
			Foreground(p.fg),
	}
}

func renderMarkdown(text string, width int) string {
	md := opencodeMDStyle()
	var out strings.Builder
	lines := strings.Split(text, "\n")
	inCodeBlock := false
	inList := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				out.WriteString(md.codeBlock.Render(" "))
				out.WriteString("\n")
			} else {
				out.WriteString(md.codeBlock.Render(" "))
				out.WriteString("\n")
			}
			continue
		}

		if inCodeBlock {
			out.WriteString(md.codeBlock.Render(trimmed))
			out.WriteString("\n")
			continue
		}

		if trimmed == "" {
			inList = false
			out.WriteString("\n")
			continue
		}

		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			h := strings.TrimLeft(trimmed, "# ")
			out.WriteString(md.heading.Render(h))
			out.WriteString("\n")
			continue
		}

		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			inList = true
			content := trimmed[2:]
			rendered := renderInline(content, md, width-2)
			out.WriteString("  ")
			out.WriteString(md.bullet.Render("- "))
			out.WriteString(rendered)
			out.WriteString("\n")
			continue
		}

		if inList {
			rendered := renderInline(trimmed, md, width-2)
			out.WriteString("     ")
			out.WriteString(rendered)
			out.WriteString("\n")
			continue
		}

		if strings.HasPrefix(trimmed, "> ") {
			content := trimmed[2:]
			rendered := renderInline(content, md, width-4)
			out.WriteString("  ")
			out.WriteString(md.code.Copy().Padding(0, 0).Background(lipgloss.Color("#141414")).Render("│ "))
			out.WriteString(rendered)
			out.WriteString("\n")
			continue
		}

		rendered := renderInline(line, md, width)
		out.WriteString(rendered)
		out.WriteString("\n")
	}

	return strings.TrimRight(out.String(), "\n")
}

func renderInline(text string, md mdStyle, _ int) string {
	var out strings.Builder
	i := 0

	for i < len(text) {
		if strings.HasPrefix(text[i:], "**") {
			end := findClosing(text, i+2, "**")
			if end > i+2 {
				inner := text[i+2 : end]
				out.WriteString(md.bold.Render(inner))
				i = end + 2
				continue
			}
		}

		if strings.HasPrefix(text[i:], "`") {
			end := findClosing(text, i+1, "`")
			if end > i+1 {
				inner := text[i+1 : end]
				out.WriteString(md.code.
					Copy().
					Padding(0, 0).
					Background(lipgloss.Color("#1e1e1e")).
					Render(inner))
				i = end + 1
				continue
			}
		}

		if strings.HasPrefix(text[i:], "[") {
			// [text](url)
			closeB := findClosing(text, i+1, "]")
			if closeB > i+1 && closeB+1 < len(text) && text[closeB+1] == '(' {
				closeP := findAnyByte(text, closeB+2, ')')
				if closeP > closeB+2 {
					linkText := text[i+1 : closeB]
					out.WriteString(md.link.Render(linkText))
					i = closeP + 1
					continue
				}
			}
		}

		out.WriteByte(text[i])
		i++
	}

	return out.String()
}

func findClosing(s string, start int, delim string) int {
	for j := start; j <= len(s)-len(delim); j++ {
		if s[j:j+len(delim)] == delim {
			return j
		}
	}
	return -1
}

func findAnyByte(s string, start int, b byte) int {
	for j := start; j < len(s); j++ {
		if s[j] == b {
			return j
		}
	}
	return -1
}
