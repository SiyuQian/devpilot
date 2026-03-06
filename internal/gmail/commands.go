package gmail

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/internal/auth"
)

func RegisterCommands(parent *cobra.Command) {
	gmailCmd := &cobra.Command{
		Use:   "gmail",
		Short: "Manage Gmail messages",
	}

	listCmd.Flags().Bool("unread", false, "Show only unread messages")
	listCmd.Flags().String("after", "", "Show messages after this date (YYYY-MM-DD)")
	listCmd.Flags().Int("limit", 20, "Maximum number of messages to return")

	gmailCmd.AddCommand(listCmd)
	gmailCmd.AddCommand(readCmd)
	gmailCmd.AddCommand(markReadCmd)

	parent.AddCommand(gmailCmd)
}

func requireLogin() (*Client, error) {
	token, err := auth.LoadOAuthToken("gmail")
	if err != nil {
		return nil, fmt.Errorf("Not logged in to Gmail. Run: devpilot login gmail")
	}
	return NewClientFromToken(token), nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List emails",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireLogin()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

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
			os.Exit(1)
		}

		if len(refs) == 0 {
			if unread {
				fmt.Println("No unread messages.")
			} else {
				fmt.Println("No messages found.")
			}
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tFROM\tSUBJECT\tDATE")
		for _, ref := range refs {
			msg, err := client.GetMessage(ref.ID)
			if err != nil {
				continue
			}
			from := GetHeader(msg, "From")
			subject := GetHeader(msg, "Subject")
			date := GetHeader(msg, "Date")

			// Truncate long fields for table display
			if len(from) > 30 {
				from = from[:27] + "..."
			}
			if len(subject) > 40 {
				subject = subject[:37] + "..."
			}
			if len(date) > 20 {
				date = date[:20]
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ref.ID, from, subject, date)
		}
		w.Flush()
	},
}

var readCmd = &cobra.Command{
	Use:   "read <message-id>",
	Short: "Read a specific email",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireLogin()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		msg, err := client.GetMessage(args[0])
		if err != nil {
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
				fmt.Fprintf(os.Stderr, "Message not found: %s\n", args[0])
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			os.Exit(1)
		}

		fmt.Printf("From: %s\n", GetHeader(msg, "From"))
		fmt.Printf("Subject: %s\n", GetHeader(msg, "Subject"))
		fmt.Printf("Date: %s\n", GetHeader(msg, "Date"))
		fmt.Println()
		fmt.Println(GetBody(msg))
	},
}

var markReadCmd = &cobra.Command{
	Use:   "mark-read <id>...",
	Short: "Mark one or more emails as read",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireLogin()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := client.BatchModify(args, []string{"UNREAD"}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Marked %d message(s) as read.\n", len(args))
	},
}
