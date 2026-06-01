package slack

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type fakeSlackClient struct {
	openID     string
	resolveID  string
	postErr    error
	resolveErr error
	openErr    error
	postedTo   string
	postedText string
}

func (c *fakeSlackClient) OpenConversation(userID string) (string, error) {
	return c.openID, c.openErr
}

func (c *fakeSlackClient) ResolveChannel(name string) (string, error) {
	return c.resolveID, c.resolveErr
}

func (c *fakeSlackClient) PostMessage(channelID, text string) error {
	c.postedTo = channelID
	c.postedText = text
	return c.postErr
}

func TestRunSendChannelAndDM(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("channel", "#general", "")
	cmd.Flags().String("message", "hello", "")
	client := &fakeSlackClient{resolveID: "C1", openID: "D1"}
	out := captureSlackStdout(t, func() {
		if code := runSend(cmd, client, strings.NewReader("")); code != 0 {
			t.Fatalf("runSend channel code = %d", code)
		}
	})
	if !strings.Contains(out, "to #general") || client.postedTo != "C1" || client.postedText != "hello" {
		t.Fatalf("channel send failed output=%s client=%#v", out, client)
	}

	cmd = &cobra.Command{}
	cmd.Flags().String("channel", "U123", "")
	cmd.Flags().String("message", "", "")
	client = &fakeSlackClient{resolveID: "C1", openID: "D1"}
	out = captureSlackStdout(t, func() {
		if code := runSend(cmd, client, strings.NewReader("from stdin\n")); code != 0 {
			t.Fatalf("runSend dm code = %d", code)
		}
	})
	if !strings.Contains(out, "as DM") || client.postedTo != "D1" || client.postedText != "from stdin" {
		t.Fatalf("dm send failed output=%s client=%#v", out, client)
	}
}

func TestRunSendErrors(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("channel", "#general", "")
	cmd.Flags().String("message", "", "")
	if code := runSend(cmd, &fakeSlackClient{}, strings.NewReader("")); code == 0 {
		t.Fatalf("empty message code = 0")
	}
	cmd.Flags().Set("message", "hi") //nolint:errcheck
	if code := runSend(cmd, &fakeSlackClient{resolveErr: errors.New("no channel")}, strings.NewReader("")); code == 0 {
		t.Fatalf("resolve error code = 0")
	}
	if code := runSend(cmd, &fakeSlackClient{resolveID: "C1", postErr: errors.New("post")}, strings.NewReader("")); code == 0 {
		t.Fatalf("post error code = 0")
	}

	dm := &cobra.Command{}
	dm.Flags().String("channel", "U123", "")
	dm.Flags().String("message", "hi", "")
	if code := runSend(dm, &fakeSlackClient{openErr: errors.New("open")}, strings.NewReader("")); code == 0 {
		t.Fatalf("open error code = 0")
	}
}

func TestRegisterCommands(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	RegisterCommands(root)
	cmd, _, err := root.Find([]string{"slack", "send"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if cmd.Name() != "send" {
		t.Fatalf("cmd = %q, want send", cmd.Name())
	}
}

func captureSlackStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy: %v", err)
	}
	return buf.String()
}
