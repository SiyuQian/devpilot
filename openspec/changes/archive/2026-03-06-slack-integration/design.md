## Context

DevPilot is adding Slack as the third external service (after Trello and Gmail). The `oauth-browser-flow` change provides the shared OAuth component. Slack's OAuth flow is simpler than Gmail's: bot tokens don't expire, so no refresh logic is needed.

Slack Web API: `https://slack.com/api/`

## Goals / Non-Goals

**Goals:**
- `devpilot login slack` authenticates via OAuth browser flow, installs the devpilot bot to the user's workspace
- `devpilot slack send --channel <channel> --message <text>` sends a message
- Support markdown-formatted messages (Slack's mrkdwn)

**Non-Goals:**
- Reading Slack messages
- Managing channels, threads, or reactions
- Interactive Slack bot / slash commands
- Slack Events API / Socket Mode

## Decisions

### 1. Package structure: domain package owns everything Slack-specific

```
internal/slack/
├── client.go         # Slack Web API client
├── service.go        # auth.Service implementation + Slack OAuth config (client ID, scopes, URLs)
├── commands.go       # Cobra subcommands
└── client_test.go    # Tests
```

The Slack package owns its OAuth client ID/secret, scopes, and auth/token URLs. `service.go` constructs an `auth.OAuthConfig` and calls `auth.StartFlow()`. Slack-specific token semantics (bot token, no refresh, workspace metadata) are handled entirely within this package.

### 2. Slack OAuth with bot token scopes

Slack OAuth V2 (`https://slack.com/oauth/v2/authorize`) with bot token scopes:
- `chat:write` — send messages to channels where the bot is a member
- `channels:read` — list public channels (for discovery/validation)

The OAuth response returns a `bot_token` (starts with `xoxb-`). This token does NOT expire, so no refresh logic needed.

**Why**: Bot tokens are simpler and more appropriate than user tokens for automated messaging.

### 3. Store bot token as credential

```json
{
  "slack": {
    "access_token": "xoxb-...",
    "team_id": "T0123...",
    "team_name": "My Workspace"
  }
}
```

No refresh_token or expiry needed — Slack bot tokens are long-lived.

### 4. Channel resolution by name

The `--channel` flag accepts channel names (with or without `#`). The client resolves names to channel IDs via `conversations.list` API before posting.

**Why**: Users think in channel names, not IDs.

### 5. Message formatting via Slack mrkdwn

Messages are sent as Slack `mrkdwn` blocks. The skill can format summaries with bold, bullets, links, etc.

### 6. Slack App credentials in `internal/slack/`

```go
// internal/slack/service.go
const (
    slackClientID     = "xxxx.xxxx"
    slackClientSecret = "xxxx"
    slackAuthURL      = "https://slack.com/oauth/v2/authorize"
    slackTokenURL     = "https://slack.com/api/oauth.v2.access"
)
```

Provider-specific constants live in the domain package. The `Login()` method constructs an `auth.OAuthConfig` from these and passes it to `auth.StartFlow()`. The Slack App is created under the devpilot project and distributed via OAuth install flow.

## Risks / Trade-offs

- **[Bot must be in channel]** → `chat:write` only works in channels where the bot has been added. Mitigation: clear error message telling user to `/invite @devpilot` to the channel.
- **[Slack App distribution]** → For public distribution, the Slack App needs to be submitted for review. Mitigation: users can install from the OAuth URL directly; formal listing is optional.
- **[Channel name collisions]** → Multiple channels could match a name in large workspaces. Mitigation: exact match only; error if ambiguous.
