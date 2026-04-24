// Package generate implements the `devpilot commit` command, which shells
// out to `claude --print` to author a conventional commit message from
// staged changes.
package generate

import "embed"

//go:embed prompts
var promptsFS embed.FS
