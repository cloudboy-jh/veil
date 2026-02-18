package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var markerFiles = []string{
	"package.json",
	"go.mod",
	"Cargo.toml",
	"pyproject.toml",
	"composer.json",
	"Gemfile",
}

var prefixGroups = map[string]string{
	"OPENAI_":      "API Keys",
	"ANTHROPIC_":   "API Keys",
	"STRIPE_":      "Payments",
	"SUPABASE_":    "Database",
	"DATABASE_":    "Database",
	"AWS_":         "AWS",
	"GITHUB_":      "GitHub",
	"NEXT_PUBLIC_": "Frontend",
	"REDIS_":       "Database",
	"POSTGRES_":    "Database",
}

var invalidProjectName = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

type ProjectSummary struct {
	Name  string
	Path  string
	Count int
}

func (a *App) ResolveProject(projectFlag string) (string, string, error) {
	if _, err := a.LoadConfig(); err != nil {
		return "", "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("read cwd: %w", err)
	}
	cwd = normalizePath(cwd)

	if projectFlag != "" {
		if mapped, ok := a.config.Projects[projectFlag]; ok {
			return projectFlag, mapped, nil
		}
		return projectFlag, cwd, nil
	}

	if marker, err := os.ReadFile(filepath.Join(cwd, ".veil")); err == nil {
		name := strings.TrimSpace(string(marker))
		if name != "" {
			return name, cwd, nil
		}
	}

	type pathMatch struct {
		Path string
		Name string
	}
	matches := make([]pathMatch, 0, len(a.config.PathProjects))
	for path, name := range a.config.PathProjects {
		if strings.HasPrefix(cwd, path) {
			matches = append(matches, pathMatch{Path: path, Name: name})
		}
	}
	sort.Slice(matches, func(i, j int) bool { return len(matches[i].Path) > len(matches[j].Path) })
	if len(matches) > 0 {
		return matches[0].Name, matches[0].Path, nil
	}

	for _, marker := range markerFiles {
		if _, err := os.Stat(filepath.Join(cwd, marker)); err == nil {
			return filepath.Base(cwd), cwd, nil
		}
	}

	base := filepath.Base(cwd)
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = defaultProjectName
	}
	return base, cwd, nil
}

func sanitizeProjectName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultProjectName
	}
	name = invalidProjectName.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return defaultProjectName
	}
	return strings.ToLower(name)
}

func (a *App) projectFilePath(name string) string {
	return filepath.Join(a.StoreDir, sanitizeProjectName(name)+".json.age")
}

func (a *App) LoadProject(name, path string) (*ProjectBundle, error) {
	if !a.IsInitialized() {
		return nil, errors.New("veil is not initialized (run `veil init`)")
	}
	if _, err := a.LoadConfig(); err != nil {
		return nil, err
	}
	identity, err := a.LoadIdentity()
	if err != nil {
		return nil, err
	}
	bundle := &ProjectBundle{
		Project: name,
		Path:    normalizePath(path),
		Secrets: []Secret{},
	}
	filePath := a.projectFilePath(name)
	b, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return bundle, nil
		}
		return nil, fmt.Errorf("read project file: %w", err)
	}
	plain, err := decryptJSON(string(b), identity)
	if err != nil {
		return nil, fmt.Errorf("decrypt project %q: %w", name, err)
	}
	if err := json.Unmarshal(plain, bundle); err != nil {
		return nil, fmt.Errorf("decode project %q: %w", name, err)
	}
	if bundle.Secrets == nil {
		bundle.Secrets = []Secret{}
	}
	if bundle.Path == "" {
		bundle.Path = normalizePath(path)
	}
	return bundle, nil
}

func (a *App) SaveProject(bundle *ProjectBundle) error {
	if _, err := a.LoadConfig(); err != nil {
		return err
	}
	identity, err := a.LoadIdentity()
	if err != nil {
		return err
	}
	bundle.Project = sanitizeProjectName(bundle.Project)
	bundle.Path = normalizePath(bundle.Path)

	recipients := uniqueStrings(append(a.config.Recipients, identity.Recipient().String()))
	a.config.Recipients = recipients

	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("encode project: %w", err)
	}
	ciphertext, err := encryptJSON(data, recipients)
	if err != nil {
		return err
	}
	filePath := a.projectFilePath(bundle.Project)
	if err := os.WriteFile(filePath, []byte(ciphertext), 0o600); err != nil {
		return fmt.Errorf("write project file: %w", err)
	}
	a.registerProject(bundle.Project, bundle.Path)
	return a.SaveConfig()
}

func (a *App) ListProjects() ([]ProjectSummary, error) {
	if _, err := a.LoadConfig(); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for name := range a.config.Projects {
		seen[sanitizeProjectName(name)] = struct{}{}
	}
	entries, err := os.ReadDir(a.StoreDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json.age") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".json.age")
			seen[name] = struct{}{}
		}
	}
	out := make([]ProjectSummary, 0, len(seen))
	for name := range seen {
		path := a.config.Projects[name]
		bundle, err := a.LoadProject(name, path)
		if err != nil {
			continue
		}
		out = append(out, ProjectSummary{Name: name, Path: bundle.Path, Count: len(bundle.Secrets)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func detectGroup(key string) string {
	upper := strings.ToUpper(strings.TrimSpace(key))
	for prefix, group := range prefixGroups {
		if strings.HasPrefix(upper, prefix) {
			return group
		}
	}
	return "General"
}

func UpsertSecret(bundle *ProjectBundle, key, value, group string) (created bool) {
	now := nowRFC3339()
	for i := range bundle.Secrets {
		if bundle.Secrets[i].Key == key {
			bundle.Secrets[i].Value = value
			if group != "" {
				bundle.Secrets[i].Group = group
			}
			bundle.Secrets[i].UpdatedAt = now
			return false
		}
	}
	if group == "" {
		group = detectGroup(key)
	}
	bundle.Secrets = append(bundle.Secrets, Secret{
		Key:       key,
		Value:     value,
		Group:     group,
		CreatedAt: now,
		UpdatedAt: now,
	})
	return true
}

func RemoveSecret(bundle *ProjectBundle, key string) bool {
	for i := range bundle.Secrets {
		if bundle.Secrets[i].Key == key {
			bundle.Secrets = append(bundle.Secrets[:i], bundle.Secrets[i+1:]...)
			return true
		}
	}
	return false
}

func GetSecret(bundle *ProjectBundle, key string) (Secret, bool) {
	for _, secret := range bundle.Secrets {
		if secret.Key == key {
			return secret, true
		}
	}
	return Secret{}, false
}

func MaskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return strings.Repeat("*", len(value))
	}
	return value[:6] + strings.Repeat("*", len(value)-6)
}

func latestUpdate(bundle *ProjectBundle) time.Time {
	var latest time.Time
	for _, secret := range bundle.Secrets {
		t, err := time.Parse(time.RFC3339, secret.UpdatedAt)
		if err != nil {
			continue
		}
		if t.After(latest) {
			latest = t
		}
	}
	return latest
}
