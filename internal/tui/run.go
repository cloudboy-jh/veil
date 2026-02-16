package tui

import tea "github.com/charmbracelet/bubbletea"

func Run(svc Service) error {
	p := tea.NewProgram(newModel(svc))
	_, err := p.Run()
	return err
}
