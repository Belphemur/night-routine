package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/rs/zerolog"
)

// SettingsHandler manages settings page functionality
type SettingsHandler struct {
	*BaseHandler
	configStore     *database.ConfigStore
	scheduler       *scheduler.Scheduler
	tokenManager    *token.TokenManager
	calendarService *calendar.Service
}

// NewSettingsHandler creates a new settings page handler
func NewSettingsHandler(baseHandler *BaseHandler, configStore *database.ConfigStore, sched *scheduler.Scheduler, tokenMgr *token.TokenManager, calSvc *calendar.Service) *SettingsHandler {
	return &SettingsHandler{
		BaseHandler:     baseHandler,
		configStore:     configStore,
		scheduler:       sched,
		tokenManager:    tokenMgr,
		calendarService: calSvc,
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

// handleSettings shows the settings page
func (h *SettingsHandler) handleSettings(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleSettings").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling settings page request")

	// Always allow access to settings (no authentication check needed)

	// Get current configuration
	parentA, parentB, err := h.configStore.GetParents()
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get parent configuration")
		h.RenderTemplate(w, "settings.html", SettingsPageData{
			IsAuthenticated: true, // Always authenticated for settings
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
			IsAuthenticated: true, // Always authenticated for settings
			ErrorMessage:    "Failed to load configuration. Please try again.",
			AllDaysOfWeek:   getAllDaysOfWeek(),
		})
		return
	}

	// Process messages
	errorMessage := GetErrorMessage(r.URL.Query().Get("error"))
	successMessage := GetSuccessMessage(r.URL.Query().Get("success"))

	// Only show unknown error if there was actually an error param
	if r.URL.Query().Get("error") == "" {
		errorMessage = ""
	}

	data := SettingsPageData{
		IsAuthenticated:        true, // Always authenticated for settings
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

	// No authentication check - settings are always accessible

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to parse form")
		http.Redirect(w, r, "/settings?error="+ErrCodeInvalidFormData, http.StatusSeeOther)
		return
	}

	// Extract parent names
	parentA := strings.TrimSpace(r.FormValue("parent_a"))
	parentB := strings.TrimSpace(r.FormValue("parent_b"))

	// Extract availability (checkboxes)
	parentAUnavailable := r.Form["parent_a_unavailable"]
	parentBUnavailable := r.Form["parent_b_unavailable"]

	// Validate unavailable days
	for _, day := range parentAUnavailable {
		if !constants.IsValidDayOfWeek(day) {
			handlerLogger.Error().Str("invalid_day", day).Msg("Invalid day in parent A availability")
			http.Redirect(w, r, "/settings?error="+ErrCodeInvalidDayOfWeek, http.StatusSeeOther)
			return
		}
	}
	for _, day := range parentBUnavailable {
		if !constants.IsValidDayOfWeek(day) {
			handlerLogger.Error().Str("invalid_day", day).Msg("Invalid day in parent B availability")
			http.Redirect(w, r, "/settings?error="+ErrCodeInvalidDayOfWeek, http.StatusSeeOther)
			return
		}
	}

	// Extract schedule settings
	updateFrequency := r.FormValue("update_frequency")
	lookAheadDaysStr := r.FormValue("look_ahead_days")
	pastEventThresholdDaysStr := r.FormValue("past_event_threshold_days")

	// Validate and convert numeric values with upper bounds
	lookAheadDays, err := strconv.Atoi(lookAheadDaysStr)
	if err != nil || lookAheadDays < 1 || lookAheadDays > 365 {
		handlerLogger.Error().Err(err).Str("value", lookAheadDaysStr).Msg("Invalid look ahead days")
		http.Redirect(w, r, "/settings?error="+ErrCodeInvalidLookAheadDays, http.StatusSeeOther)
		return
	}

	pastEventThresholdDays, err := strconv.Atoi(pastEventThresholdDaysStr)
	if err != nil || pastEventThresholdDays < 0 || pastEventThresholdDays > 30 {
		handlerLogger.Error().Err(err).Str("value", pastEventThresholdDaysStr).Msg("Invalid past event threshold days")
		http.Redirect(w, r, "/settings?error="+ErrCodeInvalidPastEventThreshold, http.StatusSeeOther)
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
		http.Redirect(w, r, "/settings?error="+ErrCodeFailedSaveParent, http.StatusSeeOther)
		return
	}

	// Save availability configuration
	if err := h.configStore.SaveAvailability("parent_a", parentAUnavailable); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save parent A availability")
		http.Redirect(w, r, "/settings?error="+ErrCodeFailedSaveAvailability, http.StatusSeeOther)
		return
	}

	if err := h.configStore.SaveAvailability("parent_b", parentBUnavailable); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save parent B availability")
		http.Redirect(w, r, "/settings?error="+ErrCodeFailedSaveAvailability, http.StatusSeeOther)
		return
	}

	// Save schedule configuration
	if err := h.configStore.SaveSchedule(updateFrequency, lookAheadDays, pastEventThresholdDays); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to save schedule configuration")
		http.Redirect(w, r, "/settings?error="+ErrCodeFailedSaveSchedule, http.StatusSeeOther)
		return
	}

	handlerLogger.Info().Msg("Configuration updated successfully")

	// Trigger automatic sync after settings update
	if err := h.triggerSync(r.Context(), handlerLogger); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to trigger automatic sync after settings update")
		http.Redirect(w, r, "/settings?success="+SuccessCodeSettingsUpdatedSyncFailed, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/settings?success="+SuccessCodeSettingsUpdated, http.StatusSeeOther)
}

