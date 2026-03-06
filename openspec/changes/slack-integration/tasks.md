## 1. Slack Auth Service

- [ ] 1.1 Create `internal/slack/service.go` implementing `auth.Service` interface (Name, Login, Logout, IsLoggedIn)
- [ ] 1.2 Define Slack-specific OAuth constants in `internal/slack/service.go` (client ID, client secret, auth URL, token URL, bot scopes) — these live in the domain package, not in `internal/auth/`
- [ ] 1.3 Implement `Login()`: construct `auth.OAuthConfig` from Slack constants, call `auth.StartFlow()`, parse response for bot token + workspace metadata (team_id, team_name), save via `auth.Save()`
- [ ] 1.4 Register Slack service in `internal/auth/service.go` init()

## 2. Slack API Client

- [ ] 2.1 Create `internal/slack/client.go` with `NewClient(botToken string, opts ...Option)` and functional options
- [ ] 2.2 Implement `ListConversations()` — calls `conversations.list` API to list public channels
- [ ] 2.3 Implement `ResolveChannel(name string)` — finds channel ID by exact name match from conversations list
- [ ] 2.4 Implement `PostMessage(channelID string, text string)` — calls `chat.postMessage` API

## 3. CLI Commands

- [ ] 3.1 Create `internal/slack/commands.go` with `slack` parent command and register in root
- [ ] 3.2 Implement `devpilot slack send --channel <name> --message <text>` — resolves channel, posts message
- [ ] 3.3 Add stdin support: read message from stdin when `--message` is not provided
- [ ] 3.4 Add login check to all slack subcommands with helpful error message

## 4. Tests

- [ ] 4.1 Unit test Slack service Login/Logout/IsLoggedIn with mock OAuth flow
- [ ] 4.2 Unit test client ListConversations/ResolveChannel/PostMessage with httptest mock server
- [ ] 4.3 Unit test channel name resolution (exact match, not found, hash prefix stripping)
- [ ] 4.4 Unit test stdin message reading
