package gmail

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "is:unread" {
			t.Fatalf("unexpected query: %s", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("maxResults") != "5" {
			t.Fatalf("unexpected maxResults: %s", r.URL.Query().Get("maxResults"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		resp := MessageListResponse{
			Messages: []MessageRef{
				{ID: "msg1", ThreadID: "thread1"},
				{ID: "msg2", ThreadID: "thread2"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	refs, err := client.ListMessages("is:unread", 5)
	if err != nil {
		t.Fatalf("ListMessages error: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(refs))
	}
	if refs[0].ID != "msg1" {
		t.Fatalf("expected msg1, got %s", refs[0].ID)
	}
}

func TestListMessagesEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(MessageListResponse{})
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	refs, err := client.ListMessages("", 10)
	if err != nil {
		t.Fatalf("ListMessages error: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(refs))
	}
}

func TestGetMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages/msg1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("format") != "full" {
			t.Fatalf("unexpected format: %s", r.URL.Query().Get("format"))
		}
		msg := Message{
			ID: "msg1",
			Payload: Payload{
				Headers: []Header{
					{Name: "From", Value: "alice@example.com"},
					{Name: "Subject", Value: "Test Subject"},
					{Name: "Date", Value: "2024-01-15 09:30"},
				},
				MimeType: "text/plain",
				Body: Body{
					Data: base64.URLEncoding.EncodeToString([]byte("Hello World")),
				},
			},
		}
		json.NewEncoder(w).Encode(msg)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	msg, err := client.GetMessage("msg1")
	if err != nil {
		t.Fatalf("GetMessage error: %v", err)
	}
	if msg.ID != "msg1" {
		t.Fatalf("expected msg1, got %s", msg.ID)
	}
	if GetHeader(msg, "From") != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %s", GetHeader(msg, "From"))
	}
	if GetHeader(msg, "Subject") != "Test Subject" {
		t.Fatalf("expected Test Subject, got %s", GetHeader(msg, "Subject"))
	}
	body := GetBody(msg)
	if body != "Hello World" {
		t.Fatalf("expected 'Hello World', got '%s'", body)
	}
}

func TestBatchModify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages/batchModify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		ids := body["ids"].([]any)
		if len(ids) != 2 {
			t.Fatalf("expected 2 ids, got %d", len(ids))
		}
		labels := body["removeLabelIds"].([]any)
		if labels[0].(string) != "UNREAD" {
			t.Fatalf("expected UNREAD label, got %s", labels[0])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	err := client.BatchModify([]string{"msg1", "msg2"}, []string{"UNREAD"})
	if err != nil {
		t.Fatalf("BatchModify error: %v", err)
	}
}

func TestGetBodyMultipart(t *testing.T) {
	msg := &Message{
		ID: "msg1",
		Payload: Payload{
			MimeType: "multipart/alternative",
			Parts: []Payload{
				{
					MimeType: "text/plain",
					Body: Body{
						Data: base64.URLEncoding.EncodeToString([]byte("Plain text body")),
					},
				},
				{
					MimeType: "text/html",
					Body: Body{
						Data: base64.URLEncoding.EncodeToString([]byte("<p>HTML body</p>")),
					},
				},
			},
		},
	}
	body := GetBody(msg)
	if body != "Plain text body" {
		t.Fatalf("expected 'Plain text body', got '%s'", body)
	}
}

func TestGetBodyHTMLFallback(t *testing.T) {
	msg := &Message{
		ID: "msg1",
		Payload: Payload{
			MimeType: "multipart/alternative",
			Parts: []Payload{
				{
					MimeType: "text/html",
					Body: Body{
						Data: base64.URLEncoding.EncodeToString([]byte("<p>Hello &amp; welcome</p>")),
					},
				},
			},
		},
	}
	body := GetBody(msg)
	if body != "Hello & welcome" {
		t.Fatalf("expected 'Hello & welcome', got '%s'", body)
	}
}

func TestGetHeaderCaseInsensitive(t *testing.T) {
	msg := &Message{
		Payload: Payload{
			Headers: []Header{
				{Name: "from", Value: "test@example.com"},
			},
		},
	}
	if GetHeader(msg, "From") != "test@example.com" {
		t.Fatalf("expected case-insensitive match")
	}
}

func TestGetHeaderMissing(t *testing.T) {
	msg := &Message{
		Payload: Payload{
			Headers: []Header{},
		},
	}
	if GetHeader(msg, "From") != "" {
		t.Fatalf("expected empty string for missing header")
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<b>Bold</b> and <i>italic</i>", "Bold and italic"},
		{"No tags", "No tags"},
		{"&amp; &lt; &gt; &quot; &#39;", "& < > \" '"},
		{"<div>Line1</div><div>Line2</div>", "Line1Line2"},
	}
	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.expected {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestServiceName(t *testing.T) {
	svc := NewGmailService()
	if svc.Name() != "gmail" {
		t.Fatalf("expected 'gmail', got '%s'", svc.Name())
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"message": "Not Found"}}`))
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	_, err := client.GetMessage("invalid-id")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}
