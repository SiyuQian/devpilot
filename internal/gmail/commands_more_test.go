package gmail

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/spf13/cobra"
)

type fakeMessageClient struct {
	refs       []MessageRef
	ids        []string
	messages   map[string]*Message
	err        error
	batchErr   error
	batchCalls int
}

func (c *fakeMessageClient) ListMessages(query string, limit int) ([]MessageRef, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.refs, nil
}

func (c *fakeMessageClient) ListAllMessageIDs(query string) ([]string, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.ids, nil
}

func (c *fakeMessageClient) GetMessage(id string) (*Message, error) {
	if c.err != nil {
		return nil, c.err
	}
	msg := c.messages[id]
	if msg == nil {
		return nil, errors.New("404 Not Found")
	}
	return msg, nil
}

func (c *fakeMessageClient) BatchModify(ids []string, removeLabelIDs []string) error {
	c.batchCalls++
	return c.batchErr
}

func TestGmailCommandHelpers(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	RegisterCommands(root)
	if cmd, _, err := root.Find([]string{"gmail", "list"}); err != nil || cmd.Name() != "list" {
		t.Fatalf("registered gmail list missing: cmd=%v err=%v", cmd, err)
	}

	msg := &Message{Payload: Payload{
		Headers: []Header{
			{Name: "From", Value: strings.Repeat("a", 35)},
			{Name: "Subject", Value: strings.Repeat("b", 45)},
			{Name: "Date", Value: "Mon, 01 Jun 2026 12:34:56 +0000"},
		},
		Body: Body{Data: "SGVsbG8="},
	}}
	client := &fakeMessageClient{
		refs:     []MessageRef{{ID: "m1"}, {ID: "missing"}},
		ids:      []string{"m1", "m2", "m3"},
		messages: map[string]*Message{"m1": msg},
	}

	list := &cobra.Command{}
	list.Flags().Bool("unread", true, "")
	list.Flags().String("after", "2026-06-01", "")
	list.Flags().Int("limit", 10, "")
	out := captureGmailStdout(t, func() {
		if code := runList(list, client); code != 0 {
			t.Fatalf("runList code = %d", code)
		}
	})
	for _, want := range []string{"ID", "m1", "aaa...", "bbb..."} {
		if !strings.Contains(out, want) {
			t.Errorf("runList output missing %q:\n%s", want, out)
		}
	}

	out = captureGmailStdout(t, func() {
		if code := runRead("m1", client); code != 0 {
			t.Fatalf("runRead code = %d", code)
		}
	})
	if !strings.Contains(out, "Hello") {
		t.Errorf("runRead output = %s", out)
	}

	out = captureGmailStdout(t, func() {
		if code := runBulkMarkRead("from:a", client); code != 0 {
			t.Fatalf("runBulkMarkRead code = %d", code)
		}
	})
	if !strings.Contains(out, "Marked 3 message") || client.batchCalls != 1 {
		t.Errorf("bulk output=%s batchCalls=%d", out, client.batchCalls)
	}

	out = captureGmailStdout(t, func() {
		if code := runMarkRead([]string{"m1", "m2"}, client); code != 0 {
			t.Fatalf("runMarkRead code = %d", code)
		}
	})
	if !strings.Contains(out, "Marked 2 message") {
		t.Errorf("mark output=%s", out)
	}
}

func TestGmailCommandHelperErrorsAndEmpty(t *testing.T) {
	dir := t.TempDir()
	restore := auth.OverrideConfigDir(dir)
	defer restore()
	if _, err := requireLogin(); err == nil {
		t.Fatalf("requireLogin succeeded without token")
	}

	errClient := &fakeMessageClient{err: errors.New("boom")}
	list := &cobra.Command{}
	list.Flags().Bool("unread", false, "")
	list.Flags().String("after", "", "")
	list.Flags().Int("limit", 0, "")
	if code := runList(list, errClient); code == 0 {
		t.Fatalf("runList error code = 0")
	}
	if code := runRead("missing", errClient); code == 0 {
		t.Fatalf("runRead error code = 0")
	}
	if code := runBulkMarkRead("q", errClient); code == 0 {
		t.Fatalf("runBulkMarkRead error code = 0")
	}
	if code := runMarkRead([]string{"m"}, &fakeMessageClient{batchErr: errors.New("bad")}); code == 0 {
		t.Fatalf("runMarkRead error code = 0")
	}

	emptyClient := &fakeMessageClient{}
	out := captureGmailStdout(t, func() {
		if code := runList(list, emptyClient); code != 0 {
			t.Fatalf("runList empty code = %d", code)
		}
	})
	if !strings.Contains(out, "No messages found") {
		t.Errorf("empty list output=%s", out)
	}
	out = captureGmailStdout(t, func() {
		if code := runBulkMarkRead("q", emptyClient); code != 0 {
			t.Fatalf("runBulkMarkRead empty code = %d", code)
		}
	})
	if !strings.Contains(out, "No matching unread") {
		t.Errorf("empty bulk output=%s", out)
	}
}

