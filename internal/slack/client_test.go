package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siyuqian/devpilot/internal/auth"
)

func TestListConversations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("types") != "public_channel" {
			t.Fatalf("unexpected types: %s", r.URL.Query().Get("types"))
		}
		resp := conversationsListResponse{
			OK: true,
			Channels: []Channel{
				{ID: "C001", Name: "general"},
				{ID: "C002", Name: "random"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	channels, err := client.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations error: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
	if channels[0].Name != "general" {
		t.Fatalf("expected general, got %s", channels[0].Name)
	}
}

func TestListConversationsPaginated(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			resp := conversationsListResponse{
				OK:       true,
				Channels: []Channel{{ID: "C001", Name: "general"}},
			}
			resp.ResponseMetadata.NextCursor = "cursor123"
			json.NewEncoder(w).Encode(resp)
		} else {
			if r.URL.Query().Get("cursor") != "cursor123" {
				t.Fatalf("expected cursor123, got %s", r.URL.Query().Get("cursor"))
			}
			resp := conversationsListResponse{
				OK:       true,
				Channels: []Channel{{ID: "C002", Name: "random"}},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	channels, err := client.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations error: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
}

func TestListConversationsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := conversationsListResponse{OK: false, Error: "invalid_auth"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	_, err := client.ListConversations()
	if err == nil {
		t.Fatal("expected error for invalid_auth")
	}
}

func TestResolveChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := conversationsListResponse{
			OK: true,
			Channels: []Channel{
				{ID: "C001", Name: "general"},
				{ID: "C002", Name: "random"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))

	id, err := client.ResolveChannel("general")
	if err != nil {
		t.Fatalf("ResolveChannel error: %v", err)
	}
	if id != "C001" {
		t.Fatalf("expected C001, got %s", id)
	}
}

func TestResolveChannelWithHash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := conversationsListResponse{
			OK:       true,
			Channels: []Channel{{ID: "C001", Name: "general"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))

	id, err := client.ResolveChannel("#general")
	if err != nil {
		t.Fatalf("ResolveChannel error: %v", err)
	}
	if id != "C001" {
		t.Fatalf("expected C001, got %s", id)
	}
}

func TestResolveChannelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := conversationsListResponse{
			OK:       true,
			Channels: []Channel{{ID: "C001", Name: "general"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))

	_, err := client.ResolveChannel("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent channel")
	}
}

func TestPostMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Fatalf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}

		r.ParseForm()
		if r.PostForm.Get("channel") != "C001" {
			t.Fatalf("expected C001, got %s", r.PostForm.Get("channel"))
		}
		if r.PostForm.Get("text") != "Hello world" {
			t.Fatalf("expected 'Hello world', got '%s'", r.PostForm.Get("text"))
		}

		resp := postMessageResponse{OK: true}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	err := client.PostMessage("C001", "Hello world")
	if err != nil {
		t.Fatalf("PostMessage error: %v", err)
	}
}

func TestPostMessageNotInChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := postMessageResponse{OK: false, Error: "not_in_channel"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	err := client.PostMessage("C001", "Hello")
	if err == nil {
		t.Fatal("expected error for not_in_channel")
	}
}

func TestPostMessageError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := postMessageResponse{OK: false, Error: "invalid_auth"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	err := client.PostMessage("C001", "Hello")
	if err == nil {
		t.Fatal("expected error for invalid_auth")
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"ok":false,"error":"internal_error"}`))
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	_, err := client.ListConversations()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestServiceName(t *testing.T) {
	svc := NewSlackService()
	if svc.Name() != "slack" {
		t.Fatalf("expected 'slack', got '%s'", svc.Name())
	}
}

func TestServiceIsLoggedIn(t *testing.T) {
	restore := auth.OverrideConfigDir(t.TempDir())
	defer restore()

	svc := NewSlackService()
	// Without credentials saved, should return false
	if svc.IsLoggedIn() {
		t.Fatal("expected IsLoggedIn to return false without credentials")
	}
}

func TestParseSlackError(t *testing.T) {
	body := []byte(`{"ok":false,"error":"invalid_auth"}`)
	got := parseSlackError(body)
	if got != "invalid_auth" {
		t.Fatalf("expected 'invalid_auth', got '%s'", got)
	}
}

func TestParseSlackErrorInvalidJSON(t *testing.T) {
	body := []byte(`not json`)
	got := parseSlackError(body)
	if got != "not json" {
		t.Fatalf("expected raw body, got '%s'", got)
	}
}
