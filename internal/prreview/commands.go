package prreview

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// RegisterCommands installs the `pr-review` command tree.
func RegisterCommands(parent *cobra.Command) {
	parent.AddCommand(newCommand(ghRunner{}))
}

func newCommand(r Runner) *cobra.Command {
	c := &cobra.Command{
		Use:   "pr-review",
		Short: "Deterministic halves of the pr-review skill",
	}
	c.AddCommand(newPreflightCommand(r), newPostCommand(r))
	return c
}

func newPreflightCommand(r Runner) *cobra.Command {
	var opts PreflightOptions
	c := &cobra.Command{
		Use:   "preflight <pr-url>",
		Short: "Eligibility gate, PR load, and data collection for a review",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			opts.URL = args[0]
			res, err := RunPreflight(cmd.Context(), r, opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			fmt.Println(preflightSummary(res, opts.OutPath))
		},
	}
	c.Flags().StringVar(&opts.OutPath, "out", "", "Path to write the preflight JSON")
	c.Flags().StringVar(&opts.DiffOut, "diff-out", "", "Path to write the diff (default: <out-dir>/pr_<num>_diff.patch)")
	c.Flags().BoolVar(&opts.Force, "force", false, "Skip the already_reviewed_at_head stop")
	_ = c.MarkFlagRequired("out")
	return c
}

func preflightSummary(res *PreflightResult, outPath string) string {
	if res.Gate.Decision == "stop" {
		return fmt.Sprintf("stop (%s): %s", *res.Gate.StopReason, *res.Gate.StopMessage)
	}
	return fmt.Sprintf("proceed (%s): %d files, %d existing comments -> %s",
		res.ReviewMode, len(res.PR.Files), len(res.ExistingComments), outPath)
}

func newPostCommand(r Runner) *cobra.Command {
	var opts PostOptions
	c := &cobra.Command{
		Use:   "post <pr-url>",
		Short: "Validate anchors and post one combined review",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			opts.URL = args[0]
			out, code, err := RunPost(cmd.Context(), r, opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(code)
			}
			fmt.Println(out)
		},
	}
	c.Flags().StringVar(&opts.FindingsPath, "findings", "", "JSON array of inline findings")
	c.Flags().StringVar(&opts.BodyPath, "body", "", "Markdown file with the review body")
	c.Flags().StringVar(&opts.Event, "event", "", "REQUEST_CHANGES | COMMENT | APPROVE")
	c.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Validate everything, POST nothing")
	c.Flags().StringVar(&opts.ExpectHeadSHA, "expect-head-sha", "", "Fail with head_moved if the PR head differs")
	_ = c.MarkFlagRequired("findings")
	_ = c.MarkFlagRequired("body")
	_ = c.MarkFlagRequired("event")
	return c
}

// execGraphPreflight shells out to this same binary's `graph preflight`,
// which owns the cache lookup and envelope emission.
func execGraphPreflight(ctx context.Context, base, head string) ([]byte, error) {
	bin, err := os.Executable()
	if err != nil {
		bin = "devpilot"
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolving working directory: %w", err)
	}
	cmd := exec.CommandContext(ctx, bin, "graph", "preflight", "--repo", cwd, "--base", base, "--head", head)
	out, err := cmd.Output()
	// graph preflight exits non-zero on fallback conditions but still emits
	// an envelope; keep the output when it is valid JSON.
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("graph preflight: %w", err)
	}
	return out, nil
}
