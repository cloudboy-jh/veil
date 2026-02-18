package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderSectionTitle(title string, width int, info ...string) string {
	label := m.styles.SectionTitle.Render(title)
	lineWidth := width - lipgloss.Width(label) - 1
	extra := strings.TrimSpace(strings.Join(info, " "))
	if extra != "" {
		extra = " " + m.styles.Muted.Render(extra)
		lineWidth -= lipgloss.Width(extra)
	}
	if lineWidth < 0 {
		lineWidth = 0
	}
	line := strings.Repeat("-", lineWidth)
	return label + " " + m.styles.Muted.Render(line) + extra
}

func (m model) renderKeyHints(hints string, width int) string {
	return m.styles.KeyHints.Width(width).Render(hints)
}

func (m model) renderStatusLine(status string, width int) string {
	status = strings.TrimSpace(status)
	if status == "" || strings.EqualFold(status, "Ready") {
		return m.styles.StatusInfo.Width(width).Render("")
	}

	lower := strings.ToLower(status)
	style := m.styles.StatusInfo
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "invalid") {
		style = m.styles.StatusError
	} else if strings.Contains(lower, "warn") || strings.Contains(lower, "cancel") {
		style = m.styles.StatusWarn
	}

	return style.Width(width).Render(status)
}

func (m model) renderInputBlock(title, detail, inputView string) string {
	help := m.styles.Muted.Render("[enter] confirm  [esc] cancel")
	inputLine := m.styles.InputBox.Render(inputView)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Accent.Render(title),
		m.styles.Muted.Render(detail),
		"",
		inputLine,
		"",
		help,
	)
}

func (m model) renderModalShell(content string, width int) string {
	if width < 32 {
		width = 32
	}
	return m.styles.Modal.Width(width).Render(content)
}
