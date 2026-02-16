package main

import "github.com/jackhorton/veil/internal/tui"

type tuiService struct {
	app *App
}

func newTUIService(app *App) tui.Service {
	return &tuiService{app: app}
}

func (s *tuiService) IsInitialized() bool {
	return s.app.IsInitialized()
}

func (s *tuiService) Init(keyStorage, machineName string) error {
	return s.app.Init(keyStorage, machineName)
}

func (s *tuiService) ListProjects() ([]tui.ProjectSummary, error) {
	projects, err := s.app.ListProjects()
	if err != nil {
		return nil, err
	}
	out := make([]tui.ProjectSummary, 0, len(projects))
	for _, p := range projects {
		out = append(out, tui.ProjectSummary{Name: p.Name, Path: p.Path, Count: p.Count})
	}
	return out, nil
}

func (s *tuiService) ResolveProject(projectFlag string) (string, string, error) {
	return s.app.ResolveProject(projectFlag)
}

func (s *tuiService) LoadProject(name, path string) (*tui.ProjectBundle, error) {
	bundle, err := s.app.LoadProject(name, path)
	if err != nil {
		return nil, err
	}
	return convertBundleToTUI(bundle), nil
}

func (s *tuiService) SaveProject(bundle *tui.ProjectBundle) error {
	return s.app.SaveProject(convertBundleFromTUI(bundle))
}

func (s *tuiService) Sync(token string) error {
	return s.app.Sync(token)
}

func (s *tuiService) LoadSettings() (tui.SettingsView, error) {
	config, err := s.app.LoadConfig()
	if err != nil {
		return tui.SettingsView{}, err
	}
	return tui.SettingsView{
		GistID:       config.Gist.ID,
		LastSyncedAt: config.Gist.LastSyncedAt,
		MachineName:  config.Machine.Name,
		KeyStorage:   config.KeyStorage,
		ExportFormat: config.Prefs.ExportFormat,
	}, nil
}

func (s *tuiService) ParseEnvContent(content string) ([]tui.EnvPair, error) {
	pairs, err := parseEnvContent(content)
	if err != nil {
		return nil, err
	}
	out := make([]tui.EnvPair, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, tui.EnvPair{Key: p.Key, Value: p.Value})
	}
	return out, nil
}

func (s *tuiService) RenderEnv(bundle *tui.ProjectBundle) string {
	return renderEnv(convertBundleFromTUI(bundle))
}

func (s *tuiService) RenderProjectJSON(bundle *tui.ProjectBundle) (string, error) {
	return renderProjectJSON(convertBundleFromTUI(bundle))
}

func convertBundleToTUI(bundle *ProjectBundle) *tui.ProjectBundle {
	secrets := make([]tui.Secret, 0, len(bundle.Secrets))
	for _, sec := range bundle.Secrets {
		secrets = append(secrets, tui.Secret{
			Key:       sec.Key,
			Value:     sec.Value,
			Group:     sec.Group,
			CreatedAt: sec.CreatedAt,
			UpdatedAt: sec.UpdatedAt,
		})
	}
	return &tui.ProjectBundle{Project: bundle.Project, Path: bundle.Path, Secrets: secrets}
}

func convertBundleFromTUI(bundle *tui.ProjectBundle) *ProjectBundle {
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
