package gmail

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/spf13/cobra"
)

type messageClient interface {
	ListMessages(query string, limit int) ([]MessageRef, error)
	ListAllMessageIDs(query string) ([]string, error)
	GetMessage(id string) (*Message, error)
	BatchModify(ids []string, removeLabelIDs []string) error
}

var (
	lookPathFn     = exec.LookPath
	runClaudeFn    = RunClaude
	sendToSlackFn  = SendToSlack
	fetchEmailsFn  = FetchEmails
	requireLoginFn = requireLogin
)

// RegisterCommands adds the `gmail` subtree to parent.
func RegisterCommands(parent *cobra.Command) {
	gmailCmd := &cobra.Command{
		Use:   "gmail",
		Short: "Manage Gmail messages",
	}

	listCmd.Flags().Bool("unread", false, "Show only unread messages")
	listCmd.Flags().String("after", "", "Show messages after this date (YYYY-MM-DD)")
	listCmd.Flags().Int("limit", 20, "Maximum number of messages to return")

	bulkMarkReadCmd.Flags().String("query", "", "Gmail search query (e.g. 'category:promotions')")
	_ = bulkMarkReadCmd.MarkFlagRequired("query")

	summaryCmd.Flags().String("channel", "", "Send summary to a Slack channel")
	summaryCmd.Flags().String("dm", "", "Send summary as a DM to a Slack user ID")
	summaryCmd.Flags().Bool("no-mark-read", false, "Skip marking emails as read (preview mode)")

	gmailCmd.AddCommand(listCmd)
	gmailCmd.AddCommand(readCmd)
	gmailCmd.AddCommand(markReadCmd)
	gmailCmd.AddCommand(bulkMarkReadCmd)
	gmailCmd.AddCommand(summaryCmd)

	parent.AddCommand(gmailCmd)
}

func requireLogin() (*Client, error) {
	token, err := auth.LoadOAuthToken("gmail")
	if err != nil {
		return nil, fmt.Errorf("not logged in to Gmail, run: devpilot login gmail")
	}
	return NewClientFromToken(token), nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List emails",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireLoginFn()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(runList(cmd, client))
	},
}

func runList(cmd *cobra.Command, client messageClient) int {
	unread, _ := cmd.Flags().GetBool("unread")
	after, _ := cmd.Flags().GetString("after")
	limit, _ := cmd.Flags().GetInt("limit")

	var queryParts []string
	if unread {
		queryParts = append(queryParts, "is:unread")
	}
	if after != "" {
		queryParts = append(queryParts, "after:"+after)
	}
	query := strings.Join(queryParts, " ")

	refs, err := client.ListMessages(query, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if len(refs) == 0 {
		if unread {
			fmt.Println("No unread messages.")
		} else {
			fmt.Println("No messages found.")
		}
		return 0
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tFROM\tSUBJECT\tDATE")
	for _, ref := range refs {
		msg, err := client.GetMessage(ref.ID)
		if err != nil {
			continue
		}
		from := GetHeader(msg, "From")
		subject := GetHeader(msg, "Subject")
		date := GetHeader(msg, "Date")
		if len(from) > 30 {
			from = from[:27] + "..."
		}
		if len(subject) > 40 {
			subject = subject[:37] + "..."
		}
		if len(date) > 20 {
			date = date[:20]
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ref.ID, from, subject, date)
	}
	_ = w.Flush()
	return 0
}

var readCmd = &cobra.Command{
	Use:   "read <message-id>",
	Short: "Read a specific email",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireLoginFn()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(runRead(args[0], client))
	},
}

func runRead(id string, client messageClient) int {
	msg, err := client.GetMessage(id)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			fmt.Fprintf(os.Stderr, "Message not found: %s\n", id)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	fmt.Printf("From: %s\n", GetHeader(msg, "From"))
	fmt.Printf("Subject: %s\n", GetHeader(msg, "Subject"))
	fmt.Printf("Date: %s\n", GetHeader(msg, "Date"))
	fmt.Println()
	fmt.Println(GetBody(msg))
	return 0
}

var bulkMarkReadCmd = &cobra.Command{
	Use:   "bulk-mark-read --query <gmail-query>",
	Short: "Mark all emails matching a query as read",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireLoginFn()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		query, _ := cmd.Flags().GetString("query")
		os.Exit(runBulkMarkRead(query, client))
	},
}

