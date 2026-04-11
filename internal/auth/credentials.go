// Package auth manages authentication credentials for external services
// used by devpilot. It provides on-disk credential storage, an OAuth 2.0
// browser flow helper, and a pluggable Service registry for service-specific
// login/logout commands.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ServiceCredentials is an opaque set of key/value credential fields for a
// single service (for example "api_key" and "token" for Trello).
type ServiceCredentials map[string]string

// AllCredentials maps service names to their stored credentials.
type AllCredentials map[string]ServiceCredentials

var configDir = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devpilot")
}

// OverrideConfigDir replaces the config directory function and returns
// a restore function. Intended for use in tests outside the auth package.
func OverrideConfigDir(dir string) (restore func()) {
	orig := configDir
	configDir = func() string { return dir }
	return func() { configDir = orig }
}

func credentialsPath() string {
	return filepath.Join(configDir(), "credentials.json")
}

func loadAll() (AllCredentials, error) {
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return AllCredentials{}, nil
		}
		return nil, err
	}
	var all AllCredentials
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("corrupt credentials file: %w", err)
	}
	return all, nil
}

func saveAll(all AllCredentials) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(credentialsPath(), data, 0600)
}

// Save persists creds for the named service, replacing any existing entry.
func Save(service string, creds ServiceCredentials) error {
	all, err := loadAll()
	if err != nil {
		return err
	}
	all[service] = creds
	return saveAll(all)
}

// Load returns the stored credentials for the named service, or an error if
// no credentials are saved.
func Load(service string) (ServiceCredentials, error) {
	all, err := loadAll()
	if err != nil {
		return nil, err
	}
	creds, ok := all[service]
	if !ok {
		return nil, fmt.Errorf("no credentials found for %s", service)
	}
	return creds, nil
}

// Remove deletes the stored credentials for the named service. It is a
// no-op if no credentials exist for that service.
func Remove(service string) error {
	all, err := loadAll()
	if err != nil {
		return err
	}
	delete(all, service)
	return saveAll(all)
}

// ListServices returns the sorted names of services with stored credentials.
func ListServices() []string {
	all, err := loadAll()
	if err != nil {
		return nil
	}
	services := make([]string, 0, len(all))
	for name := range all {
		services = append(services, name)
	}
	sort.Strings(services)
	return services
}
