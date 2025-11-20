package handlers

import (
	"net/http"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	gcal "google.golang.org/api/calendar/v3"
)

// CalendarHandler manages calendar selection functionality
type CalendarHandler struct {
	*BaseHandler
	CalendarManager *calendar.Manager
	RuntimeConfig   *config.RuntimeConfig
}

// NewCalendarHandler creates a new calendar handler
func NewCalendarHandler(baseHandler *BaseHandler, runtimeCfg *config.RuntimeConfig, calendarManager *calendar.Manager) *CalendarHandler {
	// Logger is inherited from BaseHandler
	return &CalendarHandler{
		BaseHandler:     baseHandler,
		CalendarManager: calendarManager,
		RuntimeConfig:   runtimeCfg,
	}
}

// RegisterRoutes registers calendar related routes
func (h *CalendarHandler) RegisterRoutes() {
	http.HandleFunc("/calendars", h.handleCalendarList)
}

// CalendarPageData contains data for the calendar selection page
type CalendarPageData struct {
	BasePageData
	Calendars *gcal.CalendarList
	Selected  string
	Error     string
}

// handleCalendarList shows available calendars and allows selection
func (h *CalendarHandler) handleCalendarList(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleCalendarList").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling calendar list request")

	if r.Method == http.MethodPost {
		handlerLogger.Debug().Msg("Request method is POST, delegating to handleCalendarSelection")
		h.handleCalendarSelection(w, r)
		return
	}

	handlerLogger.Debug().Msg("Fetching available calendars")
	// Get available calendars
	calendars, err := h.CalendarManager.GetCalendarList(r.Context())
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to fetch calendars")

		// If fetching calendars fails (likely auth issue), clear the token and redirect
		handlerLogger.Warn().Msg("Clearing token due to calendar fetch error")
		if tokenErr := h.TokenManager.ClearToken(r.Context()); tokenErr != nil {
			// Log this error but proceed with redirect
			handlerLogger.Error().Err(tokenErr).Msg("Failed to clear token after calendar fetch error")
		}

		http.Redirect(w, r, "/?error=calendar_fetch_error", http.StatusSeeOther)
		return
	}
	handlerLogger.Debug().Int("calendar_count", len(calendars.Items)).Msg("Successfully fetched calendars")

	// Get currently selected calendar
	handlerLogger.Debug().Msg("Fetching selected calendar")
	selected, err := h.CalendarManager.GetSelectedCalendar()
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get selected calendar")
		http.Error(w, "Failed to get selected calendar", http.StatusInternalServerError)
		return
	}
	handlerLogger.Debug().Str("selected_calendar", selected).Msg("Successfully fetched selected calendar")

	data := CalendarPageData{
		BasePageData: h.NewBasePageData(r, true), // Assuming authenticated if we got here
		Calendars:    calendars,
		Selected:     selected,
	}

	handlerLogger.Debug().Msg("Rendering calendar selection template")
	h.RenderTemplate(w, "calendars.html", data) // Assuming template name is calendars.html
}

// handleCalendarSelection processes calendar selection
func (h *CalendarHandler) handleCalendarSelection(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleCalendarSelection").Logger()
	handlerLogger.Info().Msg("Handling calendar selection POST request")

	if err := r.ParseForm(); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to parse form")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	calendarID := r.FormValue("calendar_id")
	handlerLogger = handlerLogger.With().Str("selected_calendar_id", calendarID).Logger() // Add selected ID to context
	if calendarID == "" {
		handlerLogger.Warn().Msg("No calendar_id provided in form")
		http.Error(w, "No calendar selected", http.StatusBadRequest)
		return
	}
	handlerLogger.Debug().Msg("Calendar ID received")

	// Use the calendar manager to select the calendar
	handlerLogger.Debug().Msg("Attempting to select calendar via manager")
	if err := h.CalendarManager.SelectCalendar(r.Context(), calendarID); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save calendar selection")
		http.Error(w, "Failed to save calendar selection", http.StatusInternalServerError)
		return
	}
	handlerLogger.Info().Msg("Successfully selected calendar")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
