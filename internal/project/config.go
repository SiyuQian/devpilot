package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFile = ".devkit.json"

// Config represents project-level configuration stored in .devkit.json.
type Config struct {
	Board string `json:"board,omitempty"`
}

// Load reads .devkit.json from dir. Returns a zero-value Config (not an error)
// if the file does not exist.
func Load(dir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes cfg to .devkit.json in dir, creating intermediate directories.
func Save(dir string, cfg *Config) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, configFile), data, 0644)
}

// Exists checks if .devkit.json exists in dir.
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, configFile))
	return err == nil
}
