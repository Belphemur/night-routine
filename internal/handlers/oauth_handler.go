package handlers

import (
	"html/template"
	"net/http"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

// OAuthHandler manages OAuth2 authentication and token storage
type OAuthHandler struct {
	OAuthConfig  *oauth2.Config
	Templates    *template.Template // Keep templates if needed specifically here, otherwise remove if BaseHandler is used
	TokenStore   *database.TokenStore
	TokenManager *token.TokenManager
	Config       *config.Config
	logger       zerolog.Logger
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(cfg *config.Config, tokenStore *database.TokenStore, tokenManager *token.TokenManager) (*OAuthHandler, error) {
	logger := logging.GetLogger("oauth-handler")

	// Parse templates here instead of init()
	logger.Debug().Msg("Parsing templates for OAuth handler")
	tmpl, err := template.New("").ParseGlob("internal/handlers/templates/*.html")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse templates")
		// Return error instead of Fatalf
		return nil, err
	}
	logger.Debug().Msg("Templates parsed successfully")

	return &OAuthHandler{
		OAuthConfig:  cfg.OAuth,
		Templates:    tmpl, // Assign parsed templates
		TokenStore:   tokenStore,
		TokenManager: tokenManager,
		Config:       cfg,
		logger:       logger,
	}, nil
}

// RegisterRoutes registers the OAuth routes
func (h *OAuthHandler) RegisterRoutes() {
	http.HandleFunc("/auth", h.handleAuth)
	http.HandleFunc("/oauth/callback", h.handleCallback)
}

// handleAuth initiates the OAuth flow
func (h *OAuthHandler) handleAuth(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleAuth").Logger()
	handlerLogger.Info().Msg("Initiating OAuth flow")
	// Consider adding state generation and validation for security
	state := "pseudo-random-state" // Replace with actual random state generation
	url := h.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce) // Force approval prompt
	handlerLogger.Debug().Str("redirect_url", url).Msg("Redirecting user to Google for authentication")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// handleCallback processes the OAuth callback
func (h *OAuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
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
	token, err := h.OAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Token exchange failed")
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	handlerLogger.Info().Msg("Token exchange successful")

	// Use TokenManager's SaveToken method which emits a signal, passing the request context
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
