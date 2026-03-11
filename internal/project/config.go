package project

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const configFile = ".devpilot.yaml"

// SkillEntry records an installed skill in the project config.
type SkillEntry struct {
	Name        string    `yaml:"name"`
	Source      string    `yaml:"source"`
	Version     string    `yaml:"version"`
	InstalledAt time.Time `yaml:"installedAt"`
}

// Config represents project-level configuration stored in .devpilot.yaml.
type Config struct {
	Board              string            `yaml:"board,omitempty"`
	Source             string            `yaml:"source,omitempty"` // "trello" or "github"
	Models             map[string]string `yaml:"models,omitempty"`
	OpenSpecMinVersion string            `yaml:"openspecMinVersion,omitempty"`
	Skills             []SkillEntry      `yaml:"skills,omitempty"`
}

// ResolveSource returns the effective task source: flag value takes priority,
// then the config file value, then "trello" as the default.
func (c *Config) ResolveSource(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if c.Source != "" {
		return c.Source
	}
	return "trello"
}

// ModelFor returns the configured model for a command, falling back to "default", then "".
func (c *Config) ModelFor(command string) string {
	if c.Models == nil {
		return ""
	}
	if m, ok := c.Models[command]; ok {
		return m
	}
	return c.Models["default"]
}

// UpsertSkill adds or updates a skill entry by name.
func (c *Config) UpsertSkill(entry SkillEntry) {
	for i, s := range c.Skills {
		if s.Name == entry.Name {
			c.Skills[i] = entry
			return
		}
	}
	c.Skills = append(c.Skills, entry)
}

// Load reads .devpilot.yaml from dir. Returns a zero-value Config (not an error)
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes cfg to .devpilot.yaml in dir, creating intermediate directories.
func Save(dir string, cfg *Config) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, configFile), data, 0644)
}

// Exists checks if .devpilot.yaml exists in dir.
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, configFile))
	return err == nil
}
