package tui

import appcore "github.com/jackhorton/veil/internal/app"

type tuiService struct {
	app *appcore.App
}

func newTUIService(app *appcore.App) Service {
	return &tuiService{app: app}
}

func (s *tuiService) IsInitialized() bool {
	return s.app.IsInitialized()
}

func (s *tuiService) Init(keyStorage, machineName string) error {
	return s.app.Init(keyStorage, machineName)
}

func (s *tuiService) ListProjects() ([]ProjectSummary, error) {
	projects, err := s.app.ListProjects()
	if err != nil {
		return nil, err
	}
	out := make([]ProjectSummary, 0, len(projects))
	for _, p := range projects {
		out = append(out, ProjectSummary{Name: p.Name, Path: p.Path, Count: p.Count})
	}
	return out, nil
}

func (s *tuiService) ResolveProject(projectFlag string) (string, string, error) {
	return s.app.ResolveProject(projectFlag)
}

func (s *tuiService) LoadProject(name, path string) (*ProjectBundle, error) {
	bundle, err := s.app.LoadProject(name, path)
	if err != nil {
		return nil, err
	}
	return convertBundleToTUI(bundle), nil
}

func (s *tuiService) SaveProject(bundle *ProjectBundle) error {
	return s.app.SaveProject(convertBundleFromTUI(bundle))
}

func (s *tuiService) Sync(token string) error {
	return s.app.Sync(token)
}

func (s *tuiService) LoadSettings() (SettingsView, error) {
	config, err := s.app.LoadConfig()
	if err != nil {
		return SettingsView{}, err
	}
	return SettingsView{
		GistID:       config.Gist.ID,
		LastSyncedAt: config.Gist.LastSyncedAt,
		MachineName:  config.Machine.Name,
		KeyStorage:   config.KeyStorage,
		ExportFormat: config.Prefs.ExportFormat,
	}, nil
}

func (s *tuiService) ParseEnvContent(content string) ([]EnvPair, error) {
	pairs, err := appcore.ParseEnvContent(content)
	if err != nil {
		return nil, err
	}
	out := make([]EnvPair, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, EnvPair{Key: p.Key, Value: p.Value})
	}
	return out, nil
}

func (s *tuiService) RenderEnv(bundle *ProjectBundle) string {
	return appcore.RenderEnv(convertBundleFromTUI(bundle))
}

func (s *tuiService) RenderProjectJSON(bundle *ProjectBundle) (string, error) {
	return appcore.RenderProjectJSON(convertBundleFromTUI(bundle))
}

func convertBundleToTUI(bundle *appcore.ProjectBundle) *ProjectBundle {
	secrets := make([]Secret, 0, len(bundle.Secrets))
	for _, sec := range bundle.Secrets {
		secrets = append(secrets, Secret{
			Key:       sec.Key,
			Value:     sec.Value,
			Group:     sec.Group,
			CreatedAt: sec.CreatedAt,
			UpdatedAt: sec.UpdatedAt,
		})
	}
	return &ProjectBundle{Project: bundle.Project, Path: bundle.Path, Secrets: secrets}
}

func convertBundleFromTUI(bundle *ProjectBundle) *appcore.ProjectBundle {
	secrets := make([]appcore.Secret, 0, len(bundle.Secrets))
	for _, sec := range bundle.Secrets {
		secrets = append(secrets, appcore.Secret{
			Key:       sec.Key,
			Value:     sec.Value,
			Group:     sec.Group,
			CreatedAt: sec.CreatedAt,
			UpdatedAt: sec.UpdatedAt,
		})
	}
	return &appcore.ProjectBundle{Project: bundle.Project, Path: bundle.Path, Secrets: secrets}
}
