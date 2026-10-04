package library

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the small per-user settings file kept outside any library.
type Config struct {
	Recent []string `json:"recent"`
	// Language is the UI language: "en" (default) or "sr" (Serbian Cyrillic).
	Language string `json:"language,omitempty"`
}

const maxRecent = 8

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(dir, "Yearfold")
	if err := adoptLegacyDir(filepath.Join(dir, legacyConfigDir), configDir); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

func LoadConfig() Config {
	var c Config
	p, err := configPath()
	if err != nil {
		return c
	}
	if data, err := os.ReadFile(p); err == nil {
		json.Unmarshal(data, &c)
	}
	return c
}

func (c Config) Save() error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// AddRecent puts root at the front of the recent list.
func (c *Config) AddRecent(root string) {
	recent := []string{root}
	for _, r := range c.Recent {
		if r != root && len(recent) < maxRecent {
			recent = append(recent, r)
		}
	}
	c.Recent = recent
}
