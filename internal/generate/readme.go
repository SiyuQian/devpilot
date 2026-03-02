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
	FileTree       string
	PackageInfo    string
	ExistingReadme string
}

func buildReadmePrompt(fileTree, packageInfo, existingReadme string) (string, error) {
	var buf bytes.Buffer
	err := readmeTmpl.Execute(&buf, readmeData{
		FileTree:       fileTree,
		PackageInfo:    packageInfo,
		ExistingReadme: existingReadme,
	})
	return buf.String(), err
}

func collectFileTree(dir string) (string, error) {
	var lines []string
	excludes := map[string]bool{
		".git": true, "node_modules": true, "vendor": true,
		"bin": true, ".claude": true, "__pycache__": true,
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		if d.IsDir() && excludes[d.Name()] {
			return filepath.SkipDir
		}
		lines = append(lines, rel)
		if len(lines) >= 200 {
			return filepath.SkipAll
		}
		return nil
	})
	return strings.Join(lines, "\n"), err
}

func collectPackageInfo(dir string) string {
	candidates := []string{"go.mod", "package.json", "pyproject.toml", "Cargo.toml"}
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		lines := strings.SplitN(string(data), "\n", 31)
		if len(lines) > 30 {
			lines = lines[:30]
		}
		return fmt.Sprintf("(%s)\n%s", name, strings.Join(lines, "\n"))
	}
	return ""
}

func RunReadme(ctx context.Context, model string, dryRun bool) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	fileTree, err := collectFileTree(dir)
	if err != nil {
		return err
	}

	packageInfo := collectPackageInfo(dir)

	var existingReadme string
	if data, err := os.ReadFile(filepath.Join(dir, "README.md")); err == nil {
		existingReadme = string(data)
	}

	prompt, err := buildReadmePrompt(fileTree, packageInfo, existingReadme)
	if err != nil {
		return fmt.Errorf("build prompt: %w", err)
	}

	fmt.Println("Generating README...")
	content, err := Generate(ctx, prompt, model)
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
	fmt.Scanln(&choice)
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
