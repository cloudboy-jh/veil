package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jackhorton/veil/branding"
)

func (m model) renderHeader(width int) string {
	logo := strings.Split(branding.LogoStyle.Render(branding.WordmarkSmall), "\n")
	if len(logo) > 0 && logo[len(logo)-1] == "" {
		logo = logo[:len(logo)-1]
	}

	tagline := m.styles.Muted.Render("encrypted secrets for developers") + "  " + m.styles.Muted.Render("> "+m.pageLabel())

	metaParts := []string{m.styles.Muted.Render("project:")}
	project := m.current
	if strings.TrimSpace(project) == "" {
		project = "none"
	}
	metaParts = append(metaParts, m.styles.Text.Render(project))
	if !m.hasActiveModal() {
		if s := strings.TrimSpace(m.status); s != "" && s != "Ready" {
			metaParts = append(metaParts, m.styles.Muted.Render("|"), m.styles.Accent.Render(s))
		}
	}
	meta := strings.Join(metaParts, " ")

	content := lipgloss.JoinVertical(lipgloss.Left, strings.Join(logo, "\n"), tagline, meta)
	return m.styles.Header.Width(width).Height(headerHeight).Render(m.fitHeight(content, headerHeight))
}
