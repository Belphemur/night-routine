package handlers

import (
	"net/http"
	"strings"
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

// CalendarDayJSON represents a calendar day in JSON format for client-side use
type CalendarDayJSON struct {
	DateStr          string `json:"dateStr"`
	DayOfMonth       int    `json:"dayOfMonth"`
	IsCurrentMonth   bool   `json:"isCurrentMonth"`
	AssignmentID     int64  `json:"assignmentId,omitempty"`
	AssignmentParent string `json:"assignmentParent,omitempty"`
	AssignmentReason string `json:"assignmentReason,omitempty"`
	IsOverridden     bool   `json:"isOverridden"`
	CSSClasses       string `json:"cssClasses"`
}

// MobileCalendarData contains the flattened calendar data and boundaries
type MobileCalendarData struct {
	Days      []CalendarDayJSON `json:"days"`
	StartDate string            `json:"startDate"`
	EndDate   string            `json:"endDate"`
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
	CalendarData   MobileCalendarData // Flattened calendar data for mobile view with boundaries
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
			data.CalendarData = h.flattenCalendarData(calendarWeeks)
		}
	}

	handlerLogger.Debug().Msg("Rendering home template")
	h.RenderTemplate(w, "home.html", data)
}

// flattenCalendarData converts CalendarWeeks to a MobileCalendarData struct for mobile view
func (h *HomeHandler) flattenCalendarData(weeks [][]viewhelpers.CalendarDay) MobileCalendarData {
	var days []CalendarDayJSON
	var startDate, endDate string

	if len(weeks) > 0 {
		if len(weeks[0]) > 0 {
			startDate = weeks[0][0].Date.Format("2006-01-02")
		}
		lastWeek := weeks[len(weeks)-1]
		if len(lastWeek) > 0 {
			endDate = lastWeek[len(lastWeek)-1].Date.Format("2006-01-02")
		}
	}

	for _, week := range weeks {
		for _, day := range week {
			dayJSON := CalendarDayJSON{
				DateStr:        day.Date.Format("2006-01-02"),
				DayOfMonth:     day.DayOfMonth,
				IsCurrentMonth: day.IsCurrentMonth,
			}

			// Build base CSS classes shared by all days
			baseClasses := []string{"border", "border-slate-200", "text-center", "align-top", "relative"}
			if day.IsCurrentMonth {
				baseClasses = append(baseClasses, "bg-white", "hover:shadow-lg")
			} else {
				baseClasses = append(baseClasses, "bg-slate-50", "text-slate-400")
			}

			if day.Assignment != nil {
				dayJSON.AssignmentID = day.Assignment.ID
				dayJSON.AssignmentParent = day.Assignment.Parent
				dayJSON.AssignmentReason = string(day.Assignment.DecisionReason)
				dayJSON.IsOverridden = day.Assignment.DecisionReason == "Override"

				// Add assignment-specific classes
				classes := append(baseClasses, "cursor-pointer", "transition-all", "duration-200")

				if day.Assignment.ParentType.String() == "ParentA" {
					classes = append(classes, "bg-linear-to-br", "from-blue-50", "to-indigo-100", "text-indigo-900", "border-indigo-200", "hover:from-blue-100", "hover:to-indigo-200")
				} else if day.Assignment.ParentType.String() == "ParentB" {
					classes = append(classes, "bg-linear-to-br", "from-amber-50", "to-orange-100", "text-orange-900", "border-orange-200", "hover:from-amber-100", "hover:to-orange-200")
				}

				if dayJSON.IsOverridden {
					classes = append(classes, "overridden")
				}

				dayJSON.CSSClasses = strings.Join(classes, " ")
			} else {
				// No assignment - use base classes only
				dayJSON.CSSClasses = strings.Join(baseClasses, " ")
			}

			days = append(days, dayJSON)
		}
	}

	return MobileCalendarData{
		Days:      days,
		StartDate: startDate,
		EndDate:   endDate,
	}
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
