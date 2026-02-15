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
	Quit     key.Binding
	Home     key.Binding
	Project  key.Binding
	Settings key.Binding
	Add      key.Binding
	Edit     key.Binding
	Import   key.Binding
	Export   key.Binding
	Delete   key.Binding
	Reveal   key.Binding
	Filter   key.Binding
	Sync     key.Binding
	Back     key.Binding
	Confirm  key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Home:     key.NewBinding(key.WithKeys("1", "h"), key.WithHelp("1", "home")),
		Project:  key.NewBinding(key.WithKeys("2", "p"), key.WithHelp("2", "project")),
		Settings: key.NewBinding(key.WithKeys("3", "s"), key.WithHelp("3", "settings")),
		Add:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Import:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import")),
		Export:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "export")),
		Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Reveal:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reveal")),
		Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Sync:     key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Confirm:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Home, k.Project, k.Settings, k.Add, k.Delete, k.Reveal, k.Filter, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Home, k.Project, k.Settings, k.Sync},
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
	bodyWidth     int
	bodyHeight    int
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
	input.Prompt = "> "
	input.CharLimit = 8192
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
					m.input.Prompt = "> "
					m.input.Blur()
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
		case "1", "h":
			m.page = pageHome
		case "2", "p":
			m.page = pageProject
		case "3", "s":
			m.page = pageSettings
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
		case "a":
			if m.needsInit {
				break
			}
			if err := m.ensureCurrentBundle(); err != nil {
				m.status = err.Error()
				break
			}
			m.mode = modeAddKey
			m.input.Prompt = "key> "
			m.input.SetValue("")
			m.input.Focus()
			m.status = "Enter key name"
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
		actions := m.renderActionBar()
		return lipgloss.JoinVertical(lipgloss.Left, header, body, actions, m.renderFooter())
	}
	body := m.viewport.View()
	if m.page == pageProject {
		body = m.renderProject()
	}
	if m.mode != modeNormal {
		body = m.renderInputPanel()
	}

	actions := m.renderActionBar()
	parts := []string{header, body, actions}
	parts = append(parts, m.renderFooter())
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m tuiModel) renderHeader() string {
	logo := branding.Render()
	if m.page == pageHome && m.width >= 128 {
		logo = branding.RenderFull()
	}
	logo = strings.Trim(logo, "\n")

	tabs := []string{"1 Home", "2 Project", "3 Settings"}
	for i := range tabs {
		if page(i) == m.page {
			tabs[i] = m.styles.Accent.Render(tabs[i])
		} else {
			tabs[i] = m.styles.Tabs.Render(tabs[i])
		}
	}
	tabBar := strings.Join(tabs, "  |  ")
	header := lipgloss.JoinVertical(lipgloss.Left, logo, tabBar)
	if m.page == pageHome && m.width > 0 {
		return lipgloss.Place(m.width, lipgloss.Height(header), lipgloss.Center, lipgloss.Top, header)
	}
	return header
}

func (m tuiModel) renderHome() string {
	quick := "[a] Add Secret    [S] Sync    [i] Import"
	last := ""
	if m.bundle != nil && len(m.bundle.Secrets) > 0 {
		recent := append([]Secret(nil), m.bundle.Secrets...)
		sort.Slice(recent, func(i, j int) bool { return recent[i].UpdatedAt > recent[j].UpdatedAt })
		limit := 3
		if len(recent) < limit {
			limit = len(recent)
		}
		parts := make([]string, 0, limit)
		for i := 0; i < limit; i++ {
			parts = append(parts, fmt.Sprintf("%s  %s", recent[i].Key, maskValue(recent[i].Value)))
		}
		last = strings.Join(parts, "\n")
	}
	if last == "" {
		last = "No recent secrets"
	}

	syncStatus := "not linked"
	if config, err := m.app.LoadConfig(); err == nil {
		if config.Gist.ID != "" {
			syncStatus = "never"
			if config.Gist.LastSyncedAt != "" {
				syncStatus = config.Gist.LastSyncedAt
			}
		}
	}

	content := strings.Join([]string{
		m.styles.Accent.Render("Quick actions"),
		quick,
		m.styles.Muted.Render("Last 3 secrets"),
		m.styles.Muted.Render(last),
		m.styles.Muted.Render("Sync status: " + syncStatus),
	}, "\n\n")
	panel := m.styles.Panel.Render(content)
	if m.bodyWidth > 0 && m.bodyHeight > 0 {
		return lipgloss.Place(m.bodyWidth, m.bodyHeight, lipgloss.Center, lipgloss.Top, panel)
	}
	return panel
}

