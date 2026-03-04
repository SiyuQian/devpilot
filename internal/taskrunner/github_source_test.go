package taskrunner

import "testing"

func TestGitHubSource_FilterReady(t *testing.T) {
	issues := []ghIssue{
		{Number: 1, Title: "Ready task", Body: "Do this", URL: "https://github.com/o/r/issues/1",
			Labels: []ghLabel{{Name: "devpilot"}}},
		{Number: 2, Title: "In progress", URL: "https://github.com/o/r/issues/2",
			Labels: []ghLabel{{Name: "devpilot"}, {Name: "in-progress"}}},
		{Number: 3, Title: "Failed task", URL: "https://github.com/o/r/issues/3",
			Labels: []ghLabel{{Name: "devpilot"}, {Name: "failed"}}},
	}

	tasks := issuesToReadyTasks(issues)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "1" || tasks[0].Name != "Ready task" {
		t.Errorf("unexpected task: %+v", tasks[0])
	}
}

func TestGitHubPriority(t *testing.T) {
	cases := []struct {
		labels   []ghLabel
		expected int
	}{
		{[]ghLabel{{Name: "P0-critical"}, {Name: "devpilot"}}, 0},
		{[]ghLabel{{Name: "p1-high"}, {Name: "devpilot"}}, 1},
		{[]ghLabel{{Name: "devpilot"}}, 2},
	}
	for _, c := range cases {
		issue := ghIssue{Labels: c.labels}
		got := ghPriority(issue)
		if got != c.expected {
			t.Errorf("labels %v: expected %d, got %d", c.labels, c.expected, got)
		}
	}
}
