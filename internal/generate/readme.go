package generate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var readmeTmpl = template.Must(template.ParseFS(promptsFS, "prompts/readme.tmpl"))

type readmeData struct {
	ExistingReadme string
}

func buildReadmePrompt(existingReadme string) (string, error) {
	var buf bytes.Buffer
	err := readmeTmpl.Execute(&buf, readmeData{
		ExistingReadme: existingReadme,
	})
	return buf.String(), err
}

// RunReadme generates a README by letting Claude explore the project with tools.
func RunReadme(ctx context.Context, model string, dryRun bool) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	var existingReadme string
	if data, err := os.ReadFile(filepath.Join(dir, "README.md")); err == nil {
		existingReadme = string(data)
	}

	prompt, err := buildReadmePrompt(existingReadme)
	if err != nil {
		return fmt.Errorf("build prompt: %w", err)
	}

	fmt.Println("Generating README (exploring project)...")
	content, err := GenerateWithTools(ctx, prompt, model)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s\n\n", content)

	if dryRun {
		fmt.Println("(dry-run: not writing)")
		return nil
	}

	fmt.Print("Save to README.md? [y/n] ")
	var choice string
	_, _ = fmt.Scanln(&choice)
	if strings.ToLower(strings.TrimSpace(choice)) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(content+"\n"), 0644); err != nil {
		return err
	}
	fmt.Println("Saved to README.md")
	return nil
}
