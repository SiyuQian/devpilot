package graph

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunBuildEmitsValidEnvelope(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "go.mod"),
		[]byte("module example.com/cli-build-test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "main.go"),
		[]byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVPILOT_HOME", t.TempDir())

	buf := captureStdout(t, func() {
		if code := runBuild(repo); code != 0 {
			t.Fatalf("runBuild rc=%d (output: %s)", code, "")
		}
	})

	if !bytes.Contains(buf, []byte(`"ok":true`)) {
		t.Fatalf("not ok: %s", buf)
	}
	var env map[string]any
	if err := json.Unmarshal(buf, &env); err != nil {
		t.Fatalf("parse: %v\n%s", err, buf)
	}
	if env["command"] != "graph.build" {
		t.Errorf("command=%v", env["command"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("data not object: %v", env["data"])
	}
	if data["mode"] != "full" {
		t.Errorf("mode=%v", data["mode"])
	}
}

func TestRunBuildGoNoModule(t *testing.T) {
	repo := t.TempDir()
	// A .go file without a go.mod/go.work must surface as go_no_module.
	if err := os.WriteFile(filepath.Join(repo, "main.go"),
		[]byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVPILOT_HOME", t.TempDir())

	buf := captureStdout(t, func() {
		if code := runBuild(repo); code != 1 {
			t.Errorf("expected rc=1, got %d", code)
		}
	})

	var env map[string]any
	if err := json.Unmarshal(buf, &env); err != nil {
		t.Fatalf("parse: %v\n%s", err, buf)
	}
	errInfo, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("error not object: %v", env["error"])
	}
	if errInfo["code"] != "go_no_module" {
		t.Errorf("code=%v, want go_no_module (%s)", errInfo["code"], buf)
	}
	sugs, ok := env["next_tool_suggestions"].([]any)
	if !ok || len(sugs) == 0 {
		t.Errorf("expected suggestion preserved on go_no_module envelope: %s", buf)
	}
}

func TestRunBuildBadRepo(t *testing.T) {
	t.Setenv("DEVPILOT_HOME", t.TempDir())
	buf := captureStdout(t, func() {
		if code := runBuild("/definitely/not/here"); code != 1 {
			t.Errorf("expected rc=1, got %d", code)
		}
	})
	if !bytes.Contains(buf, []byte(`"code":"repo_invalid"`)) {
		t.Errorf("missing repo_invalid: %s", buf)
	}
}
