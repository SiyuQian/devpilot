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

// Override in tests to avoid hitting GitHub.
var fetchCatalogFn = func(ctx context.Context, owner, repo, ref string) ([]CatalogEntry, error) {
	return FetchCatalog(ctx, owner, repo, ref)
}

// Override in tests to avoid hitting GitHub.
var fetchSkillFn = func(owner, repo, name, ref string) ([]SkillFile, error) {
	return FetchSkill(owner, repo, name, ref)
}

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

// resolveInstallLevel applies precedence: --level flag > interactive prompt
// (if reader != nil) > project-level default.
// levelFlag must be "", "project", or "user"; any other value is an error.
func resolveInstallLevel(levelFlag, projectDir string, reader *bufio.Reader) (baseDir string, userLevel bool, err error) {
	switch levelFlag {
	case "project":
		return filepath.Join(projectDir, InstallDir), false, nil
	case "user":
		userDir, derr := UserSkillDir()
		if derr != nil {
			return "", false, derr
		}
		return userDir, true, nil
	case "":
		base, ul := promptInstallLevel(projectDir, reader)
		return base, ul, nil
	default:
		return "", false, fmt.Errorf("invalid --level value %q: must be 'project' or 'user'", levelFlag)
	}
}

// validateSkillAddArgs enforces that exactly one of {positional name, --all}
// is provided. Called from skillAddCmd.Args.
func validateSkillAddArgs(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool("all")
	switch {
	case all && len(args) > 0:
		return fmt.Errorf("cannot combine --all with a skill name")
	case !all && len(args) == 0:
		return fmt.Errorf("skill name is required (or use --all to install the entire catalog)")
	case !all && len(args) > 1:
		return fmt.Errorf("skill add accepts exactly one skill name")
	}
	return nil
}

var skillAddCmd = &cobra.Command{
	Use:   "add [name[@ref]]",
	Short: "Install a skill from the devpilot catalog",
	Long: `Install a skill from the devpilot catalog.

Use a positional argument to install a single named skill, or pass --all to
install every skill in the catalog. Use --level to pick the install
destination non-interactively (project or user).`,
	Args: validateSkillAddArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		all, _ := cmd.Flags().GetBool("all")
		levelFlag, _ := cmd.Flags().GetString("level")

		// Validate --level early, before any network / filesystem work.
		if levelFlag != "" && levelFlag != "project" && levelFlag != "user" {
			return fmt.Errorf("invalid --level value %q: must be 'project' or 'user'", levelFlag)
		}

		var reader *bufio.Reader
		if term.IsTerminal(int(os.Stdin.Fd())) {
			reader = bufio.NewReader(os.Stdin)
		}

		if all {
			return runBulkInstall(dir, levelFlag, reader)
		}

		return runSingleInstall(dir, args[0], levelFlag, reader)
	},
}

// runSingleInstall installs one skill (the existing default behavior).
func runSingleInstall(dir, arg, levelFlag string, reader *bufio.Reader) error {
	name, ref, err := parseSkillArg(arg)
	if err != nil {
		return err
	}
	if ref == "" {
		ref = defaultRef
	}

	fmt.Printf("Fetching skill %q...\n", name)
	files, err := FetchSkill(defaultOwner, defaultRepo, name, ref)
	if err != nil {
		return fmt.Errorf("fetching skill: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("skill %q not found in devpilot catalog", name)
	}

	baseDir, userLevel, err := resolveInstallLevel(levelFlag, dir, reader)
	if err != nil {
		return err
	}

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
		InstalledAt: time.Now().UTC(),
	})
	if err := project.Save(configDir, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	displayPath := filepath.Join(baseDir, name) + "/"
	fmt.Printf("Installed skill %q into %s\n", name, displayPath)
	return nil
}

