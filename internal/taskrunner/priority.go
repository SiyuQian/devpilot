package taskrunner

import (
	"sort"
	"strings"
)

// SortByPriority sorts tasks by Priority (0=highest, 2=lowest).
// Within the same priority, tasks are sorted by CreatedAt ascending (FIFO).
// When CreatedAt is zero (e.g. Trello tasks), the original slice order is
// preserved via stable sort, keeping existing behaviour unchanged.
func SortByPriority(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority < tasks[j].Priority
		}
		// Same priority: earlier creation time runs first (FIFO).
		// If either timestamp is zero (backend doesn't provide it), preserve
		// relative order via the stable sort guarantee.
		if tasks[i].CreatedAt != 0 && tasks[j].CreatedAt != 0 {
			return tasks[i].CreatedAt < tasks[j].CreatedAt
		}
		return false
	})
}

// priorityFromLabelNames returns the task priority (0–2) from a slice of label
// names. Labels starting with P0/P1/P2 (case-insensitive) are recognised.
// Returns 2 (lowest) when no priority label is found.
func priorityFromLabelNames(names []string) int {
	for _, n := range names {
		upper := strings.ToUpper(n)
		if strings.HasPrefix(upper, "P0") {
			return 0
		}
		if strings.HasPrefix(upper, "P1") {
			return 1
		}
		if strings.HasPrefix(upper, "P2") {
			return 2
		}
	}
	return 2
}
