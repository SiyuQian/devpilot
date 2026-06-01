package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// RegisterCommands installs the `github` command tree.
func RegisterCommands(parent *cobra.Command) {
	parent.AddCommand(newCommand(ghRunner{}))
}

func newCommand(r Runner) *cobra.Command {
	c := &cobra.Command{
		Use:   "github",
		Short: "GitHub pull request and repository activity helpers",
	}
	c.AddCommand(newPRsCommand(r), newRepoCommand(r))
	return c
}

func newPRsCommand(r Runner) *cobra.Command {
	c := &cobra.Command{
		Use:   "prs",
		Short: "List GitHub pull requests",
	}
	c.AddCommand(newReviewQueueCommand(r), newAuthoredCommand(r))
	return c
}

func newReviewQueueCommand(r Runner) *cobra.Command {
	var q PRQuery
	var jsonOut bool
	c := &cobra.Command{
		Use:   "review-queue",
		Short: "List pull requests requesting review from a user",
		Run: func(cmd *cobra.Command, args []string) {
			q.Mode = "review-requested"
			prs, err := listPRs(cmd.Context(), r, q)
			exitOnErr(err)
			printPRs(prs, jsonOut)
		},
	}
	addPRFlags(c, &q, &jsonOut)
	c.Flags().BoolVar(&q.Direct, "direct", false, "Only include PRs directly requesting the user, excluding team requests")
	return c
}

func newAuthoredCommand(r Runner) *cobra.Command {
	var q PRQuery
	var jsonOut bool
	c := &cobra.Command{
		Use:   "authored",
		Short: "List pull requests authored by a user",
		Run: func(cmd *cobra.Command, args []string) {
			q.Mode = "authored"
			prs, err := listPRs(cmd.Context(), r, q)
			exitOnErr(err)
			printPRs(prs, jsonOut)
		},
	}
	addPRFlags(c, &q, &jsonOut)
	return c
}

func addPRFlags(c *cobra.Command, q *PRQuery, jsonOut *bool) {
	c.Flags().StringVar(&q.User, "user", "@me", "GitHub user login or @me")
	c.Flags().StringVar(&q.State, "state", "open", "PR state: open, closed, or all")
	c.Flags().IntVar(&q.Limit, "limit", 200, "Maximum number of pull requests to return")
	c.Flags().StringVar(&q.Repo, "repo", "", "Restrict to one repository, owner/name")
	c.Flags().StringVar(&q.Owner, "owner", "", "Restrict to repositories owned by this user or organization")
	c.Flags().BoolVar(&q.IncludeDrafts, "include-drafts", false, "Include draft pull requests")
	c.Flags().BoolVar(jsonOut, "json", false, "Print normalized JSON")
}

func newRepoCommand(r Runner) *cobra.Command {
	c := &cobra.Command{
		Use:   "repo",
		Short: "Inspect GitHub repositories",
	}
	c.AddCommand(newActivityCommand(r))
	return c
}

func newActivityCommand(r Runner) *cobra.Command {
	var q ActivityQuery
	var since string
	var jsonOut bool
	c := &cobra.Command{
		Use:   "activity <owner/repo>",
		Short: "Show repository activity for a day or recent duration",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			q.Repo = args[0]
			if since != "" {
				d, err := time.ParseDuration(since)
				exitOnErr(err)
				q.Since = d
			}
			digest, err := repoActivity(cmd.Context(), r, q)
			exitOnErr(err)
			printActivity(digest, jsonOut)
		},
	}
	c.Flags().StringVar(&q.Date, "date", "", "Date to inspect in YYYY-MM-DD, default today in the selected timezone")
	c.Flags().StringVar(&since, "since", "", "Recent duration to inspect, for example 24h")
	c.Flags().StringVar(&q.Timezone, "timezone", "", "IANA timezone for date windows, default local timezone")
	c.Flags().IntVar(&q.Limit, "limit", 200, "Maximum items to fetch per source")
	c.Flags().BoolVar(&jsonOut, "json", false, "Print normalized JSON")
	return c
}

func printPRs(prs []PR, jsonOut bool) {
	if jsonOut {
		printJSON(prs)
		return
	}
	if len(prs) == 0 {
		fmt.Println("No pull requests found.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "REPO\tPR\tSTATE\tDRAFT\tAUTHOR\tTITLE\tURL")
	for _, pr := range prs {
		_, _ = fmt.Fprintln(w, strings.Join(prRow(pr), "\t"))
	}
	_ = w.Flush()
}

func printActivity(d *ActivityDigest, jsonOut bool) {
	if jsonOut {
		printJSON(d)
		return
	}
	fmt.Printf("%s activity from %s to %s\n", d.Repo, d.From.Format(time.RFC3339), d.To.Format(time.RFC3339))

	if len(d.Events) == 0 && len(d.PullRequests) == 0 && len(d.Issues) == 0 && len(d.WorkflowRuns) == 0 && len(d.Releases) == 0 {
		fmt.Println("No activity found.")
		return
	}
	if len(d.PullRequests) > 0 {
		fmt.Println("\nPull Requests")
		for _, pr := range d.PullRequests {
			fmt.Printf("  %s  #%d  %-8s  %s  %s\n", pr.UpdatedAt.Format("15:04"), pr.Number, strings.ToLower(pr.State), pr.Title, pr.URL)
		}
	}
	if len(d.Issues) > 0 {
		fmt.Println("\nIssues")
		for _, issue := range d.Issues {
			fmt.Printf("  %s  #%d  %-8s  %s  %s\n", issue.UpdatedAt.Format("15:04"), issue.Number, strings.ToLower(issue.State), issue.Title, issue.URL)
		}
	}
	if len(d.Events) > 0 {
		fmt.Println("\nEvents")
		for _, ev := range d.Events {
			fmt.Printf("  %s  %-18s  %s\n", ev.CreatedAt.Format("15:04"), ev.Actor, ev.Summary)
		}
	}
	if len(d.WorkflowRuns) > 0 {
		fmt.Println("\nWorkflow Runs")
		for _, run := range d.WorkflowRuns {
			status := firstNonEmpty(run.Conclusion, run.Status)
			fmt.Printf("  %s  %-12s  %s on %s  %s\n", run.CreatedAt.Format("15:04"), status, firstNonEmpty(run.DisplayName, run.Name), run.Branch, run.URL)
		}
	}
	if len(d.Releases) > 0 {
		fmt.Println("\nReleases")
		for _, release := range d.Releases {
			fmt.Printf("  %s  %s  %s\n", release.PublishedAt.Format("15:04"), release.TagName, release.Name)
		}
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	exitOnErr(enc.Encode(v))
}

func exitOnErr(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

func init() {
	cobra.EnableCommandSorting = false
}
