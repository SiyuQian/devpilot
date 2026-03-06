## Context

DevPilot authenticates with Trello via manually entered API key + token. The upcoming Gmail and Slack integrations both require OAuth 2.0 Authorization Code flow with browser redirect. Both flows follow the same pattern: redirect user to provider, receive callback with auth code, exchange for tokens, store them. Building a shared component avoids duplication and ensures consistent UX.

Current auth infrastructure:
- `internal/auth/service.go` — `Service` interface with `Login()/Logout()/IsLoggedIn()`
- `internal/auth/credentials.go` — JSON-based credential storage at `~/.config/devpilot/credentials.json`
- `internal/auth/trello_service.go` — manual key/token entry pattern

## Goals / Non-Goals

**Goals:**
- Provide a reusable `OAuthFlow` component that any service can call from its `Login()` method
- Handle the full OAuth 2.0 authorization code flow: browser open → local callback → token exchange
- Support automatic token refresh for providers that issue refresh tokens (Gmail)
- Store OAuth tokens through the existing `credentials.go` infrastructure
- Provide shared error types for common OAuth failures (expired token, revoked access, etc.)

**Non-Goals:**
- PKCE support (not required by Gmail or Slack OAuth; can add later)
- Device code flow (for headless environments — future enhancement)
- Modifying the existing Trello auth flow
- Implementing Gmail or Slack commands themselves (separate changes)
- Embedding provider-specific client IDs/secrets (those belong in each domain package)

## Decisions

### 1. OAuthConfig struct + StartFlow function (not interface)

Use a simple config struct + function rather than an interface hierarchy.

```go
type OAuthConfig struct {
    ProviderName string
    AuthURL      string
    TokenURL     string
    ClientID     string
    ClientSecret string
    Scopes       []string
    RedirectPort int // default: random available port
}

type OAuthToken struct {
    AccessToken  string
    RefreshToken string
    Expiry       time.Time
    TokenType    string
}

func StartFlow(cfg OAuthConfig) (*OAuthToken, error)
```

**Why**: A function with config is simpler than an interface for this use case. Each service creates its config and calls `StartFlow`. No need for polymorphism — the variation is in config, not behavior.

**Alternative considered**: `OAuthProvider` interface — rejected as over-engineering since all OAuth 2.0 flows follow the same protocol.

### 2. Local HTTP server on random port

Start a temporary `net/http` server on `localhost` with a random available port. The redirect URI becomes `http://localhost:<port>/callback`.

**Why**: Random port avoids conflicts. Google OAuth supports `http://localhost` redirect URIs with any port for native apps (uses "loopback redirect" mode). Slack also supports localhost redirects.

**Alternative considered**: Fixed port (e.g., 9876) — rejected because it can conflict with other services.

### 3. Browser opening via `open` command (macOS) / `xdg-open` (Linux)

Use `os/exec` to call platform-specific browser-open commands. No external dependencies needed.

**Why**: Go stdlib is sufficient. macOS `open` and Linux `xdg-open` cover the two relevant platforms.

### 4. Token storage in existing credentials.json

Store OAuth tokens as `ServiceCredentials` map fields:

```json
{
  "gmail": {
    "access_token": "...",
    "refresh_token": "...",
    "expiry": "2024-01-15T10:30:00Z",
    "token_type": "Bearer"
  }
}
```

**Why**: Reuses existing infrastructure. No schema change needed — `ServiceCredentials` is already `map[string]string`.

### 5. Token refresh as a helper function

```go
func RefreshToken(cfg OAuthConfig, refreshToken string) (*OAuthToken, error)
```

Services call `RefreshToken` when they detect an expired access token. The refresh logic is not automatic/background — it's invoked on-demand by the service's API client.

**Why**: On-demand refresh is simpler and more predictable than background refresh goroutines. The service knows when it gets a 401 and can refresh + retry.

### 6. Hybrid architecture: shared infra in auth, provider config in domain packages

`internal/auth/` owns the shared OAuth machinery:
- `oauth.go` — `OAuthConfig`, `OAuthToken`, `StartFlow()`, `RefreshToken()`
- `credentials.go` — token storage (existing)
- `service.go` — `Service` interface + registry (existing)
- `errors.go` — shared OAuth error types (`ErrTokenExpired`, `ErrReauthRequired`)

Each domain package (`internal/gmail/`, `internal/slack/`) owns its provider-specific details:
- Client ID / secret constants
- OAuth scopes, URLs
- How to construct `OAuthConfig` for `auth.StartFlow()`
- How/when to call `auth.RefreshToken()` from its API client
- Workspace metadata, token interpretation

**Why**: The OAuth protocol is identical across providers — that's shared. But each provider has unique scopes, token semantics (Gmail refresh tokens vs Slack long-lived bot tokens), and metadata (Slack team_id). Keeping those in domain packages avoids `internal/auth/` becoming a dumping ground for every provider's quirks.

## Risks / Trade-offs

- **[Google OAuth "testing" mode]** → If the Google Cloud project stays in "testing" mode, refresh tokens expire after 7 days. Mitigation: publish the app for production approval, or use "internal" mode if the user has Google Workspace.
- **[Client secret in binary]** → The OAuth client secret is extractable from the binary. Mitigation: this is standard practice for native/CLI apps (Google, Slack, and GitHub all document this pattern). The secret alone cannot access user data without the user's authorization.
- **[Port conflicts]** → Random port could theoretically be taken. Mitigation: retry with a different port if bind fails.
- **[No headless support]** → Won't work on servers without a browser. Mitigation: document this limitation; device code flow is a future enhancement.
