package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// Global variables used in init()
var (
	templates  *template.Template
	oauthHandler *OAuthHandler
)

// OAuthHandler manages OAuth2 authentication and token storage
type OAuthHandler struct {
	OAuthConfig *oauth2.Config
	Templates   *template.Template
	TokenStore  *database.TokenStore
	Config      *config.Config
}

// init initializes the templates for all handlers
func init() {
	var err error
	templates, err = template.New("").ParseGlob("internal/handlers/templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(cfg *config.Config, tokenStore *database.TokenStore) (*OAuthHandler, error) {
	oauthConf := &oauth2.Config{
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		RedirectURL:  cfg.OAuth.RedirectURL,
		Scopes: []string{
			calendar.CalendarEventsScope,
			calendar.CalendarCalendarlistReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}

	return &OAuthHandler{
		OAuthConfig: oauthConf,
		Templates:   templates,
		TokenStore:  tokenStore,
		Config:      cfg,
	}, nil
}

// RegisterRoutes registers the OAuth routes
func (h *OAuthHandler) RegisterRoutes() {
	http.HandleFunc("/auth", h.handleAuth)
	http.HandleFunc("/oauth/callback", h.handleCallback)
}

// GetOAuthConfig returns the OAuth config for use by other handlers
func (h *OAuthHandler) GetOAuthConfig() *oauth2.Config {
	return h.OAuthConfig
}

// RenderTemplate is a helper method to render HTML templates
func (h *OAuthHandler) RenderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.Templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// handleAuth initiates the OAuth flow
func (h *OAuthHandler) handleAuth(w http.ResponseWriter, r *http.Request) {
	url := h.OAuthConfig.AuthCodeURL("state")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// handleCallback processes the OAuth callback
func (h *OAuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := h.OAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("Token exchange error: %v", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	if err := h.TokenStore.SaveToken(token); err != nil {
		log.Printf("Token save error: %v", err)
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		return
	}

	// Redirect to calendar selection page
	http.Redirect(w, r, "/calendars", http.StatusTemporaryRedirect)
}