package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
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
	m.projectTable.SetStyles(m.tableStyles())
	m.load()
	return m
}

func (m model) tableStyles() table.Styles {
	s := table.DefaultStyles()
	bg := m.styles.Background
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#9AA8C7")).
		Foreground(lipgloss.Color("#DCE5FF")).
		Background(bg).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#F4F7FF")).
		Background(lipgloss.Color("#4A5572")).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(lipgloss.Color("#D1DBF4")).
		Background(bg)
	return s
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
	modal, ok := m.currentModal()
	if !ok {
		return ""
	}
	return m.renderInputBlock(modal.Title, modal.Detail, m.input.View())
}

func (m *model) resizeProjectColumns(tableWidth int) {
	if tableWidth <= 0 {
		return
	}

	available := max(24, tableWidth-4)
	groupWidth := 10
	keyWidth := 18
	valueWidth := 16
	baseTotal := groupWidth + keyWidth + valueWidth

	if available > baseTotal {
		extra := available - baseTotal
		keyExtra := extra * 55 / 100
		valueExtra := extra * 35 / 100
		groupExtra := extra - keyExtra - valueExtra
		groupWidth += groupExtra
		keyWidth += keyExtra
		valueWidth += valueExtra
	} else if available < baseTotal {
		groupWidth = max(8, available/5)
		remaining := max(12, available-groupWidth)
		keyWidth = max(12, remaining/2)
		valueWidth = max(8, available-groupWidth-keyWidth)
		if groupWidth+keyWidth+valueWidth > available {
			over := groupWidth + keyWidth + valueWidth - available
			if keyWidth-over >= 12 {
				keyWidth -= over
			} else {
				valueWidth = max(8, valueWidth-over)
			}
		}
	}

	m.projectTable.SetColumns([]table.Column{
		{Title: "Group", Width: groupWidth},
		{Title: "Key", Width: keyWidth},
		{Title: "Value", Width: valueWidth},
	})
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
