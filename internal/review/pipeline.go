package review

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/siyuqian/devpilot/internal/executor"
)

// PipelineResult holds the final output of the review pipeline.
type PipelineResult struct {
	Summary    string
	Assessment string
	Findings   []ScoredFinding
	Verdict    string // "APPROVE" or "REQUEST_CHANGES"
	// PRInfo for posting
	PR *PRInfo
}

// PipelineOption configures the review pipeline.
type PipelineOption func(*pipelineConfig)

type pipelineConfig struct {
	reviewModel  string
	scoringModel string
	threshold    int
	postToGitHub bool
	execOpts     []executor.ExecutorOption
	eventHandler executor.ClaudeEventHandler
}

// WithReviewModel sets the model for Round 1 (default Opus).
func WithReviewModel(model string) PipelineOption {
	return func(c *pipelineConfig) {
		c.reviewModel = model
	}
}

// WithScoringModel sets the model for Round 2 (default Haiku).
func WithScoringModel(model string) PipelineOption {
	return func(c *pipelineConfig) {
		c.scoringModel = model
	}
}

// WithThreshold sets the confidence score cutoff (default 50).
func WithThreshold(threshold int) PipelineOption {
	return func(c *pipelineConfig) {
		c.threshold = threshold
	}
}

// WithPipelinePostToGitHub controls whether posting instructions are included.
func WithPipelinePostToGitHub(post bool) PipelineOption {
	return func(c *pipelineConfig) {
		c.postToGitHub = post
	}
}

// WithPipelineExecutorOptions passes additional executor options.
func WithPipelineExecutorOptions(opts ...executor.ExecutorOption) PipelineOption {
	return func(c *pipelineConfig) {
		c.execOpts = append(c.execOpts, opts...)
	}
}

// WithPipelineEventHandler sets a callback for streaming events.
func WithPipelineEventHandler(handler executor.ClaudeEventHandler) PipelineOption {
	return func(c *pipelineConfig) {
		c.eventHandler = handler
	}
}

const (
	DefaultReviewModel  = "claude-opus-4-6"
	DefaultScoringModel = "claude-haiku-4-5-20251001"
	DefaultThreshold    = 50
)

// RunPipeline executes the full multi-round review pipeline.
func RunPipeline(ctx context.Context, prURL string, opts ...PipelineOption) (*PipelineResult, error) {
	pr, err := ParsePRURL(prURL)
	if err != nil {
		return nil, err
	}

	cfg := &pipelineConfig{
		reviewModel:  DefaultReviewModel,
		scoringModel: DefaultScoringModel,
		threshold:    DefaultThreshold,
		postToGitHub: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Phase 1: Gather context
	fmt.Fprintln(os.Stderr, "[phase] Gathering PR context...")
	rc, err := GatherContext(pr)
	if err != nil {
		return nil, fmt.Errorf("gather context: %w", err)
	}

	// Phase 2: Round 1 — Review
	fmt.Fprintln(os.Stderr, "[phase] Reviewing code...")
	chunks := ChunkDiff(rc.Diff)
	var allFindings []Finding
	var summary, assessment string

	for i, chunk := range chunks {
		prompt := BuildPrompt(rc, chunk.Diff)
		reviewExec := newRoundExecutor(cfg.reviewModel, cfg.eventHandler, cfg.execOpts)
		result, err := reviewExec.Run(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("round 1: %w", err)
		}

		output, err := ParseReviewOutput(result.Stdout)
		if err != nil {
			// Retry once
			result, err = reviewExec.Run(ctx, prompt+"\n\nYour previous response was not valid JSON. Respond with ONLY a valid JSON object.")
			if err != nil {
				return nil, fmt.Errorf("round 1 retry: %w", err)
			}
			output, err = ParseReviewOutput(result.Stdout)
			if err != nil {
				return nil, fmt.Errorf("round 1 parse after retry: %w", err)
			}
		}

		allFindings = append(allFindings, output.Findings...)
		// Use the last chunk's summary/assessment
		if i == len(chunks)-1 {
			summary = output.Summary
			assessment = output.Assessment
		}
	}

	// Phase 3: Round 2 — Scoring (skip if no findings)
	var scored []ScoredFinding
	if len(allFindings) == 0 {
		fmt.Fprintln(os.Stderr, "[phase] No findings — skipping scoring")
	} else {
		fmt.Fprintf(os.Stderr, "[phase] Scoring %d findings...\n", len(allFindings))
		scoringPrompt := BuildScoringPrompt(rc, allFindings)
		scoringExec := newRoundExecutor(cfg.scoringModel, nil, nil)
		scoreResult, err := scoringExec.Run(ctx, scoringPrompt)
		if err != nil {
			return nil, fmt.Errorf("round 2: %w", err)
		}

		scores, err := ParseScores(scoreResult.Stdout)
		if err != nil {
			// Retry once
			scoreResult, err = scoringExec.Run(ctx, scoringPrompt+"\n\nYour previous response was not valid JSON. Respond with ONLY a valid JSON array.")
			if err != nil {
				return nil, fmt.Errorf("round 2 retry: %w", err)
			}
			scores, err = ParseScores(scoreResult.Stdout)
			if err != nil {
				return nil, fmt.Errorf("round 2 parse after retry: %w", err)
			}
		}

		// Merge scores with findings and filter
		scored = mergeAndFilter(allFindings, scores, cfg.threshold)
	}

	// Determine verdict
	verdict := "APPROVE"
	for _, f := range scored {
		if f.Severity == "CRITICAL" {
			verdict = "REQUEST_CHANGES"
			break
		}
	}

	result := &PipelineResult{
		Summary:    summary,
		Assessment: assessment,
		Findings:   scored,
		Verdict:    verdict,
		PR:         pr,
	}

	// Phase 4: Post to GitHub
	if cfg.postToGitHub {
		fmt.Fprintln(os.Stderr, "[phase] Posting review to GitHub...")
		if err := PostReview(pr, result, rc.Diff); err != nil {
			// Try LLM fallback
			fallbackErr := postReviewFallback(ctx, pr, result, rc.Diff, err)
			if fallbackErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to post review to GitHub: %v (fallback also failed: %v)\n", err, fallbackErr)
			}
		}
	}

	return result, nil
}

