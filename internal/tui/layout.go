package tui

func (m *model) relayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	m.projectTable.SetWidth(max(60, m.width-6))
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
