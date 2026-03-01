---
name: devpilot:trello
description: Interact with Trello boards, lists, and cards directly from Claude Code. Use when the user wants to view boards, search/create/move/update Trello cards, add comments, or get a board overview. Triggers on any mention of Trello, kanban boards, task cards, or project board management.
---

# Trello

Manage Trello boards and cards using direct REST API calls with credentials stored by the devpilot CLI.

## Setup

Run `devpilot login trello` to authenticate. This stores your API key and token at `~/.config/devpilot/credentials.json`.

If not logged in, tell the user to run `devpilot login trello` and stop.

## Reading Credentials

Extract credentials from the devpilot config:

```bash
TRELLO_KEY=$(cat ~/.config/devpilot/credentials.json | python3 -c "import sys,json; print(json.load(sys.stdin)['trello']['api_key'])")
TRELLO_TOKEN=$(cat ~/.config/devpilot/credentials.json | python3 -c "import sys,json; print(json.load(sys.stdin)['trello']['token'])")
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
