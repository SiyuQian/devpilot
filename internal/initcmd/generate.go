package initcmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/internal/project"
	"github.com/siyuqian/devpilot/internal/skillmgr"
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

// ConfigureBoard sets up the board name in .devpilot.yaml.
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

	cfg, err := project.Load(opts.Dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	cfg.Board = boardName
	if err := project.Save(opts.Dir, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("  Configured board: %s\n", boardName)
	return nil
}

// ghLabel describes a GitHub label to create for DevPilot.
type ghLabel struct {
	name  string
	color string
	desc  string
}

// ghRequiredLabels are the labels DevPilot needs on a GitHub repository.
var ghRequiredLabels = []ghLabel{
	{name: "devpilot", color: "0075ca", desc: "Task managed by DevPilot"},
	{name: "in-progress", color: "e4e669", desc: "Task is currently being executed by DevPilot"},
	{name: "failed", color: "d93f0b", desc: "DevPilot task execution failed"},
	{name: "P0-critical", color: "b60205", desc: "Highest priority — execute first"},
	{name: "P1-high", color: "e99695", desc: "High priority"},
	{name: "P2-normal", color: "c5def5", desc: "Normal priority (default)"},
}

// ConfigureGitHubSource saves source=github to .devpilot.yaml and creates the
// required labels on the current GitHub repository via the gh CLI.
func ConfigureGitHubSource(opts GenerateOpts) error {
	cfg, err := project.Load(opts.Dir)
	if err != nil {
		cfg = &project.Config{}
	}
	cfg.Source = "github"
	if err := project.Save(opts.Dir, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Println("  Configured task source: GitHub Issues")

	fmt.Println("  Creating required GitHub labels (--force skips existing)...")
	for _, l := range ghRequiredLabels {
		out, err := exec.Command("gh", "label", "create", l.name,
			"--color", l.color,
			"--description", l.desc,
			"--force",
		).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not create label %q: %s\n", l.name, strings.TrimSpace(string(out)))
		} else {
			fmt.Printf("  Label created: %s\n", l.name)
		}
	}
	return nil
}

// gitignoreContains reports whether existing already declares entry as a
// gitignore line. Comparison is line-by-line (not substring): each line is
// trimmed of whitespace and a single leading "!" (gitignore's negate prefix);
// blank and comment lines are skipped. The entry is normalized the same way
// for symmetry, so callers can pass "!foo" or "foo" interchangeably.
func gitignoreContains(existing, entry string) bool {
	want := strings.TrimSpace(entry)
	want = strings.TrimPrefix(want, "!")
	for _, line := range strings.Split(existing, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "!")
		if line == want {
			return true
		}
	}
	return false
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
		if !gitignoreContains(existing, entry) {
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
	defer func() { _ = f.Close() }()

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

// SkillInstallOpts holds injectable functions for InstallSkills.
// All fields are optional; nil values use production defaults.
type SkillInstallOpts struct {
	// SelectFn presents a skill catalog and returns the names the user selected.
	SelectFn func(catalog []skillmgr.CatalogEntry) ([]string, error)

	// FetchCatalogFn returns the available skill catalog.
	FetchCatalogFn func() ([]skillmgr.CatalogEntry, error)

	// FetchSkillFn fetches skill files for a given name and tag.
	FetchSkillFn func(name, tag string) ([]skillmgr.SkillFile, error)
}

// InstallSkills presents a multi-select checklist of devpilot's built-in skills
// and installs the selected ones. Skipped in non-interactive mode.
func InstallSkills(opts GenerateOpts, installOpts SkillInstallOpts) error {
	if !opts.Interactive {
		return nil
	}

	selectFn := installOpts.SelectFn
	if selectFn == nil {
		selectFn = skillmgr.SelectSkillsFromCatalog
	}

	fetchCatalogFn := installOpts.FetchCatalogFn
	if fetchCatalogFn == nil {
		fetchCatalogFn = func() ([]skillmgr.CatalogEntry, error) {
			fmt.Println("  Discovering available skills...")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			catalog, err := skillmgr.FetchCatalog(ctx, "siyuqian", "devpilot", "main")
			if err != nil {
				return nil, fmt.Errorf("fetching skill catalog: %w", err)
			}
			return catalog, nil
		}
	}

	catalog, err := fetchCatalogFn()
	if err != nil {
		return err
	}
	if len(catalog) == 0 {
		fmt.Println("  No skills found in catalog.")
		return nil
	}

	selected, err := selectFn(catalog)
	if err != nil {
		return fmt.Errorf("skill selection: %w", err)
	}
	if len(selected) == 0 {
		return nil
	}

	fetchSkillFn := installOpts.FetchSkillFn
	if fetchSkillFn == nil {
		fetchSkillFn = func(name, ref string) ([]skillmgr.SkillFile, error) {
			return skillmgr.FetchSkill("siyuqian", "devpilot", name, ref)
		}
	}

	cfg, err := project.Load(opts.Dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	for _, name := range selected {
		fmt.Printf("  Installing skill %q...\n", name)
		files, err := fetchSkillFn(name, "main")
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to fetch skill %q: %v\n", name, err)
			continue
		}
		if err := skillmgr.InstallSkill(filepath.Join(opts.Dir, skillmgr.InstallDir), name, files); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to install skill %q: %v\n", name, err)
			continue
		}
		cfg.UpsertSkill(project.SkillEntry{
			Name:        name,
			Source:      skillmgr.DefaultSource,
			InstalledAt: time.Now().UTC(),
		})
		fmt.Printf("  Installed %s/%s/\n", skillmgr.InstallDir, name)
	}

	return project.Save(opts.Dir, cfg)
}
