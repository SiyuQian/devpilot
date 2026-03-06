# Remove .env dependency — OAuth credentials via interactive prompt

## Problem

OAuth client ID/secret for Gmail and Slack are loaded from `.env` via `godotenv`.
This is wrong for a distributed CLI tool: other users would authenticate through
the original developer's OAuth app, creating privacy and trust issues.

## Design

Follow the existing Trello login pattern: prompt users for their OAuth client
credentials during `devpilot login gmail/slack`, then store them alongside the
access tokens in `~/.config/devpilot/credentials.json`.

### Changes

1. **Gmail/Slack `Login()`** — Before starting OAuth flow, prompt for client ID
   and client secret via stdin (same as Trello prompts for API key/token).
   Store as `client_id` and `client_secret` in credentials.
2. **Gmail/Slack `oauthConfig()`** — Read client ID/secret from credentials
   store instead of `os.Getenv()`.
3. **`cmd/devpilot/main.go`** — Remove `godotenv.Load()` and import.
4. **`go.mod`** — Remove `github.com/joho/godotenv` dependency.
5. **Delete** `.env` and `.env.example`.
6. **`.gitignore`** — Remove `.env` entry (no longer needed).

### What stays the same

- OAuth flow logic (`auth.StartFlow`)
- Access token / refresh token storage
- `EDITOR` env var in `generate/commit_plan.go`
