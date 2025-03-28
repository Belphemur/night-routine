package handlers

import (
	"net/http"
)

// HomeHandler manages home page functionality
type HomeHandler struct {
	*BaseHandler
}

// NewHomeHandler creates a new home page handler
func NewHomeHandler(baseHandler *BaseHandler) *HomeHandler {
	return &HomeHandler{
		BaseHandler: baseHandler,
	}
}

// RegisterRoutes registers home page related routes
func (h *HomeHandler) RegisterRoutes() {
	http.HandleFunc("/", h.handleHome)
}

// HomePageData contains data for the home page template
type HomePageData struct {
	IsAuthenticated bool
	CalendarID      string
	ErrorMessage    string
	SuccessMessage  string
}

// handleHome shows the main page with auth status
func (h *HomeHandler) handleHome(w http.ResponseWriter, r *http.Request) {
	// Use TokenManager to check if we have a valid token
	hasToken, err := h.TokenManager.HasToken()
	if err != nil {
		http.Error(w, "Failed to check auth status", http.StatusInternalServerError)
		return
	}

	isAuthenticated := false
	if hasToken {
		// Only attempt to get a valid token if we know one exists
		token, err := h.TokenManager.GetValidToken(r.Context())
		if err == nil && token != nil && token.Valid() {
			isAuthenticated = true
		}
	}

	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil {
		http.Error(w, "Failed to get selected calendar", http.StatusInternalServerError)
		return
	}

	// Get error and success messages from query parameters if any
	errorParam := r.URL.Query().Get("error")
	successParam := r.URL.Query().Get("success")

	var errorMessage, successMessage string
	if errorParam == "calendar_client_error" {
		errorMessage = "Failed to connect to Google Calendar. Please try authenticating again."
	} else if errorParam == "calendar_fetch_error" {
		errorMessage = "Failed to fetch your calendars. Please try authenticating again."
	} else if errorParam == "sync_error" {
		errorMessage = "Failed to sync schedule. Please try again."
	} else if errorParam == "authentication_required" {
		errorMessage = "Authentication required. Please connect your Google Calendar first."
	} else if errorParam == "calendar_selection_required" {
		errorMessage = "Please select a calendar first."
	}

	if successParam == "sync_complete" {
		successMessage = "Schedule successfully synced with Google Calendar."
	}

	data := HomePageData{
		IsAuthenticated: isAuthenticated,
		CalendarID:      calendarID,
		ErrorMessage:    errorMessage,
		SuccessMessage:  successMessage,
	}

	h.RenderTemplate(w, "home.html", data)
}
