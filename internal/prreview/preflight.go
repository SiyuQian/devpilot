package prreview

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	schemaPreflight = "devpilot.pr-review.preflight/v1"
	reviewMarker    = "<!-- devpilot:pr-review"
	graphTimeout    = 30 * time.Second
)

// Gate is the eligibility decision for a PR.
type Gate struct {
	Decision    string  `json:"decision"` // "proceed" or "stop"
	StopReason  *string `json:"stop_reason"`
	StopMessage *string `json:"stop_message"`
}

// PRFile is one changed file in the PR.
type PRFile struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// PRInfo is the PR metadata embedded in the preflight output.
type PRInfo struct {
	Owner   string   `json:"owner"`
	Repo    string   `json:"repo"`
	Number  int      `json:"number"`
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Author  string   `json:"author"`
	BaseSHA string   `json:"base_sha"`
	HeadSHA string   `json:"head_sha"`
	Files   []PRFile `json:"files"`
}

// ExistingComment is an inline review comment already on the PR.
type ExistingComment struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Side     string `json:"side"`
	Body     string `json:"body"`
	User     string `json:"user"`
	CommitID string `json:"commit_id"`
}

// PreflightResult is the JSON written to --out.
type PreflightResult struct {
	Schema             string            `json:"schema"`
	Gate               Gate              `json:"gate"`
	ReviewMode         string            `json:"review_mode"`
	LastReviewedSHA    *string           `json:"last_reviewed_sha"`
	PR                 PRInfo            `json:"pr"`
	DiffPath           string            `json:"diff_path"`
	ExistingComments   []ExistingComment `json:"existing_comments"`
	Graph              json.RawMessage   `json:"graph"`
	DependencyManifest Manifest          `json:"dependency_manifest"`
}

// PreflightOptions are the flags of `devpilot pr-review preflight`.
type PreflightOptions struct {
	URL     string
	OutPath string
	DiffOut string
	Force   bool
	// GraphPreflight runs `devpilot graph preflight` and returns its stdout.
	// Overridable in tests; nil selects the self-exec implementation.
	GraphPreflight func(ctx context.Context, base, head string) ([]byte, error)
}

var prURLRE = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/pull/(\d+)`)

func parsePRURL(url string) (owner, repo string, num int, err error) {
	m := prURLRE.FindStringSubmatch(strings.TrimSpace(url))
	if m == nil {
		return "", "", 0, fmt.Errorf("not a GitHub PR URL: %q", url)
	}
	num, err = strconv.Atoi(m[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("parsing PR number: %w", err)
	}
	return m[1], m[2], num, nil
}

var automationAuthors = map[string]bool{
	"dependabot[bot]": true, "dependabot-preview[bot]": true,
	"renovate[bot]": true, "renovate-bot": true, "mend[bot]": true,
	"release-please[bot]": true, "github-actions[bot]": true,
	"snyk-bot": true, "greenkeeper[bot]": true,
}

var generatedPatterns = []string{
	"*.lock", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "go.sum",
	"*.pb.go", "*_generated.go", "*.gen.go", "*.generated.*",
	"generated/*", "*/generated/*", "vendor/*", "dist/*", "node_modules/*",
	"*.min.js", "*.min.css", "*.snap",
}

func isGeneratedPath(path string) bool {
	base := filepath.Base(path)
	for _, p := range generatedPatterns {
		if strings.Contains(p, "/") {
			if ok, _ := filepath.Match(p, path); ok {
				return true
			}
			continue
		}
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
	}
	return false
}

func stop(reason, message string) Gate {
	return Gate{Decision: "stop", StopReason: &reason, StopMessage: &message}
}

type prAPI struct {
	State  string `json:"state"`
	Merged bool   `json:"merged"`
	Draft  bool   `json:"draft"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	User   struct {
		Login string `json:"login"`
	} `json:"user"`
	Base struct {
		SHA string `json:"sha"`
	} `json:"base"`
	Head struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

