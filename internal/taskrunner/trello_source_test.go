package taskrunner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siyuqian/devpilot/internal/trello"
)

func TestTrelloSource_FetchReady_MapsToTasks(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Task 1", Desc: "Do something", ShortURL: "https://trello.com/c/c1",
			Labels: []trello.Label{{Name: "P0-critical"}}},
		{ID: "c2", Name: "Task 2", Desc: "Do another thing", ShortURL: "https://trello.com/c/c2"},
	}
	data, _ := json.Marshal(cards)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	client := trello.NewClient("key", "token", trello.WithBaseURL(ts.URL))
	source := &TrelloSource{client: client, readyListID: "list1"}

	tasks, err := source.FetchReady()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "c1" || tasks[0].Name != "Task 1" || tasks[0].Priority != 0 {
		t.Errorf("task 0 mismatch: %+v", tasks[0])
	}
	if tasks[1].Priority != 2 {
		t.Errorf("expected default P2 for unlabeled task, got %d", tasks[1].Priority)
	}
}

func TestParseTrelloPriority(t *testing.T) {
	cases := []struct {
		labels   []trello.Label
		expected int
	}{
		{[]trello.Label{{Name: "P0-critical"}}, 0},
		{[]trello.Label{{Name: "P1-high"}}, 1},
		{[]trello.Label{{Name: "P2-normal"}}, 2},
		{[]trello.Label{{Name: "p0-critical"}}, 0}, // case insensitive
		{[]trello.Label{{Name: "bug"}}, 2},          // non-priority label defaults to P2
		{nil, 2},
	}
	for _, c := range cases {
		card := trello.Card{Labels: c.labels}
		got := trelloPriority(card)
		if got != c.expected {
			t.Errorf("labels %v: expected %d, got %d", c.labels, c.expected, got)
		}
	}
}
