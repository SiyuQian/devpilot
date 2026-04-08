package skillmgr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/siyuqian/devpilot/internal/project"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// fetchCatalogFn is the function used to fetch the skill catalog.
// Override in tests to avoid hitting GitHub.
var fetchCatalogFn = func(ctx context.Context, owner, repo, ref string) ([]CatalogEntry, error) {
	return FetchCatalog(ctx, owner, repo, ref)
}

// fetchLatestTagFn is the function used to resolve the latest release tag.
// Override in tests to avoid hitting GitHub.
var fetchLatestTagFn = FetchLatestTag

// descriptionLimit is the max length for skill descriptions in list output.
const descriptionLimit = 40

// RegisterCommands adds the skill command to the parent command.
func RegisterCommands(parent *cobra.Command) {
	skillCmd.AddCommand(skillAddCmd)
	skillCmd.AddCommand(skillListCmd)
	parent.AddCommand(skillCmd)
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Claude Code skills",
}

// userConfigDirFn is the function used to resolve the user config directory.
// Override in tests to avoid writing to the real user config.
var userConfigDirFn = project.UserConfigDir

// UserSkillDir returns the directory where user-level skills are installed.
func UserSkillDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "skills"), nil
}

// promptInstallLevel asks the user to select project or user level.
// Returns the resolved base directory for skill installation and whether
// it is a user-level install. Defaults to project level.
func promptInstallLevel(projectDir string, reader *bufio.Reader) (baseDir string, userLevel bool) {
	projectBase := filepath.Join(projectDir, InstallDir)

	if reader == nil {
		return projectBase, false
	}

	userDir, err := UserSkillDir()
	if err != nil {
		return projectBase, false
	}

	fmt.Println("Install level:")
	fmt.Printf("  1) Project (%s/)\n", InstallDir)
	fmt.Printf("  2) User (%s/)\n", userDir)
	fmt.Print("Select [1]: ")

	line, readErr := reader.ReadString('\n')
	if readErr != nil {
		return projectBase, false
	}

	if strings.TrimSpace(line) == "2" {
		return userDir, true
	}
	return projectBase, false
}

var skillAddCmd = &cobra.Command{
	Use:   "add <name[@version]>",
	Short: "Install a skill from the devpilot catalog",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		name, version, err := parseSkillArg(args[0])
		if err != nil {
			return err
		}

		if version == "" {
			fmt.Printf("Resolving latest version of %q...\n", name)
			version, err = FetchLatestTag(defaultOwner, defaultRepo)
			if err != nil {
				return fmt.Errorf("resolving latest tag: %w", err)
			}
		}

		fmt.Printf("Fetching skill %q at %s...\n", name, version)
		files, err := FetchSkill(defaultOwner, defaultRepo, name, version)
		if err != nil {
			return fmt.Errorf("fetching skill: %w", err)
		}
		if len(files) == 0 {
			return fmt.Errorf("skill %q not found in devpilot catalog", name)
		}

		// Prompt for install level (project vs user) when running interactively.
		var reader *bufio.Reader
		if term.IsTerminal(int(os.Stdin.Fd())) {
			reader = bufio.NewReader(os.Stdin)
		}
		baseDir, userLevel := promptInstallLevel(dir, reader)

		if err := InstallSkill(baseDir, name, files); err != nil {
			return fmt.Errorf("installing skill: %w", err)
		}

		configDir := dir
		if userLevel {
			ud, err := userConfigDirFn()
			if err != nil {
				return fmt.Errorf("resolving user config dir: %w", err)
			}
			configDir = ud
		}
		cfg, err := project.Load(configDir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		cfg.UpsertSkill(project.SkillEntry{
			Name:        name,
			Source:      DefaultSource,
			Version:     version,
			InstalledAt: time.Now().UTC(),
		})
		if err := project.Save(configDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		displayPath := InstallDir + "/" + name + "/"
		if userLevel {
			displayPath = baseDir + "/" + name + "/"
		}
		fmt.Printf("Installed skill %q (%s) into %s\n", name, version, displayPath)
		return nil
	},
}

type skillWithLevel struct {
	project.SkillEntry
	Level string
}

// truncateDescription truncates s to descriptionLimit runes, appending "..." if truncated.
func truncateDescription(s string) string {
	runes := []rune(s)
	if len(runes) <= descriptionLimit {
		return s
	}
	return string(runes[:descriptionLimit]) + "..."
}

func init() {
	skillListCmd.Flags().BoolP("installed", "i", false, "Show only installed skills")
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skills and their installation status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		installedOnly, _ := cmd.Flags().GetBool("installed")

		// Load installed skills from both levels.
		installed := loadInstalledSkills()

		if installedOnly {
			return printInstalledOnly(installed)
		}

		// Fetch catalog.
		ctx := context.Background()
		ref, err := fetchLatestTagFn(defaultOwner, defaultRepo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not resolve latest version: %v\nShowing installed skills only.\n", err)
			return printInstalledOnly(installed)
		}

		catalog, err := fetchCatalogFn(ctx, defaultOwner, defaultRepo, ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not fetch skill catalog: %v\nShowing installed skills only.\n", err)
			return printInstalledOnly(installed)
		}

		return printCatalogView(catalog, installed)
	},
}

