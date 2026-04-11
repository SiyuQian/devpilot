// Package generate implements the `devpilot commit` and `devpilot readme`
// commands, which shell out to `claude --print` to author commit messages
// and README files.
package generate

import "embed"

//go:embed prompts
var promptsFS embed.FS
