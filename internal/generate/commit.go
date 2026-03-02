package generate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

var commitTmpl = template.Must(template.ParseFS(promptsFS, "prompts/commit.tmpl"))

type commitData struct {
	FileList string
	DiffStat string
	Context  string
}

func buildCommitPrompt(fileList, diffStat, userContext string) (string, error) {
	var buf bytes.Buffer
	err := commitTmpl.Execute(&buf, commitData{
		FileList: fileList,
		DiffStat: diffStat,
		Context:  userContext,
	})
	return buf.String(), err
}

// gitOutput runs a git command and returns trimmed stdout.
func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RunCommit stages all changes, generates a commit message via Claude, and commits.
func RunCommit(ctx context.Context, model, userContext string, dryRun bool) error {
	// Stage all changes
	if out, err := exec.Command("git", "add", ".").CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}

	// Check for staged changes
	diffCached, err := gitOutput("diff", "--cached", "--stat")
	if err != nil {
		return err
	}
	if diffCached == "" {
		fmt.Println("No changes to commit.")
		return nil
	}

	fileList, err := gitOutput("diff", "--cached", "--name-only")
	if err != nil {
		return err
	}

	prompt, err := buildCommitPrompt(fileList, diffCached, userContext)
	if err != nil {
		return fmt.Errorf("build prompt: %w", err)
	}

	fmt.Println("Generating commit message...")
	message, err := Generate(ctx, prompt, model)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s\n\n", message)

	if dryRun {
		fmt.Println("(dry-run: not committing)")
		return nil
	}

	// Prompt user for confirmation
	fmt.Print("Commit with this message? [y/n/e(dit)] ")
	var choice string
	fmt.Scanln(&choice)
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "y", "yes", "":
		// proceed
	case "e", "edit":
		edited, err := editInTerminal(message)
		if err != nil {
			return err
		}
		message = edited
	default:
		fmt.Println("Aborted.")
		return nil
	}

	if out, err := exec.Command("git", "commit", "-m", message).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	fmt.Println("Committed.")
	return nil
}

// editInTerminal opens $EDITOR for the user to edit the message.
func editInTerminal(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	tmpFile, err := os.CreateTemp("", "devpilot-commit-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(initial); err != nil {
		return "", err
	}
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor: %w", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
