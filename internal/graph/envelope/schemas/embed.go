// Package schemas embeds the JSON-schema files that the envelope validator
// consults. New schemas land in this directory and are picked up automatically.
package schemas

import "embed"

// FS holds every *.json schema shipped with the envelope package.
//
//go:embed *.json
var FS embed.FS
