package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jackhorton/veil/branding"
)

type styles struct {
	Muted   lipgloss.Style
	Accent  lipgloss.Style
	Warn    lipgloss.Style
	Success lipgloss.Style
	Panel   lipgloss.Style
}

func newStyles() styles {
	return styles{
		Muted:   lipgloss.NewStyle().Foreground(branding.Slate),
		Accent:  lipgloss.NewStyle().Foreground(branding.Violet).Bold(true),
		Warn:    lipgloss.NewStyle().Foreground(branding.Amber).Bold(true),
		Success: lipgloss.NewStyle().Foreground(branding.Emerald).Bold(true),
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(branding.Slate).
			Padding(1, 2),
	}
}

var inputBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(branding.Slate).
	Padding(0, 1).
	Width(50)