// runBulkInstall installs every skill in the catalog at the resolved level.
// Individual failures do not abort the batch; a summary is printed at the end
// and a non-nil error is returned if any skill failed.
func runBulkInstall(dir, levelFlag string, reader *bufio.Reader) error {
	ctx := context.Background()
	catalog, err := fetchCatalogFn(ctx, defaultOwner, defaultRepo, defaultRef)
	if err != nil {
		return fmt.Errorf("fetching skill catalog: %w", err)
	}
	if len(catalog) == 0 {
		return fmt.Errorf("skill catalog is empty")
	}

	baseDir, userLevel, err := resolveInstallLevel(levelFlag, dir, reader)
	if err != nil {
		return err
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

	type failure struct {
		name string
		err  error
	}
	var failures []failure
	installed := 0

	fmt.Printf("Installing %d skills into %s/...\n", len(catalog), baseDir)
	for _, entry := range catalog {
		name := entry.Name
		fmt.Printf("  • %s ... ", name)
		files, ferr := fetchSkillFn(defaultOwner, defaultRepo, name, defaultRef)
		if ferr != nil {
			fmt.Println("FAIL")
			failures = append(failures, failure{name, fmt.Errorf("fetching: %w", ferr)})
			continue
		}
		if len(files) == 0 {
			fmt.Println("FAIL")
			failures = append(failures, failure{name, fmt.Errorf("not found in catalog")})
			continue
		}
		if ierr := InstallSkill(baseDir, name, files); ierr != nil {
			fmt.Println("FAIL")
			failures = append(failures, failure{name, fmt.Errorf("installing: %w", ierr)})
			continue
		}
		cfg.UpsertSkill(project.SkillEntry{
			Name:        name,
			Source:      DefaultSource,
			InstalledAt: time.Now().UTC(),
		})
		installed++
		fmt.Println("ok")
	}

	// Persist config once at the end (even on partial failure so successful
	// installs are recorded).
	if installed > 0 {
		if serr := project.Save(configDir, cfg); serr != nil {
			return fmt.Errorf("saving config: %w", serr)
		}
	}

	fmt.Printf("\nInstalled %d/%d skills into %s/\n", installed, len(catalog), baseDir)
	if len(failures) > 0 {
		fmt.Println("Failed:")
		for _, f := range failures {
			fmt.Printf("  - %s: %v\n", f.name, f.err)
		}
		return fmt.Errorf("%d skill(s) failed to install", len(failures))
	}
	return nil
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
	skillAddCmd.Flags().Bool("all", false, "Install every skill in the devpilot catalog")
	skillAddCmd.Flags().String("level", "", "Install level: 'project' or 'user' (bypasses interactive prompt)")
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

		ctx := context.Background()
		catalog, err := fetchCatalogFn(ctx, defaultOwner, defaultRepo, defaultRef)
		if err != nil {
			warnFallback("could not fetch skill catalog", err)
			return printInstalledOnly(installed)
		}

		return printCatalogView(catalog, installed)
	},
}

func warnFallback(label string, err error) {
	fmt.Fprintf(os.Stderr, "Warning: %s: %v\nShowing installed skills only.\n", label, err)
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
	_, _ = fmt.Fprintln(w, "NAME\tINSTALLED\tLEVEL")
	for _, s := range installed {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, formatInstallDate(s.InstalledAt), s.Level)
	}
	return w.Flush()
}

// printCatalogView displays all catalog skills with installation status.
func printCatalogView(catalog []CatalogEntry, installed []skillWithLevel) error {
	lookup := make(map[string]skillWithLevel, len(installed))
	for _, s := range installed {
		lookup[s.Name] = s
	}

	seen := make(map[string]bool, len(catalog))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDESCRIPTION\tINSTALLED\tLEVEL")
	for _, c := range catalog {
		seen[c.Name] = true
		desc := truncateDescription(c.Description)
		if s, ok := lookup[c.Name]; ok {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Name, desc, formatInstallDate(s.InstalledAt), s.Level)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%s\t—\t—\n", c.Name, desc)
		}
	}

	// Append installed skills that are not in the catalog (e.g. removed or renamed).
	for _, s := range installed {
		if !seen[s.Name] {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, "(not in catalog)", formatInstallDate(s.InstalledAt), s.Level)
		}
	}

	return w.Flush()
}

// formatInstallDate formats a time as "2006-01-02" for display.
// Returns "—" for the zero time.
func formatInstallDate(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02")
}

// parseSkillArg splits "pm@v1.2.3" into ("pm", "v1.2.3").
// Returns ("pm", "") if no ref is specified.
func parseSkillArg(arg string) (name, ref string, err error) {
	parts := strings.SplitN(arg, "@", 2)
	name = parts[0]
	if name == "" {
		return "", "", fmt.Errorf("skill name cannot be empty")
	}
	if len(parts) == 2 {
		ref = parts[1]
	}
	return name, ref, nil
}
