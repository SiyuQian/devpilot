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

func TestSortByPriority_CreatedAtTiebreaker(t *testing.T) {
	// Within the same priority, older tasks (smaller CreatedAt) should run first.
	tasks := []Task{
		{ID: "newer", Priority: 1, CreatedAt: 2000},
		{ID: "oldest", Priority: 1, CreatedAt: 1000},
		{ID: "middle", Priority: 1, CreatedAt: 1500},
	}
	SortByPriority(tasks)
	want := []string{"oldest", "middle", "newer"}
	for i, w := range want {
		if tasks[i].ID != w {
			t.Errorf("position %d: got %s, want %s", i, tasks[i].ID, w)
		}
	}
}

func TestSortByPriority_CreatedAtWithPriority(t *testing.T) {
	// Priority always beats creation time: a newer P0 beats an older P1.
	tasks := []Task{
		{ID: "p1-old", Priority: 1, CreatedAt: 1000},
		{ID: "p0-new", Priority: 0, CreatedAt: 9000},
	}
	SortByPriority(tasks)
	if tasks[0].ID != "p0-new" {
		t.Errorf("expected P0 first regardless of age, got %s", tasks[0].ID)
	}
}

func TestSortByPriority_ZeroCreatedAtPreservesOrder(t *testing.T) {
	// When CreatedAt is zero (Trello tasks), original slice order is preserved.
	tasks := []Task{
		{ID: "first", Priority: 2, CreatedAt: 0},
		{ID: "second", Priority: 2, CreatedAt: 0},
	}
	SortByPriority(tasks)
	if tasks[0].ID != "first" || tasks[1].ID != "second" {
		t.Errorf("zero-CreatedAt order not preserved: got %s, %s", tasks[0].ID, tasks[1].ID)
	}
}
