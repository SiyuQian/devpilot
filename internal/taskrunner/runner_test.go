package taskrunner

import (
	"strings"
	"testing"
)

func TestBuildPrompt_withOpenSpec(t *testing.T) {
	r := &Runner{config: Config{UseOpenSpec: true}}
	task := Task{Name: "add-auth", Description: "the plan"}
	prompt := r.buildPrompt(task)
	if !strings.Contains(prompt, "/opsx:apply add-auth") {
		t.Errorf("expected prompt to contain opsx:apply, got:\n%s", prompt)
	}
}

func TestBuildPrompt_withoutOpenSpec(t *testing.T) {
	r := &Runner{config: Config{UseOpenSpec: false}}
	task := Task{Name: "add-auth", Description: "the plan"}
	prompt := r.buildPrompt(task)
	if !strings.Contains(prompt, "the plan") {
		t.Errorf("expected prompt to contain plan description, got:\n%s", prompt)
	}
}
