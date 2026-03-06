package auth

import "errors"

var (
	ErrTokenExpired   = errors.New("oauth: access token has expired")
	ErrReauthRequired = errors.New("oauth: re-authorization required (refresh token expired or revoked)")
	ErrAuthDenied     = errors.New("oauth: user denied authorization")
)