// RunPreflight executes the full preflight and writes --out (and the diff
// file when reached). It returns the result for the one-line summary; error
// means infrastructure failure (exit 1).
func RunPreflight(ctx context.Context, r Runner, opts PreflightOptions) (*PreflightResult, error) {
	owner, repo, num, err := parsePRURL(opts.URL)
	if err != nil {
		return nil, err
	}
	res := &PreflightResult{
		Schema:             schemaPreflight,
		Gate:               Gate{Decision: "proceed"},
		ReviewMode:         "full",
		ExistingComments:   []ExistingComment{},
		Graph:              json.RawMessage(`{"mode":"fallback","reason":"not run"}`),
		DependencyManifest: Manifest{Go: []Artifact{}, NPM: []Artifact{}, Python: []Artifact{}, Rust: []Artifact{}},
		PR:                 PRInfo{Owner: owner, Repo: repo, Number: num, Files: []PRFile{}},
	}
	prPath := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, num)

	out, err := r.Run(ctx, "api", prPath)
	if err != nil {
		return nil, fmt.Errorf("fetching PR: %w", err)
	}
	var pr prAPI
	if err := json.Unmarshal(out, &pr); err != nil {
		return nil, fmt.Errorf("decoding PR: %w", err)
	}
	res.PR.Title, res.PR.Body, res.PR.Author = pr.Title, pr.Body, pr.User.Login
	res.PR.BaseSHA, res.PR.HeadSHA = pr.Base.SHA, pr.Head.SHA

	// Gate checks, in eligibility.md's order.
	switch {
	case pr.Merged:
		res.Gate = stop("merged", "PR is MERGED; nothing to review.")
	case pr.State == "closed":
		res.Gate = stop("closed", "PR is CLOSED; nothing to review.")
	case pr.Draft:
		res.Gate = stop("draft", "PR is a draft. Want me to review it anyway?")
	case automationAuthors[pr.User.Login]:
		res.Gate = stop("automation_only", "Looks like an automated PR ("+pr.User.Login+"). Quick sanity check only, or full review?")
	}
	if res.Gate.Decision == "stop" {
		return res, writeResult(res, opts.OutPath)
	}

	files, err := fetchFiles(ctx, r, prPath)
	if err != nil {
		return nil, err
	}
	res.PR.Files = files
	switch {
	case len(files) == 0:
		res.Gate = stop("empty_diff", "Empty diff.")
	case allGenerated(files):
		res.Gate = stop("generated_only", "Diff is generated files only; no behavior to review.")
	}
	if res.Gate.Decision == "stop" {
		return res, writeResult(res, opts.OutPath)
	}

	// Incremental detection via REST (GraphQL lacks commit_id).
	lastSHA, err := lastDevpilotReviewSHA(ctx, r, prPath)
	if err != nil {
		return nil, err
	}
	diffFrom := ""
	if lastSHA != "" {
		if lastSHA == pr.Head.SHA && !opts.Force {
			res.Gate = stop("already_reviewed_at_head", "I already reviewed this exact commit. Want me to re-run anyway?")
			res.LastReviewedSHA = &lastSHA
			return res, writeResult(res, opts.OutPath)
		}
		if lastSHA != pr.Head.SHA {
			res.ReviewMode = "incremental"
			res.LastReviewedSHA = &lastSHA
			diffFrom = lastSHA
		}
	}

	diff, err := fetchDiff(ctx, r, owner, repo, num, diffFrom, pr.Head.SHA)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(diff) == "" {
		res.Gate = stop("empty_diff", "Empty diff.")
		return res, writeResult(res, opts.OutPath)
	}
	diffOut := opts.DiffOut
	if diffOut == "" {
		diffOut = filepath.Join(filepath.Dir(opts.OutPath), fmt.Sprintf("pr_%d_diff.patch", num))
	}
	if err := os.WriteFile(diffOut, []byte(diff), 0o644); err != nil {
		return nil, fmt.Errorf("writing diff: %w", err)
	}
	res.DiffPath, err = filepath.Abs(diffOut)
	if err != nil {
		res.DiffPath = diffOut
	}

	comments, err := fetchExistingComments(ctx, r, prPath)
	if err != nil {
		return nil, err
	}
	res.ExistingComments = comments

	res.Graph = runGraphPreflight(ctx, opts.GraphPreflight, pr.Base.SHA, pr.Head.SHA)

	parsed := parseDiff(diff)
	res.DependencyManifest = extractManifest(parsed, fetchBaseManifests(ctx, r, owner, repo, pr.Base.SHA, parsed))

	return res, writeResult(res, opts.OutPath)
}

