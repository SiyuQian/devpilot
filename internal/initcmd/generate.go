package initcmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/siyuqian/devpilot/internal/project"
)

// Board is a simple struct for board listing (avoids importing trello package).
type Board struct {
	Name string
}

// GenerateOpts configures generator behavior.
type GenerateOpts struct {
	Dir         string
	Interactive bool
	Reader      *bufio.Reader
}

// ProjectType holds detected project language/framework info.
type ProjectType struct {
	Name     string
	BuildCmd string
	TestCmd  string
}

func detectProjectType(dir string) ProjectType {
	// Go
	if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		name := parseGoModuleName(data)
		return ProjectType{
			Name:     name,
			BuildCmd: "go build ./...",
			TestCmd:  "go test ./...",
		}
	}

	// Node
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		name := parsePackageJSONName(data)
		return ProjectType{
			Name:     name,
			BuildCmd: "npm run build",
			TestCmd:  "npm test",
		}
	}

	// Python (pyproject.toml)
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		return ProjectType{
			Name:     filepath.Base(dir),
			BuildCmd: "python -m build",
			TestCmd:  "python -m pytest",
		}
	}

	// Python (requirements.txt)
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		return ProjectType{
			Name:     filepath.Base(dir),
			BuildCmd: "",
			TestCmd:  "python -m pytest",
		}
	}

	// Fallback
	return ProjectType{
		Name: filepath.Base(dir),
	}
}

func parseGoModuleName(data []byte) string {
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if mod, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(mod)
		}
	}
	return ""
}

func parsePackageJSONName(data []byte) string {
	var pkg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &pkg); err == nil {
		return pkg.Name
	}
	return ""
}

// GenerateClaudeMD creates a CLAUDE.md file from the detected project type.
func GenerateClaudeMD(opts GenerateOpts) error {
	pt := detectProjectType(opts.Dir)

	tmpl, err := template.New("claude").Parse(claudeMDTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"ProjectName": pt.Name,
		"BuildCmd":    pt.BuildCmd,
		"TestCmd":     pt.TestCmd,
	}); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(opts.Dir, "CLAUDE.md"), buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Println("  Created CLAUDE.md")
	return nil
}

// ConfigureBoard sets up the board name in .devpilot.json.
func ConfigureBoard(opts GenerateOpts, listBoards func() ([]Board, error)) error {
	if !opts.Interactive {
		fmt.Println("  Skipped: board configuration (use devpilot init without --yes to configure)")
		return nil
	}

	var boardName string

	if listBoards != nil {
		boards, err := listBoards()
		if err != nil {
			return fmt.Errorf("listing boards: %w", err)
		}

		fmt.Println("  Available boards:")
		for i, b := range boards {
			fmt.Printf("    %d) %s\n", i+1, b.Name)
		}
		fmt.Print("  Select board number: ")

		line, err := opts.Reader.ReadString('\n')
		if err != nil {
			return err
		}

		idx, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || idx < 1 || idx > len(boards) {
			return fmt.Errorf("invalid selection: %s", strings.TrimSpace(line))
		}
		boardName = boards[idx-1].Name
	} else {
		fmt.Print("  Enter board name: ")
		line, err := opts.Reader.ReadString('\n')
		if err != nil {
			return err
		}
		boardName = strings.TrimSpace(line)
	}

	if boardName == "" {
		return nil
	}

	if err := project.Save(opts.Dir, &project.Config{Board: boardName}); err != nil {
		return err
	}

	fmt.Printf("  Configured board: %s\n", boardName)
	return nil
}

// EnsureGitignore ensures that the given entries exist in .gitignore.
func EnsureGitignore(dir string, entries []string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	existing := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	}

	var toAdd []string
	for _, entry := range entries {
		if !strings.Contains(existing, entry) {
			toAdd = append(toAdd, entry)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Ensure we start on a new line
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	block := "\n# DevPilot\n"
	for _, entry := range toAdd {
		block += entry + "\n"
	}
	if _, err := f.WriteString(block); err != nil {
		return err
	}

	fmt.Printf("  Updated .gitignore: added %s\n", strings.Join(toAdd, ", "))
	return nil
}

// CreateSkill creates an initial skill directory with a SKILL.md file.
func CreateSkill(opts GenerateOpts) error {
	name := "my-skill"

	if opts.Interactive {
		fmt.Printf("  Skill name [my-skill]: ")
		line, err := opts.Reader.ReadString('\n')
		if err != nil {
			return err
		}
		input := strings.TrimSpace(line)
		if input != "" {
			name = input
		}
	}

	skillDir := filepath.Join(opts.Dir, ".claude", "skills", name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	tmpl, err := template.New("skill").Parse(skillMDTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"SkillName": name,
	}); err != nil {
		return err
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Printf("  Created .claude/skills/%s/SKILL.md\n", name)
	return nil
}
