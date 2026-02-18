package tui

import "strings"

func (m model) renderFooterStatus(width int) string {
	if m.hasActiveModal() {
		return m.styles.Footer.Width(width).Render("")
	}
	status := m.renderStatusLine(m.status, width)
	status = strings.TrimRight(status, "\n")
	return m.styles.Footer.Width(width).Render(status)
}

func (m model) renderFooterHints(width int) string {
	if m.hasActiveModal() {
		return m.styles.Footer.Width(width).Render("")
	}
	hints := m.renderFooter()
	return m.styles.Footer.Width(width).Render(m.renderKeyHints(hints, width))
}
