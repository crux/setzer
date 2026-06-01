package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// keyringService namespaces Setzer's secrets in the OS keychain.
const keyringService = "setzer"

// Config holds the non-secret settings for the single site Setzer manages.
// The access token is NOT stored here — it lives in the OS keychain.
type Config struct {
	RepoURL     string `json:"repo_url"`     // e.g. https://github.com/owner/repo.git
	Branch      string `json:"branch"`       // e.g. main
	ContentPath string `json:"content_path"` // path within the repo, e.g. content.json
	SiteDir     string `json:"site_dir"`     // serve root within the repo ("." or e.g. "site")
}

// Configured reports whether enough is set to operate.
func (c *Config) Configured() bool {
	return c != nil && c.RepoURL != "" && c.ContentPath != ""
}

// configPath returns the config file path, creating its directory.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "setzer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig reads the config file. A missing file yields an empty
// (unconfigured) Config rather than an error.
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// Save writes the non-secret config to disk (owner-only).
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// SetToken stores the PAT in the OS keychain, keyed by repo URL.
func SetToken(repoURL, token string) error {
	return keyring.Set(keyringService, repoURL, token)
}

// GetToken retrieves the PAT for a repo from the OS keychain.
func GetToken(repoURL string) (string, error) {
	return keyring.Get(keyringService, repoURL)
}
