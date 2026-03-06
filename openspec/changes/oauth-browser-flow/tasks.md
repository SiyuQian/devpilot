## 1. OAuth Config, Token Types, and Error Types

- [x] 1.1 Create `internal/auth/oauth.go` with `OAuthConfig` struct (ProviderName, AuthURL, TokenURL, ClientID, ClientSecret, Scopes, RedirectPort) and `OAuthToken` struct (AccessToken, RefreshToken, Expiry, TokenType)
- [x] 1.2 Add helper functions to convert between `OAuthToken` and `ServiceCredentials` map (SaveOAuthToken, LoadOAuthToken)
- [x] 1.3 Create `internal/auth/errors.go` with shared OAuth error types (`ErrTokenExpired`, `ErrReauthRequired`, `ErrAuthDenied`)

## 2. Local Callback Server

- [x] 2.1 Implement local HTTP server that binds to a random available port on localhost with `/callback` handler
- [x] 2.2 Add CSRF state parameter generation (crypto/rand) and validation on callback
- [x] 2.3 Add port bind retry logic (up to 3 attempts with different random ports)
- [x] 2.4 Add graceful server shutdown after flow completes (success or failure)

## 3. Browser Opening

- [x] 3.1 Implement platform-aware browser opening (`open` on macOS, `xdg-open` on Linux)
- [x] 3.2 Add fallback: print authorization URL to stdout if browser open fails

## 4. Token Exchange

- [x] 4.1 Implement authorization code → token exchange via POST to token URL
- [x] 4.2 Parse token response JSON into `OAuthToken` (handle both `expires_in` and `expiry` fields)
- [x] 4.3 Handle error responses from token endpoint (invalid_grant, access_denied, etc.)

## 5. Token Refresh

- [x] 5.1 Implement `RefreshToken(OAuthConfig, refreshToken)` function that POSTs with grant_type=refresh_token
- [x] 5.2 Handle refresh failure (expired/revoked refresh token) with clear error message indicating re-auth needed

## 6. StartFlow Orchestration

- [x] 6.1 Implement `StartFlow(OAuthConfig)` that orchestrates: start server → build auth URL → open browser → wait for callback → exchange code → shutdown server → return token
- [x] 6.2 Add 2-minute timeout for the entire flow
- [x] 6.3 Handle user-denied authorization (error parameter in callback)

## 7. Tests

- [x] 7.1 Unit test CSRF state generation and validation
- [x] 7.2 Unit test token exchange with mock HTTP server
- [x] 7.3 Unit test token refresh with mock HTTP server
- [x] 7.4 Unit test OAuthToken ↔ ServiceCredentials conversion
- [x] 7.5 Unit test callback server error handling (invalid state, denied auth, timeout)
