package tui

import tea "github.com/charmbracelet/bubbletea"

import appcore "github.com/jackhorton/veil/internal/app"

func Run(svc Service) error {
	p := tea.NewProgram(newModel(svc))
	_, err := p.Run()
	return err
}

func RunTUI(app *appcore.App) error {
	return Run(newTUIService(app))
}
