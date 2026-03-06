## 1. Gmail Auth Service

- [ ] 1.1 Create `internal/gmail/service.go` implementing `auth.Service` interface (Name, Login, Logout, IsLoggedIn)
- [ ] 1.2 Define Gmail-specific OAuth constants in `internal/gmail/service.go` (client ID, client secret, auth URL, token URL, scopes) — these live in the domain package, not in `internal/auth/`
- [ ] 1.3 Implement `Login()`: construct `auth.OAuthConfig` from Gmail constants, call `auth.StartFlow()`, save tokens via `auth.Save()`
- [ ] 1.4 Register Gmail service in `internal/auth/service.go` init()

## 2. Gmail API Client

- [ ] 2.1 Create `internal/gmail/client.go` with `NewClient(accessToken string, opts ...Option)` and functional options (WithBaseURL, WithHTTPClient)
- [ ] 2.2 Implement `ListMessages(query string, limit int)` — calls `GET /gmail/v1/users/me/messages` with query param
- [ ] 2.3 Implement `GetMessage(id string)` — calls `GET /gmail/v1/users/me/messages/{id}` with format=full
- [ ] 2.4 Implement `BatchModify(ids []string, removeLabelIds []string)` — calls `POST /gmail/v1/users/me/messages/batchModify`
- [ ] 2.5 Add automatic token refresh: check expiry before requests, call `auth.RefreshToken()` on 401, save new tokens, retry

## 3. Email Parsing

- [ ] 3.1 Parse message list response to extract message IDs and thread IDs
- [ ] 3.2 Parse full message response to extract From, Subject, Date headers
- [ ] 3.3 Extract plain text body from message payload (handle multipart/alternative, text/plain parts)
- [ ] 3.4 Add HTML-to-text fallback for emails with only text/html parts (strip tags)

## 4. CLI Commands

- [ ] 4.1 Create `internal/gmail/commands.go` with `gmail` parent command and register in root
- [ ] 4.2 Implement `devpilot gmail list --unread [--after DATE] [--limit N]` — builds Gmail search query, calls ListMessages, formats table output
- [ ] 4.3 Implement `devpilot gmail read <message-id>` — calls GetMessage, formats header + body output
- [ ] 4.4 Implement `devpilot gmail mark-read <id>...` — calls BatchModify to remove UNREAD label
- [ ] 4.5 Add login check to all gmail subcommands with helpful error message

## 5. Tests

- [ ] 5.1 Unit test Gmail service Login/Logout/IsLoggedIn with mock OAuth flow
- [ ] 5.2 Unit test client ListMessages/GetMessage/BatchModify with httptest mock server
- [ ] 5.3 Unit test email parsing: multipart messages, HTML-only fallback, header extraction
- [ ] 5.4 Unit test CLI command output formatting
