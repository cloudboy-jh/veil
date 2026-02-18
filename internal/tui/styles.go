package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jackhorton/veil/branding"
)

type styles struct {
	Background    lipgloss.Color
	Text          lipgloss.Style
	Muted         lipgloss.Style
	Accent        lipgloss.Style
	Warn          lipgloss.Style
	Success       lipgloss.Style
	Panel         lipgloss.Style
	Frame         lipgloss.Style
	AppBackground lipgloss.Style
	Surface       lipgloss.Style
	Header        lipgloss.Style
	Body          lipgloss.Style
	Footer        lipgloss.Style
	InputBar      lipgloss.Style
	Modal         lipgloss.Style
	ModalBackdrop lipgloss.Style
	InputBox      lipgloss.Style
	SectionTitle  lipgloss.Style
	KeyHints      lipgloss.Style
	StatusInfo    lipgloss.Style
	StatusError   lipgloss.Style
	StatusWarn    lipgloss.Style
}

func newStyles() styles {
	bg := lipgloss.Color("#2E354A")
	panelBg := lipgloss.Color("#2A334B")
	modalBg := lipgloss.Color("#273048")
	inputBg := lipgloss.Color("#242C42")
	fgText := lipgloss.Color("#F4F7FF")
	fgMuted := lipgloss.Color("#A7B4D2")
	border := lipgloss.Color("#9AA8C7")
	accent := lipgloss.Color("#A78BFA")

	return styles{
		Background: bg,
		Text:       lipgloss.NewStyle().Foreground(fgText),
		Muted:      lipgloss.NewStyle().Foreground(fgMuted),
		Accent:     lipgloss.NewStyle().Foreground(accent).Bold(true),
		Warn:       lipgloss.NewStyle().Foreground(branding.Amber).Bold(true),
		Success:    lipgloss.NewStyle().Foreground(branding.Emerald).Bold(true),
		Frame:      lipgloss.NewStyle().Background(bg).Foreground(fgText),
		AppBackground: lipgloss.NewStyle().
			Background(bg).
			Foreground(fgText),
		Header: lipgloss.NewStyle().
			Background(bg).
			Foreground(fgText),
		Body: lipgloss.NewStyle().
			Background(bg).
			Foreground(fgText),
		Footer: lipgloss.NewStyle().
			Background(bg).
			Foreground(fgMuted),
		InputBar: lipgloss.NewStyle().
			Background(bg).
			Foreground(fgMuted),
		ModalBackdrop: lipgloss.NewStyle().
			Background(panelBg).
			Foreground(fgMuted),
		Surface: lipgloss.NewStyle().
			Background(bg).
			Foreground(fgText),
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Background(panelBg).
			Foreground(fgText).
			Padding(1, 2),
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Background(modalBg).
			Foreground(fgText).
			Padding(1, 2),
		InputBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(border).
			Background(inputBg).
			Foreground(fgText).
			Padding(0, 1),
		SectionTitle: lipgloss.NewStyle().
			Foreground(fgMuted).
			Bold(true),
		KeyHints: lipgloss.NewStyle().
			Foreground(fgMuted),
		StatusInfo: lipgloss.NewStyle().
			Foreground(fgText),
		StatusError: lipgloss.NewStyle().
			Foreground(branding.White).
			Background(lipgloss.Color("#8F3A4D")).
			Padding(0, 1),
		StatusWarn: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1B1B1B")).
			Background(branding.Amber).
			Padding(0, 1),
	}
}
