package trello

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBoards(t *testing.T) {
	boards := []Board{{ID: "board1", Name: "Sprint Board"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/members/me/boards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("filter") != "open" {
			t.Error("expected filter=open")
		}
		if r.URL.Query().Get("key") == "" || r.URL.Query().Get("token") == "" {
			t.Error("missing auth params")
		}
		json.NewEncoder(w).Encode(boards)
	}))
	defer server.Close()

	client := NewClient("testkey", "testtoken", WithBaseURL(server.URL))
	result, err := client.GetBoards()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Name != "Sprint Board" {
		t.Errorf("unexpected boards: %+v", result)
	}
}

func TestGetBoardLists(t *testing.T) {
	lists := []List{{ID: "list1", Name: "Ready"}, {ID: "list2", Name: "Done"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/boards/board1/lists" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(lists)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	result, err := client.GetBoardLists("board1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 || result[0].Name != "Ready" {
		t.Errorf("unexpected lists: %+v", result)
	}
}

func TestGetListCards(t *testing.T) {
	cards := []Card{{ID: "card1", Name: "Fix bug", Desc: "the plan"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/lists/list1/cards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(cards)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	result, err := client.GetListCards("list1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Desc != "the plan" {
		t.Errorf("unexpected cards: %+v", result)
	}
}

func TestMoveCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/1/cards/card1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":"card1"}`)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	err := client.MoveCard("card1", "list2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/1/cards/card1/actions/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	err := client.AddComment("card1", "task done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindBoardByName(t *testing.T) {
	boards := []Board{{ID: "b1", Name: "Sprint Board"}, {ID: "b2", Name: "Backlog"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(boards)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))

	board, err := client.FindBoardByName("Sprint Board")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if board.ID != "b1" {
		t.Errorf("expected b1, got %s", board.ID)
	}

	_, err = client.FindBoardByName("Nonexistent")
	if err == nil {
		t.Error("expected error for missing board")
	}
}

func TestFindListByName(t *testing.T) {
	lists := []List{{ID: "l1", Name: "Ready"}, {ID: "l2", Name: "Done"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(lists)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))

	list, err := client.FindListByName("board1", "Ready")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.ID != "l1" {
		t.Errorf("expected l1, got %s", list.ID)
	}

	_, err = client.FindListByName("board1", "Nonexistent")
	if err == nil {
		t.Error("expected error for missing list")
	}
}
