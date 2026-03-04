package taskrunner

import "testing"

func TestSortByPriority_AllPriorities(t *testing.T) {
	tasks := []Task{
		{ID: "c3", Name: "Low", Priority: 2},
		{ID: "c1", Name: "Critical", Priority: 0},
		{ID: "c2", Name: "High", Priority: 1},
	}
	SortByPriority(tasks)
	if tasks[0].ID != "c1" {
		t.Errorf("expected P0 first, got %s", tasks[0].ID)
	}
	if tasks[1].ID != "c2" {
		t.Errorf("expected P1 second, got %s", tasks[1].ID)
	}
	if tasks[2].ID != "c3" {
		t.Errorf("expected P2 third, got %s", tasks[2].ID)
	}
}

func TestSortByPriority_DefaultP2(t *testing.T) {
	tasks := []Task{
		{ID: "c1", Name: "No priority", Priority: 2},
		{ID: "c2", Name: "Critical", Priority: 0},
	}
	SortByPriority(tasks)
	if tasks[0].ID != "c2" {
		t.Errorf("expected P0 first, got %s", tasks[0].ID)
	}
}

func TestSortByPriority_StableSort(t *testing.T) {
	tasks := []Task{
		{ID: "c1", Priority: 1},
		{ID: "c2", Priority: 1},
		{ID: "c3", Priority: 1},
	}
	SortByPriority(tasks)
	if tasks[0].ID != "c1" || tasks[1].ID != "c2" || tasks[2].ID != "c3" {
		t.Errorf("stable sort not preserved: got %s, %s, %s", tasks[0].ID, tasks[1].ID, tasks[2].ID)
	}
}

func TestSortByPriority_EmptySlice(t *testing.T) {
	var tasks []Task
	SortByPriority(tasks) // should not panic
}
