# internal/token

OAuth2 token lifecycle management with automatic refresh.

## Purpose

Abstracts Google OAuth2 token storage, retrieval, and automatic refresh. Emits signals when token state changes so other components (e.g., calendar service) can react.

## Key Types

- `TokenManager` — Manages token lifecycle using `TokenStore` + `oauth2.Config`.

## Key Methods

| Method | Purpose |
|--------|---------|
| `HasToken() (bool, error)` | Check if a token exists in storage |
| `GetValidToken(ctx) (*oauth2.Token, error)` | Get current token, auto-refreshing if expired |
| `SaveToken(ctx, token) error` | Persist token and emit `TokenSetup` signal |
| `ClearToken(ctx) error` | Delete token and emit `TokenSetup` signal |

## Signal Integration

- Emits `signals.TokenSetup` when token is saved or cleared.
- This triggers calendar service initialization in `main.go`.

## Dependencies

- Uses: `internal/database` (TokenStore), `internal/signals`
- Used by: `cmd/night-routine`, `internal/handlers/oauth_handler`, `internal/calendar`
