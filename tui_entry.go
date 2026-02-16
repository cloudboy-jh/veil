package main

import "github.com/jackhorton/veil/internal/tui"

func runTUI(app *App) error {
	return tui.Run(newTUIService(app))
}
