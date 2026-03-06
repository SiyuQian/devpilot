## Why

DevPilot needs to read, summarize, and manage Gmail messages as part of the email-assistant workflow. Adding Gmail as a first-class service enables `devpilot login gmail` (OAuth browser flow) and CLI commands for listing, reading, and marking emails — the same pattern as Trello integration.

## What Changes

- Add Gmail as an auth service implementing `auth.Service`, using the shared OAuth browser flow from `oauth-browser-flow` change
- Embed Google Cloud OAuth client ID/secret in the binary (devpilot-managed app)
- Add `devpilot gmail list` command to list unread emails with filtering options
- Add `devpilot gmail read` command to read a specific email's content
- Add `devpilot gmail mark-read` command to mark one or more emails as read
- Register Gmail service in auth registry so `devpilot login gmail`, `devpilot logout gmail`, and `devpilot status` work automatically

## Capabilities

### New Capabilities
- `gmail-auth`: Gmail OAuth authentication with scopes for reading and modifying emails (gmail.readonly, gmail.modify). Implements auth.Service interface.
- `gmail-commands`: CLI commands for listing unread emails, reading email content, and marking emails as read via the Gmail REST API.

### Modified Capabilities

## Impact

- `internal/gmail/` — new package with client, service, and commands
- `internal/auth/service.go` — register Gmail service in `init()`
- Depends on `oauth-browser-flow` change for OAuth login flow
- New external dependency: Gmail REST API (https://gmail.googleapis.com/gmail/v1/)
- Requires devpilot Google Cloud project with Gmail API enabled
