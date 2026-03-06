package taskrunner

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	BoardName     string
	Interval      time.Duration
	Timeout       time.Duration
	ReviewTimeout time.Duration // 0 disables code review
	Once          bool
	DryRun        bool
	WorkDir       string
	UseOpenSpec   bool
}

type Runner struct {
	config       Config
	source       TaskSource
	executor     *Executor
	reviewer     *Reviewer
	git          *GitOps
	logger       *log.Logger
	eventHandler EventHandler
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithEventHandler sets an event handler that receives runner lifecycle events.
func WithEventHandler(handler EventHandler) RunnerOption {
	return func(r *Runner) {
		r.eventHandler = handler
	}
}

func New(cfg Config, source TaskSource, opts ...RunnerOption) *Runner {
	r := &Runner{
		config: cfg,
		source: source,
		git:    NewGitOps(cfg.WorkDir),
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
	for _, opt := range opts {
		opt(r)
	}

	// When event handler is set, silence the logger to avoid duplicate output.
	// All information is conveyed through events instead.
	if r.eventHandler != nil {
		r.logger = log.New(io.Discard, "", 0)
	}

	// When event handler is set, enable streaming output on executor
	var execOpts []ExecutorOption
	if r.eventHandler != nil {
		bridge := newEventBridge(r.eventHandler)
		execOpts = append(execOpts, WithClaudeEventHandler(bridge.Handle))
	}
	r.executor = NewExecutor(execOpts...)

	if cfg.ReviewTimeout > 0 {
		r.reviewer = NewReviewer()
	}
	return r
}

func (r *Runner) emit(e Event) {
	if r.eventHandler != nil {
		r.eventHandler(e)
	}
}

func (r *Runner) init() error {
	r.logger.Printf("Initializing task source...")
	info, err := r.source.Init()
	if err != nil {
		return err
	}
	r.logger.Printf("Connected: %s", info.DisplayName)
	r.emit(RunnerStartedEvent{BoardName: info.DisplayName, BoardID: info.BoardID, Lists: info.Lists})
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
			r.emit(RunnerStoppedEvent{})
			return nil
		default:
		}

		r.emit(PollingEvent{})
		tasks, err := r.source.FetchReady()
		if err != nil {
			r.logger.Printf("Error polling: %v. Retrying in %s...", err, r.config.Interval)
			r.emit(RunnerErrorEvent{Err: err})
			if !r.sleep(ctx, r.config.Interval) {
				r.logger.Println("Shutting down.")
				r.emit(RunnerStoppedEvent{})
				return nil
			}
			continue
		}

		if len(tasks) == 0 {
			r.logger.Printf("No tasks. Sleeping %s...", r.config.Interval)
			r.emit(NoTasksEvent{NextPoll: r.config.Interval})
			if !r.sleep(ctx, r.config.Interval) {
				r.logger.Println("Shutting down.")
				r.emit(RunnerStoppedEvent{})
				return nil
			}
			continue
		}

		SortByPriority(tasks)
		task := tasks[0]
		r.processCard(ctx, task)

		if r.config.Once {
			r.logger.Println("--once flag set. Exiting.")
			r.emit(RunnerStoppedEvent{})
			return nil
		}
	}
}

