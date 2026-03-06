## ADDED Requirements

### Requirement: OAuth authorization code flow
The system SHALL provide a reusable `StartFlow(OAuthConfig)` function that executes the full OAuth 2.0 authorization code flow: start local callback server, open browser to authorization URL, receive callback with auth code, exchange code for tokens, and return the token set.

#### Scenario: Successful authorization
- **WHEN** a service calls `StartFlow` with a valid `OAuthConfig` (provider name, auth URL, token URL, client ID, client secret, scopes)
- **THEN** the system starts a local HTTP server on localhost, opens the user's default browser to the provider's authorization URL with correct query parameters (client_id, redirect_uri, response_type=code, scope, state), and waits for the callback

#### Scenario: Successful token exchange
- **WHEN** the provider redirects back to the local callback with a valid authorization code
- **THEN** the system exchanges the code for tokens via POST to the token URL and returns an `OAuthToken` containing access_token, refresh_token (if provided), expiry, and token_type

#### Scenario: Authorization denied by user
- **WHEN** the provider redirects back with an error parameter (e.g., access_denied)
- **THEN** the system SHALL return an error indicating the user denied authorization and shut down the local server

#### Scenario: Authorization timeout
- **WHEN** the user does not complete authorization within 2 minutes
- **THEN** the system SHALL return a timeout error and shut down the local server

### Requirement: CSRF protection via state parameter
The system SHALL generate a cryptographically random state parameter for each OAuth flow and validate it on callback to prevent CSRF attacks.

#### Scenario: Valid state parameter
- **WHEN** the callback includes a state parameter matching the one sent in the authorization request
- **THEN** the system SHALL proceed with token exchange

#### Scenario: Invalid state parameter
- **WHEN** the callback includes a state parameter that does not match
- **THEN** the system SHALL reject the callback with an error and NOT exchange the code

### Requirement: Local callback server
The system SHALL start a temporary HTTP server on `localhost` with a random available port to receive the OAuth callback.

#### Scenario: Server starts on available port
- **WHEN** `StartFlow` is called
- **THEN** the system binds to a random available port on localhost and uses `http://localhost:<port>/callback` as the redirect URI

#### Scenario: Port bind failure with retry
- **WHEN** the initial port bind fails
- **THEN** the system SHALL retry with a different random port up to 3 times before returning an error

#### Scenario: Server shuts down after flow completes
- **WHEN** the token exchange completes (success or failure)
- **THEN** the local HTTP server SHALL shut down gracefully

### Requirement: Browser opening
The system SHALL open the user's default browser to the authorization URL using platform-appropriate commands.

#### Scenario: macOS browser open
- **WHEN** the platform is macOS
- **THEN** the system SHALL use the `open` command to launch the browser

#### Scenario: Linux browser open
- **WHEN** the platform is Linux
- **THEN** the system SHALL use `xdg-open` to launch the browser

#### Scenario: Browser open failure
- **WHEN** the browser command fails
- **THEN** the system SHALL print the authorization URL to stdout so the user can manually open it, and continue waiting for the callback

### Requirement: Token refresh
The system SHALL provide a `RefreshToken(OAuthConfig, refreshToken)` function that exchanges a refresh token for a new access token.

#### Scenario: Successful token refresh
- **WHEN** `RefreshToken` is called with a valid refresh token
- **THEN** the system SHALL POST to the token URL with grant_type=refresh_token and return the new access token and expiry

#### Scenario: Refresh token expired or revoked
- **WHEN** the token endpoint returns an error (e.g., invalid_grant)
- **THEN** the system SHALL return an error indicating re-authorization is needed

### Requirement: Token storage via credentials system
OAuth tokens SHALL be stored and loaded through the existing `auth.Save()` and `auth.Load()` functions as `ServiceCredentials` map entries.

#### Scenario: Save OAuth tokens
- **WHEN** a service completes OAuth flow and receives tokens
- **THEN** the tokens (access_token, refresh_token, expiry, token_type) SHALL be saved via `auth.Save(serviceName, creds)`

#### Scenario: Load OAuth tokens
- **WHEN** a service needs to make an authenticated API call
- **THEN** it SHALL load tokens via `auth.Load(serviceName)` and check expiry before use
