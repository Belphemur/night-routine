package token

import (
	"context"
	"fmt"

	"github.com/belphemur/night-routine/internal/database"
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
