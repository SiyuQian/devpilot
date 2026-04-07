package review

import _ "embed"

//go:embed review-prompt.md
var reviewPromptMD string

//go:embed review-template.md
var reviewTemplateMD string
