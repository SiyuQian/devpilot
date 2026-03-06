package slack

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func RegisterCommands(parent *cobra.Command) {
	slackCmd := &cobra.Command{
		Use:   "slack",
		Short: "Manage Slack messages",
	}

	sendCmd.Flags().String("channel", "", "Channel name to send message to (required)")
	sendCmd.Flags().String("message", "", "Message text (reads from stdin if not provided)")
	sendCmd.MarkFlagRequired("channel") //nolint:errcheck

	slackCmd.AddCommand(sendCmd)

	parent.AddCommand(slackCmd)
}

func requireSlackLogin() (*Client, error) {
	token, err := loadBotToken()
	if err != nil {
		return nil, err
	}
	return NewClient(token), nil
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a message to a Slack channel",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := requireSlackLogin()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		channel, _ := cmd.Flags().GetString("channel")
		message, _ := cmd.Flags().GetString("message")

		if message == "" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				os.Exit(1)
			}
			message = strings.TrimSpace(string(data))
		}

		if message == "" {
			fmt.Fprintln(os.Stderr, "Error: message is required (use --message or pipe via stdin)")
			os.Exit(1)
		}

		channelName := strings.TrimPrefix(channel, "#")

		channelID, err := client.ResolveChannel(channelName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := client.PostMessage(channelID, message); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Message sent to #%s.\n", channelName)
	},
}