func TestRunSummary(t *testing.T) {
	oldLookPath, oldRunClaude, oldSendToSlack := lookPathFn, runClaudeFn, sendToSlackFn
	defer func() {
		lookPathFn = oldLookPath
		runClaudeFn = oldRunClaude
		sendToSlackFn = oldSendToSlack
	}()
	lookPathFn = func(string) (string, error) { return "/bin/claude", nil }
	runClaudeFn = func(string) (string, error) { return "summary", nil }
	var sentTarget string
	sendToSlackFn = func(summary, target string) error {
		sentTarget = target
		return nil
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/gmail/v1/users/me/messages":
			_, _ = w.Write([]byte(`{"messages":[{"id":"m1"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/gmail/v1/users/me/messages/m1":
			_, _ = w.Write([]byte(`{"id":"m1","payload":{"headers":[{"name":"Subject","value":"Hi"}],"body":{"data":"SGVsbG8="}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/gmail/v1/users/me/messages/batchModify":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient("token", WithBaseURL(server.URL))
	cmd := &cobra.Command{}
	cmd.Flags().String("channel", "daily", "")
	cmd.Flags().String("dm", "", "")
	cmd.Flags().Bool("no-mark-read", false, "")
	out := captureGmailStdout(t, func() {
		if code := runSummary(cmd, client); code != 0 {
			t.Fatalf("runSummary code = %d", code)
		}
	})
	if !strings.Contains(out, "summary") || sentTarget != "daily" {
		t.Fatalf("summary output=%s sentTarget=%q", out, sentTarget)
	}

	lookPathFn = func(string) (string, error) { return "", errors.New("missing") }
	if code := runSummary(cmd, client); code == 0 {
		t.Fatalf("runSummary missing claude code = 0")
	}
}

func TestRunSummaryBranches(t *testing.T) {
	oldLookPath, oldRunClaude, oldSendToSlack, oldFetchEmails := lookPathFn, runClaudeFn, sendToSlackFn, fetchEmailsFn
	defer func() {
		lookPathFn = oldLookPath
		runClaudeFn = oldRunClaude
		sendToSlackFn = oldSendToSlack
		fetchEmailsFn = oldFetchEmails
	}()
	lookPathFn = func(string) (string, error) { return "/bin/claude", nil }
	runClaudeFn = func(string) (string, error) { return "summary", nil }
	sendToSlackFn = func(summary, target string) error { return errors.New("slack down") }

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/gmail/v1/users/me/messages":
			_, _ = w.Write([]byte(`{"messages":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/gmail/v1/users/me/messages/batchModify":
			http.Error(w, "bad", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cmd := &cobra.Command{}
	cmd.Flags().String("channel", "", "")
	cmd.Flags().String("dm", "", "")
	cmd.Flags().Bool("no-mark-read", false, "")
	client := NewClient("token", WithBaseURL(server.URL))
	if code := runSummary(cmd, client); code != 0 {
		t.Fatalf("runSummary empty code = %d", code)
	}

	fetchEmailsFn = func(*Client, []string) ([]EmailSummary, error) {
		return []EmailSummary{{ID: "m1", Subject: "s", Body: "b"}}, nil
	}
	fullServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/gmail/v1/users/me/messages":
			_, _ = w.Write([]byte(`{"messages":[{"id":"m1"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/gmail/v1/users/me/messages/batchModify":
			http.Error(w, "bad", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer fullServer.Close()
	cmd.Flags().Set("channel", "daily") //nolint:errcheck
	if code := runSummary(cmd, NewClient("token", WithBaseURL(fullServer.URL))); code == 0 {
		t.Fatalf("runSummary batch error code = 0")
	}

	runClaudeFn = func(string) (string, error) { return "", errors.New("claude failed") }
	cmd.Flags().Set("no-mark-read", "true") //nolint:errcheck
	if code := runSummary(cmd, NewClient("token", WithBaseURL(fullServer.URL))); code == 0 {
		t.Fatalf("runSummary claude error code = 0")
	}
}

func captureGmailStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
