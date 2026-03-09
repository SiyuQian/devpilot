package gmail

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		maxLen   int
		expected string
	}{
		{"short body", "hello", 10, "hello"},
		{"exact limit", "hello", 5, "hello"},
		{"over limit", "hello world", 5, "hello[truncated]"},
		{"empty body", "", 10, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateBody(tt.body, tt.maxLen)
			if got != tt.expected {
				t.Errorf("TruncateBody(%q, %d) = %q, want %q", tt.body, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	emails := []EmailSummary{
		{
			ID:      "msg1",
			From:    "alice@example.com",
			Subject: "Meeting Tomorrow",
			Date:    "Mon, 1 Jan 2024 09:00:00",
			Body:    "Let's meet at 3pm.",
		},
		{
			ID:      "msg2",
			From:    "bob@example.com",
			Subject: "Newsletter",
			Date:    "Mon, 1 Jan 2024 10:00:00",
			Body:    "Check out our latest updates.",
		},
	}

	prompt := BuildPrompt(emails)

	// Check prompt contains key elements
	if !strings.Contains(prompt, "--- 2 emails ---") {
		t.Error("expected prompt to contain email count")
	}
	if !strings.Contains(prompt, "Email 1:") {
		t.Error("expected prompt to contain Email 1:")
	}
	if !strings.Contains(prompt, "Email 2:") {
		t.Error("expected prompt to contain Email 2:")
	}
	if !strings.Contains(prompt, "From: alice@example.com") {
		t.Error("expected prompt to contain alice's email")
	}
	if !strings.Contains(prompt, "Subject: Meeting Tomorrow") {
		t.Error("expected prompt to contain first subject")
	}
	if !strings.Contains(prompt, "Body:\nLet's meet at 3pm.") {
		t.Error("expected prompt to contain first body")
	}
	if !strings.Contains(prompt, "From: bob@example.com") {
		t.Error("expected prompt to contain bob's email")
	}
	if !strings.Contains(prompt, "ACTION REQUIRED") {
		t.Error("expected prompt to contain classification instructions")
	}
}

func TestBuildPromptWithTruncation(t *testing.T) {
	longBody := strings.Repeat("a", 1500)
	emails := []EmailSummary{
		{
			ID:      "msg1",
			From:    "test@example.com",
			Subject: "Long Email",
			Date:    "Mon, 1 Jan 2024",
			Body:    TruncateBody(longBody, 1000),
		},
	}

	prompt := BuildPrompt(emails)
	if !strings.Contains(prompt, "[truncated]") {
		t.Error("expected prompt to contain truncated body")
	}
	// The body in the prompt should be 1000 chars + "[truncated]"
	if !strings.Contains(prompt, strings.Repeat("a", 1000)+"[truncated]") {
		t.Error("expected body to be truncated at 1000 chars")
	}
}

func TestTodayQuery(t *testing.T) {
	query := TodayQuery()
	today := time.Now().Format("2006/01/02")
	expected := "is:unread after:" + today
	if query != expected {
		t.Errorf("TodayQuery() = %q, want %q", query, expected)
	}
}

func TestFetchEmails(t *testing.T) {
	// Mock Gmail API server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract message ID from path
		parts := strings.Split(r.URL.Path, "/")
		msgID := parts[len(parts)-1]

		var subject, body string
		switch msgID {
		case "msg1":
			subject = "First Email"
			body = "Body of first email"
		case "msg2":
			subject = "Second Email"
			body = "Body of second email"
		case "msg3":
			subject = "Third Email"
			body = strings.Repeat("x", 1500) // Will be truncated
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		msg := Message{
			ID: msgID,
			Payload: Payload{
				Headers: []Header{
					{Name: "From", Value: msgID + "@example.com"},
					{Name: "Subject", Value: subject},
					{Name: "Date", Value: "2024-01-15"},
				},
				MimeType: "text/plain",
				Body: Body{
					Data: base64.URLEncoding.EncodeToString([]byte(body)),
				},
			},
		}
		json.NewEncoder(w).Encode(msg)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	emails, err := FetchEmails(client, []string{"msg1", "msg2", "msg3"})
	if err != nil {
		t.Fatalf("FetchEmails error: %v", err)
	}

	if len(emails) != 3 {
		t.Fatalf("expected 3 emails, got %d", len(emails))
	}

	// Check first email
	found := false
	for _, e := range emails {
		if e.ID == "msg1" {
			found = true
			if e.From != "msg1@example.com" {
				t.Errorf("expected msg1@example.com, got %s", e.From)
			}
			if e.Subject != "First Email" {
				t.Errorf("expected First Email, got %s", e.Subject)
			}
		}
	}
	if !found {
		t.Error("msg1 not found in results")
	}

	// Check truncation on third email
	for _, e := range emails {
		if e.ID == "msg3" {
			if !strings.HasSuffix(e.Body, "[truncated]") {
				t.Errorf("expected msg3 body to be truncated, got %d chars", len(e.Body))
			}
		}
	}
}

func TestFetchEmailsPartialFailure(t *testing.T) {
	// Mock server that fails for one message
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		msgID := parts[len(parts)-1]

		if msgID == "msg2" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
			return
		}

		msg := Message{
			ID: msgID,
			Payload: Payload{
				Headers: []Header{
					{Name: "From", Value: "test@example.com"},
					{Name: "Subject", Value: "Test"},
					{Name: "Date", Value: "2024-01-15"},
				},
				MimeType: "text/plain",
				Body: Body{
					Data: base64.URLEncoding.EncodeToString([]byte("body")),
				},
			},
		}
		json.NewEncoder(w).Encode(msg)
	}))
	defer srv.Close()

	client := NewClient("test-token", WithBaseURL(srv.URL))
	emails, err := FetchEmails(client, []string{"msg1", "msg2", "msg3"})
	if err != nil {
		t.Fatalf("FetchEmails error: %v", err)
	}

	// Should get 2 emails (msg2 failed)
	if len(emails) != 2 {
		t.Fatalf("expected 2 emails (1 failed), got %d", len(emails))
	}
}
