package taskrunner

import (
	"sort"
	"strings"

	"github.com/siyuqian/devpilot/internal/trello"
)

// SortByPriority sorts cards by priority label (P0 > P1 > P2).
// Cards without a priority label are treated as P2.
// Stable sort preserves original list order within the same priority.
func SortByPriority(cards []trello.Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		return cardPriority(cards[i]) < cardPriority(cards[j])
	})
}

func cardPriority(card trello.Card) int {
	for _, label := range card.Labels {
		name := strings.ToUpper(label.Name)
		if strings.HasPrefix(name, "P0") {
			return 0
		}
		if strings.HasPrefix(name, "P1") {
			return 1
		}
		if strings.HasPrefix(name, "P2") {
			return 2
		}
	}
	return 2 // default: lowest priority
}
