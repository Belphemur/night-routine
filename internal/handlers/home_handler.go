package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/viewhelpers"
	"github.com/rs/zerolog"
)

// HomeHandler manages home page functionality
type HomeHandler struct {
	*BaseHandler
	Scheduler *scheduler.Scheduler
}

// NewHomeHandler creates a new home page handler
func NewHomeHandler(baseHandler *BaseHandler, scheduler *scheduler.Scheduler) *HomeHandler {
	return &HomeHandler{
		BaseHandler: baseHandler,
		Scheduler:   scheduler,
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
	CurrentMonth    string
	CalendarWeeks   [][]viewhelpers.CalendarDay
}

// handleHome shows the main page with auth status and potentially the calendar
func (h *HomeHandler) handleHome(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleHome").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling home page request")

	isAuthenticated := h.checkAuthentication(r.Context(), handlerLogger)
	calendarID := h.getSelectedCalendarID(handlerLogger)
	errorMessage, successMessage := h.processMessages(r, handlerLogger)

	data := HomePageData{
		IsAuthenticated: isAuthenticated,
		CalendarID:      calendarID,
		ErrorMessage:    errorMessage,
		SuccessMessage:  successMessage,
	}

	if isAuthenticated {
		calendarMonth, calendarWeeks, calendarErr := h.generateCalendarData(handlerLogger)
		if calendarErr != nil {
			// Use the existing error message mechanism if calendar generation fails
			data.ErrorMessage = "Error generating assignment calendar. Please try again later."
		} else {
			data.CurrentMonth = calendarMonth
			data.CalendarWeeks = calendarWeeks
		}
	}

	handlerLogger.Debug().Msg("Rendering home template")
	h.RenderTemplate(w, "home.html", data)
}

// checkAuthentication verifies if the user has a valid session token.
func (h *HomeHandler) checkAuthentication(ctx context.Context, logger zerolog.Logger) bool {
	logger.Debug().Msg("Checking token existence")
	hasToken, err := h.TokenManager.HasToken()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check token existence")
		// Treat as unauthenticated if check fails
		return false
	}
	if !hasToken {
		logger.Debug().Msg("No token found")
		return false
	}

	logger.Debug().Msg("Attempting to validate existing token")
	token, err := h.TokenManager.GetValidToken(ctx)
	if err == nil && token != nil && token.Valid() {
		logger.Debug().Msg("Token is valid")
		return true
	}

	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get/validate token, treating as unauthenticated")
	} else {
		logger.Debug().Msg("Token is nil or invalid, treating as unauthenticated")
	}
	return false
}

// getSelectedCalendarID retrieves the currently selected Google Calendar ID.
func (h *HomeHandler) getSelectedCalendarID(logger zerolog.Logger) string {
	logger.Debug().Msg("Fetching selected calendar ID")
	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get selected calendar ID from store")
		return "" // Return empty string if fetch fails
	}
	logger.Debug().Str("selected_calendar_id", calendarID).Msg("Selected calendar ID fetched")
	return calendarID
}

// processMessages extracts and translates error/success codes from query parameters.
func (h *HomeHandler) processMessages(r *http.Request, logger zerolog.Logger) (errorMessage, successMessage string) {
	errorParam := r.URL.Query().Get("error")
	successParam := r.URL.Query().Get("success")
	logger.Debug().Str("error_param", errorParam).Str("success_param", successParam).Msg("Checked query parameters")

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
		case "calendar_generation_error": // Kept this case for potential future use if redirecting on error
			errorMessage = "Failed to generate the assignment calendar. Please check logs or try again later."
		default:
			errorMessage = "An unknown error occurred."
		}
		logger.Warn().Str("error_code", errorParam).Str("error_message", errorMessage).Msg("Processing error message")
	}

	if successParam == "sync_complete" {
		successMessage = "Schedule successfully synced with Google Calendar."
		logger.Info().Str("success_code", successParam).Str("success_message", successMessage).Msg("Processing success message")
	}
	return errorMessage, successMessage
}

// generateCalendarData calculates the date range, generates the schedule, and structures it for the template.
func (h *HomeHandler) generateCalendarData(logger zerolog.Logger) (monthName string, weeks [][]viewhelpers.CalendarDay, err error) {
	logger.Debug().Msg("Generating calendar view data")
	refTime := time.Now()
	startDate, endDate := viewhelpers.CalculateCalendarRange(refTime)
	logger.Debug().Time("start_date", startDate).Time("end_date", endDate).Msg("Calculated calendar range")

	assignments, err := h.Scheduler.GenerateSchedule(startDate, endDate)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate schedule for calendar view")
		return "", nil, err // Return error to the caller
	}

	logger.Debug().Int("assignment_count", len(assignments)).Msg("Successfully generated schedule")
	monthName, weeks = viewhelpers.StructureAssignmentsForTemplate(startDate, endDate, assignments)
	logger.Debug().Str("month_name", monthName).Int("week_count", len(weeks)).Msg("Structured calendar data for template")
	return monthName, weeks, nil
}
