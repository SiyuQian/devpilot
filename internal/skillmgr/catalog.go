package skillmgr

// CatalogEntry describes a skill available from the default source.
type CatalogEntry struct {
	Name        string
	Description string
}

// BuiltinCatalog lists all skills shipped with devpilot.
var BuiltinCatalog = []CatalogEntry{
	{
		Name:        "confluence-reviewer",
		Description: "Review Atlassian Confluence pages and leave inline and page-level comments",
	},
	{
		Name:        "google-go-style",
		Description: "Google Go Style Guide enforcement for writing and reviewing Go code",
	},
	{
		Name:        "openspec-apply-change",
		Description: "Implement tasks from an OpenSpec change",
	},
	{
		Name:        "openspec-archive-change",
		Description: "Archive a completed OpenSpec change after implementation",
	},
	{
		Name:        "openspec-explore",
		Description: "Explore ideas and clarify requirements before implementation",
	},
	{
		Name:        "openspec-propose",
		Description: "Propose a new change and generate all artifacts in one step",
	},
	{
		Name:        "pm",
		Description: "Product manager skill for market research and feature discovery",
	},
	{
		Name:        "task-executor",
		Description: "Executes a task plan autonomously (used by devpilot run)",
	},
	{
		Name:        "task-refiner",
		Description: "Improve Trello card task plans for the devpilot runner",
	},
	{
		Name:        "trello",
		Description: "Interact with Trello boards, lists, and cards directly from Claude Code",
	},
}
