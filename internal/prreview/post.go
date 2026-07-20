package prreview

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const maxReviewBodyBytes = 65 * 1024

// Finding is one inline review comment to post.
type Finding struct {
	Path      string  `json:"path"`
	Line      int     `json:"line"`
	Side      string  `json:"side"`
	StartLine *int    `json:"start_line"`
	StartSide *string `json:"start_side"`
	Body      string  `json:"body"`
}

// PostOptions are the flags of `devpilot pr-review post`.
type PostOptions struct {
	URL           string
	FindingsPath  string
	BodyPath      string
	Event         string
	DryRun        bool
	ExpectHeadSHA string // optional: fail with head_moved if the PR head differs
}

// Exit codes for `pr-review post`.
const (
	postExitOK         = 0
	postExitValidation = 1
	postExitRejected   = 2
)

// RunPost validates the findings against the PR's current diff and issues one
// combined review POST. It returns the review URL (or dry-run summary) and
// the process exit code; err carries the failure detail for stderr.
func RunPost(ctx context.Context, r Runner, opts PostOptions) (string, int, error) {
	owner, repo, num, err := parsePRURL(opts.URL)
	if err != nil {
		return "", postExitValidation, err
	}
	if opts.Event != "REQUEST_CHANGES" && opts.Event != "COMMENT" && opts.Event != "APPROVE" {
		return "", postExitValidation, fmt.Errorf("--event must be REQUEST_CHANGES, COMMENT, or APPROVE, got %q", opts.Event)
	}

	findingsRaw, err := os.ReadFile(opts.FindingsPath)
	if err != nil {
		return "", postExitValidation, fmt.Errorf("reading findings: %w", err)
	}
	var findings []Finding
	if err := json.Unmarshal(findingsRaw, &findings); err != nil {
		return "", postExitValidation, fmt.Errorf("decoding findings: %w", err)
	}
	body, err := os.ReadFile(opts.BodyPath)
	if err != nil {
		return "", postExitValidation, fmt.Errorf("reading body: %w", err)
	}

	if opts.Event == "APPROVE" && len(findings) > 0 {
		return "", postExitValidation, fmt.Errorf("APPROVE with %d inline findings: severity/event mismatch", len(findings))
	}
	if len(body) >= maxReviewBodyBytes {
		return "", postExitValidation, fmt.Errorf("review body is %d bytes, over the %d-byte limit", len(body), maxReviewBodyBytes)
	}

	prPath := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, num)
	out, err := r.Run(ctx, "api", prPath)
	if err != nil {
		return "", postExitValidation, fmt.Errorf("fetching PR: %w", err)
	}
	var pr prAPI
	if err := json.Unmarshal(out, &pr); err != nil {
		return "", postExitValidation, fmt.Errorf("decoding PR: %w", err)
	}
	if opts.ExpectHeadSHA != "" && pr.Head.SHA != opts.ExpectHeadSHA {
		return "", postExitValidation, fmt.Errorf("head_moved: %s -> %s", opts.ExpectHeadSHA, pr.Head.SHA)
	}

	diff, err := fetchDiff(ctx, r, owner, repo, num, "", pr.Head.SHA)
	if err != nil {
		return "", postExitValidation, err
	}
	anchors := diffAnchors(parseDiff(diff))
	for i, f := range findings {
		if f.Side == "" {
			findings[i].Side = "RIGHT"
			f.Side = "RIGHT"
		}
		if f.Side != "RIGHT" && f.Side != "LEFT" {
			return "", postExitValidation, fmt.Errorf("finding %d: side must be RIGHT or LEFT, got %q", i, f.Side)
		}
		if strings.TrimSpace(f.Body) == "" {
			return "", postExitValidation, fmt.Errorf("finding %d: empty body", i)
		}
		if f.StartLine != nil && *f.StartLine > f.Line {
			return "", postExitValidation, fmt.Errorf("finding %d: start_line %d > line %d", i, *f.StartLine, f.Line)
		}
		if !anchors[anchorKey(f.Path, f.Side, f.Line)] {
			return "", postExitValidation, fmt.Errorf("finding %d: (%s, %d, %s) is not in the diff at head %s", i, f.Path, f.Line, f.Side, pr.Head.SHA)
		}
	}

	payload, err := json.Marshal(reviewPayload(string(body), opts.Event, findings))
	if err != nil {
		return "", postExitValidation, fmt.Errorf("encoding review payload: %w", err)
	}

	if opts.DryRun {
		summary := fmt.Sprintf("dry-run ok: event=%s, %d inline comments, body %d bytes, head %s",
			opts.Event, len(findings), len(body), pr.Head.SHA)
		return summary, postExitOK, nil
	}

	resp, err := r.RunInput(ctx, payload, "api", "-X", "POST", prPath+"/reviews", "--input", "-")
	if err != nil {
		return "", postExitRejected, fmt.Errorf("github rejected the review POST: %w", err)
	}
	var posted struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(resp, &posted); err != nil || posted.HTMLURL == "" {
		return fmt.Sprintf("posted review on %s/%s#%d", owner, repo, num), postExitOK, nil
	}
	return posted.HTMLURL, postExitOK, nil
}

func reviewPayload(body, event string, findings []Finding) map[string]any {
	comments := make([]map[string]any, 0, len(findings))
	for _, f := range findings {
		c := map[string]any{"path": f.Path, "line": f.Line, "side": f.Side, "body": f.Body}
		if f.StartLine != nil {
			c["start_line"] = *f.StartLine
		}
		if f.StartSide != nil {
			c["start_side"] = *f.StartSide
		}
		comments = append(comments, c)
	}
	return map[string]any{"event": event, "body": body, "comments": comments}
}
