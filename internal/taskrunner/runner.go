package taskrunner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/siyuqian/developer-kit/internal/trello"
)

type Config struct {
	BoardName     string
	Interval      time.Duration
	Timeout       time.Duration
	ReviewTimeout time.Duration // 0 disables code review
	Once          bool
	DryRun        bool
	WorkDir       string
}

type Runner struct {
	config   Config
	trello   *trello.Client
	executor *Executor
	reviewer *Reviewer
	git      *GitOps
	logger   *log.Logger

	// Resolved IDs
	boardID      string
	readyListID  string
	inProgListID string
	doneListID   string
	failedListID string
}

func New(cfg Config, trelloClient *trello.Client) *Runner {
	r := &Runner{
		config:   cfg,
		trello:   trelloClient,
		executor: NewExecutor(),
		git:      NewGitOps(cfg.WorkDir),
		logger:   log.New(os.Stdout, "", log.LstdFlags),
	}
	if cfg.ReviewTimeout > 0 {
		r.reviewer = NewReviewer()
	}
	return r
}

func (r *Runner) init() error {
	r.logger.Printf("Resolving board: %s", r.config.BoardName)
	board, err := r.trello.FindBoardByName(r.config.BoardName)
	if err != nil {
		return fmt.Errorf("find board: %w", err)
	}
	r.boardID = board.ID
	r.logger.Printf("Board found: %s (%s)", board.Name, board.ID)

	listNames := map[string]*string{
		"Ready":       &r.readyListID,
		"In Progress": &r.inProgListID,
		"Done":        &r.doneListID,
		"Failed":      &r.failedListID,
	}
	for name, idPtr := range listNames {
		list, err := r.trello.FindListByName(r.boardID, name)
		if err != nil {
			return fmt.Errorf("find list %q: %w", name, err)
		}
		*idPtr = list.ID
		r.logger.Printf("List %q → %s", name, list.ID)
	}
	return nil
}

func (r *Runner) Run(ctx context.Context) error {
	if err := r.init(); err != nil {
		return err
	}

	// Pre-flight: ensure working directory is clean
	clean, err := r.git.IsClean()
	if err != nil {
		return fmt.Errorf("check working directory: %w", err)
	}
	if !clean {
		return fmt.Errorf("working directory has uncommitted changes; commit or stash them before running")
	}

	r.logger.Println("Runner started. Polling for tasks...")

	for {
		select {
		case <-ctx.Done():
			r.logger.Println("Shutting down.")
			return nil
		default:
		}

		cards, err := r.trello.GetListCards(r.readyListID)
		if err != nil {
			r.logger.Printf("Error polling: %v. Retrying in %s...", err, r.config.Interval)
			if !r.sleep(ctx, r.config.Interval) {
				r.logger.Println("Shutting down.")
				return nil
			}
			continue
		}

		if len(cards) == 0 {
			r.logger.Printf("No tasks. Sleeping %s...", r.config.Interval)
			if !r.sleep(ctx, r.config.Interval) {
				r.logger.Println("Shutting down.")
				return nil
			}
			continue
		}

		card := cards[0]
		r.processCard(ctx, card)

		if r.config.Once {
			r.logger.Println("--once flag set. Exiting.")
			return nil
		}
	}
}

