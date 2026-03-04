package taskrunner

import "testing"

func TestTaskDefaultPriority(t *testing.T) {
	task := Task{ID: "1", Name: "Test"}
	if task.Priority != 0 {
		t.Errorf("expected zero Priority, got %d", task.Priority)
	}
}

func TestSourceInfoZeroValue(t *testing.T) {
	var info SourceInfo
	if info.DisplayName != "" || info.BoardID != "" || info.Lists != nil {
		t.Error("SourceInfo zero value should have empty fields")
	}
}
