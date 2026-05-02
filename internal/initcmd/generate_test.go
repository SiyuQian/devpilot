package initcmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/project"
)

func TestConfigureBoard_NonInteractiveSkips(t *testing.T) {
	dir := t.TempDir()

	opts := GenerateOpts{Dir: dir, Interactive: false}
	if err := ConfigureBoard(opts, nil); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	// Should not have created .devpilot.yaml
	if _, err := os.Stat(filepath.Join(dir, ".devpilot.yaml")); !os.IsNotExist(err) {
		t.Error(".devpilot.yaml should not exist in non-interactive mode")
	}
}

func TestConfigureBoard_InteractiveWithListBoards(t *testing.T) {
	dir := t.TempDir()

	input := strings.NewReader("1\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	listBoards := func() ([]Board, error) {
		return []Board{{Name: "Dev Board"}, {Name: "Other Board"}}, nil
	}

	if err := ConfigureBoard(opts, listBoards); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".devpilot.yaml"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "Dev Board") {
		t.Errorf(".devpilot.yaml does not contain board name, got: %s", string(data))
	}
}

func TestConfigureBoard_InteractiveFreeText(t *testing.T) {
	dir := t.TempDir()

	input := strings.NewReader("My Custom Board\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if err := ConfigureBoard(opts, nil); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".devpilot.yaml"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "My Custom Board") {
		t.Errorf(".devpilot.yaml does not contain board name, got: %s", string(data))
	}
}

func TestConfigureBoard_PreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()

	// Write a config with existing skills entry.
	initial := []byte("skills:\n- name: pm\n  source: github.com/siyuqian/devpilot\n  installedAt: 2026-01-01T00:00:00Z\n")
	if err := os.WriteFile(filepath.Join(dir, ".devpilot.yaml"), initial, 0644); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader("My Board\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if err := ConfigureBoard(opts, nil); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Board != "My Board" {
		t.Errorf("Board = %q, want %q", cfg.Board, "My Board")
	}
}

// gitignoreHasLine reports whether existing contains entry as a line-equal
// match (after trimming whitespace and a single leading "!"), ignoring blank
// and comment lines. Used by tests to assert post-call .gitignore content.
func gitignoreHasLine(existing, entry string) bool {
	want := strings.TrimSpace(entry)
	want = strings.TrimPrefix(want, "!")
	for _, line := range strings.Split(existing, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "!")
		if line == want {
			return true
		}
	}
	return false
}

func TestEnsureGitignore(t *testing.T) {
	tests := []struct {
		name         string
		initial      string // initial .gitignore contents; empty means file does not exist
		writeInitial bool   // whether to write initial file at all
		entries      []string
		wantPresent  []string // entries that must be present (line-equal) after the call
		wantAbsent   []string // entries that must NOT have been added by the call (i.e., still absent)
	}{
		{
			name:         "adds entry to missing file",
			writeInitial: false,
			entries:      []string{".devpilot/logs/"},
			wantPresent:  []string{".devpilot/logs/"},
		},
		{
			name:         "adds entry to empty file",
			writeInitial: true,
			initial:      "",
			entries:      []string{".devpilot/logs/"},
			wantPresent:  []string{".devpilot/logs/"},
		},
		{
			name:         "adds entry when only substring matches exist",
			writeInitial: true,
			// The requested entry is a substring of an existing (longer)
			// line; the buggy substring check would skip adding. The
			// existing line is NOT a line-equal match, so the entry MUST
			// be appended.
			initial:     ".devpilot/logs/extra\n",
			entries:     []string{".devpilot/logs/"},
			wantPresent: []string{".devpilot/logs/"},
		},
		{
			name:         "skips entry when exact line match exists",
			writeInitial: true,
			initial:      ".devpilot/logs/\n",
			entries:      []string{".devpilot/logs/"},
			wantPresent:  []string{".devpilot/logs/"},
		},
		{
			name:         "skips entry when negated line match exists",
			writeInitial: true,
			initial:      "!.devpilot/logs/\n",
			entries:      []string{".devpilot/logs/"},
			wantPresent:  []string{".devpilot/logs/"},
		},
		{
			name:         "treats lines with surrounding whitespace as equal",
			writeInitial: true,
			initial:      "  .devpilot/logs/  \n",
			entries:      []string{".devpilot/logs/"},
			wantPresent:  []string{".devpilot/logs/"},
		},
		{
			name:         "ignores comment lines",
			writeInitial: true,
			initial:      "# .devpilot/logs/\n",
			entries:      []string{".devpilot/logs/"},
			wantPresent:  []string{".devpilot/logs/"},
		},
		{
			name:         "multiple entries partial overlap",
			writeInitial: true,
			initial:      "a\n",
			entries:      []string{"a", "b"},
			wantPresent:  []string{"a", "b"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".gitignore")
			if tc.writeInitial {
				if err := os.WriteFile(path, []byte(tc.initial), 0644); err != nil {
					t.Fatalf("WriteFile failed: %v", err)
				}
			}

			if err := EnsureGitignore(dir, tc.entries); err != nil {
				t.Fatalf("EnsureGitignore failed: %v", err)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile failed: %v", err)
			}
			got := string(data)

			for _, want := range tc.wantPresent {
				if !gitignoreHasLine(got, want) {
					t.Errorf("entry %q missing from .gitignore; got:\n%s", want, got)
				}
			}
			for _, absent := range tc.wantAbsent {
				if gitignoreHasLine(got, absent) {
					t.Errorf("entry %q unexpectedly present in .gitignore; got:\n%s", absent, got)
				}
			}
		})
	}
}

// TestEnsureGitignore_DoesNotDuplicateExactMatch asserts the regression-bug
// counterpart: when an exact line already exists, the file is not modified.
func TestEnsureGitignore_DoesNotDuplicateExactMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	initial := ".devpilot/logs/\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := EnsureGitignore(dir, []string{".devpilot/logs/"}); err != nil {
		t.Fatalf("EnsureGitignore failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != initial {
		t.Errorf(".gitignore was modified despite exact match.\nbefore:\n%s\nafter:\n%s", initial, string(data))
	}
}

