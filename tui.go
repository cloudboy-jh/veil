package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jackhorton/veil/branding"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type page int

const (
	pageHome page = iota
	pageProject
	pageSettings
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeAddKey
	modeAddValue
	modeEditValue
	modeFilter
	modeImportPath
	modeExportPath
	modePageSelect
)

type projectItem struct {
	name  string
	count int
	path  string
}

func (p projectItem) FilterValue() string { return p.name }
func (p projectItem) Title() string       { return fmt.Sprintf("%s (%d)", p.name, p.count) }
func (p projectItem) Description() string { return p.path }

type keyMap struct {
	Quit    key.Binding
	Pages   key.Binding
	Add     key.Binding
	Edit    key.Binding
	Import  key.Binding
	Export  key.Binding
	Delete  key.Binding
	Reveal  key.Binding
	Filter  key.Binding
	Sync    key.Binding
	Back    key.Binding
	Confirm key.Binding
	List    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Pages:   key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "pages")),
		Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Import:  key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import")),
		Export:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "export")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Reveal:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reveal")),
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Sync:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync")),
		Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		List:    key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "list")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Pages, k.Add, k.Delete, k.Reveal, k.Filter, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Pages, k.Sync, k.List},
		{k.Add, k.Edit, k.Import, k.Export},
		{k.Delete, k.Reveal, k.Filter, k.Back, k.Confirm, k.Quit},
	}
}

type tuiModel struct {
	app           *App
	page          page
	mode          inputMode
	width         int
	height        int
	projects      []ProjectSummary
	projectList   list.Model
	projectTable  table.Model
	viewport      viewport.Model
	help          help.Model
	keys          keyMap
	input         textinput.Model
	status        string
	current       string
	bundle        *ProjectBundle
	filterQuery   string
	pendingKey    string
	pendingValue  string
	revealKey     string
	pendingReveal string
	pendingDelete string
	needsInit     bool
	styles        tuiStyles
}

type tuiStyles struct {
	Muted   lipgloss.Style
	Accent  lipgloss.Style
	Warn    lipgloss.Style
	Success lipgloss.Style
	Panel   lipgloss.Style
	Tabs    lipgloss.Style
}

func newStyles() tuiStyles {
	return tuiStyles{
		Muted:   lipgloss.NewStyle().Foreground(branding.Slate),
		Accent:  lipgloss.NewStyle().Foreground(branding.Violet).Bold(true),
		Warn:    lipgloss.NewStyle().Foreground(branding.Amber).Bold(true),
		Success: lipgloss.NewStyle().Foreground(branding.Emerald).Bold(true),
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(branding.Slate).
			Padding(1, 2),
		Tabs: lipgloss.NewStyle().Foreground(branding.White),
	}
}

func newTUIModel(app *App) tuiModel {
	keys := defaultKeyMap()
	delegate := list.NewDefaultDelegate()
	projectList := list.New([]list.Item{}, delegate, 40, 12)
	projectList.Title = "Projects"
	projectList.SetShowStatusBar(false)
	projectList.SetFilteringEnabled(false)
	projectList.SetShowHelp(false)

	columns := []table.Column{
		{Title: "Group", Width: 14},
		{Title: "Key", Width: 36},
		{Title: "Value", Width: 40},
	}
	tbl := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	tbl.SetStyles(table.DefaultStyles())

	input := textinput.New()
	input.Prompt = "key> "
	input.CharLimit = 8192
	input.Focus()
	vp := viewport.New(0, 0)

	m := tuiModel{
		app:          app,
		page:         pageHome,
		mode:         modeNormal,
		projectList:  projectList,
		projectTable: tbl,
		viewport:     vp,
		help:         help.New(),
		keys:         keys,
		input:        input,
		status:       "Ready",
		styles:       newStyles(),
	}
	m.load()
	return m
}

func (m *tuiModel) load() {
	m.needsInit = !m.app.IsInitialized()
	if m.needsInit {
		m.status = "Run init: press i for file key storage or k for keychain"
		return
	}
	projects, err := m.app.ListProjects()
	if err != nil {
		m.status = err.Error()
		return
	}
	m.projects = projects
	items := make([]list.Item, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectItem{name: project.Name, count: project.Count, path: project.Path})
	}
	m.projectList.SetItems(items)
	if m.current == "" && len(projects) > 0 {
		m.current = projects[0].Name
	}
	m.loadBundle()
}

