package generate

import (
	"errors"
	"strings"
	"testing"
)

func TestCommitModelViews(t *testing.T) {
	plan := commitPlan{
		Commits: []commitEntry{
			{Message: "feat: add x\n\nbody", Files: []string{"a.go", "gone.go"}},
			{Message: "fix: y", Files: []string{"b.go"}},
		},
		Excluded: []excludedFile{{File: "scratch.txt", Reason: "unrelated"}},
	}
	base := commitModel{
		plan:       plan,
		nameStatus: "A\ta.go\nD\tgone.go\nM\tb.go\n",
		warnings:   []string{"check this"},
	}

	cases := []struct {
		name  string
		model commitModel
		want  []string
	}{
		{
			name:  "staging",
			model: commitModel{phase: phaseStagingFiles},
			want:  []string{"Staging changes"},
		},
		{
			name:  "analyzing",
			model: commitModel{phase: phaseAnalyzing},
			want:  []string{"Analyzing changes"},
		},
		{
			name: "plan multi",
			model: func() commitModel {
				m := base
				m.phase = phasePlan
				return m
			}(),
			want: []string{"Commit Plan", "feat:", "scratch.txt", "check this", "[a]ccept all"},
		},
		{
			name: "plan dry run",
			model: func() commitModel {
				m := base
				m.phase = phasePlan
				m.dryRun = true
				return m
			}(),
			want: []string{"dry-run"},
		},
		{
			name: "executing",
			model: commitModel{
				phase:            phaseExecuting,
				plan:             plan,
				currentCommit:    1,
				completedCommits: []commitResult{{hash: "abc123", message: "feat: add x"}},
			},
			want: []string{"Committing", "abc123", "fix: y"},
		},
		{
			name: "done success",
			model: commitModel{
				phase:            phaseDone,
				plan:             plan,
				completedCommits: []commitResult{{hash: "abc123", message: "feat: add x"}},
			},
			want: []string{"Done", "abc123", "feat: add x"},
		},
		{
			name:  "done aborted",
			model: commitModel{phase: phaseDone, err: errAborted},
			want:  []string{"Aborted"},
		},
		{
			name:  "done no changes",
			model: commitModel{phase: phaseDone, err: errNoChanges},
			want:  []string{"No changes to commit"},
		},
		{
			name: "done error with completed",
			model: commitModel{
				phase:            phaseDone,
				err:              errors.New("failed"),
				completedCommits: []commitResult{{hash: "abc123", message: "feat: add x"}},
			},
			want: []string{"Error: failed", "Completed before failure", "abc123"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.model.View()
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("View() missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestCommitViewHelpers(t *testing.T) {
	if got := firstLineOf("one\ntwo"); got != "one" {
		t.Errorf("firstLineOf() = %q, want one", got)
	}
	if got := firstLineOf("one"); got != "one" {
		t.Errorf("firstLineOf() = %q, want one", got)
	}
	status := parseNameStatus("A\ta.go\nD\tgone.go\nbad\n")
	if status["a.go"] != "A" || status["gone.go"] != "D" {
		t.Fatalf("parseNameStatus() = %#v", status)
	}
	for _, file := range []string{"a.go", "gone.go", "other.go"} {
		if got := formatFileEntry(file, status); !strings.Contains(got, file) {
			t.Errorf("formatFileEntry(%q) = %q", file, got)
		}
	}
	if got := formatCommitMessage("feat: add x\n\nbody"); !strings.Contains(got, "add x") {
		t.Errorf("formatCommitMessage() = %q", got)
	}
}
