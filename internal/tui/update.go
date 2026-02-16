package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer m.relayout()

	if m.mode == modePageSelect {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "1":
				m.page = pageHome
				m.mode = modeNormal
				m.resetInputForPage()
				m.status = "Ready"
			case "2":
				m.page = pageProject
				m.mode = modeNormal
				m.resetInputForPage()
				m.status = "Ready"
			case "3":
				m.page = pageSettings
				m.mode = modeNormal
				m.resetInputForPage()
				m.status = "Ready"
			case "esc":
				m.mode = modeNormal
				m.status = "Ready"
				m.resetInputForPage()
			}
			return m, nil
		}
	}

	if m.mode == modeAddKey || m.mode == modeAddValue || m.mode == modeEditValue || m.mode == modeFilter || m.mode == modeImportPath || m.mode == modeExportPath {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		switch keyMsg := msg.(type) {
		case tea.KeyMsg:
			switch keyMsg.String() {
			case "esc":
				m.mode = modeNormal
				m.resetInputForPage()
				m.status = "Cancelled"
			case "enter":
				switch m.mode {
				case modeAddKey:
					if err := m.ensureCurrentBundle(); err != nil {
						m.status = err.Error()
						m.mode = modeNormal
						m.resetInputForPage()
						return m, cmd
					}
					raw := strings.TrimSpace(m.input.Value())
					key, value, ok := strings.Cut(raw, "=")
					key = strings.TrimSpace(key)
					if !ok || key == "" {
						m.status = "Use format KEY=VALUE"
						return m, cmd
					}
					upsertSecret(m.bundle, key, value, "")
					if err := m.svc.SaveProject(m.bundle); err != nil {
						m.status = err.Error()
					} else {
						m.status = formatSavedStatus(key)
						m.load()
					}
					m.pendingKey = ""
					m.pendingValue = ""
					m.mode = modeNormal
					m.resetInputForPage()
				case modeAddValue:
					value := m.input.Value()
					if m.bundle == nil {
						m.status = "Select a project first"
						m.mode = modeNormal
						m.resetInputForPage()
						break
					}
					upsertSecret(m.bundle, m.pendingKey, value, "")
					if err := m.svc.SaveProject(m.bundle); err != nil {
						m.status = err.Error()
					} else {
						m.status = formatSavedStatus(m.pendingKey)
						m.load()
					}
					m.pendingKey = ""
					m.pendingValue = ""
					m.mode = modeNormal
					m.resetInputForPage()
				case modeEditValue:
					value := m.input.Value()
					if m.bundle == nil || m.pendingKey == "" {
						m.status = "Nothing selected"
						m.mode = modeNormal
						m.resetInputForPage()
						break
					}
					upsertSecret(m.bundle, m.pendingKey, value, "")
					if err := m.svc.SaveProject(m.bundle); err != nil {
						m.status = err.Error()
					} else {
						m.status = "Updated secret " + m.pendingKey
						m.load()
					}
					m.pendingKey = ""
					m.pendingValue = ""
					m.mode = modeNormal
					m.resetInputForPage()
				case modeFilter:
					m.filterQuery = strings.TrimSpace(m.input.Value())
					m.mode = modeNormal
					m.resetInputForPage()
					m.status = "Filter applied"
					m.refreshTable()
				case modeImportPath:
					path := strings.TrimSpace(m.input.Value())
					if m.bundle == nil || path == "" {
						m.status = "Import path is required"
						break
					}
					raw, err := os.ReadFile(path)
					if err != nil {
						m.status = err.Error()
						break
					}
					pairs, err := m.svc.ParseEnvContent(string(raw))
					if err != nil {
						m.status = err.Error()
						break
					}
					for _, pair := range pairs {
						upsertSecret(m.bundle, pair.Key, pair.Value, "")
					}
					if err := m.svc.SaveProject(m.bundle); err != nil {
						m.status = err.Error()
					} else {
						m.status = fmt.Sprintf("Imported %d keys", len(pairs))
						m.load()
					}
					m.mode = modeNormal
					m.resetInputForPage()
				case modeExportPath:
					path := strings.TrimSpace(m.input.Value())
					if m.bundle == nil || path == "" {
						m.status = "Export path is required"
						break
					}
					output := m.svc.RenderEnv(m.bundle)
					if strings.HasSuffix(strings.ToLower(path), ".json") {
						jsonOutput, err := m.svc.RenderProjectJSON(m.bundle)
						if err != nil {
							m.status = err.Error()
							break
						}
						output = jsonOutput
					}
					if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
						m.status = err.Error()
						break
					}
					m.mode = modeNormal
					m.resetInputForPage()
					m.status = "Exported to " + path
				}
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.relayout()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.page == pageHome && m.mode == modeNormal {
				m.input.SetValue("")
				m.status = "Ready"
			}
		case "P":
			if m.mode == modeNormal {
				m.mode = modePageSelect
				m.status = "Select page"
			}
		case "l":
			if m.mode == modeNormal && m.page == pageHome {
				m.page = pageProject
				m.resetInputForPage()
				m.status = "Ready"
			}
		case "i":
			if m.needsInit {
				if err := m.svc.Init("file", ""); err != nil {
					m.status = err.Error()
				} else {
					m.status = "Initialized with file key storage"
					m.load()
				}
				break
			}
			if m.page != pageHome && m.page != pageProject {
				break
			}
			if err := m.ensureCurrentBundle(); err != nil {
				m.status = err.Error()
				break
			}
			m.mode = modeImportPath
			m.input.Prompt = "import path (.env)> "
			m.input.SetValue("")
			m.input.Focus()
			m.status = "Enter .env file path"
		case "k":
			if m.needsInit {
				if err := m.svc.Init("keychain", ""); err != nil {
					m.status = err.Error()
				} else {
					m.status = "Initialized with keychain key storage"
					m.load()
				}
			}
		case "S":
			if m.needsInit {
				break
			}
			if err := m.svc.Sync(""); err != nil {
				m.status = err.Error()
			} else {
				m.status = "Synced"
				m.load()
			}
		case "a":
			if m.needsInit {
				break
			}
			if err := m.ensureCurrentBundle(); err != nil {
				m.status = err.Error()
				break
			}
			if m.page == pageHome || m.page == pageProject {
				m.mode = modeAddKey
				m.input.Prompt = "key=value> "
				m.input.SetValue("")
				m.input.Focus()
				m.status = "Enter KEY=VALUE"
			}
		case "e":
			if m.page != pageProject || m.bundle == nil {
				break
			}
			row := m.projectTable.SelectedRow()
			if len(row) < 2 {
				break
			}
			selected, ok := getSecret(m.bundle, row[1])
			if !ok {
				break
			}
			m.pendingKey = selected.Key
			m.pendingValue = selected.Value
			m.mode = modeEditValue
			m.input.Prompt = "value> "
			m.input.SetValue(selected.Value)
			m.input.Focus()
			m.status = "Edit value for " + selected.Key
		case "x":
			if m.page != pageProject || m.bundle == nil || m.needsInit {
				break
			}
			m.mode = modeExportPath
			m.input.Prompt = "export path (.env/.json)> "
			m.input.SetValue(m.current + ".env")
			m.input.Focus()
			m.status = "Enter export destination"
		case "/":
			if m.page == pageProject {
				m.mode = modeFilter
				m.input.Prompt = "filter> "
				m.input.SetValue(m.filterQuery)
				m.input.Focus()
				m.status = "Type filter query"
			}
		case "r":
			if m.page != pageProject || m.bundle == nil {
				break
			}
			row := m.projectTable.SelectedRow()
			if len(row) < 2 {
				break
			}
			if m.revealKey == row[1] {
				m.revealKey = ""
				m.pendingReveal = ""
				m.status = "Masked"
			} else {
				if m.pendingReveal != row[1] {
					m.pendingReveal = row[1]
					m.status = "Press r again to reveal " + row[1]
					break
				}
				m.revealKey = row[1]
				m.pendingReveal = ""
				m.status = "Revealed " + row[1]
			}
			m.refreshTable()
		case "d":
			if m.page != pageProject || m.bundle == nil {
				break
			}
			row := m.projectTable.SelectedRow()
			if len(row) < 2 {
				break
			}
			if m.pendingDelete != row[1] {
				m.pendingDelete = row[1]
				m.status = "Press d again to delete " + row[1]
				break
			}
			m.pendingDelete = ""
			if !removeSecret(m.bundle, row[1]) {
				break
			}
			if err := m.svc.SaveProject(m.bundle); err != nil {
				m.status = err.Error()
			} else {
				m.status = "Deleted " + row[1]
				if m.revealKey == row[1] {
					m.revealKey = ""
				}
				m.load()
			}
		}
	}

	if m.page == pageProject {
		var cmd tea.Cmd
		m.projectTable, cmd = m.projectTable.Update(msg)
		return m, cmd
	}

	return m, nil
}