func (r *Runner) processCard(ctx context.Context, card trello.Card) {
	start := time.Now()
	r.logger.Printf("Processing card: %q (%s)", card.Name, card.ID)

	if card.Desc == "" {
		r.logger.Printf("Card has empty description, marking as failed")
		r.trello.MoveCard(card.ID, r.failedListID)
		r.trello.AddComment(card.ID, "❌ Task failed\nError: Empty plan — card description is empty")
		return
	}

	if r.config.DryRun {
		r.logger.Printf("[DRY RUN] Would process card: %q", card.Name)
		return
	}

	// Move to In Progress
	if err := r.trello.MoveCard(card.ID, r.inProgListID); err != nil {
		r.logger.Printf("Failed to move card to In Progress: %v", err)
	}

	// Git: checkout main, pull, create branch
	branch := r.git.BranchName(card.ID, card.Name)
	if err := r.git.CheckoutMain(); err != nil {
		r.failCard(card, start, fmt.Sprintf("git checkout main: %v", err))
		return
	}
	r.git.Pull() // best-effort
	if err := r.git.CreateBranch(branch); err != nil {
		r.failCard(card, start, fmt.Sprintf("git create branch: %v", err))
		return
	}

	// Build prompt
	prompt := r.buildPrompt(card)

	// Execute
	taskCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	result, err := r.executor.Run(taskCtx, prompt)

	// Save log
	r.saveLog(card.ID, result)

	if err != nil || result.ExitCode != 0 {
		errMsg := "non-zero exit code"
		if result.TimedOut {
			errMsg = "execution timed out"
		} else if result.Stderr != "" {
			errMsg = truncate(result.Stderr, 500)
		}
		r.failCard(card, start, errMsg)
		r.git.CheckoutMain()
		return
	}

	// Verify claude produced commits before pushing
	hasCommits, err := r.git.HasNewCommits(branch)
	if err != nil {
		r.failCard(card, start, fmt.Sprintf("check commits: %v", err))
		r.git.CheckoutMain()
		return
	}
	if !hasCommits {
		r.failCard(card, start, "claude produced no commits on task branch")
		r.git.CheckoutMain()
		return
	}

	// Push and create PR
	if err := r.git.Push(branch); err != nil {
		r.failCard(card, start, fmt.Sprintf("git push: %v", err))
		r.git.CheckoutMain()
		return
	}

	cardURL := fmt.Sprintf("https://trello.com/c/%s", card.ID)
	prBody := fmt.Sprintf("## Task\n%s\n\n🤖 Executed by devkit runner", cardURL)
	prURL, err := r.git.CreatePR(card.Name, prBody)
	if err != nil {
		r.failCard(card, start, fmt.Sprintf("create PR: %v", err))
		r.git.CheckoutMain()
		return
	}

	// Code review (non-blocking)
	if r.reviewer != nil {
		r.logger.Printf("Running code review for PR: %s", prURL)
		reviewCtx, reviewCancel := context.WithTimeout(ctx, r.config.ReviewTimeout)
		reviewResult, reviewErr := r.reviewer.Review(reviewCtx, prURL)
		reviewCancel()
		if reviewErr != nil {
			r.logger.Printf("Code review error: %v", reviewErr)
		} else if reviewResult.ExitCode != 0 {
			r.logger.Printf("Code review finished with non-zero exit: %d", reviewResult.ExitCode)
		} else {
			r.logger.Printf("Code review completed for PR: %s", prURL)
		}
	}

	if err := r.git.MergePR(); err != nil {
		r.logger.Printf("Auto-merge failed (may need approval): %v", err)
	}

	// Move to Done
	duration := time.Since(start).Round(time.Second)
	r.trello.MoveCard(card.ID, r.doneListID)
	r.trello.AddComment(card.ID, fmt.Sprintf("✅ Task completed by devkit runner\nDuration: %s\nPR: %s", duration, prURL))
	r.logger.Printf("Card %q completed in %s. PR: %s", card.Name, duration, prURL)

	r.git.CheckoutMain()
	r.git.Pull()
}

func (r *Runner) buildPrompt(card trello.Card) string {
	return fmt.Sprintf(`Execute the following task plan. Use /superpowers:test-driven-development and /superpowers:verification-before-completion skills during execution.

Task: %s

Plan:
%s

When done:
- Commit all changes with a descriptive message
- Push to the appropriate branch`, card.Name, card.Desc)
}

func (r *Runner) failCard(card trello.Card, start time.Time, errMsg string) {
	duration := time.Since(start).Round(time.Second)
	logPath := filepath.Join(os.Getenv("HOME"), ".config", "devkit", "logs", card.ID+".log")
	comment := fmt.Sprintf("❌ Task failed\nDuration: %s\nError: %s\nSee full log: %s", duration, errMsg, logPath)
	r.trello.MoveCard(card.ID, r.failedListID)
	r.trello.AddComment(card.ID, comment)
	r.logger.Printf("Card %q failed: %s", card.Name, errMsg)
}

func (r *Runner) saveLog(cardID string, result *ExecuteResult) {
	if result == nil {
		return
	}
	logDir := filepath.Join(os.Getenv("HOME"), ".config", "devkit", "logs")
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, cardID+".log")
	content := fmt.Sprintf("=== STDOUT ===\n%s\n\n=== STDERR ===\n%s\n", result.Stdout, result.Stderr)
	os.WriteFile(logPath, []byte(content), 0644)
}

func (r *Runner) sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