func runBulkMarkRead(query string, client messageClient) int {
	fullQuery := "is:unread " + query
	fmt.Printf("Searching for emails matching: %s\n", fullQuery)
	ids, err := client.ListAllMessageIDs(fullQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if len(ids) == 0 {
		fmt.Println("No matching unread messages found.")
		return 0
	}

	fmt.Printf("Found %d messages. Marking as read...\n", len(ids))
	batchSize := 1000
	marked := 0
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := client.BatchModify(ids[i:end], []string{"UNREAD"}); err != nil {
			fmt.Fprintf(os.Stderr, "Error at batch %d-%d: %v\n", i, end, err)
			return 1
		}
		marked += end - i
		fmt.Printf("  Progress: %d/%d\n", marked, len(ids))
	}
	fmt.Printf("Done. Marked %d message(s) as read.\n", len(ids))
	return 0
}

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Summarize today's unread emails using AI",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireLoginFn()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(runSummary(cmd, client))
	},
}

func runSummary(cmd *cobra.Command, client *Client) int {
	if _, err := lookPathFn("claude"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: Claude Code CLI is required but not found on PATH. Install it from https://claude.ai/code")
		return 1
	}

	channel, _ := cmd.Flags().GetString("channel")
	dm, _ := cmd.Flags().GetString("dm")
	noMarkRead, _ := cmd.Flags().GetBool("no-mark-read")
	hasOutputTarget := channel != "" || dm != ""
	if !hasOutputTarget && !cmd.Flags().Changed("no-mark-read") {
		noMarkRead = true
	}

	query := UnreadQuery()
	fmt.Printf("Fetching unread emails (%s)...\n", query)
	ids, err := client.ListAllMessageIDs(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching emails: %v\n", err)
		return 1
	}
	if len(ids) == 0 {
		fmt.Println("No unread emails for today.")
		return 0
	}

	fmt.Printf("Found %d unread email(s). Fetching content...\n", len(ids))
	emails, err := fetchEmailsFn(client, ids)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching email content: %v\n", err)
		return 1
	}

	prompt := BuildPrompt(emails)
	fmt.Println("Generating summary with Claude...")
	summary, err := runClaudeFn(prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println(summary)

	slackTarget := channel
	if dm != "" {
		slackTarget = dm
	}
	if slackTarget != "" {
		fmt.Printf("\nSending summary to Slack (%s)...\n", slackTarget)
		if err := sendToSlackFn(summary, slackTarget); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		} else {
			fmt.Println("Summary sent to Slack.")
		}
	}

	if !noMarkRead {
		fmt.Printf("Marking %d email(s) as read...\n", len(ids))
		batchSize := 1000
		for i := 0; i < len(ids); i += batchSize {
			end := i + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			if err := client.BatchModify(ids[i:end], []string{"UNREAD"}); err != nil {
				fmt.Fprintf(os.Stderr, "Error marking emails as read: %v\n", err)
				return 1
			}
		}
		fmt.Println("Done.")
	}
	return 0
}

var markReadCmd = &cobra.Command{
	Use:   "mark-read <id>...",
	Short: "Mark one or more emails as read",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireLoginFn()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(runMarkRead(args, client))
	},
}

func runMarkRead(ids []string, client messageClient) int {
	if err := client.BatchModify(ids, []string{"UNREAD"}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("Marked %d message(s) as read.\n", len(ids))
	return 0
}