func fetchFiles(ctx context.Context, r Runner, prPath string) ([]PRFile, error) {
	out, err := r.Run(ctx, "api", "--paginate", prPath+"/files")
	if err != nil {
		return nil, fmt.Errorf("fetching PR files: %w", err)
	}
	var raw []struct {
		Filename  string `json:"filename"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
	}
	if err := json.Unmarshal(normalizePaginated(out), &raw); err != nil {
		return nil, fmt.Errorf("decoding PR files: %w", err)
	}
	files := make([]PRFile, 0, len(raw))
	for _, f := range raw {
		files = append(files, PRFile{Path: f.Filename, Additions: f.Additions, Deletions: f.Deletions})
	}
	return files, nil
}

// normalizePaginated joins `gh api --paginate` output, which concatenates
// one JSON array per page, into a single array.
func normalizePaginated(out []byte) []byte {
	s := strings.TrimSpace(string(out))
	if s == "" {
		return []byte("[]")
	}
	s = strings.ReplaceAll(s, "][", ",")
	s = strings.ReplaceAll(s, "]\n[", ",")
	return []byte(s)
}

func allGenerated(files []PRFile) bool {
	for _, f := range files {
		if !isGeneratedPath(f.Path) {
			return false
		}
	}
	return true
}

func lastDevpilotReviewSHA(ctx context.Context, r Runner, prPath string) (string, error) {
	out, err := r.Run(ctx, "api", "--paginate", prPath+"/reviews")
	if err != nil {
		return "", fmt.Errorf("fetching PR reviews: %w", err)
	}
	var reviews []struct {
		Body        string `json:"body"`
		CommitID    string `json:"commit_id"`
		SubmittedAt string `json:"submitted_at"`
	}
	if err := json.Unmarshal(normalizePaginated(out), &reviews); err != nil {
		return "", fmt.Errorf("decoding PR reviews: %w", err)
	}
	marked := reviews[:0]
	for _, rv := range reviews {
		if strings.Contains(rv.Body, reviewMarker) {
			marked = append(marked, rv)
		}
	}
	if len(marked) == 0 {
		return "", nil
	}
	sort.Slice(marked, func(i, j int) bool { return marked[i].SubmittedAt > marked[j].SubmittedAt })
	return marked[0].CommitID, nil
}

func fetchExistingComments(ctx context.Context, r Runner, prPath string) ([]ExistingComment, error) {
	out, err := r.Run(ctx, "api", "--paginate", prPath+"/comments")
	if err != nil {
		return nil, fmt.Errorf("fetching PR comments: %w", err)
	}
	var raw []struct {
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Side     string `json:"side"`
		Body     string `json:"body"`
		CommitID string `json:"commit_id"`
		User     struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := json.Unmarshal(normalizePaginated(out), &raw); err != nil {
		return nil, fmt.Errorf("decoding PR comments: %w", err)
	}
	comments := make([]ExistingComment, 0, len(raw))
	for _, c := range raw {
		comments = append(comments, ExistingComment{
			Path: c.Path, Line: c.Line, Side: c.Side,
			Body: c.Body, User: c.User.Login, CommitID: c.CommitID,
		})
	}
	return comments, nil
}

func fetchDiff(ctx context.Context, r Runner, owner, repo string, num int, from, head string) (string, error) {
	accept := "Accept: application/vnd.github.diff"
	var out []byte
	var err error
	if from == "" {
		out, err = r.Run(ctx, "api", "-H", accept, fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, num))
	} else {
		out, err = r.Run(ctx, "api", "-H", accept, fmt.Sprintf("repos/%s/%s/compare/%s...%s", owner, repo, from, head))
	}
	if err != nil {
		return "", fmt.Errorf("fetching diff: %w", err)
	}
	return string(out), nil
}

func runGraphPreflight(ctx context.Context, fn func(context.Context, string, string) ([]byte, error), base, head string) json.RawMessage {
	if fn == nil {
		fn = execGraphPreflight
	}
	fallback := func(reason string) json.RawMessage {
		b, _ := json.Marshal(map[string]string{"mode": "fallback", "reason": reason})
		return b
	}
	gctx, cancel := context.WithTimeout(ctx, graphTimeout)
	defer cancel()
	out, err := fn(gctx, base, head)
	if err != nil {
		return fallback(err.Error())
	}
	if !json.Valid(out) {
		return fallback("graph preflight emitted invalid JSON")
	}
	return json.RawMessage(out)
}

func fetchBaseManifests(ctx context.Context, r Runner, owner, repo, baseSHA string, files []FileDiff) baseManifests {
	var b baseManifests
	need := map[string]*string{}
	for _, f := range files {
		name := f.Path[strings.LastIndex(f.Path, "/")+1:]
		if strings.HasSuffix(name, ".go") {
			need["go.mod"] = &b.goMod
		}
		if name == "package.json" || isJSSource(name) {
			need["package.json"] = &b.packageJSON
		}
	}
	for path, dst := range need {
		out, err := r.Run(ctx, "api", "-H", "Accept: application/vnd.github.raw",
			fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, path, baseSHA))
		if err == nil {
			*dst = string(out)
		}
	}
	return b
}

func writeResult(res *PreflightResult, outPath string) error {
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding preflight result: %w", err)
	}
	if err := os.WriteFile(outPath, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing preflight result: %w", err)
	}
	return nil
}
