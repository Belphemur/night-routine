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

// HomePageData contains data for the home page template
type HomePageData struct {
	BasePageData
	CalendarID     string
	CalendarName   string
	ErrorMessage   string
	SuccessMessage string
	CurrentMonth   string
	CalendarWeeks  [][]viewhelpers.CalendarDay
	CalendarData   []CalendarDayJSON // Flattened calendar data for mobile view
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

// flattenCalendarData converts CalendarWeeks to a flat array of CalendarDayJSON for mobile view
func (h *HomeHandler) flattenCalendarData(weeks [][]viewhelpers.CalendarDay) []CalendarDayJSON {
	var result []CalendarDayJSON

	for _, week := range weeks {
		for _, day := range week {
			dayJSON := CalendarDayJSON{
				DateStr:        day.Date.Format("2006-01-02"),
				DayOfMonth:     day.DayOfMonth,
				IsCurrentMonth: day.IsCurrentMonth,
			}

			if day.Assignment != nil {
				dayJSON.AssignmentID = day.Assignment.ID
				dayJSON.AssignmentParent = day.Assignment.Parent
				dayJSON.AssignmentReason = string(day.Assignment.DecisionReason)
				dayJSON.IsOverridden = day.Assignment.DecisionReason == "Override"

				// Build CSS classes based on assignment
				var classes []string
				classes = append(classes, "border", "border-slate-200", "text-center", "align-top", "relative", "cursor-pointer", "transition-all", "duration-200")

				if day.IsCurrentMonth {
					classes = append(classes, "bg-white", "hover:shadow-lg")
				} else {
					classes = append(classes, "bg-slate-50", "text-slate-400")
				}

				if day.Assignment.ParentType.String() == "ParentA" {
					classes = append(classes, "bg-gradient-to-br", "from-blue-50", "to-indigo-100", "text-indigo-900", "border-indigo-200", "hover:from-blue-100", "hover:to-indigo-200")
				} else if day.Assignment.ParentType.String() == "ParentB" {
					classes = append(classes, "bg-gradient-to-br", "from-amber-50", "to-orange-100", "text-orange-900", "border-orange-200", "hover:from-amber-100", "hover:to-orange-200")
				}

				if dayJSON.IsOverridden {
					classes = append(classes, "overridden")
				}

				dayJSON.CSSClasses = joinClasses(classes)
			} else {
				// No assignment
				var classes []string
				classes = append(classes, "border", "border-slate-200", "text-center", "align-top", "relative")

				if day.IsCurrentMonth {
					classes = append(classes, "bg-white", "hover:shadow-lg")
				} else {
					classes = append(classes, "bg-slate-50", "text-slate-400")
				}

				dayJSON.CSSClasses = joinClasses(classes)
			}

			result = append(result, dayJSON)
		}
	}

	return result
}

// joinClasses joins CSS class names with spaces
func joinClasses(classes []string) string {
	result := ""
	for i, class := range classes {
		if i > 0 {
			result += " "
		}
		result += class
	}
	return result
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