func newRoundExecutor(model string, handler executor.ClaudeEventHandler, extraOpts []executor.ExecutorOption) *executor.Executor {
	args := []string{"-p", "--output-format", "stream-json", "--model", model, "--verbose"}
	opts := []executor.ExecutorOption{executor.WithCommand("claude", args...)}
	if handler != nil {
		opts = append(opts, executor.WithClaudeEventHandler(handler))
	}
	opts = append(opts, extraOpts...)
	return executor.NewExecutor(opts...)
}

// mergeAndFilter combines findings with scores and filters by threshold.
func mergeAndFilter(findings []Finding, scores []ScoreEntry, threshold int) []ScoredFinding {
	scoreMap := make(map[int]int)
	for _, s := range scores {
		scoreMap[s.Index] = s.Score
	}

	var result []ScoredFinding
	for i, f := range findings {
		score, ok := scoreMap[i]
		if !ok {
			score = 50 // default if scorer missed this finding
		}
		if score >= threshold {
			result = append(result, ScoredFinding{Finding: f, Score: score})
		}
	}
	return result
}

// FormatMarkdown renders the pipeline result as human-readable markdown.
func FormatMarkdown(result *PipelineResult) string {
	var b strings.Builder

	b.WriteString("## Summary\n\n")
	b.WriteString(result.Summary)
	b.WriteString("\n\n")

	b.WriteString("## Verdict\n\n")
	b.WriteString(result.Verdict)
	b.WriteString("\n\n")

	if len(result.Findings) > 0 {
		b.WriteString("## Findings\n\n")
		// Group by file
		fileFindings := make(map[string][]ScoredFinding)
		var fileOrder []string
		for _, f := range result.Findings {
			if _, seen := fileFindings[f.File]; !seen {
				fileOrder = append(fileOrder, f.File)
			}
			fileFindings[f.File] = append(fileFindings[f.File], f)
		}
		for _, file := range fileOrder {
			fmt.Fprintf(&b, "### `%s`\n\n", file)
			for _, f := range fileFindings[file] {
				if f.EndLine > 0 {
					fmt.Fprintf(&b, "**[%s]** Line %d-%d: %s (score: %d)\n\n", f.Severity, f.Line, f.EndLine, f.Title, f.Score)
				} else {
					fmt.Fprintf(&b, "**[%s]** Line %d: %s (score: %d)\n\n", f.Severity, f.Line, f.Title, f.Score)
				}
				b.WriteString(f.Explanation)
				b.WriteString("\n\n")
				if f.Suggestion != "" {
					b.WriteString("```suggestion\n")
					b.WriteString(f.Suggestion)
					b.WriteString("\n```\n\n")
				}
			}
		}
	}

	b.WriteString("## Overall Assessment\n\n")
	b.WriteString(result.Assessment)
	b.WriteString("\n")

	return b.String()
}

// IsApprovedResult checks if a pipeline result indicates approval.
func IsApprovedResult(result *PipelineResult) bool {
	return result.Verdict == "APPROVE"
}
