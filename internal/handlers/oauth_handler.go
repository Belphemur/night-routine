package handlers

import (
	"net/http"

	"github.com/belphemur/night-routine/internal/logging"
	"golang.org/x/oauth2"
)

// OAuthHandler manages OAuth2 authentication and token storage
type OAuthHandler struct {
	*BaseHandler // Embed BaseHandler
	OAuthConfig  *oauth2.Config
}

// NewOAuthHandler creates a new OAuth handler using the BaseHandler
func NewOAuthHandler(baseHandler *BaseHandler) (*OAuthHandler, error) {
	// Logger is inherited from BaseHandler
	baseHandler.logger.Debug().Msg("Initializing OAuth handler")

	// OAuthConfig is derived from the Config within BaseHandler
	oauthConfig := baseHandler.Config.OAuth

	return &OAuthHandler{
		BaseHandler: baseHandler,
		OAuthConfig: oauthConfig,
	}, nil
}

// RegisterRoutes registers the OAuth routes
func (h *OAuthHandler) RegisterRoutes() {
	http.HandleFunc("/auth", h.handleAuth)
	http.HandleFunc("/oauth/callback", h.handleCallback)
}

// handleAuth initiates the OAuth flow
func (h *OAuthHandler) handleAuth(w http.ResponseWriter, r *http.Request) {
	// Use logger from embedded BaseHandler
	handlerLogger := h.logger.With().Str("handler", "handleAuth").Logger()
	handlerLogger.Info().Msg("Initiating OAuth flow")
	// Consider adding state generation and validation for security
	state := "pseudo-random-state" // Replace with actual random state generation
	// Use OAuthConfig from the struct
	url := h.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce) // Force approval prompt
	handlerLogger.Debug().Str("redirect_url", url).Msg("Redirecting user to Google for authentication")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// handleCallback processes the OAuth callback
func (h *OAuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Use logger from embedded BaseHandler
	handlerLogger := h.logger.With().Str("handler", "handleCallback").Logger()
	handlerLogger.Info().Msg("Handling OAuth callback")

	// Add state validation here
	// state := r.URL.Query().Get("state")
	// if state != "pseudo-random-state" { // Compare with stored state
	// 	handlerLogger.Error().Str("received_state", state).Msg("Invalid OAuth state")
	// 	http.Error(w, "Invalid state", http.StatusBadRequest)
	// 	return
	// }

	code := r.URL.Query().Get("code")
	if code == "" {
		handlerLogger.Error().Msg("No authorization code received in callback")
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}
	handlerLogger.Debug().Msg("Authorization code received")

	handlerLogger.Debug().Msg("Exchanging authorization code for token")
	// Use OAuthConfig from the struct
	token, err := h.OAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Token exchange failed")
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	handlerLogger.Info().Msg("Token exchange successful")

	// Use TokenManager from embedded BaseHandler
	handlerLogger.Debug().Msg("Saving token using TokenManager")
	if err := h.TokenManager.SaveToken(r.Context(), token); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save token")
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		return
	}
	handlerLogger.Info().Msg("Token saved successfully")

	// Redirect to calendar selection page
	handlerLogger.Debug().Msg("Redirecting to calendar selection page")
	http.Redirect(w, r, "/calendars", http.StatusSeeOther) // Use SeeOther for POST-redirect-GET
}