// triggerSync triggers an automatic schedule sync
func (h *SettingsHandler) triggerSync(ctx context.Context, logger zerolog.Logger) error {
	logger.Info().Msg("Triggering automatic sync after settings update")

	// Check if we have a token
	hasToken, err := h.tokenManager.HasToken()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check token existence")
		return fmt.Errorf("failed to check token: %w", err)
	}
	if !hasToken {
		logger.Warn().Msg("No token found, skipping automatic sync")
		return fmt.Errorf("no authentication token available")
	}

	// Verify token is valid
	token, err := h.tokenManager.GetValidToken(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to validate token, skipping automatic sync")
		return fmt.Errorf("invalid token: %w", err)
	}
	if token == nil {
		logger.Error().Msg("Token is nil after validation")
		return fmt.Errorf("token validation failed")
	}

	// Check if a calendar is selected
	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get selected calendar")
		return fmt.Errorf("failed to get calendar: %w", err)
	}
	if calendarID == "" {
		logger.Warn().Msg("No calendar selected, skipping automatic sync")
		return fmt.Errorf("no calendar selected")
	}

	// Ensure calendar service is initialized
	if !h.calendarService.IsInitialized() {
		logger.Info().Msg("Initializing calendar service for automatic sync")
		if err := h.calendarService.Initialize(ctx); err != nil {
			logger.Error().Err(err).Msg("Failed to initialize calendar service")
			return fmt.Errorf("failed to initialize calendar service: %w", err)
		}
	}

	// Generate and sync schedule
	logger.Info().Msg("Generating schedule for automatic sync")
	now := time.Now()

	// Fetch lookAheadDays from database to use the latest settings
	_, lookAheadDays, _, err := h.configStore.GetSchedule()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch lookAheadDays from database")
		return fmt.Errorf("failed to fetch lookAheadDays: %w", err)
	}
	end := now.AddDate(0, 0, lookAheadDays)

	assignments, err := h.scheduler.GenerateSchedule(now, end, time.Now())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate schedule")
		return fmt.Errorf("failed to generate schedule: %w", err)
	}

	logger.Info().Int("assignments", len(assignments)).Msg("Syncing schedule with calendar")
	if err := h.calendarService.SyncSchedule(ctx, assignments); err != nil {
		logger.Error().Err(err).Msg("Failed to sync schedule with calendar")
		return fmt.Errorf("failed to sync calendar: %w", err)
	}

	logger.Info().Msg("Automatic sync completed successfully")
	return nil
}

// getAllDaysOfWeek returns all days of the week for the UI
func getAllDaysOfWeek() []string {
	return constants.GetAllDaysOfWeek()
}
