package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type ServiceCredentials map[string]string

type AllCredentials map[string]ServiceCredentials

var configDir = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devpilot")
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

func Save(service string, creds ServiceCredentials) error {
	all, err := loadAll()
	if err != nil {
		return err
	}
	all[service] = creds
	return saveAll(all)
}

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

func Remove(service string) error {
	all, err := loadAll()
	if err != nil {
		return err
	}
	delete(all, service)
	return saveAll(all)
}

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
