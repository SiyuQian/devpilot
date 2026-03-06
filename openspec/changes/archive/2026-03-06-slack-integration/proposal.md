## Why

The email-assistant workflow needs to send digest summaries to Slack. Adding Slack as a first-class devpilot service with OAuth login and a `send` command keeps the CLI consistent with Gmail and Trello integrations — every external service follows the same `devpilot login/logout` + domain commands pattern.

## What Changes

- Add Slack as an auth service implementing `auth.Service`, using the shared OAuth browser flow
- Embed Slack App OAuth client ID/secret in the binary (devpilot-managed Slack App)
- Add `devpilot slack send` command to send messages to Slack channels
- Register Slack service in auth registry

## Capabilities

### New Capabilities
- `slack-auth`: Slack OAuth authentication via browser flow with bot token scopes (chat:write, channels:read).
- `slack-commands`: CLI command for sending messages to Slack channels via Slack Web API.

### Modified Capabilities

## Impact

- `internal/slack/` — new package with client, service, and commands
- `internal/auth/service.go` — register Slack service in `init()`
- Depends on `oauth-browser-flow` change for OAuth login flow
- New external dependency: Slack Web API (https://slack.com/api/)
- Requires devpilot Slack App with bot token scopes
