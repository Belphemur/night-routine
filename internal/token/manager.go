package token

import (
	"context"
	"fmt"

	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/signals"
	"golang.org/x/oauth2"
)

// TokenManager handles OAuth token storage and refreshing
type TokenManager struct {
	tokenStore  *database.TokenStore
	oauthConfig *oauth2.Config
}

// NewTokenManager creates a new TokenManager
func NewTokenManager(tokenStore *database.TokenStore, oauthConfig *oauth2.Config) *TokenManager {
	return &TokenManager{
		tokenStore:  tokenStore,
		oauthConfig: oauthConfig,
	}
}

// HasToken checks if a token exists in the store without validating it
func (tm *TokenManager) HasToken() (bool, error) {
	token, err := tm.tokenStore.GetToken()
	if err != nil {
		return false, fmt.Errorf("failed to retrieve token: %w", err)
	}
	return token != nil, nil
}

// GetValidToken retrieves a valid token, refreshing it if necessary
func (tm *TokenManager) GetValidToken(ctx context.Context) (*oauth2.Token, error) {
	token, err := tm.tokenStore.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	if token == nil {
		return nil, fmt.Errorf("no token found")
	}

	if !token.Valid() {
		newToken, err := tm.oauthConfig.TokenSource(ctx, token).Token()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		if err := tm.tokenStore.SaveToken(newToken); err != nil {
			return nil, fmt.Errorf("failed to save refreshed token: %w", err)
		}

		token = newToken
	}

	return token, nil
}

// SaveToken saves a token to the store and emits a signal
func (tm *TokenManager) SaveToken(ctx context.Context, token *oauth2.Token) error {
	if err := tm.tokenStore.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	// Emit token setup signal with the updated context
	signals.EmitTokenSetup(ctx, true)

	return nil
}
