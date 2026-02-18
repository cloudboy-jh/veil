package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	headerHeight      = 6
	separatorCount    = 2
	separatorHeight   = 1
	footerHeight      = 2
	inputHeight       = 1
	verticalChrome    = 0
	horizontalChrome  = 0
	minFrameWidth     = 58
	maxFrameWidth     = 104
	minReadableWidth  = 40
	maxReadableWidth  = 120
	minTableHeight    = 3
	maxContentHeight  = 20
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
	m.resizeProjectColumns(tableWidth)
}

func (m model) renderFrame(content string) string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	frameOuterWidth := m.frameWidth()
	frameInnerWidth := max(1, frameOuterWidth-horizontalChrome)
	contentWidth := m.innerWidth()
	frameHeight := headerHeight + (separatorCount * separatorHeight) + m.contentHeight() + footerHeight + inputHeight
	if m.height > 0 && frameHeight > m.height {
		frameHeight = m.height
	}
	innerHeight := max(1, frameHeight-verticalChrome)

	header := m.placeSection(m.renderHeader(contentWidth), headerHeight, frameInnerWidth)
	sep := m.placeSection(m.frameSeparator(contentWidth), separatorHeight, frameInnerWidth)
	body := m.placeSection(m.styles.Body.Width(frameInnerWidth).Render(m.fitHeight(content, m.contentHeight())), m.contentHeight(), frameInnerWidth)
	footer := m.placeSection(m.frameFooter(contentWidth), footerHeight, frameInnerWidth)
	input := m.placeSection(m.frameInput(contentWidth), inputHeight, frameInnerWidth)

	inner := lipgloss.JoinVertical(lipgloss.Left, header, sep, body, sep, footer, input)
	inner = m.styles.Surface.Width(frameInnerWidth).Height(innerHeight).Render(inner)

	frame := m.styles.Frame.
		Width(frameInnerWidth).
		Height(innerHeight).
		Render(inner)

	placed := lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, frame)
	return m.styles.AppBackground.Width(m.width).Height(m.height).Render(placed)
}

func (m model) frameSeparator(width int) string {
	return m.styles.Muted.Width(width).Render(strings.Repeat("─", max(1, width)))
}

func (m model) frameInput(width int) string {
	if m.hasActiveModal() {
		return m.styles.InputBar.Width(width).Render("")
	}
	return m.styles.InputBar.Width(width).Render(m.input.View())
}

func (m model) frameFooter(width int) string {
	status := m.renderFooterStatus(width)
	hints := m.renderFooterHints(width)
	return lipgloss.JoinVertical(lipgloss.Left, status, hints)
}

func (m model) frameWidth() int {
	w := m.width
	if w < 1 {
		return 1
	}
	if w > maxFrameWidth {
		return maxFrameWidth
	}
	if w < minFrameWidth {
		return w
	}
	return w
}

func (m model) contentHeight() int {
	h := m.height - verticalChrome - headerHeight - (separatorCount * separatorHeight) - footerHeight - inputHeight
	if h < 0 {
		return 0
	}
	if h > maxContentHeight {
		return maxContentHeight
	}
	return h
}

func (m model) frameInnerWidth() int {
	w := m.frameWidth() - horizontalChrome
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
