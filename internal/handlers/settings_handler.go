package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/belphemur/night-routine/internal/database"
	"github.com/rs/zerolog"
)

// SettingsHandler manages settings page functionality
type SettingsHandler struct {
	*BaseHandler
	configStore *database.ConfigStore
}

// NewSettingsHandler creates a new settings page handler
func NewSettingsHandler(baseHandler *BaseHandler, configStore *database.ConfigStore) *SettingsHandler {
	return &SettingsHandler{
		BaseHandler: baseHandler,
		configStore: configStore,
	}
}

// RegisterRoutes registers settings related routes
func (h *SettingsHandler) RegisterRoutes() {
	http.HandleFunc("/settings", h.handleSettings)
	http.HandleFunc("/settings/update", h.handleUpdateSettings)
}

// SettingsPageData contains data for the settings page template
type SettingsPageData struct {
	IsAuthenticated        bool
	ParentA                string
	ParentB                string
	ParentAUnavailable     []string
	ParentBUnavailable     []string
	UpdateFrequency        string
	LookAheadDays          int
	PastEventThresholdDays int
	ErrorMessage           string
	SuccessMessage         string
	AllDaysOfWeek          []string
}

// checkAuthentication verifies if the user has a valid session token
func (h *SettingsHandler) checkAuthentication(logger zerolog.Logger) bool {
	logger.Debug().Msg("Checking token existence")
	hasToken, err := h.TokenManager.HasToken()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check token existence")
		return false
	}
	if !hasToken {
		logger.Debug().Msg("No token found")
		return false
	}
	
	logger.Debug().Msg("Token exists, user is authenticated")
	return true
}

// handleSettings shows the settings page
func (h *SettingsHandler) handleSettings(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleSettings").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling settings page request")
	
	// Check authentication
	isAuthenticated := h.checkAuthentication(handlerLogger)
	
	// Get current configuration
	parentA, parentB, err := h.configStore.GetParents()
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get parent configuration")
		h.RenderTemplate(w, "settings.html", SettingsPageData{
			IsAuthenticated: isAuthenticated,
			ErrorMessage:    "Failed to load configuration. Please try again.",
			AllDaysOfWeek:   getAllDaysOfWeek(),
		})
		return
	}
	
	parentAUnavailable, err := h.configStore.GetAvailability("parent_a")
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get parent A availability")
		parentAUnavailable = []string{}
	}
	
	parentBUnavailable, err := h.configStore.GetAvailability("parent_b")
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get parent B availability")
		parentBUnavailable = []string{}
	}
	
	updateFrequency, lookAheadDays, pastEventThresholdDays, err := h.configStore.GetSchedule()
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get schedule configuration")
		h.RenderTemplate(w, "settings.html", SettingsPageData{
			IsAuthenticated: isAuthenticated,
			ErrorMessage:    "Failed to load configuration. Please try again.",
			AllDaysOfWeek:   getAllDaysOfWeek(),
		})
		return
	}
	
	// Process messages
	errorMessage := r.URL.Query().Get("error")
	successMessage := r.URL.Query().Get("success")
	
	data := SettingsPageData{
		IsAuthenticated:        isAuthenticated,
		ParentA:                parentA,
		ParentB:                parentB,
		ParentAUnavailable:     parentAUnavailable,
		ParentBUnavailable:     parentBUnavailable,
		UpdateFrequency:        updateFrequency,
		LookAheadDays:          lookAheadDays,
		PastEventThresholdDays: pastEventThresholdDays,
		ErrorMessage:           errorMessage,
		SuccessMessage:         successMessage,
		AllDaysOfWeek:          getAllDaysOfWeek(),
	}
	
	handlerLogger.Debug().Msg("Rendering settings template")
	h.RenderTemplate(w, "settings.html", data)
}

// handleUpdateSettings processes settings form submission
func (h *SettingsHandler) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleUpdateSettings").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling settings update request")
	
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	
	// Parse form data
	if err := r.ParseForm(); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to parse form")
		http.Redirect(w, r, "/settings?error=Invalid+form+data", http.StatusSeeOther)
		return
	}
	
	// Extract parent names
	parentA := strings.TrimSpace(r.FormValue("parent_a"))
	parentB := strings.TrimSpace(r.FormValue("parent_b"))
	
	// Extract availability (checkboxes)
	parentAUnavailable := r.Form["parent_a_unavailable"]
	parentBUnavailable := r.Form["parent_b_unavailable"]
	
	// Extract schedule settings
	updateFrequency := r.FormValue("update_frequency")
	lookAheadDaysStr := r.FormValue("look_ahead_days")
	pastEventThresholdDaysStr := r.FormValue("past_event_threshold_days")
	
	// Validate and convert numeric values
	lookAheadDays, err := strconv.Atoi(lookAheadDaysStr)
	if err != nil || lookAheadDays < 1 {
		handlerLogger.Error().Err(err).Str("value", lookAheadDaysStr).Msg("Invalid look ahead days")
		http.Redirect(w, r, "/settings?error=Invalid+look+ahead+days", http.StatusSeeOther)
		return
	}
	
	pastEventThresholdDays, err := strconv.Atoi(pastEventThresholdDaysStr)
	if err != nil || pastEventThresholdDays < 0 {
		handlerLogger.Error().Err(err).Str("value", pastEventThresholdDaysStr).Msg("Invalid past event threshold days")
		http.Redirect(w, r, "/settings?error=Invalid+past+event+threshold+days", http.StatusSeeOther)
		return
	}
	
	handlerLogger.Info().
		Str("parent_a", parentA).
		Str("parent_b", parentB).
		Str("update_frequency", updateFrequency).
		Int("look_ahead_days", lookAheadDays).
		Int("past_event_threshold_days", pastEventThresholdDays).
		Msg("Updating configuration")
	
	// Save parent configuration
	if err := h.configStore.SaveParents(parentA, parentB); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save parent configuration")
		http.Redirect(w, r, "/settings?error=Failed+to+save+parent+names", http.StatusSeeOther)
		return
	}
	
	// Save availability configuration
	if err := h.configStore.SaveAvailability("parent_a", parentAUnavailable); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save parent A availability")
		http.Redirect(w, r, "/settings?error=Failed+to+save+availability", http.StatusSeeOther)
		return
	}
	
	if err := h.configStore.SaveAvailability("parent_b", parentBUnavailable); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save parent B availability")
		http.Redirect(w, r, "/settings?error=Failed+to+save+availability", http.StatusSeeOther)
		return
	}
	
	// Save schedule configuration
	if err := h.configStore.SaveSchedule(updateFrequency, lookAheadDays, pastEventThresholdDays); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save schedule configuration")
		http.Redirect(w, r, "/settings?error=Failed+to+save+schedule+settings", http.StatusSeeOther)
		return
	}
	
	handlerLogger.Info().Msg("Configuration updated successfully")
	http.Redirect(w, r, "/settings?success=Settings+updated+successfully", http.StatusSeeOther)
}

// getAllDaysOfWeek returns all days of the week for the UI
func getAllDaysOfWeek() []string {
	return []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
}