// loadInstalledSkills loads skills from both project and user configs.
func loadInstalledSkills() []skillWithLevel {
	var all []skillWithLevel

	dir, err := os.Getwd()
	if err == nil {
		projCfg, err := project.Load(dir)
		if err == nil {
			for _, s := range projCfg.Skills {
				all = append(all, skillWithLevel{s, "project"})
			}
		}
	}

	userDir, err := userConfigDirFn()
	if err == nil {
		userCfg, err := project.Load(userDir)
		if err == nil {
			for _, s := range userCfg.Skills {
				all = append(all, skillWithLevel{s, "user"})
			}
		}
	}

	return all
}

// printInstalledOnly displays only installed skills (the --installed view).
func printInstalledOnly(installed []skillWithLevel) error {
	if len(installed) == 0 {
		fmt.Println("No skills installed. Use 'devpilot skill add <name>' to install one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tVERSION\tLEVEL")
	for _, s := range installed {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Version, s.Level)
	}
	return w.Flush()
}

// printCatalogView displays all catalog skills with installation status.
func printCatalogView(catalog []CatalogEntry, installed []skillWithLevel) error {
	// Build lookup: skill name → installed info.
	type installInfo struct {
		Version string
		Level   string
	}
	lookup := make(map[string]installInfo, len(installed))
	for _, s := range installed {
		lookup[s.Name] = installInfo{Version: s.Version, Level: s.Level}
	}

	// Track which installed skills appear in the catalog.
	seen := make(map[string]bool, len(catalog))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDESCRIPTION\tVERSION\tLEVEL")
	for _, c := range catalog {
		seen[c.Name] = true
		desc := truncateDescription(c.Description)
		if info, ok := lookup[c.Name]; ok {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Name, desc, info.Version, info.Level)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%s\t—\t—\n", c.Name, desc)
		}
	}

	// Append installed skills that are not in the catalog (e.g. removed or renamed).
	for _, s := range installed {
		if !seen[s.Name] {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, "(not in catalog)", s.Version, s.Level)
		}
	}

	return w.Flush()
}

// parseSkillArg splits "pm@v1.2.3" into ("pm", "v1.2.3").
// Returns ("pm", "") if no version is specified.
func parseSkillArg(arg string) (name, version string, err error) {
	parts := strings.SplitN(arg, "@", 2)
	name = parts[0]
	if name == "" {
		return "", "", fmt.Errorf("skill name cannot be empty")
	}
	if len(parts) == 2 {
		version = parts[1]
	}
	return name, version, nil
}
