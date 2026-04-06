package auth

import (
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origFunc := configDir
	configDir = func() string { return tmpDir }
	defer func() { configDir = origFunc }()

	creds := ServiceCredentials{
		"api_key": "test-key",
		"token":   "test-token",
	}

	if err := Save("trello", creds); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load("trello")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded["api_key"] != "test-key" || loaded["token"] != "test-token" {
		t.Fatalf("unexpected creds: %v", loaded)
	}
}

func TestLoadMissing(t *testing.T) {
	tmpDir := t.TempDir()
	origFunc := configDir
	configDir = func() string { return tmpDir }
	defer func() { configDir = origFunc }()

	_, err := Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing service")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	origFunc := configDir
	configDir = func() string { return tmpDir }
	defer func() { configDir = origFunc }()

	creds := ServiceCredentials{"api_key": "k", "token": "t"}
	if err := Save("trello", creds); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := Remove("trello"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err := Load("trello")
	if err == nil {
		t.Fatal("expected error after removal")
	}
}

func TestListServices(t *testing.T) {
	tmpDir := t.TempDir()
	origFunc := configDir
	configDir = func() string { return tmpDir }
	defer func() { configDir = origFunc }()

	services := ListServices()
	if len(services) != 0 {
		t.Fatalf("expected 0 services, got %d", len(services))
	}

	if err := Save("trello", ServiceCredentials{"api_key": "k", "token": "t"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	services = ListServices()
	if len(services) != 1 || services[0] != "trello" {
		t.Fatalf("expected [trello], got %v", services)
	}
}
