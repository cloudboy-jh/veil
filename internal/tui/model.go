package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
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

type model struct {
	svc           Service
	page          page
	mode          inputMode
	width         int
	height        int
	projects      []ProjectSummary
	projectTable  table.Model
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
	styles        styles
}

func newModel(svc Service) model {
	columns := []table.Column{
		{Title: "Group", Width: 14},
		{Title: "Key", Width: 36},
		{Title: "Value", Width: 40},
	}
	tbl := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(12))
	tbl.SetStyles(table.DefaultStyles())

	input := textinput.New()
	input.Prompt = "key=value> "
	input.CharLimit = 8192
	input.Blur()

	m := model{
		svc:          svc,
		page:         pageHome,
		mode:         modeNormal,
		projectTable: tbl,
		input:        input,
		status:       "Ready",
		styles:       newStyles(),
	}
	m.load()
	return m
}

func (m *model) load() {
	m.needsInit = !m.svc.IsInitialized()
	if m.needsInit {
		m.status = "Run init: press i for file key storage or k for keychain"
		return
	}
	projects, err := m.svc.ListProjects()
	if err != nil {
		m.status = err.Error()
		return
	}
	m.projects = projects
	if m.current == "" && len(projects) > 0 {
		m.current = projects[0].Name
	}
	m.loadBundle()
}

func (m *model) loadBundle() {
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
	bundle, err := m.svc.LoadProject(m.current, projectPath)
	if err != nil {
		m.status = err.Error()
		return
	}
	m.bundle = bundle
	m.refreshTable()
}

func (m *model) refreshTable() {
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

func (m *model) ensureCurrentBundle() error {
	if m.bundle != nil {
		return nil
	}
	project, path, err := m.svc.ResolveProject("")
	if err != nil {
		return err
	}
	bundle, err := m.svc.LoadProject(project, path)
	if err != nil {
		return err
	}
	m.current = project
	m.bundle = bundle
	m.refreshTable()
	return nil
}

func (m model) renderInputPanel() string {
	title := "Input"
	switch m.mode {
	case modeAddKey:
		title = "Add Secret"
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
	content := m.styles.Panel.
		Width(m.innerWidth()).
		Render(m.styles.Accent.Render(title) + "\n\n" + m.input.View())
	return m.fitHeight(content, m.contentHeight())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func formatSavedStatus(key string) string {
	return fmt.Sprintf("Saved secret %s", key)
}
