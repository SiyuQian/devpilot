package skillmgr

import (
	"strings"
	"testing"
)

func TestParseSkillArg(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
		wantErr bool
	}{
		{input: "pm", name: "pm", version: ""},
		{input: "pm@v1.2.3", name: "pm", version: "v1.2.3"},
		{input: "google-go-style@v0.4.0", name: "google-go-style", version: "v0.4.0"},
		{input: "@v1.0.0", wantErr: true},
	}

	for _, tt := range tests {
		name, version, err := parseSkillArg(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSkillArg(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSkillArg(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if name != tt.name {
			t.Errorf("parseSkillArg(%q) name = %q, want %q", tt.input, name, tt.name)
		}
		if version != tt.version {
			t.Errorf("parseSkillArg(%q) version = %q, want %q", tt.input, version, tt.version)
		}
	}
}

func TestSkillAddRequiresConfig(t *testing.T) {
	t.TempDir() // isolate but don't create config
	cmd := skillAddCmd
	cmd.ResetFlags()
	err := cmd.RunE(cmd, []string{"pm"})
	if err == nil {
		t.Fatal("expected error when no .devpilot.yaml, got nil")
	}
	if !strings.Contains(err.Error(), "devpilot init") {
		t.Errorf("error should mention 'devpilot init', got: %v", err)
	}
}

func TestSkillListRequiresConfig(t *testing.T) {
	t.TempDir()
	cmd := skillListCmd
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when no .devpilot.yaml, got nil")
	}
	if !strings.Contains(err.Error(), "devpilot init") {
		t.Errorf("error should mention 'devpilot init', got: %v", err)
	}
}
