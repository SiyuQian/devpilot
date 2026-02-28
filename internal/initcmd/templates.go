package initcmd

const claudeMDTemplate = `# CLAUDE.md

## Project Overview

{{.ProjectName}} — [TODO: add project description]

## Build & Development Commands

` + "```bash" + `
{{- if .BuildCmd}}
{{.BuildCmd}}
{{- end}}
{{- if .TestCmd}}
{{.TestCmd}}
{{- end}}
` + "```" + `

## Project Structure

[TODO: document key directories]
`

const skillMDTemplate = `---
name: {{.SkillName}}
description: "[TODO: describe what this skill does]"
---

# {{.SkillName}}

[TODO: add skill instructions]
`
