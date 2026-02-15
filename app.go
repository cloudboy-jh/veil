package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"filippo.io/age"
	"github.com/zalando/go-keyring"
)

type App struct {
	HomeDir     string
	StoreDir    string
	ConfigPath  string
	config      Config
	configReady bool
	identity    *age.X25519Identity
}

func NewApp() (*App, error) {
	home := os.Getenv("VEIL_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
		home = filepath.Join(userHome, ".veil")
	}
	home = filepath.Clean(home)
	return &App{
		HomeDir:    home,
		StoreDir:   filepath.Join(home, "store"),
		ConfigPath: filepath.Join(home, "config.json"),
	}, nil
}

func (a *App) ensureLayout() error {
	if err := os.MkdirAll(a.StoreDir, 0o700); err != nil {
		return fmt.Errorf("create store directory: %w", err)
	}
	return nil
}

func (a *App) defaultConfig() Config {
	ts := nowRFC3339()
	return Config{
		Version:      configVersion,
		CreatedAt:    ts,
		UpdatedAt:    ts,
		Projects:     map[string]string{},
		PathProjects: map[string]string{},
		Recipients:   []string{},
		Prefs: Preferences{
			ExportFormat: "env",
		},
	}
}

func (a *App) LoadConfig() (*Config, error) {
	if a.configReady {
		return &a.config, nil
	}
	a.config = a.defaultConfig()
	if err := a.ensureLayout(); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(a.ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			a.configReady = true
			return &a.config, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, &a.config); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if a.config.Projects == nil {
		a.config.Projects = map[string]string{}
	}
	if a.config.PathProjects == nil {
		a.config.PathProjects = map[string]string{}
	}
	if a.config.Recipients == nil {
		a.config.Recipients = []string{}
	}
	if a.config.Prefs.ExportFormat == "" {
		a.config.Prefs.ExportFormat = "env"
	}
	a.configReady = true
	return &a.config, nil
}

func (a *App) SaveConfig() error {
	if !a.configReady {
		if _, err := a.LoadConfig(); err != nil {
			return err
		}
	}
	a.config.UpdatedAt = nowRFC3339()
	b, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(a.ConfigPath, b, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (a *App) IsInitialized() bool {
	if !a.configReady {
		_, _ = a.LoadConfig()
	}
	return a.config.Machine.ID != "" && a.config.KeyStorage != ""
}

func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (a *App) Init(keyStorage string, machineName string) error {
	if keyStorage == "" {
		keyStorage = "file"
	}
	if keyStorage != "file" && keyStorage != "keychain" {
		return fmt.Errorf("invalid key storage %q (use file or keychain)", keyStorage)
	}
	if _, err := a.LoadConfig(); err != nil {
		return err
	}
	if machineName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			machineName = "veil-machine"
		} else {
			machineName = hostname
		}
	}
	if a.IsInitialized() {
		return nil
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("generate age identity: %w", err)
	}
	machineID, err := randomID()
	if err != nil {
		return fmt.Errorf("generate machine id: %w", err)
	}
	a.config.Machine = MachineConfig{
		ID:        machineID,
		Name:      machineName,
		PublicKey: id.Recipient().String(),
		AddedAt:   nowRFC3339(),
	}
	a.config.KeyStorage = keyStorage
	a.identity = id
	if err := a.saveIdentity(id); err != nil {
		return err
	}
	a.config.Recipients = uniqueStrings(append(a.config.Recipients, id.Recipient().String()))
	return a.SaveConfig()
}

func (a *App) saveIdentity(id *age.X25519Identity) error {
	if a.config.KeyStorage == "keychain" {
		user := "age_" + a.config.Machine.ID
		if err := keyring.Set(serviceName, user, id.String()); err != nil {
			return fmt.Errorf("save identity to keychain: %w", err)
		}
		a.config.KeyFile = ""
		return nil
	}
	keyDir := filepath.Join(a.HomeDir, "keys")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("create key directory: %w", err)
	}
	path := filepath.Join(keyDir, a.config.Machine.ID+".txt")
	if err := os.WriteFile(path, []byte(id.String()+"\n"), 0o600); err != nil {
		return fmt.Errorf("write identity file: %w", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(path, 0o600)
	}
	a.config.KeyFile = path
	return nil
}

func (a *App) LoadIdentity() (*age.X25519Identity, error) {
	if a.identity != nil {
		return a.identity, nil
	}
	if _, err := a.LoadConfig(); err != nil {
		return nil, err
	}
	if !a.IsInitialized() {
		return nil, errors.New("veil is not initialized (run `veil init`)")
	}
	if a.config.KeyStorage == "keychain" {
		secret, err := keyring.Get(serviceName, "age_"+a.config.Machine.ID)
		if err == nil {
			id, parseErr := age.ParseX25519Identity(strings.TrimSpace(secret))
			if parseErr == nil {
				a.identity = id
				return id, nil
			}
		}
	}
	if a.config.KeyFile == "" {
		return nil, errors.New("missing key file path")
	}
	b, err := os.ReadFile(a.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("read identity file: %w", err)
	}
	id, err := age.ParseX25519Identity(strings.TrimSpace(string(b)))
	if err != nil {
		return nil, fmt.Errorf("parse identity file: %w", err)
	}
	a.identity = id
	return id, nil
}

func (a *App) registerProject(name, path string) {
	if name == "" {
		return
	}
	norm := normalizePath(path)
	a.config.Projects[name] = norm
	a.config.PathProjects[norm] = name
}

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

func uniqueStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