func (r *Runner) processCard(ctx context.Context, task Task) {
	start := time.Now()
	r.logger.Printf("Processing card: %q (%s)", task.Name, task.ID)

	if task.Description == "" {
		r.logger.Printf("Card has empty description, marking as failed")
		r.source.MarkFailed(task.ID, "❌ Task failed\nError: Empty plan — card description is empty")
		return
	}

	if r.config.DryRun {
		r.logger.Printf("[DRY RUN] Would process card: %q", task.Name)
		return
	}

	// Move to In Progress
	if err := r.source.MarkInProgress(task.ID); err != nil {
		r.logger.Printf("Failed to move card to In Progress: %v", err)
	}

	// Git: checkout main, pull, create branch
	branch := r.git.BranchName(task.ID, task.Name)
	if err := r.git.CheckoutMain(); err != nil {
		r.failCard(task, start, fmt.Sprintf("git checkout main: %v", err))
		return
	}
	r.git.Pull() // best-effort
	if err := r.git.CreateBranch(branch); err != nil {
		r.failCard(task, start, fmt.Sprintf("git create branch: %v", err))
		return
	}
	r.emit(CardStartedEvent{CardID: task.ID, CardName: task.Name, Branch: branch})

	// Build prompt
	prompt := r.buildPrompt(task)

	// Execute
	taskCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	result, err := r.executor.Run(taskCtx, prompt)

	// Save log
	r.saveLog(task.ID, result)

	if err != nil || result.ExitCode != 0 {
		errMsg := "non-zero exit code"
		if result.TimedOut {
			errMsg = "execution timed out"
		} else if result.Stderr != "" {
			errMsg = truncate(result.Stderr, 500)
		}
		r.failCard(task, start, errMsg)
		r.git.CheckoutMain()
		return
	}

	// Verify claude produced commits before pushing
	hasCommits, err := r.git.HasNewCommits(branch)
	if err != nil {
		r.failCard(task, start, fmt.Sprintf("check commits: %v", err))
		r.git.CheckoutMain()
		return
	}
	if !hasCommits {
		r.failCard(task, start, "claude produced no commits on task branch")
		r.git.CheckoutMain()
		return
	}

	// Push and create PR
	if err := r.git.Push(branch); err != nil {
		r.failCard(task, start, fmt.Sprintf("git push: %v", err))
		r.git.CheckoutMain()
		return
	}

	prBody := fmt.Sprintf("## Task\n%s\n\n🤖 Executed by devpilot runner", task.URL)
	prURL, err := r.git.CreatePR(task.Name, prBody)
	if err != nil {
		r.failCard(task, start, fmt.Sprintf("create PR: %v", err))
		r.git.CheckoutMain()
		return
	}

	// Code review gate (blocking with self-heal loop)
	if r.reviewer != nil {
		approved := false
		for attempt := 0; attempt <= MaxReviewRetries; attempt++ {
			r.logger.Printf("Running code review for PR: %s (attempt %d)", prURL, attempt+1)
			r.emit(ReviewStartedEvent{PRURL: prURL})
			reviewCtx, reviewCancel := context.WithTimeout(ctx, r.config.ReviewTimeout)
			reviewResult, reviewErr := r.reviewer.Review(reviewCtx, prURL)
			reviewCancel()

			if reviewErr != nil {
				r.logger.Printf("Code review error: %v", reviewErr)
				r.emit(ReviewDoneEvent{PRURL: prURL, ExitCode: -1})
				break
			}

			r.emit(ReviewDoneEvent{PRURL: prURL, ExitCode: reviewResult.ExitCode})

			if IsApproved(reviewResult.Stdout) {
				r.logger.Printf("Code review approved for PR: %s", prURL)
				approved = true
				break
			}

			// Review found issues — attempt fix if retries remain
			if attempt < MaxReviewRetries {
				r.logger.Printf("Review found issues, attempting fix (attempt %d/%d)", attempt+1, MaxReviewRetries)
				r.emit(FixStartedEvent{PRURL: prURL, Attempt: attempt + 1})
				fixCtx, fixCancel := context.WithTimeout(ctx, r.config.ReviewTimeout)
				fixResult, fixErr := r.reviewer.Fix(fixCtx, prURL)
				fixCancel()

				fixExitCode := -1
				if fixErr == nil {
					fixExitCode = fixResult.ExitCode
				}
				r.emit(FixDoneEvent{PRURL: prURL, Attempt: attempt + 1, ExitCode: fixExitCode})

				if fixErr != nil {
					r.logger.Printf("Fix attempt failed: %v", fixErr)
					continue
				}

				// Push the fix
				if err := r.git.Push(branch); err != nil {
					r.logger.Printf("Failed to push fix: %v", err)
					r.failCard(task, start, fmt.Sprintf("push fix: %v", err))
					r.git.CheckoutMain()
					return
				}
			}
		}

		if !approved {
			r.failCard(task, start, fmt.Sprintf("code review failed after %d attempts", MaxReviewRetries+1))
			r.git.CheckoutMain()
			return
		}
	}

	if err := r.git.MergePR(); err != nil {
		r.logger.Printf("Auto-merge failed (may need approval): %v", err)
	}

	// Move to Done
	duration := time.Since(start).Round(time.Second)
	r.emit(CardDoneEvent{CardID: task.ID, CardName: task.Name, PRURL: prURL, Duration: duration})
	comment := fmt.Sprintf("✅ Task completed by devpilot runner\nDuration: %s\nPR: %s", duration, prURL)
	r.source.MarkDone(task.ID, comment)
	r.logger.Printf("Card %q completed in %s. PR: %s", task.Name, duration, prURL)

	r.git.CheckoutMain()
	r.git.Pull()
}

func (r *Runner) buildPrompt(task Task) string {
	if r.config.UseOpenSpec {
		return fmt.Sprintf(`Execute the following OpenSpec change autonomously from start to finish. This runs unattended — never stop to ask for feedback, confirmation, or approval.

Run: /opsx:apply %s

Rules:
- Execute ALL tasks without stopping
- Commit after each logical unit of work
- Never ask for user input or feedback
- If a task is blocked, skip it and continue with the next task
- When ALL tasks are complete, push to the current branch`, task.Name)
	}
	return fmt.Sprintf(`Execute the following task plan autonomously from start to finish. This runs unattended — never stop to ask for feedback, confirmation, or approval. Execute ALL steps/batches continuously without pausing.

Use /superpowers:test-driven-development and /superpowers:verification-before-completion skills during execution.

Task: %s

Plan:
%s

Rules:
- Execute ALL steps in the plan without stopping. Do NOT pause between batches or steps for review.
- Commit after each logical unit of work
- Never ask for user input or feedback
- If a step is blocked, skip it and continue with the next step
- When ALL steps are complete, push to the current branch`, task.Name, task.Description)
}

func (r *Runner) failCard(task Task, start time.Time, errMsg string) {
	duration := time.Since(start).Round(time.Second)
	r.emit(CardFailedEvent{CardID: task.ID, CardName: task.Name, ErrMsg: errMsg, Duration: duration})
	logPath := filepath.Join(r.config.WorkDir, ".devpilot", "logs", task.ID+".log")
	comment := fmt.Sprintf("❌ Task failed\nDuration: %s\nError: %s\nSee full log: %s", duration, errMsg, logPath)
	r.source.MarkFailed(task.ID, comment)
	r.logger.Printf("Card %q failed: %s", task.Name, errMsg)
}

func (r *Runner) saveLog(cardID string, result *ExecuteResult) {
	if result == nil {
		return
	}
	logDir := filepath.Join(r.config.WorkDir, ".devpilot", "logs")
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

