package openspec

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siyuqian/devpilot/internal/trello"
)

func TestTrelloTarget_FindByName(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "add-auth", Desc: "old plan"},
		{ID: "c2", Name: "fix-bug", Desc: "other"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(cards)
	}))
	defer server.Close()

	client := trello.NewClient("k", "t", trello.WithBaseURL(server.URL))
	target := NewTrelloTarget(client, "list1")

	// Found case
	id, err := target.FindByName("add-auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "c1" {
		t.Errorf("expected c1, got %s", id)
	}

	// Not found case
	id, err = target.FindByName("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "" {
		t.Errorf("expected empty id, got %s", id)
	}
}

func TestTrelloTarget_Create(t *testing.T) {
	var gotMethod, gotPath, gotName, gotDesc string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotName = r.URL.Query().Get("name")
		gotDesc = r.URL.Query().Get("desc")
		fmt.Fprint(w, `{"id":"card99","name":"add-auth","desc":"the plan","idList":"list1"}`)
	}))
	defer server.Close()

	client := trello.NewClient("k", "t", trello.WithBaseURL(server.URL))
	target := NewTrelloTarget(client, "list1")

	err := target.Create("add-auth", "the plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/1/cards" {
		t.Errorf("expected /1/cards, got %s", gotPath)
	}
	if gotName != "add-auth" {
		t.Errorf("expected name=add-auth, got %s", gotName)
	}
	if gotDesc != "the plan" {
		t.Errorf("expected desc=the plan, got %s", gotDesc)
	}
}

func TestTrelloTarget_Update(t *testing.T) {
	var gotMethod, gotPath, gotDesc string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotDesc = r.URL.Query().Get("desc")
		fmt.Fprint(w, `{"id":"card1","name":"add-auth","desc":"updated"}`)
	}))
	defer server.Close()

	client := trello.NewClient("k", "t", trello.WithBaseURL(server.URL))
	target := NewTrelloTarget(client, "list1")

	err := target.Update("card1", "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if gotPath != "/1/cards/card1" {
		t.Errorf("expected /1/cards/card1, got %s", gotPath)
	}
	if gotDesc != "updated" {
		t.Errorf("expected desc=updated, got %s", gotDesc)
	}
}