func (m *tuiModel) loadBundle() {
	if m.current == "" || m.needsInit {
		m.bundle = nil
		m.projectTable.SetRows([]table.Row{})
		return
	}
	projectPath := ""
	for _, summary := range m.projects {
		if summary.Name == m.current {
			projectPath = summary.Path
			break
		}
	}
	bundle, err := m.app.LoadProject(m.current, projectPath)
	if err != nil {
		m.status = err.Error()
		return
	}
	m.bundle = bundle
	m.refreshTable()
}

func (m *tuiModel) refreshTable() {
	if m.bundle == nil {
		m.projectTable.SetRows([]table.Row{})
		return
	}
	secrets := append([]Secret(nil), m.bundle.Secrets...)
	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Group == secrets[j].Group {
			return secrets[i].Key < secrets[j].Key
		}
		return secrets[i].Group < secrets[j].Group
	})
	rows := make([]table.Row, 0, len(secrets))
	query := strings.ToLower(strings.TrimSpace(m.filterQuery))
	for _, secret := range secrets {
		if query != "" && !strings.Contains(strings.ToLower(secret.Key+" "+secret.Group), query) {
			continue
		}
		value := maskValue(secret.Value)
		if m.revealKey == secret.Key {
			value = secret.Value
		}
		rows = append(rows, table.Row{secret.Group, secret.Key, value})
	}
	m.projectTable.SetRows(rows)
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer m.relayout()

	// Handle page select mode first
	if m.mode == modePageSelect {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "1":
				m.page = pageHome
				m.mode = modeNormal
				m.input.Prompt = "key> "
				m.input.SetValue("")
				m.input.Focus()
				m.status = "Ready"
			case "2":
				m.page = pageProject
				m.mode = modeNormal
				m.input.Prompt = "filter> "
				m.input.SetValue("")
				m.input.Focus()
				m.status = "Ready"
			case "3":
				m.page = pageSettings
				m.mode = modeNormal
				m.input.Blur()
				m.status = "Ready"
			case "esc":
				m.mode = modeNormal
				m.status = "Ready"
				if m.page == pageHome || m.page == pageProject {
					m.input.Focus()
				}
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
				m.input.Blur()
				m.input.SetValue("")
				m.status = "Cancelled"
			case "enter":
				switch m.mode {
				case modeAddKey:
					m.pendingKey = strings.TrimSpace(m.input.Value())
					if m.pendingKey == "" {
						m.status = "Key is required"
						return m, cmd
					}
					m.input.SetValue("")
					m.input.Prompt = "value> "
					m.mode = modeAddValue
					m.status = "Enter secret value"
				case modeAddValue:
					value := m.input.Value()
					if m.bundle == nil {
						m.status = "Select a project first"
						m.mode = modeNormal
						break
					}
					upsertSecret(m.bundle, m.pendingKey, value, "")
					if err := m.app.SaveProject(m.bundle); err != nil {
						m.status = err.Error()
					} else {
						m.status = "Saved secret " + m.pendingKey
						m.load()
					}
					m.pendingKey = ""
					m.pendingValue = ""
					m.mode = modeNormal
					m.input.SetValue("")
					// Reset to appropriate prompt based on page
					if m.page == pageHome {
						m.input.Prompt = "key> "
						m.input.Focus()
					} else if m.page == pageProject {
						m.input.Prompt = "filter> "
						m.input.Focus()
					} else {
						m.input.Prompt = "> "
						m.input.Blur()
					}
				case modeEditValue:
					value := m.input.Value()
					if m.bundle == nil || m.pendingKey == "" {
						m.status = "Nothing selected"
						m.mode = modeNormal
						break
					}
					upsertSecret(m.bundle, m.pendingKey, value, "")
					if err := m.app.SaveProject(m.bundle); err != nil {
						m.status = err.Error()
					} else {
						m.status = "Updated secret " + m.pendingKey
						m.load()
					}
					m.pendingKey = ""
					m.pendingValue = ""
					m.mode = modeNormal
					m.input.SetValue("")
					m.input.Prompt = "> "
					m.input.Blur()
				case modeFilter:
					m.filterQuery = strings.TrimSpace(m.input.Value())
					m.mode = modeNormal
					m.input.SetValue("")
					m.input.Blur()
					m.status = "Filter applied"
					m.refreshTable()
				case modeImportPath:
					path := strings.TrimSpace(m.input.Value())
					if m.bundle == nil || path == "" {
						m.status = "Import path is required"
						break
					}
					raw, err := readImportInput(path)
					if err != nil {
						m.status = err.Error()
						break
					}
					pairs, err := parseEnvContent(string(raw))
					if err != nil {
						m.status = err.Error()
						break
					}
					for _, pair := range pairs {
						upsertSecret(m.bundle, pair.Key, pair.Value, "")
					}
					if err := m.app.SaveProject(m.bundle); err != nil {
						m.status = err.Error()
					} else {
						m.status = fmt.Sprintf("Imported %d keys", len(pairs))
						m.load()
					}
					m.mode = modeNormal
					m.input.SetValue("")
					m.input.Prompt = "> "
					m.input.Blur()
				case modeExportPath:
					path := strings.TrimSpace(m.input.Value())
					if m.bundle == nil || path == "" {
						m.status = "Export path is required"
						break
					}
					output := renderEnv(m.bundle)
					if strings.HasSuffix(strings.ToLower(path), ".json") {
						jsonOutput, err := renderProjectJSON(m.bundle)
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
					m.input.SetValue("")
					m.input.Prompt = "> "
					m.input.Blur()
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
		case "P":
			if m.mode == modeNormal {
				m.mode = modePageSelect
				m.status = "Select page"
				m.input.Blur()
			}
		case "l":
			if m.mode == modeNormal && m.page == pageHome {
				m.page = pageProject
				m.input.Prompt = "filter> "
				m.input.SetValue("")
				m.input.Focus()
				m.status = "Ready"
			}
		case "i":
			if m.needsInit {
				if err := m.app.Init("file", ""); err != nil {
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
				if err := m.app.Init("keychain", ""); err != nil {
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
			if err := m.app.Sync(""); err != nil {
				m.status = err.Error()
			} else {
				m.status = "Synced"
				m.load()
			}
		case "enter":
			if m.page == pageHome && m.mode == modeNormal {
				val := strings.TrimSpace(m.input.Value())
				if val != "" {
					if err := m.ensureCurrentBundle(); err != nil {
						m.status = err.Error()
						break
					}
					m.pendingKey = val
					m.input.SetValue("")
					m.input.Prompt = "value> "
					m.mode = modeAddValue
					m.status = "Enter value for " + val
				}
			}
		case "a":
			if m.needsInit {
				break
			}
			if err := m.ensureCurrentBundle(); err != nil {
				m.status = err.Error()
				break
			}
			if m.page == pageProject {
				// On project page, switch filter input to add mode
				m.input.Prompt = "key> "
				m.input.SetValue("")
				m.input.Focus()
				m.status = "Enter key name"
			} else {
				m.mode = modeAddKey
				m.input.Prompt = "key> "
				m.input.SetValue("")
				m.input.Focus()
				m.status = "Enter key name"
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
			if err := m.app.SaveProject(m.bundle); err != nil {
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
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	header := m.renderHeader()
	if m.needsInit {
		body := m.styles.Panel.Render("Veil is not initialized.\n\nPress i for file key storage or k for keychain.")
		return lipgloss.JoinVertical(lipgloss.Left, header, body, m.renderFooter())
	}

	var body string
	switch m.page {
	case pageHome:
		body = m.renderHome()
	case pageProject:
		body = m.renderProject()
	case pageSettings:
		body = m.renderSettings()
	}

	// Show input panel overlay when in input modes (except on project page with filter)
	if m.mode != modeNormal && m.mode != modePageSelect && !(m.page == pageProject && m.input.Prompt == "filter> ") {
		body = m.renderInputPanel()
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, m.renderFooter())
}

var inputBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(branding.Slate).
	Padding(0, 1).
	Width(50)

func (m tuiModel) renderInputBox() string {
	return inputBoxStyle.Render(m.input.View())
}

func (m tuiModel) renderHeader() string {
	return branding.Render()
}

func (m tuiModel) renderHome() string {
	var parts []string

	// Input box
	parts = append(parts, m.renderInputBox())
	parts = append(parts, "")

	// Recent secrets (last 3, muted, no label)
	if m.bundle != nil && len(m.bundle.Secrets) > 0 {
		recent := append([]Secret(nil), m.bundle.Secrets...)
		sort.Slice(recent, func(i, j int) bool { return recent[i].UpdatedAt > recent[j].UpdatedAt })
		limit := 3
		if len(recent) < limit {
			limit = len(recent)
		}
		for i := 0; i < limit; i++ {
			line := fmt.Sprintf("  %s       %s", recent[i].Key, maskValue(recent[i].Value))
			parts = append(parts, m.styles.Muted.Render(line))
		}
		parts = append(parts, "")
	}

	// Sync status (muted)
	syncStatus := "not linked"
	if config, err := m.app.LoadConfig(); err == nil {
		if config.Gist.ID != "" {
			syncStatus = "never"
			if config.Gist.LastSyncedAt != "" {
				syncStatus = config.Gist.LastSyncedAt
			}
		}
	}
	parts = append(parts, m.styles.Muted.Render("  synced "+syncStatus))

	return strings.Join(parts, "\n")
}

func (m tuiModel) renderProject() string {
	var parts []string

	// Filter input box
	parts = append(parts, m.renderInputBox())
	parts = append(parts, "")

	// Project info line (muted)
	if m.current != "" {
		info := fmt.Sprintf("  Project: %s", m.current)
		if m.bundle != nil {
			info += fmt.Sprintf(" · %d secrets", len(m.bundle.Secrets))
		}
		parts = append(parts, m.styles.Muted.Render(info))
		parts = append(parts, "")
	}

	// Table
	body := m.projectTable.View()
	if m.bundle == nil || len(m.bundle.Secrets) == 0 {
		body = "  No secrets"
	}
	parts = append(parts, body)

	return strings.Join(parts, "\n")
}

func (m tuiModel) renderSettings() string {
	config, err := m.app.LoadConfig()
	if err != nil {
		return "  Failed to load settings: " + err.Error()
	}
	gist := "Not linked"
	if config.Gist.ID != "" {
		gist = config.Gist.ID
	}
	syncStatus := config.Gist.LastSyncedAt
	if syncStatus == "" {
		syncStatus = "never"
	}
	content := strings.Join([]string{
		"  GitHub Gist: " + gist,
		"  Last Sync: " + syncStatus,
		"  Machine: " + config.Machine.Name,
		"  Key Storage: " + config.KeyStorage,
		"  Export Default: " + config.Prefs.ExportFormat,
	}, "\n")
	return content
}

func runTUI(app *App) error {
	p := tea.NewProgram(newTUIModel(app))
	_, err := p.Run()
	return err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func readImportInput(path string) ([]byte, error) {
	return os.ReadFile(strings.TrimSpace(path))
}

func (m *tuiModel) relayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	// Only adjust table dimensions based on available width
	m.projectTable.SetWidth(max(60, m.width-6))
}

func (m *tuiModel) setViewportContent() {
	// No longer needed - content rendered directly
}

func (m tuiModel) renderFooter() string {
	var parts []string

	// Status line (only when not Ready)
	if m.status != "Ready" {
		parts = append(parts, m.styles.Muted.Render(m.status))
	}

	// Help bar - context sensitive
	var help string
	if m.mode == modePageSelect {
		help = "[1] home  [2] project  [3] settings  [esc] cancel"
	} else if m.mode != modeNormal {
		help = "[enter] confirm  [esc] cancel"
	} else {
		switch m.page {
		case pageHome:
			help = "[a] add  [i] import  [S] sync  [l] list  [P] pages  [q] quit"
		case pageProject:
			help = "[a] add  [e] edit  [d] delete  [r] reveal  [/] filter  [i] import  [x] export  [S] sync  [P] pages  [q] quit"
		case pageSettings:
			help = "[S] sync  [P] pages  [q] quit"
		}
	}
	parts = append(parts, m.styles.Muted.Render(help))

	return strings.Join(parts, "\n")
}

func (m tuiModel) renderInputPanel() string {
	title := "Input"
	switch m.mode {
	case modeAddKey:
		title = "Add Secret Key"
	case modeAddValue:
		title = "Add Secret Value"
	case modeEditValue:
		title = "Edit Secret Value"
	case modeFilter:
		title = "Filter"
	case modeImportPath:
		title = "Import .env"
	case modeExportPath:
		title = "Export"
	}
	return m.styles.Panel.Render(m.styles.Accent.Render(title) + "\n\n" + m.input.View())
}

func (m *tuiModel) ensureCurrentBundle() error {
	if m.bundle != nil {
		return nil
	}
	project, path, err := m.app.ResolveProject("")
	if err != nil {
		return err
	}
	bundle, err := m.app.LoadProject(project, path)
	if err != nil {
		return err
	}
	m.current = project
	m.bundle = bundle
	m.refreshTable()
	return nil
}
