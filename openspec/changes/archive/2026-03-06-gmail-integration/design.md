## Context

DevPilot has an existing auth service pattern (`internal/auth/`) and the `oauth-browser-flow` change provides a reusable OAuth component. Gmail integration follows the same architectural pattern as Trello: a domain package in `internal/gmail/` that owns its API client, auth service implementation, and Cobra commands.

Gmail REST API: `https://gmail.googleapis.com/gmail/v1/users/me/messages`

## Goals / Non-Goals

**Goals:**
- `devpilot login gmail` authenticates via OAuth browser flow with Gmail scopes
- `devpilot gmail list --unread` lists unread messages with sender, subject, date
- `devpilot gmail read <id>` displays email content in plain text
- `devpilot gmail mark-read <id>...` marks messages as read
- Output is structured and parseable (for skill consumption via `claude -p`)

**Non-Goals:**
- Sending emails
- Managing labels, drafts, or attachments
- Gmail push notifications / watch
- Rich HTML rendering of emails

## Decisions

### 1. Package structure: domain package owns everything Gmail-specific

```
internal/gmail/
├── client.go         # Gmail REST API client (list, get, modify)
├── service.go        # auth.Service implementation + Gmail OAuth config (client ID, scopes, URLs)
├── commands.go       # Cobra subcommands (list, read, mark-read)
└── client_test.go    # Tests with httptest mock server
```

The Gmail package owns its OAuth client ID/secret, scopes, and auth/token URLs as constants. `service.go` constructs an `auth.OAuthConfig` and calls `auth.StartFlow()` — the shared infra in `internal/auth/` knows nothing about Gmail.

**Why**: Consistent with existing `internal/trello/` pattern. Each domain owns its commands and auth config. The hybrid approach keeps `internal/auth/` as pure shared infrastructure.

### 2. Gmail API client with functional options

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func NewClient(accessToken string, opts ...Option) *Client
```

Use functional options (`WithBaseURL`, `WithHTTPClient`) for testability, same pattern as Trello client.

### 3. OAuth scopes: gmail.modify only

Request `https://www.googleapis.com/auth/gmail.modify` which covers both reading and marking as read. This is narrower than `mail.google.com` (full access) but broader than `gmail.readonly` (can't mark as read).

**Why**: Single scope covers all v1 needs. `gmail.readonly` is insufficient for mark-read.

### 4. Output format: structured text for skill parsing

```
# devpilot gmail list --unread
ID          FROM                 SUBJECT                          DATE
18df3a...   alice@example.com    Q1 Report Review                 2024-01-15 09:30
18df3b...   bob@example.com      Meeting Tomorrow                 2024-01-15 08:15

# devpilot gmail read 18df3a
From: alice@example.com
Subject: Q1 Report Review
Date: 2024-01-15 09:30

Hi, please review the attached Q1 report...
```

**Why**: Plain text is easy for Claude to parse in a skill context. No JSON output needed for v1.

### 5. Token refresh on 401

The Gmail client checks token expiry before each request. If expired, it calls `auth.RefreshToken()`, saves the new token, and retries. If refresh fails, it returns an error telling the user to `devpilot login gmail` again.

### 6. Google OAuth credentials in `internal/gmail/`

```go
// internal/gmail/service.go
const (
    gmailClientID     = "xxxx.apps.googleusercontent.com"
    gmailClientSecret = "GOCSPX-xxxx"
    gmailAuthURL      = "https://accounts.google.com/o/oauth2/v2/auth"
    gmailTokenURL     = "https://oauth2.googleapis.com/token"
)
```

Provider-specific constants live in the domain package, not in `internal/auth/`. The `Login()` method constructs an `auth.OAuthConfig` from these and passes it to `auth.StartFlow()`. The Google Cloud project is owned and maintained as part of the devpilot project.

## Risks / Trade-offs

- **[Google OAuth app review]** → To avoid "unverified app" warnings and 7-day token expiry, the OAuth app needs Google verification. Mitigation: apply for verification once the app is stable; users can bypass the warning during testing.
- **[API rate limits]** → Gmail API has a per-user limit of 250 quota units per second. Mitigation: the email-assistant use case is low-volume (daily digest); unlikely to hit limits.
- **[Plain text extraction]** → Some emails are HTML-only with no text/plain part. Mitigation: strip HTML tags as fallback; perfect rendering is a non-goal.
