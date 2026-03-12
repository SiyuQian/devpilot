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
		Name:        "content-creator",
		Description: "SEO-optimized blog and content writing skill",
	},
	{
		Name:        "google-go-style",
		Description: "Google Go Style Guide enforcement for writing and reviewing Go code",
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
