package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/zalando/go-keyring"
)

const recipientsFileName = "recipients.txt"

type gistFile struct {
	Filename  string `json:"filename"`
	Type      string `json:"type"`
	Language  string `json:"language"`
	RawURL    string `json:"raw_url"`
	Size      int    `json:"size"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
}

type gistResponse struct {
	ID    string              `json:"id"`
	Files map[string]gistFile `json:"files"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (a *App) LoadGitHubToken() (string, error) {
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value, nil
		}
	}
	if token, err := keyring.Get(serviceName, githubTokenUser); err == nil && strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token), nil
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		token := strings.TrimSpace(string(out))
		if token != "" {
			return token, nil
		}
	}
	clientID := strings.TrimSpace(os.Getenv("VEIL_GITHUB_CLIENT_ID"))
	if clientID != "" {
		token, flowErr := githubDeviceFlow(clientID)
		if flowErr == nil {
			_ = a.StoreGitHubToken(token)
			return token, nil
		}
	}
	return "", errors.New("missing GitHub token: set GH_TOKEN/GITHUB_TOKEN, run `gh auth login`, or set VEIL_GITHUB_CLIENT_ID for device flow")
}

func (a *App) StoreGitHubToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("empty token")
	}
	if err := keyring.Set(serviceName, githubTokenUser, token); err != nil {
		return fmt.Errorf("store token in keychain: %w", err)
	}
	return nil
}

func githubDeviceFlow(clientID string) (string, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", "gist read:user")
	req, err := http.NewRequest(http.MethodPost, "https://github.com/login/device/code", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("device code request failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var code struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		Interval                int    `json:"interval"`
		ExpiresIn               int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&code); err != nil {
		return "", err
	}
	verificationURL := code.VerificationURIComplete
	if verificationURL == "" {
		verificationURL = code.VerificationURI
	}
	if verificationURL != "" {
		if qr, err := qrcode.New(verificationURL, qrcode.Low); err == nil {
			fmt.Println(qr.ToSmallString(false))
		}
	}
	fmt.Printf("Open: %s\nCode: %s\n", code.VerificationURI, code.UserCode)

	interval := time.Duration(code.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		form := url.Values{}
		form.Set("client_id", clientID)
		form.Set("device_code", code.DeviceCode)
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		pollReq, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
		pollReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		pollReq.Header.Set("Accept", "application/json")
		pollResp, err := http.DefaultClient.Do(pollReq)
		if err != nil {
			continue
		}
		var payload struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
		}
		_ = json.NewDecoder(pollResp.Body).Decode(&payload)
		_ = pollResp.Body.Close()
		if payload.AccessToken != "" {
			return payload.AccessToken, nil
		}
		switch payload.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 2 * time.Second
		default:
			if payload.Error != "" {
				return "", errors.New(payload.Error)
			}
		}
	}
	return "", errors.New("device flow timed out")
}

func githubRequest(token, method, endpoint string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultClient.Do(req)
}

func fetchRawContent(rawURL string) (string, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch raw content: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getGist(token, gistID string) (*gistResponse, error) {
	resp, err := githubRequest(token, http.MethodGet, "https://api.github.com/gists/"+gistID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github gist get failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var gist gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		return nil, err
	}
	return &gist, nil
}

func createGist(token string, files map[string]string) (*gistResponse, error) {
	requestFiles := map[string]map[string]string{}
	for name, content := range files {
		requestFiles[name] = map[string]string{"content": content}
	}
	payload := map[string]any{
		"description": "Veil encrypted secrets",
		"public":      false,
		"files":       requestFiles,
	}
	b, _ := json.Marshal(payload)
	resp, err := githubRequest(token, http.MethodPost, "https://api.github.com/gists", b)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github gist create failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var gist gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		return nil, err
	}
	return &gist, nil
}