func (m tuiModel) renderProject() string {
	if m.current == "" {
		return m.styles.Panel.Render("No project selected")
	}
	meta := fmt.Sprintf("Project: %s", m.current)
	if m.filterQuery != "" {
		meta += "   Filter: " + m.filterQuery
	}
	if m.bundle != nil {
		meta += fmt.Sprintf("   Secrets: %d", len(m.bundle.Secrets))
	}
	body := m.projectTable.View()
	if m.bundle == nil {
		body = "No secrets"
	}
	return m.styles.Panel.Render(meta + "\n\n" + body)
}

func (m tuiModel) renderSettings() string {
	if m.bundle == nil {
		m.bundle = &ProjectBundle{}
	}
	config, err := m.app.LoadConfig()
	if err != nil {
		return m.styles.Panel.Render("Failed to load settings: " + err.Error())
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
		"GitHub Gist: " + gist,
		"Last Sync: " + syncStatus,
		"Machine: " + config.Machine.Name,
		"Key Storage: " + config.KeyStorage,
		"Shell Hooks: bash, zsh, fish, powershell (manual setup)",
		"Export Default: " + config.Prefs.ExportFormat,
	}, "\n")
	return m.styles.Panel.Render(content)
}

func runTUI(app *App) error {
	p := tea.NewProgram(newTUIModel(app), tea.WithAltScreen())
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
	headerH := lipgloss.Height(m.renderHeader())
	actionH := lipgloss.Height(m.renderActionBar())
	footerH := lipgloss.Height(m.renderFooter())
	m.bodyWidth = max(40, m.width)
	m.bodyHeight = max(8, m.height-headerH-actionH-footerH)
	m.viewport.Width = m.bodyWidth
	m.viewport.Height = m.bodyHeight
	m.projectList.SetSize(max(28, m.bodyWidth/3), max(10, m.bodyHeight-2))
	m.projectTable.SetHeight(max(6, m.bodyHeight-4))
	m.projectTable.SetWidth(max(60, m.bodyWidth-6))
	m.setViewportContent()
}

func (m *tuiModel) setViewportContent() {
	var content string
	switch m.page {
	case pageHome:
		content = m.renderHome()
	case pageSettings:
		content = m.renderSettings()
	default:
		content = ""
	}
	m.viewport.SetContent(content)
}

func (m tuiModel) renderFooter() string {
	status := m.styles.Muted.Render(m.status)
	helpView := m.help.View(m.keys)
	return lipgloss.JoinVertical(lipgloss.Left, status, helpView)
}

func (m tuiModel) renderActionBar() string {
	var actions string
	if m.mode != modeNormal {
		actions = "[enter] Confirm  [esc] Cancel"
	} else {
		switch m.page {
		case pageHome:
			actions = "[a] Add  [i] Import  [S] Sync  [2] Project  [3] Settings  [q] Quit"
		case pageProject:
			actions = "[a] Add  [e] Edit  [d] Delete  [r] Reveal  [/] Filter  [i] Import  [x] Export  [S] Sync  [1] Home"
		case pageSettings:
			actions = "[S] Sync  [1] Home  [2] Project  [q] Quit"
		}
	}
	return m.styles.Accent.Render(actions)
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
	panel := m.styles.Panel.Render(m.styles.Accent.Render(title) + "\n\n" + m.input.View())
	if m.bodyWidth > 0 && m.bodyHeight > 0 {
		return lipgloss.Place(m.bodyWidth, m.bodyHeight, lipgloss.Center, lipgloss.Center, panel)
	}
	return panel
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
