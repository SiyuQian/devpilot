package auth

import "errors"

// Sentinel errors returned by the OAuth flow helpers.
var (
	// ErrTokenExpired indicates the access token has expired and must be
	// refreshed (or re-obtained) before use.
	ErrTokenExpired = errors.New("oauth: access token has expired")
	// ErrReauthRequired indicates the refresh token is no longer valid and
	// the user must re-authorize the application.
	ErrReauthRequired = errors.New("oauth: re-authorization required (refresh token expired or revoked)")
	// ErrAuthDenied indicates the user explicitly denied the authorization
	// request at the provider's consent screen.
	ErrAuthDenied = errors.New("oauth: user denied authorization")
)
