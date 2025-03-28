package handlers

import (
	"log"
	"net/http"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	gcal "google.golang.org/api/calendar/v3"
)

// CalendarHandler manages calendar selection functionality
type CalendarHandler struct {
	*BaseHandler
	CalendarManager *calendar.Manager
	Config          *config.Config
}

// NewCalendarHandler creates a new calendar handler
func NewCalendarHandler(baseHandler *BaseHandler, cfg *config.Config, calendarManager *calendar.Manager) *CalendarHandler {
	return &CalendarHandler{
		BaseHandler:     baseHandler,
		CalendarManager: calendarManager,
		Config:          cfg,
	}
}

// RegisterRoutes registers calendar related routes
func (h *CalendarHandler) RegisterRoutes() {
	http.HandleFunc("/calendars", h.handleCalendarList)
}

// CalendarPageData contains data for the calendar selection page
type CalendarPageData struct {
	Calendars *gcal.CalendarList
	Selected  string
	Error     string
}

// handleCalendarList shows available calendars and allows selection
func (h *CalendarHandler) handleCalendarList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.handleCalendarSelection(w, r)
		return
	}

	// Get available calendars
	calendars, err := h.CalendarManager.GetCalendarList(r.Context())
	if err != nil {
		log.Printf("Failed to fetch calendars: %v", err)

		// Use TokenManager's ClearToken method instead of direct TokenStore access
		if tokenErr := h.TokenManager.ClearToken(r.Context()); tokenErr != nil {
			log.Printf("Failed to clear token: %v", tokenErr)
		}

		http.Redirect(w, r, "/?error=calendar_fetch_error", http.StatusSeeOther)
		return
	}

	// Get currently selected calendar
	selected, err := h.CalendarManager.GetSelectedCalendar()
	if err != nil {
		http.Error(w, "Failed to get selected calendar", http.StatusInternalServerError)
		return
	}

	data := CalendarPageData{
		Calendars: calendars,
		Selected:  selected,
	}

	h.RenderTemplate(w, "oauth.html", data)
}

// handleCalendarSelection processes calendar selection
func (h *CalendarHandler) handleCalendarSelection(w http.ResponseWriter, r *http.Request) {
	calendarID := r.FormValue("calendar_id")
	if calendarID == "" {
		http.Error(w, "No calendar selected", http.StatusBadRequest)
		return
	}

	// Use the calendar manager to select the calendar
	if err := h.CalendarManager.SelectCalendar(r.Context(), calendarID); err != nil {
		http.Error(w, "Failed to save calendar selection", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
