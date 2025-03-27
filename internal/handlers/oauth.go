package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/belphemur/night-routine/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// OAuthHandler manages OAuth2 authentication and token storage
type OAuthHandler struct {
	oauthConf  *oauth2.Config
	templates  *template.Template
	tokenStore *TokenStore
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(config *config.Config, tokenStore *TokenStore) (*OAuthHandler, error) {
	oauthConf := &oauth2.Config{
		ClientID:     config.OAuth.ClientID,
		ClientSecret: config.OAuth.ClientSecret,
		RedirectURL:  config.OAuth.RedirectURL,
		Scopes: []string{
			calendar.CalendarEventsScope,
			calendar.CalendarCalendarlistReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}

	tmpl, err := template.New("").ParseGlob("internal/handlers/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &OAuthHandler{
		oauthConf:  oauthConf,
		templates:  tmpl,
		tokenStore: tokenStore,
	}, nil
}

// RegisterHandlers registers the OAuth routes
func (h *OAuthHandler) RegisterHandlers() {
	http.HandleFunc("/", h.handleHome)
	http.HandleFunc("/auth", h.handleAuth)
	http.HandleFunc("/oauth/callback", h.handleCallback)
	http.HandleFunc("/calendars", h.handleCalendarList)
}

// handleHome shows the main page with auth status
func (h *OAuthHandler) handleHome(w http.ResponseWriter, r *http.Request) {
	token, err := h.tokenStore.GetToken()
	if err != nil {
		http.Error(w, "Failed to check auth status", http.StatusInternalServerError)
		return
	}

	calendarID, err := h.tokenStore.GetSelectedCalendar()
	if err != nil {
		http.Error(w, "Failed to get selected calendar", http.StatusInternalServerError)
		return
	}

	// Get error message from query parameter if any
	errorParam := r.URL.Query().Get("error")
	var errorMessage string
	
	if errorParam == "calendar_client_error" {
		errorMessage = "Failed to connect to Google Calendar. Please try authenticating again."
	} else if errorParam == "calendar_fetch_error" {
		errorMessage = "Failed to fetch your calendars. Please try authenticating again."
	}

	data := struct {
		IsAuthenticated bool
		CalendarID      string
		ErrorMessage    string
	}{
		IsAuthenticated: token != nil && token.Valid(),
		CalendarID:      calendarID,
		ErrorMessage:    errorMessage,
	}

	if err := h.templates.ExecuteTemplate(w, "home.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// handleAuth initiates the OAuth flow
func (h *OAuthHandler) handleAuth(w http.ResponseWriter, r *http.Request) {
	url := h.oauthConf.AuthCodeURL("state")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// handleCallback processes the OAuth callback
func (h *OAuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := h.oauthConf.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("Token exchange error: %v", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	if err := h.tokenStore.SaveToken(token); err != nil {
		log.Printf("Token save error: %v", err)
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		return
	}

	// Redirect to calendar selection page
	http.Redirect(w, r, "/calendars", http.StatusTemporaryRedirect)
}

// handleCalendarList shows available calendars and allows selection
func (h *OAuthHandler) handleCalendarList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.handleCalendarSelection(w, r)
		return
	}

	token, err := h.tokenStore.GetToken()
	if err != nil {
		http.Error(w, "Failed to get token", http.StatusInternalServerError)
		return
	}

	client := h.oauthConf.Client(r.Context(), token)
	calendarService, err := calendar.NewService(r.Context(), option.WithHTTPClient(client))
	if err != nil {
		log.Printf("Failed to create calendar client: %v", err)
		if clearErr := h.tokenStore.ClearToken(); clearErr != nil {
			log.Printf("Failed to clear token: %v", clearErr)
		}
		http.Redirect(w, r, "/?error=calendar_client_error", http.StatusSeeOther)
		return
	}

	calendars, err := calendarService.CalendarList.List().Do()
	if err != nil {
		log.Printf("Failed to fetch calendars: %v", err)
		if clearErr := h.tokenStore.ClearToken(); clearErr != nil {
			log.Printf("Failed to clear token: %v", clearErr)
		}
		http.Redirect(w, r, "/?error=calendar_fetch_error", http.StatusSeeOther)
		return
	}

	selected, err := h.tokenStore.GetSelectedCalendar()
	if err != nil {
		http.Error(w, "Failed to get selected calendar", http.StatusInternalServerError)
		return
	}

	data := struct {
		Calendars *calendar.CalendarList
		Selected  string
	}{
		Calendars: calendars,
		Selected:  selected,
	}

	if err := h.templates.ExecuteTemplate(w, "oauth.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// handleCalendarSelection processes calendar selection
func (h *OAuthHandler) handleCalendarSelection(w http.ResponseWriter, r *http.Request) {
	calendarID := r.FormValue("calendar_id")
	if calendarID == "" {
		http.Error(w, "No calendar selected", http.StatusBadRequest)
		return
	}

	if err := h.tokenStore.SaveSelectedCalendar(calendarID); err != nil {
		http.Error(w, "Failed to save calendar selection", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
