package review

import (
	"bytes"
	"testing"

	"github.com/siyuqian/devpilot/internal/executor"
)

func TestReviewStreamer_TextBlock(t *testing.T) {
	var out, errBuf bytes.Buffer
	s := &reviewStreamer{out: &out, err: &errBuf, showTTY: true}

	s.HandleEvent(executor.ClaudeAssistantMsg{
		Content: []executor.ContentBlock{
			executor.TextBlock{Text: "Review looks good."},
		},
	})

	if got := out.String(); got != "Review looks good." {
		t.Errorf("stdout = %q, want %q", got, "Review looks good.")
	}
	if errBuf.Len() != 0 {
		t.Errorf("stderr should be empty, got %q", errBuf.String())
	}
}

func TestReviewStreamer_ToolUse(t *testing.T) {
	var out, errBuf bytes.Buffer
	s := &reviewStreamer{out: &out, err: &errBuf, showTTY: true}

	s.HandleEvent(executor.ClaudeAssistantMsg{
		Content: []executor.ContentBlock{
			executor.ToolUseBlock{Name: "Bash"},
		},
	})

	if out.Len() != 0 {
		t.Errorf("stdout should be empty, got %q", out.String())
	}
	if got := errBuf.String(); got != "[tool] Bash\n" {
		t.Errorf("stderr = %q, want %q", got, "[tool] Bash\n")
	}
}

func TestReviewStreamer_ToolUseSuppressedWhenNotTTY(t *testing.T) {
	var out, errBuf bytes.Buffer
	s := &reviewStreamer{out: &out, err: &errBuf, showTTY: false}

	s.HandleEvent(executor.ClaudeAssistantMsg{
		Content: []executor.ContentBlock{
			executor.ToolUseBlock{Name: "Bash"},
		},
	})

	if errBuf.Len() != 0 {
		t.Errorf("stderr should be empty when not TTY, got %q", errBuf.String())
	}
}

func TestReviewStreamer_ResultAddsNewline(t *testing.T) {
	var out, errBuf bytes.Buffer
	s := &reviewStreamer{out: &out, err: &errBuf, showTTY: true}

	s.HandleEvent(executor.ClaudeAssistantMsg{
		Content: []executor.ContentBlock{
			executor.TextBlock{Text: "Done"},
		},
	})
	s.HandleEvent(executor.ClaudeResultMsg{})

	if got := out.String(); got != "Done\n" {
		t.Errorf("stdout = %q, want %q", got, "Done\n")
	}
}