func updateGist(token, gistID string, files map[string]string) error {
	requestFiles := map[string]map[string]string{}
	for name, content := range files {
		requestFiles[name] = map[string]string{"content": content}
	}
	payload := map[string]any{"files": requestFiles}
	b, _ := json.Marshal(payload)
	resp, err := githubRequest(token, http.MethodPatch, "https://api.github.com/gists/"+gistID, b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github gist update failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (a *App) Link(token, gistID string) error {
	if _, err := a.LoadConfig(); err != nil {
		return err
	}
	if !a.IsInitialized() {
		return errors.New("veil is not initialized (run `veil init`)")
	}
	identity, err := a.LoadIdentity()
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		token, err = a.LoadGitHubToken()
		if err != nil {
			return err
		}
	}
	if gistID == "" {
		if a.config.Gist.ID != "" {
			gistID = a.config.Gist.ID
		} else {
			files := map[string]string{recipientsFileName: identity.Recipient().String() + "\n"}
			gist, err := createGist(token, files)
			if err != nil {
				return err
			}
			gistID = gist.ID
			a.config.Gist.Owner = gist.Owner.Login
		}
	}
	gist, err := getGist(token, gistID)
	if err != nil {
		return err
	}
	a.config.Gist.ID = gist.ID
	if gist.Owner.Login != "" {
		a.config.Gist.Owner = gist.Owner.Login
	}

	recipients := []string{identity.Recipient().String()}
	if file, ok := gist.Files[recipientsFileName]; ok {
		content := file.Content
		if content == "" && file.RawURL != "" {
			if fetched, err := fetchRawContent(file.RawURL); err == nil {
				content = fetched
			}
		}
		recipients = append(recipients, strings.Split(content, "\n")...)
	}
	recipients = uniqueStrings(append(a.config.Recipients, recipients...))
	a.config.Recipients = recipients

	if err := updateGist(token, gistID, map[string]string{recipientsFileName: strings.Join(recipients, "\n") + "\n"}); err != nil {
		return err
	}
	return a.SaveConfig()
}

func (a *App) Sync(token string) error {
	if _, err := a.LoadConfig(); err != nil {
		return err
	}
	if a.config.Gist.ID == "" {
		return errors.New("no gist connected (run `veil link`)")
	}
	identity, err := a.LoadIdentity()
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		token, err = a.LoadGitHubToken()
		if err != nil {
			return err
		}
	}
	gist, err := getGist(token, a.config.Gist.ID)
	if err != nil {
		return err
	}

	remoteRecipients := []string{}
	if file, ok := gist.Files[recipientsFileName]; ok {
		content := file.Content
		if content == "" && file.RawURL != "" {
			content, _ = fetchRawContent(file.RawURL)
		}
		remoteRecipients = strings.Split(content, "\n")
	}
	a.config.Recipients = uniqueStrings(append(a.config.Recipients, append(remoteRecipients, identity.Recipient().String())...))

	for name, file := range gist.Files {
		if !strings.HasSuffix(name, ".json.age") {
			continue
		}
		content := file.Content
		if content == "" && file.RawURL != "" {
			content, _ = fetchRawContent(file.RawURL)
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		project := strings.TrimSuffix(name, ".json.age")
		localPath := a.projectFilePath(project)
		localCipher, _ := os.ReadFile(localPath)
		if len(localCipher) == 0 {
			if err := os.WriteFile(localPath, []byte(content), 0o600); err != nil {
				return err
			}
			continue
		}
		remotePlain, remoteErr := decryptJSON(content, identity)
		localPlain, localErr := decryptJSON(string(localCipher), identity)
		if remoteErr != nil {
			continue
		}
		if localErr != nil {
			if err := os.WriteFile(localPath, []byte(content), 0o600); err != nil {
				return err
			}
			continue
		}
		var remoteBundle ProjectBundle
		var localBundle ProjectBundle
		if json.Unmarshal(remotePlain, &remoteBundle) != nil || json.Unmarshal(localPlain, &localBundle) != nil {
			continue
		}
		if latestUpdate(&remoteBundle).After(latestUpdate(&localBundle)) {
			if err := os.WriteFile(localPath, []byte(content), 0o600); err != nil {
				return err
			}
		}
	}

	files := map[string]string{}
	entries, err := os.ReadDir(a.StoreDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json.age") {
			continue
		}
		path := filepath.Join(a.StoreDir, entry.Name())
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		files[entry.Name()] = string(b)
	}
	sortedRecipients := append([]string(nil), a.config.Recipients...)
	sort.Strings(sortedRecipients)
	files[recipientsFileName] = strings.Join(sortedRecipients, "\n") + "\n"
	if err := updateGist(token, a.config.Gist.ID, files); err != nil {
		return err
	}
	a.config.Gist.LastSyncedAt = nowRFC3339()
	return a.SaveConfig()
}
