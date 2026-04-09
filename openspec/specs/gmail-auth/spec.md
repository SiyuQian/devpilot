## Purpose

Defines Gmail authentication via OAuth 2.0, including login, logout, status reporting, and automatic token refresh.

## Requirements

### Requirement: Gmail OAuth login
The system SHALL authenticate with Gmail via OAuth 2.0 browser flow using the shared `auth.StartFlow` component with scope `gmail.modify`.

#### Scenario: Successful login
- **WHEN** user runs `devpilot login gmail`
- **THEN** the system opens the browser to Google OAuth consent, completes the flow, stores access_token and refresh_token, and prints "Logged in to Gmail."

#### Scenario: Login with existing credentials
- **WHEN** user runs `devpilot login gmail` while already logged in
- **THEN** the system SHALL overwrite existing credentials with the new tokens

### Requirement: Gmail logout
The system SHALL remove stored Gmail credentials when the user logs out.

#### Scenario: Successful logout
- **WHEN** user runs `devpilot logout gmail`
- **THEN** the system removes Gmail credentials from storage and prints "Logged out of Gmail."

### Requirement: Gmail login status
The system SHALL report Gmail authentication status through the existing `devpilot status` command.

#### Scenario: Status when logged in
- **WHEN** user runs `devpilot status` and Gmail credentials exist
- **THEN** the system SHALL show "gmail: logged in"

#### Scenario: Status when not logged in
- **WHEN** user runs `devpilot status` and no Gmail credentials exist
- **THEN** the system SHALL show "gmail: not logged in"

### Requirement: Automatic token refresh
The system SHALL automatically refresh the Gmail access token when it expires, using the stored refresh token.

#### Scenario: Transparent refresh on API call
- **WHEN** a Gmail API call detects the access token is expired
- **THEN** the system SHALL call `RefreshToken`, save the new tokens, and retry the API call

#### Scenario: Refresh token revoked
- **WHEN** token refresh fails with invalid_grant
- **THEN** the system SHALL return an error instructing the user to run `devpilot login gmail`
