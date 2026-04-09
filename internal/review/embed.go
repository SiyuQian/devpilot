package review

import _ "embed"

//go:embed review-prompt.md
var reviewPromptMD string

//go:embed review-scoring.md
var reviewScoringMD string
