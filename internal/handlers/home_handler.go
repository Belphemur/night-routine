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
	// Logger is inherited from BaseHandler
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
	handlerLogger := h.logger.With().Str("handler", "handleHome").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling home page request")

	// Use TokenManager to check if we have a valid token
	handlerLogger.Debug().Msg("Checking token existence")
	hasToken, err := h.TokenManager.HasToken()
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to check token existence")
		http.Error(w, "Failed to check auth status", http.StatusInternalServerError)
		return
	}
	handlerLogger.Debug().Bool("has_token", hasToken).Msg("Token existence check complete")

	isAuthenticated := false
	if hasToken {
		// Only attempt to get a valid token if we know one exists
		handlerLogger.Debug().Msg("Attempting to validate existing token")
		token, err := h.TokenManager.GetValidToken(r.Context())
		// We only consider authenticated if we get a valid, non-nil token without error
		if err == nil && token != nil && token.Valid() {
			isAuthenticated = true
			handlerLogger.Debug().Msg("Token is valid")
		} else if err != nil {
			// Log error if getting token failed, but don't fail the request
			handlerLogger.Warn().Err(err).Msg("Failed to get/validate token, treating as unauthenticated")
		} else {
			// Token might be nil or invalid
			handlerLogger.Debug().Msg("Token is nil or invalid, treating as unauthenticated")
		}
	}

	handlerLogger.Debug().Msg("Fetching selected calendar ID")
	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil {
		// Log error but don't necessarily fail the request, page might still be useful
		handlerLogger.Error().Err(err).Msg("Failed to get selected calendar ID from store")
		// Consider if this should be a hard error or just logged
		// http.Error(w, "Failed to get selected calendar", http.StatusInternalServerError)
		// return
		calendarID = "" // Ensure calendarID is empty if fetch fails
	}
	handlerLogger.Debug().Str("selected_calendar_id", calendarID).Msg("Selected calendar ID fetched")

	// Get error and success messages from query parameters if any
	errorParam := r.URL.Query().Get("error")
	successParam := r.URL.Query().Get("success")
	handlerLogger.Debug().Str("error_param", errorParam).Str("success_param", successParam).Msg("Checked query parameters")

	var errorMessage, successMessage string
	if errorParam != "" {
		switch errorParam {
		case "calendar_client_error":
			errorMessage = "Failed to connect to Google Calendar. Please try authenticating again."
		case "calendar_fetch_error":
			errorMessage = "Failed to fetch your calendars. Please try authenticating again."
		case "sync_error":
			errorMessage = "Failed to sync schedule. Please try again."
		case "authentication_required":
			errorMessage = "Authentication required. Please connect your Google Calendar first."
		case "calendar_selection_required":
			errorMessage = "Please select a calendar first."
		default:
			errorMessage = "An unknown error occurred." // Handle unknown errors
		}
		handlerLogger.Warn().Str("error_code", errorParam).Str("error_message", errorMessage).Msg("Displaying error message")
	}

	if successParam == "sync_complete" {
		successMessage = "Schedule successfully synced with Google Calendar."
		handlerLogger.Info().Str("success_code", successParam).Str("success_message", successMessage).Msg("Displaying success message")
	}

	data := HomePageData{
		IsAuthenticated: isAuthenticated,
		CalendarID:      calendarID,
		ErrorMessage:    errorMessage,
		SuccessMessage:  successMessage,
	}

	handlerLogger.Debug().Msg("Rendering home template")
	h.RenderTemplate(w, "home.html", data)
}
