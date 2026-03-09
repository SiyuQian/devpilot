package gmail

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// EmailSummary holds a fetched email's key fields for summarization.
type EmailSummary struct {
	ID      string
	From    string
	Subject string
	Date    string
	Body    string
}

// UnreadQuery returns a Gmail query for all unread emails.
func UnreadQuery() string {
	return "is:unread"
}

// FetchEmails fetches full email content for all given message IDs concurrently.
// Uses a bounded semaphore of 10 goroutines.
func FetchEmails(client *Client, ids []string) ([]EmailSummary, error) {
	type result struct {
		idx   int
		email EmailSummary
		err   error
	}

	results := make([]result, len(ids))
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for i, id := range ids {
		wg.Add(1)
		go func(idx int, msgID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			msg, err := client.GetMessage(msgID)
			if err != nil {
				results[idx] = result{idx: idx, err: err}
				return
			}

			body := GetBody(msg)
			results[idx] = result{
				idx: idx,
				email: EmailSummary{
					ID:      msgID,
					From:    GetHeader(msg, "From"),
					Subject: GetHeader(msg, "Subject"),
					Date:    GetHeader(msg, "Date"),
					Body:    TruncateBody(body, 1000),
				},
			}
		}(i, id)
	}

	wg.Wait()

	var emails []EmailSummary
	for _, r := range results {
		if r.err != nil {
			// Skip emails that failed to fetch
			continue
		}
		emails = append(emails, r.email)
	}
	return emails, nil
}

// TruncateBody truncates a string to maxLen characters, appending "[truncated]" if truncated.
func TruncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "[truncated]"
}

// BuildPrompt constructs the prompt string for claude -p from fetched emails.
func BuildPrompt(emails []EmailSummary) string {
	var sb strings.Builder
	sb.WriteString("You are an email assistant. Summarize the following unread emails into a concise, actionable digest.\n")
	sb.WriteString("Group them by priority:\n")
	sb.WriteString("- ACTION REQUIRED: Needs response, review, decision, or action\n")
	sb.WriteString("- INFORMATIONAL: Worth knowing but no action needed\n")
	sb.WriteString("- PROMOTIONAL/NOISE: Newsletters, marketing (count only, don't summarize individually)\n\n")
	sb.WriteString("Format the output as a clean digest with counts per category. Keep each summary to one line (~100 chars max).\n\n")
	fmt.Fprintf(&sb, "--- %d emails ---\n\n", len(emails))

	for i, e := range emails {
		fmt.Fprintf(&sb, "Email %d:\n", i+1)
		fmt.Fprintf(&sb, "From: %s\n", e.From)
		fmt.Fprintf(&sb, "Subject: %s\n", e.Subject)
		fmt.Fprintf(&sb, "Date: %s\n", e.Date)
		fmt.Fprintf(&sb, "Body:\n%s\n\n", e.Body)
	}

	return sb.String()
}

// RunClaude invokes `claude -p` with the given prompt and returns the output.
func RunClaude(prompt string) (string, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("Claude Code CLI is required but not found on PATH. Install it from https://claude.ai/code")
	}

	cmd := exec.Command(claudePath, "-p", prompt)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude -p failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude -p failed: %w", err)
	}

	summary := strings.TrimSpace(string(output))
	if summary == "" {
		return "", fmt.Errorf("claude -p produced empty output")
	}
	return summary, nil
}

// SendToSlack sends a message to Slack via `devpilot slack send`.
func SendToSlack(message, channel string) error {
	devpilotPath, err := exec.LookPath("devpilot")
	if err != nil {
		return fmt.Errorf("devpilot not found on PATH")
	}

	cmd := exec.Command(devpilotPath, "slack", "send", "--channel", channel, "--message", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("slack send failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
