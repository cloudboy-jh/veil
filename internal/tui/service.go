package tui

type ProjectSummary struct {
	Name  string
	Path  string
	Count int
}

type Secret struct {
	Key       string
	Value     string
	Group     string
	CreatedAt string
	UpdatedAt string
}

type ProjectBundle struct {
	Project string
	Path    string
	Secrets []Secret
}

type SettingsView struct {
	GistID       string
	LastSyncedAt string
	MachineName  string
	KeyStorage   string
	ExportFormat string
}

type EnvPair struct {
	Key   string
	Value string
}

type Service interface {
	IsInitialized() bool
	Init(keyStorage, machineName string) error
	ListProjects() ([]ProjectSummary, error)
	ResolveProject(projectFlag string) (string, string, error)
	LoadProject(name, path string) (*ProjectBundle, error)
	SaveProject(bundle *ProjectBundle) error
	Sync(token string) error
	LoadSettings() (SettingsView, error)
	ParseEnvContent(content string) ([]EnvPair, error)
	RenderEnv(bundle *ProjectBundle) string
	RenderProjectJSON(bundle *ProjectBundle) (string, error)
}
