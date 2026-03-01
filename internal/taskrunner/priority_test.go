package taskrunner

import (
	"testing"

	"github.com/siyuqian/devpilot/internal/trello"
)

func TestSortByPriority_AllPriorities(t *testing.T) {
	cards := []trello.Card{
		{ID: "c3", Name: "Low", Labels: []trello.Label{{Name: "P2-normal"}}},
		{ID: "c1", Name: "Critical", Labels: []trello.Label{{Name: "P0-critical"}}},
		{ID: "c2", Name: "High", Labels: []trello.Label{{Name: "P1-high"}}},
	}

	SortByPriority(cards)

	if cards[0].ID != "c1" {
		t.Errorf("expected P0 card first, got %s (%s)", cards[0].ID, cards[0].Name)
	}
	if cards[1].ID != "c2" {
		t.Errorf("expected P1 card second, got %s (%s)", cards[1].ID, cards[1].Name)
	}
	if cards[2].ID != "c3" {
		t.Errorf("expected P2 card third, got %s (%s)", cards[2].ID, cards[2].Name)
	}
}

func TestSortByPriority_NoLabelsDefaultToP2(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "No label"},
		{ID: "c2", Name: "Critical", Labels: []trello.Label{{Name: "P0-critical"}}},
	}

	SortByPriority(cards)

	if cards[0].ID != "c2" {
		t.Errorf("expected P0 card first, got %s", cards[0].ID)
	}
	if cards[1].ID != "c1" {
		t.Errorf("expected unlabeled card second, got %s", cards[1].ID)
	}
}

func TestSortByPriority_StableSort(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "First P1", Labels: []trello.Label{{Name: "P1-high"}}},
		{ID: "c2", Name: "Second P1", Labels: []trello.Label{{Name: "P1-high"}}},
		{ID: "c3", Name: "Third P1", Labels: []trello.Label{{Name: "P1-high"}}},
	}

	SortByPriority(cards)

	if cards[0].ID != "c1" || cards[1].ID != "c2" || cards[2].ID != "c3" {
		t.Errorf("stable sort not preserved: got %s, %s, %s", cards[0].ID, cards[1].ID, cards[2].ID)
	}
}

func TestSortByPriority_MixedLabeledAndUnlabeled(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "No label 1"},
		{ID: "c2", Name: "P1 task", Labels: []trello.Label{{Name: "P1-high"}}},
		{ID: "c3", Name: "No label 2"},
		{ID: "c4", Name: "P0 task", Labels: []trello.Label{{Name: "P0-critical"}}},
	}

	SortByPriority(cards)

	if cards[0].ID != "c4" {
		t.Errorf("expected P0 first, got %s", cards[0].ID)
	}
	if cards[1].ID != "c2" {
		t.Errorf("expected P1 second, got %s", cards[1].ID)
	}
	// c1 and c3 are both P2 (default), stable sort preserves their order
	if cards[2].ID != "c1" || cards[3].ID != "c3" {
		t.Errorf("expected unlabeled cards in original order, got %s, %s", cards[2].ID, cards[3].ID)
	}
}

func TestSortByPriority_EmptySlice(t *testing.T) {
	var cards []trello.Card
	SortByPriority(cards) // should not panic
}

func TestSortByPriority_CaseInsensitive(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Lowercase", Labels: []trello.Label{{Name: "p0-critical"}}},
		{ID: "c2", Name: "Uppercase", Labels: []trello.Label{{Name: "P1-HIGH"}}},
	}

	SortByPriority(cards)

	if cards[0].ID != "c1" {
		t.Errorf("expected p0 card first (case insensitive), got %s", cards[0].ID)
	}
}

func TestSortByPriority_NonPriorityLabelsIgnored(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Bug label only", Labels: []trello.Label{{Name: "bug", Color: "red"}}},
		{ID: "c2", Name: "P0 task", Labels: []trello.Label{{Name: "P0-critical"}}},
	}

	SortByPriority(cards)

	if cards[0].ID != "c2" {
		t.Errorf("expected P0 first, non-priority labels should be ignored, got %s", cards[0].ID)
	}
}
