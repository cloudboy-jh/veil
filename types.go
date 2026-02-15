package main

import "time"

const (
	configVersion      = 1
	defaultProjectName = "general"
	serviceName        = "veil"
	githubTokenUser    = "github_token"
)

type Config struct {
	Version      int               `json:"version"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
	Machine      MachineConfig     `json:"machine"`
	KeyStorage   string            `json:"key_storage"`
	KeyFile      string            `json:"key_file,omitempty"`
	Projects     map[string]string `json:"projects"`
	PathProjects map[string]string `json:"path_projects"`
	Recipients   []string          `json:"recipients"`
	Gist         GistConfig        `json:"gist"`
	Prefs        Preferences       `json:"prefs"`
}

type MachineConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	AddedAt   string `json:"added_at"`
}

type GistConfig struct {
	ID           string `json:"id,omitempty"`
	Owner        string `json:"owner,omitempty"`
	LastSyncedAt string `json:"last_synced_at,omitempty"`
}

type Preferences struct {
	ExportFormat string `json:"export_format"`
}

type ProjectBundle struct {
	Project string   `json:"project"`
	Path    string   `json:"path"`
	Secrets []Secret `json:"secrets"`
}

type Secret struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Group     string `json:"group"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
