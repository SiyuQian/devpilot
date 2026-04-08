package skillmgr

import (
	"bufio"
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

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List skills installed in this project",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := project.Load(dir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if len(cfg.Skills) == 0 {
			fmt.Println("No skills installed. Use 'devpilot skill add <name>' to install one.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tSOURCE\tVERSION\tINSTALLED")
		for _, s := range cfg.Skills {
			installed := s.InstalledAt.Format("2006-01-02")
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Source, s.Version, installed)
		}
		return w.Flush()
	},
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
