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

