// Package settings stores the per-user preferences of the app.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sjakovic/yearfold/internal/legacy"
)

const (
	appDir    = "Yearfold"
	fileName  = "config.json"
	maxRecent = 8
)

type Settings struct {
	Recent   []string `json:"recent"`
	Language string   `json:"language,omitempty"`

	path string
}

func Load() Settings {
	base, err := os.UserConfigDir()
	if err != nil {
		return Settings{}
	}
	dir := filepath.Join(base, appDir)
	_ = legacy.AdoptDir(filepath.Join(base, legacy.ConfigDir), dir)
	return LoadFrom(filepath.Join(dir, fileName))
}

func LoadFrom(path string) Settings {
	s := Settings{path: path}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	return s
}

func (s Settings) Save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *Settings) AddRecent(root string) {
	recent := []string{root}
	for _, r := range s.Recent {
		if r != root && len(recent) < maxRecent {
			recent = append(recent, r)
		}
	}
	s.Recent = recent
}
