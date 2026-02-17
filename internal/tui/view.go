package tui

import (
	"fmt"
	"sort"
	"strings"
)

func (m model) View() string {
	var body string

	if m.needsInit {
		body = m.styles.Panel.Render("Veil is not initialized.\n\nPress i for file key storage or k for keychain.")
	} else {
		switch m.page {
		case pageHome:
			body = m.renderHome()
		case pageProject:
			body = m.renderProject()
		case pageSettings:
			body = m.renderSettings()
		}
	}

	if m.mode != modeNormal && m.mode != modePageSelect && !(m.page == pageProject && m.input.Prompt == "filter> ") {
		body = m.renderInputPanel()
	}

	return m.renderFrame(body)
}

func (m model) renderHome() string {
	var parts []string

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

	syncStatus := "not linked"
	if settings, err := m.svc.LoadSettings(); err == nil {
		if settings.GistID != "" {
			syncStatus = "never"
			if settings.LastSyncedAt != "" {
				syncStatus = settings.LastSyncedAt
			}
		}
	}
	parts = append(parts, m.styles.Muted.Render("  synced "+syncStatus))

	content := strings.Join(parts, "\n")
	panelWidth := 72
	if m.width > 0 {
		panelWidth = max(56, m.width-6)
		if panelWidth > 96 {
			panelWidth = 96
		}
	}
	return m.styles.Panel.Width(panelWidth).Render(content)
}

func (m model) renderProject() string {
	var parts []string

	if m.current != "" {
		info := fmt.Sprintf("  Project: %s", m.current)
		if m.bundle != nil {
			info += fmt.Sprintf(" · %d secrets", len(m.bundle.Secrets))
		}
		parts = append(parts, m.styles.Muted.Render(info))
		parts = append(parts, "")
	}

	body := m.projectTable.View()
	if m.bundle == nil || len(m.bundle.Secrets) == 0 {
		body = "  No secrets"
	}
	parts = append(parts, body)

	return strings.Join(parts, "\n")
}

func (m model) renderSettings() string {
	settings, err := m.svc.LoadSettings()
	if err != nil {
		return "  Failed to load settings: " + err.Error()
	}
	gist := "Not linked"
	if settings.GistID != "" {
		gist = settings.GistID
	}
	syncStatus := settings.LastSyncedAt
	if syncStatus == "" {
		syncStatus = "never"
	}
	content := strings.Join([]string{
		"  GitHub Gist: " + gist,
		"  Last Sync: " + syncStatus,
		"  Machine: " + settings.MachineName,
		"  Key Storage: " + settings.KeyStorage,
		"  Export Default: " + settings.ExportFormat,
	}, "\n")
	return content
}

func (m model) renderFooter() string {
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
	if m.status != "" && m.status != "Ready" {
		help = m.status + " | " + help
	}
	return m.styles.Muted.Render(help)
}
