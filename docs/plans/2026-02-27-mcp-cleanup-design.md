# MCP Server Cleanup — Design Document

**Date:** 2026-02-27
**Status:** Approved

## Summary

Remove all remaining MCP server artifacts and rewrite the Trello skill to use direct curl calls against the Trello REST API, reading credentials from `~/.config/devpilot/credentials.json`.

## Context

The devpilot CLI replaced the MCP server approach (per `2026-02-27-devpilot-cli-design.md`). The `mcps/trello-mcp-server/` directory was already deleted, but configuration files and skill documentation still reference it.

## Changes

### Delete

- `.mcp.json` — only contains the dead trello MCP server entry

### Edit: `.claude/settings.local.json`

Remove MCP-related permissions and config:
- `Bash(cd .../mcps/trello-mcp-server && npm install 2>&1)` permission
- `Bash(npm run:*)` permission
- `mcp__trello__trello_list_boards` permission
- `enabledMcpjsonServers` block

### Rewrite: `.claude/skills/trello/trello/SKILL.md`

Replace MCP tool documentation with curl-based workflows:

- **Setup:** `devpilot login trello` (no env vars, no MCP config)
- **Credential access:** Read `~/.config/devpilot/credentials.json` to extract `api_key` and `token`
- **API calls:** Direct curl to `https://api.trello.com/1/...` with key/token query params
- **Same operations:** list boards, get board, list/search/get/create/move cards, add comment, get labels, get members
- **Same workflow examples**, rewritten as curl commands

### Not changed

- Go CLI code — no modifications needed
- `docs/plans/2026-02-27-devpilot-cli-design.md` — already documents this removal
- `mcp-builder` skill — separate concern
