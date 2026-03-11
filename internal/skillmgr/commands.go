package skillmgr

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/siyuqian/devpilot/internal/project"
	"github.com/spf13/cobra"
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

var skillAddCmd = &cobra.Command{
	Use:   "add <name[@version]>",
	Short: "Install a skill from the devpilot catalog into this project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		if !project.Exists(dir) {
			return fmt.Errorf("no .devpilot.yaml found; run 'devpilot init' first")
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

		if err := InstallSkill(dir, name, files); err != nil {
			return fmt.Errorf("installing skill: %w", err)
		}

		cfg, err := project.Load(dir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		cfg.UpsertSkill(project.SkillEntry{
			Name:        name,
			Source:      DefaultSource,
			Version:     version,
			InstalledAt: time.Now().UTC(),
		})
		if err := project.Save(dir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Installed skill %q (%s) into .claude/skills/%s/\n", name, version, name)
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

		if !project.Exists(dir) {
			return fmt.Errorf("no .devpilot.yaml found; run 'devpilot init' first")
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
		fmt.Fprintln(w, "NAME\tSOURCE\tVERSION\tINSTALLED")
		for _, s := range cfg.Skills {
			installed := s.InstalledAt.Format("2006-01-02")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Source, s.Version, installed)
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
