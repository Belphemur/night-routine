package handlers

import (
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
	BasePageData
	CalendarID     string
	CalendarName   string
	ErrorMessage   string
	SuccessMessage string
	CurrentMonth   string
	CalendarWeeks  [][]viewhelpers.CalendarDay
}

// handleHome shows the main page with auth status and potentially the calendar
func (h *HomeHandler) handleHome(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleHome").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling home page request")

	isAuthenticated := h.CheckAuthentication(r.Context(), handlerLogger)
	calendarID, calendarName := h.getSelectedCalendarInfo(handlerLogger)
	errorMessage, successMessage := h.processMessages(r, handlerLogger)

	data := HomePageData{
		BasePageData:   h.NewBasePageData(r, isAuthenticated),
		CalendarID:     calendarID,
		CalendarName:   calendarName,
		ErrorMessage:   errorMessage,
		SuccessMessage: successMessage,
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

// getSelectedCalendarInfo retrieves the currently selected Google Calendar ID and name.
func (h *HomeHandler) getSelectedCalendarInfo(logger zerolog.Logger) (string, string) {
	logger.Debug().Msg("Fetching selected calendar info")
	calendarID, calendarName, err := h.TokenStore.GetSelectedCalendarWithName()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get selected calendar info from store")
		return "", "" // Return empty strings if fetch fails
	}
	logger.Debug().Str("selected_calendar_id", calendarID).Str("selected_calendar_name", calendarName).Msg("Selected calendar info fetched")
	return calendarID, calendarName
}

// processMessages extracts and translates error/success codes from query parameters.
func (h *HomeHandler) processMessages(r *http.Request, logger zerolog.Logger) (errorMessage, successMessage string) {
	errorCode := r.URL.Query().Get("error")
	successCode := r.URL.Query().Get("success")
	logger.Debug().Str("error_code", errorCode).Str("success_code", successCode).Msg("Checked query parameters")

	if errorCode != "" {
		errorMessage = GetErrorMessage(errorCode)
		logger.Warn().Str("error_code", errorCode).Str("error_message", errorMessage).Msg("Processing error message")
	}

	if successCode != "" {
		successMessage = GetSuccessMessage(successCode)
		logger.Info().Str("success_code", successCode).Str("success_message", successMessage).Msg("Processing success message")
	}
	return errorMessage, successMessage
}

// generateCalendarData calculates the date range, generates the schedule, and structures it for the template.
func (h *HomeHandler) generateCalendarData(logger zerolog.Logger) (monthName string, weeks [][]viewhelpers.CalendarDay, err error) {
	logger.Debug().Msg("Generating calendar view data")
	refTime := time.Now()
	startDate, endDate := viewhelpers.CalculateCalendarRange(refTime)
	logger.Debug().Time("start_date", startDate).Time("end_date", endDate).Msg("Calculated calendar range")

	assignments, err := h.Scheduler.GenerateSchedule(startDate, endDate, time.Now())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate schedule for calendar view")
		return "", nil, err // Return error to the caller
	}

	logger.Debug().Int("assignment_count", len(assignments)).Msg("Successfully generated schedule")
	monthName, weeks = viewhelpers.StructureAssignmentsForTemplate(startDate, endDate, assignments)
	logger.Debug().Str("month_name", monthName).Int("week_count", len(weeks)).Msg("Structured calendar data for template")
	return monthName, weeks, nil
}
