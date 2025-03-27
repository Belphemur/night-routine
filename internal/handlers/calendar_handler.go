package handlers

import (
	"log"
	"net/http"

	"github.com/belphemur/night-routine/internal/config"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// CalendarHandler manages calendar selection functionality
type CalendarHandler struct {
	*BaseHandler
	OAuthConfig *oauth2.Config
}

// Updated to use the unified OAuth configuration from the Config struct
func NewCalendarHandler(baseHandler *BaseHandler, cfg *config.Config) *CalendarHandler {
	return &CalendarHandler{
		BaseHandler: baseHandler,
		OAuthConfig: cfg.OAuth,
	}
}

// RegisterRoutes registers calendar related routes
func (h *CalendarHandler) RegisterRoutes() {
	http.HandleFunc("/calendars", h.handleCalendarList)
}

// CalendarPageData contains data for the calendar selection page
type CalendarPageData struct {
	Calendars *calendar.CalendarList
	Selected  string
}

// handleCalendarList shows available calendars and allows selection
func (h *CalendarHandler) handleCalendarList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.handleCalendarSelection(w, r)
		return
	}

	token, err := h.TokenStore.GetToken()
	if err != nil {
		http.Error(w, "Failed to get token", http.StatusInternalServerError)
		return
	}

	client := h.OAuthConfig.Client(r.Context(), token)
	calendarService, err := calendar.NewService(r.Context(), option.WithHTTPClient(client))
	if err != nil {
		log.Printf("Failed to create calendar client: %v", err)
		if clearErr := h.TokenStore.ClearToken(); clearErr != nil {
			log.Printf("Failed to clear token: %v", clearErr)
		}
		http.Redirect(w, r, "/?error=calendar_client_error", http.StatusSeeOther)
		return
	}

	calendars, err := calendarService.CalendarList.List().Do()
	if err != nil {
		log.Printf("Failed to fetch calendars: %v", err)
		if clearErr := h.TokenStore.ClearToken(); clearErr != nil {
			log.Printf("Failed to clear token: %v", clearErr)
		}
		http.Redirect(w, r, "/?error=calendar_fetch_error", http.StatusSeeOther)
		return
	}

	selected, err := h.TokenStore.GetSelectedCalendar()
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

	if err := h.TokenStore.SaveSelectedCalendar(calendarID); err != nil {
		http.Error(w, "Failed to save calendar selection", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
