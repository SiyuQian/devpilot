# Devpilot CLI — Design Document

**Date:** 2026-02-27
**Status:** Approved

## Summary

A Go CLI tool (`devpilot`) that handles authentication for external services used with Claude Code. Replaces the MCP server approach with a single installable binary. Trello is the first supported service.

## Motivation

Three pain points with the MCP server approach:
1. **Setup friction** — users must configure `.mcp.json`, run `npm install`, manage server processes
2. **Distribution** — hard to share/install MCP servers compared to `brew install`
3. **Scope** — a CLI works standalone, not just within Claude Code

## Design Decisions

### Language: Go

- Single binary, no runtime dependencies
- Cross-platform builds are trivial
- Homebrew-friendly distribution

### CLI Framework: Cobra

- Industry standard (used by gh, kubectl, docker)
- Subcommand support, auto-generated help, shell completions

### Scope: Auth-only (v1)

- CLI handles login/logout/status for external services
- Actual API interactions remain in skills or future CLI subcommands
- Trello is the only service at launch

### MCP Server: Removed

- `mcps/trello-mcp-server/` is deleted entirely
- No deprecation period — clean break

## Commands

### `devpilot login trello`

1. Prints instructions for getting API key and token from Trello
2. Prompts interactively for `API Key:` and `Token:`
3. Verifies credentials by calling Trello `/members/me` endpoint
4. On success: saves to `~/.config/devpilot/credentials.json`, prints confirmation
5. On failure: prints error, does not save

### `devpilot logout trello`

1. Removes the `trello` entry from credentials file
2. Prints confirmation

### `devpilot status`

1. Lists all services with auth status (e.g., `trello: logged in`)
2. If nothing configured: `No services configured.`

## Credentials

Stored at `~/.config/devpilot/credentials.json`:

```json
{
  "trello": {
    "api_key": "...",
    "token": "..."
  }
}
```

Each service gets its own top-level key.

## Project Structure

```
cli/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go              # Root command, version, help
│   ├── login.go             # devpilot login <service>
│   ├── logout.go            # devpilot logout <service>
│   └── status.go            # devpilot status
└── internal/
    ├── config/
    │   └── credentials.go   # Read/write ~/.config/devpilot/credentials.json
    └── services/
        └── trello.go        # Trello auth flow
```

## Service Interface

```go
type Service interface {
    Name() string
    Login() error
    Logout() error
    IsLoggedIn() bool
}
```

Services are registered in a `map[string]Service`. Adding a new service means:
1. Create a new file in `internal/services/`
2. Implement the interface
3. Register in the map

## Distribution

### Primary: Homebrew

```
brew tap siyuqian/devpilot
brew install devpilot
```

Use `goreleaser` to automate: build binaries → create GitHub Release → update Homebrew formula.

### Secondary

- `go install github.com/siyuqian/devpilot/cli@latest`
- Direct binary download from GitHub Releases

## What Gets Removed

- `mcps/trello-mcp-server/` — deleted entirely
- Trello MCP config in `.mcp.json` — removed
- `mcp-builder` skill — evaluate for removal (no longer the recommended approach)

## Non-Goals

- Full API subcommands (`devpilot trello list-boards`) — future scope
- Plugin system — YAGNI, just add files
- GUI or web-based auth flows (OAuth browser redirect) — future scope
- Windows support at launch — can add later via goreleaser
