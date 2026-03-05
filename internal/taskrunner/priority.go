package taskrunner

import (
	"sort"
	"strings"
)

// SortByPriority sorts tasks by Priority field (0=highest, 2=lowest).
// Stable sort preserves original order within the same priority.
func SortByPriority(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Priority < tasks[j].Priority
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
