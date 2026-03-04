package taskrunner

import "sort"

// SortByPriority sorts tasks by Priority field (0=highest, 2=lowest).
// Stable sort preserves original order within the same priority.
func SortByPriority(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Priority < tasks[j].Priority
	})
}
