package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jackhorton/veil/branding"
)

const (
	headerHeight      = 5
	separatorCount    = 2
	separatorHeight   = 1
	footerHeight      = 1
	inputHeight       = 1
	verticalChrome    = 2
	horizontalChrome  = 2
	minReadableWidth  = 40
	maxReadableWidth  = 120
	minTableHeight    = 3
	projectInfoHeight = 4
)

func (m *model) relayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	tableWidth := max(20, m.innerWidth()-2)
	tableHeight := max(minTableHeight, m.contentHeight()-projectInfoHeight)
	m.projectTable.SetWidth(tableWidth)
	m.projectTable.SetHeight(tableHeight)
}

func (m model) renderFrame(content string) string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	frameInnerWidth := m.frameInnerWidth()
	contentWidth := m.innerWidth()

	header := m.placeSection(m.frameHeader(contentWidth), headerHeight, frameInnerWidth)
	sep := m.placeSection(m.frameSeparator(contentWidth), separatorHeight, frameInnerWidth)
	body := m.placeSection(m.fitHeight(content, m.contentHeight()), m.contentHeight(), frameInnerWidth)
	footer := m.placeSection(m.renderFooterLine(contentWidth), footerHeight, frameInnerWidth)
	input := m.placeSection(m.frameInput(contentWidth), inputHeight, frameInnerWidth)

	inner := lipgloss.JoinVertical(lipgloss.Left, header, sep, body, sep, footer, input)

	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(branding.Slate).
		Width(frameInnerWidth).
		Height(max(1, m.height-verticalChrome)).
		Render(inner)

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, frame)
}

func (m model) frameHeader(width int) string {
	logo := strings.Split(branding.LogoStyle.Render(branding.WordmarkSmall), "\n")
	if len(logo) > 0 && logo[len(logo)-1] == "" {
		logo = logo[:len(logo)-1]
	}

	tagline := branding.TaglineStyle.Render("encrypted secrets for developers") + "    " + m.styles.Muted.Render("> "+m.pageLabel())
	lines := append(logo, tagline)
	return lipgloss.NewStyle().Width(width).Render(m.fitHeight(strings.Join(lines, "\n"), headerHeight))
}

func (m model) frameSeparator(width int) string {
	return strings.Repeat("─", max(1, width))
}

func (m model) frameInput(width int) string {
	return lipgloss.NewStyle().Width(width).Render(m.input.View())
}

func (m model) renderFooterLine(width int) string {
	footer := m.renderFooter()
	return lipgloss.NewStyle().Width(width).Render(footer)
}

func (m model) contentHeight() int {
	h := m.height - verticalChrome - headerHeight - (separatorCount * separatorHeight) - footerHeight - inputHeight
	if h < 0 {
		return 0
	}
	return h
}

func (m model) frameInnerWidth() int {
	w := m.width - horizontalChrome
	if w < 1 {
		return 1
	}
	return w
}

func (m model) innerWidth() int {
	w := m.frameInnerWidth()
	if w < minReadableWidth {
		return w
	}
	if w > maxReadableWidth {
		return maxReadableWidth
	}
	return w
}

func (m model) fitHeight(content string, height int) string {
	if height <= 0 {
		return ""
	}

	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	lines := []string{""}
	if normalized != "" {
		lines = strings.Split(normalized, "\n")
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (m model) placeSection(section string, height, frameInnerWidth int) string {
	if height <= 0 {
		return ""
	}
	return lipgloss.Place(frameInnerWidth, height, lipgloss.Left, lipgloss.Top, section)
}

func (m model) pageLabel() string {
	switch m.page {
	case pageProject:
		return "project"
	case pageSettings:
		return "settings"
	default:
		return "home"
	}
}

func (m *model) resetInputForPage() {
	m.input.SetValue("")
	switch m.page {
	case pageHome:
		m.input.Prompt = "key=value> "
		m.input.Blur()
	case pageProject:
		m.input.Prompt = "filter> "
		m.input.Blur()
	default:
		m.input.Prompt = "> "
		m.input.Blur()
	}
}
