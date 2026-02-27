# MCP Server Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove all dead MCP server artifacts and rewrite the Trello skill to use direct curl calls with devkit-managed credentials.

**Architecture:** Delete `.mcp.json`, clean MCP references from `.claude/settings.local.json`, and rewrite the Trello skill SKILL.md to instruct Claude to read credentials from `~/.config/devkit/credentials.json` and make direct curl calls to the Trello REST API.

**Tech Stack:** Bash (curl, jq), Trello REST API

---

### Task 1: Delete `.mcp.json`

**Files:**
- Delete: `.mcp.json`

**Step 1: Delete the file**

```bash
rm .mcp.json
```

**Step 2: Verify it's gone**

Run: `ls .mcp.json 2>&1`
Expected: `ls: .mcp.json: No such file or directory`

**Step 3: Commit**

```bash
git add .mcp.json
git commit -m "chore: remove .mcp.json (dead trello MCP server reference)"
```

---

### Task 2: Clean `.claude/settings.local.json`

**Files:**
- Modify: `.claude/settings.local.json`

Remove these three lines from `permissions.allow` array:
- Line 8: `"Bash(cd /Users/siyu/Works/github.com/siyuqian/developer-kit/mcps/trello-mcp-server && npm install 2>&1)"`
- Line 9: `"Bash(npm run:*)"`
- Line 10: `"mcp__trello__trello_list_boards"`

Remove the entire `enabledMcpjsonServers` block (lines 33-35):
```json
"enabledMcpjsonServers": [
    "trello"
]
```

**Step 1: Edit the file**

Result should be:

```json
{
  "permissions": {
    "allow": [
      "Bash(python3:*)",
      "Bash(chmod +x:*)",
      "Bash(bash:*)",
      "WebFetch(domain:raw.githubusercontent.com)",
      "Bash(curl:*)",
      "WebSearch",
      "WebFetch(domain:claude.com)",
      "WebFetch(domain:github.com)",
      "WebFetch(domain:docs.anthropic.com)",
      "WebFetch(domain:alexop.dev)",
      "WebFetch(domain:gist.github.com)",
      "WebFetch(domain:www.claudecodecamp.com)",
      "WebFetch(domain:addyosmani.com)",
      "WebFetch(domain:www.anthropic.com)",
      "WebFetch(domain:claudefa.st)",
      "Bash(echo:*)",
      "Bash(go version:*)",
      "Bash(mkdir:*)",
      "Bash(go get:*)",
      "Bash(go build:*)",
      "Bash(go mod:*)",
      "Bash(./devkit:*)",
      "Bash(go test:*)"
    ]
  },
  "enableAllProjectMcpServers": true
}
```

**Step 2: Verify JSON is valid**

Run: `python3 -c "import json; json.load(open('.claude/settings.local.json'))"`
Expected: No output (success)

**Step 3: Commit**

```bash
git add .claude/settings.local.json
git commit -m "chore: remove MCP permissions from settings.local.json"
```

---

### Task 3: Rewrite Trello skill

**Files:**
- Modify: `.claude/skills/trello/trello/SKILL.md`

**Step 1: Replace the entire file**

Write the following content to `.claude/skills/trello/trello/SKILL.md`:

````markdown
---
name: developerkit:trello
description: Interact with Trello boards, lists, and cards directly from Claude Code. Use when the user wants to view boards, search/create/move/update Trello cards, add comments, or get a board overview. Triggers on any mention of Trello, kanban boards, task cards, or project board management.
---

# Trello

Manage Trello boards and cards using direct REST API calls with credentials stored by the devkit CLI.

## Setup

Run `devkit login trello` to authenticate. This stores your API key and token at `~/.config/devkit/credentials.json`.

If not logged in, tell the user to run `devkit login trello` and stop.

## Reading Credentials

Extract credentials from the devkit config:

```bash
TRELLO_KEY=$(cat ~/.config/devkit/credentials.json | python3 -c "import sys,json; print(json.load(sys.stdin)['trello']['api_key'])")
TRELLO_TOKEN=$(cat ~/.config/devkit/credentials.json | python3 -c "import sys,json; print(json.load(sys.stdin)['trello']['token'])")
```

Use these in all API calls as query parameters: `key=$TRELLO_KEY&token=$TRELLO_TOKEN`

## API Reference

Base URL: `https://api.trello.com/1`

| Operation | Method | Endpoint | Key params |
|-----------|--------|----------|------------|
| List boards | GET | `/members/me/boards?filter=open` | — |
| Get board | GET | `/boards/{id}?lists=open&cards=open&card_fields=name,idList,labels,due&fields=name,desc` | board ID |
| List cards in a list | GET | `/lists/{id}/cards` | list ID |
| Search cards | GET | `/search?query={q}&modelTypes=cards` | query, optional `idBoards` |
| Get card | GET | `/cards/{id}?fields=name,desc,due,labels,idList,idBoard&members=true&actions=commentCard&actions_limit=10` | card ID |
| Create card | POST | `/cards` | `idList`, `name`, optional `desc`, `due`, `idLabels` |
| Move card | PUT | `/cards/{id}` | `idList` (new list) |
| Add comment | POST | `/cards/{id}/actions/comments` | `text` |
| Get board labels | GET | `/boards/{id}/labels` | board ID |
| Get board members | GET | `/boards/{id}/members` | board ID |

## Workflows

### "Show me my boards"

```bash
curl -s "https://api.trello.com/1/members/me/boards?filter=open&key=$TRELLO_KEY&token=$TRELLO_TOKEN"
```

### "What's on the Sprint board?"

1. List boards to find the board ID
2. Get the board with lists and cards:

```bash
curl -s "https://api.trello.com/1/boards/{boardId}?lists=open&cards=open&card_fields=name,idList,labels,due&fields=name,desc&key=$TRELLO_KEY&token=$TRELLO_TOKEN"
```

### "Find cards about authentication"

```bash
curl -s "https://api.trello.com/1/search?query=authentication&modelTypes=cards&key=$TRELLO_KEY&token=$TRELLO_TOKEN"
```

### "Create a bug card on the Backend board in To Do"

1. List boards → find Backend board ID
2. Get board → find "To Do" list ID
3. Create card:

```bash
curl -s -X POST "https://api.trello.com/1/cards?idList={listId}&name=Bug+title&desc=Description&key=$TRELLO_KEY&token=$TRELLO_TOKEN"
```

### "Move the login fix card to Done"

1. Search cards for "login fix" → get card ID
2. Get board → find "Done" list ID
3. Move card:

```bash
curl -s -X PUT "https://api.trello.com/1/cards/{cardId}?idList={doneListId}&key=$TRELLO_KEY&token=$TRELLO_TOKEN"
```

### "Add a comment on the deploy card: PR merged"

1. Search cards for "deploy" → get card ID
2. Add comment:

```bash
curl -s -X POST "https://api.trello.com/1/cards/{cardId}/actions/comments?text=PR+merged&key=$TRELLO_KEY&token=$TRELLO_TOKEN"
```

## Name Resolution

Users refer to boards, lists, and cards by name. Always resolve names to IDs first:
- Boards: list all boards, match by name
- Lists: get board with `lists=open`, match by name
- Cards: search by keyword or get board with `cards=open`, match by name
````

**Step 2: Verify the file looks correct**

Run: `head -5 .claude/skills/trello/trello/SKILL.md`
Expected:
```
---
name: developerkit:trello
description: Interact with Trello boards, lists, and cards directly from Claude Code...
---
```

**Step 3: Commit**

```bash
git add .claude/skills/trello/trello/SKILL.md
git commit -m "feat: rewrite Trello skill to use direct curl API calls

Replace MCP server references with direct Trello REST API calls.
Credentials are read from ~/.config/devkit/credentials.json
(managed by devkit login trello)."
```
