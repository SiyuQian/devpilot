## Why

DevPilot currently only supports Trello authentication via manual API key + token entry. Upcoming Gmail and Slack integrations both require OAuth 2.0 browser-based authorization. Rather than implementing OAuth twice, we need a shared OAuth browser flow component that any service can reuse. This unblocks the gmail-integration and slack-integration changes.

## What Changes

- Add a reusable OAuth browser flow module in `internal/auth/` that:
  - Starts a temporary local HTTP server to receive the OAuth callback
  - Opens the user's default browser to the provider's authorization URL
  - Exchanges the authorization code for access and refresh tokens
  - Stores tokens via the existing `credentials.go` infrastructure
  - Automatically refreshes expired access tokens using refresh tokens
  - Provides shared error types for OAuth failures (expired token, re-auth required)
- Extend `ServiceCredentials` usage to support OAuth token fields (access_token, refresh_token, expiry, etc.)
- Provider-specific config (client IDs, scopes, URLs) remains in each domain package — this change only provides the shared machinery

## Capabilities

### New Capabilities
- `oauth-flow`: Reusable OAuth 2.0 authorization code flow with browser redirect, local callback server, token exchange, and automatic token refresh. Configurable per-provider (authorize URL, token URL, scopes, client ID/secret).

### Modified Capabilities

## Impact

- `internal/auth/` — new `oauth.go` file with the shared flow; credentials storage gains OAuth token fields
- No breaking changes to existing Trello auth (it continues using manual key/token entry)
- New dependency: none (Go stdlib `net/http` for local server, `os/exec` or `browser` package for opening URLs)
- Gmail and Slack services will depend on this component for their `Login()` implementations
