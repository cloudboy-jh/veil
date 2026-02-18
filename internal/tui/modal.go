package tui

type activeModal struct {
	Title  string
	Detail string
}

func (m model) currentModal() (activeModal, bool) {
	switch m.mode {
	case modeAddKey:
		return activeModal{Title: "Add Secret", Detail: "Use KEY=VALUE format"}, true
	case modeAddValue:
		return activeModal{Title: "Add Secret Value", Detail: "Enter the secret value"}, true
	case modeEditValue:
		return activeModal{Title: "Edit Secret Value", Detail: "Update value and confirm"}, true
	case modeFilter:
		return activeModal{Title: "Filter", Detail: "Filter by key or group"}, true
	case modeImportPath:
		return activeModal{Title: "Import .env", Detail: "Enter path to .env file"}, true
	case modeExportPath:
		return activeModal{Title: "Export", Detail: "Enter export file path"}, true
	default:
		return activeModal{}, false
	}
}

func (m model) hasActiveModal() bool {
	_, ok := m.currentModal()
	return ok
}
