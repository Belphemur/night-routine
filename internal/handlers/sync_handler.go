package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/token"
)

// SyncHandler manages manual synchronization functionality
type SyncHandler struct {
	*BaseHandler    // Inherits logger
	Scheduler       *scheduler.Scheduler
	TokenManager    *token.TokenManager
	CalendarService *calendar.Service
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(baseHandler *BaseHandler, scheduler *scheduler.Scheduler, tokenManager *token.TokenManager, calendarService *calendar.Service) *SyncHandler {
	return &SyncHandler{
		BaseHandler:     baseHandler,
		Scheduler:       scheduler,
		TokenManager:    tokenManager,
		CalendarService: calendarService,
	}
}

// RegisterRoutes registers sync related routes
func (h *SyncHandler) RegisterRoutes() {
	http.HandleFunc("/sync", h.handleManualSync)
}

// handleManualSync handles manual synchronization requests
func (h *SyncHandler) handleManualSync(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleManualSync").Logger()
	handlerLogger.Info().Msg("Handling manual sync request")

	// Check if we have a token
	handlerLogger.Debug().Msg("Checking token existence")
	hasToken, err := h.TokenManager.HasToken()
	if err != nil {
		// Log the error before redirecting
		handlerLogger.Error().Err(err).Msg("Failed to check token existence")
		http.Redirect(w, r, "/?error=authentication_required", http.StatusSeeOther)
		return
	}
	if !hasToken {
		handlerLogger.Warn().Msg("No token found, redirecting for authentication")
		http.Redirect(w, r, "/?error=authentication_required", http.StatusSeeOther)
		return
	}
	handlerLogger.Debug().Msg("Token exists")

	// Verify token is valid
	handlerLogger.Debug().Msg("Validating token")
	token, err := h.TokenManager.GetValidToken(r.Context())
	if err != nil {
		handlerLogger.Warn().Err(err).Msg("Failed to validate token, redirecting for authentication")
		http.Redirect(w, r, "/?error=authentication_required", http.StatusSeeOther)
		return
	}
	if token == nil { // Should not happen if GetValidToken doesn't return error, but check anyway
		handlerLogger.Error().Msg("Token is nil after validation without error, redirecting for authentication")
		http.Redirect(w, r, "/?error=authentication_required", http.StatusSeeOther)
		return
	}
	handlerLogger.Debug().Msg("Token is valid")

	// Check if a calendar is selected
	handlerLogger.Debug().Msg("Checking for selected calendar")
	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get selected calendar from store")
		http.Redirect(w, r, "/?error=sync_error", http.StatusSeeOther) // Generic sync error
		return
	}
	if calendarID == "" {
		handlerLogger.Warn().Msg("No calendar selected, redirecting")
		http.Redirect(w, r, "/?error=calendar_selection_required", http.StatusSeeOther)
		return
	}
	handlerLogger.Debug().Str("calendar_id", calendarID).Msg("Calendar is selected")

	// Check if calendar service is initialized, initialize if not
	if !h.CalendarService.IsInitialized() {
		handlerLogger.Info().Msg("Calendar service not initialized, attempting initialization")
		if err := h.CalendarService.Initialize(r.Context()); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to initialize calendar service during manual sync")
			http.Redirect(w, r, "/?error=sync_error", http.StatusSeeOther)
			return
		}
		handlerLogger.Info().Msg("Calendar service initialized successfully")
	}

	// Run the schedule update
	handlerLogger.Info().Msg("Starting schedule update process")
	if err := h.updateSchedule(r.Context()); err != nil {
		// Error is already logged within updateSchedule
		handlerLogger.Error().Err(err).Msg("Schedule update process failed")
		http.Redirect(w, r, "/?error=sync_error", http.StatusSeeOther)
		return
	}
	handlerLogger.Info().Msg("Schedule update process completed successfully")

	// Redirect back to home with success message
	http.Redirect(w, r, "/?success=sync_complete", http.StatusSeeOther)
}

// updateSchedule generates and syncs a new schedule
func (h *SyncHandler) updateSchedule(ctx context.Context) error {
	updateLogger := h.logger.With().Str("operation", "updateSchedule").Logger()
	updateLogger.Info().Msg("Starting schedule generation and sync")

	// Calculate date range
	now := time.Now()
	end := now.AddDate(0, 0, h.RuntimeConfig.Config.Schedule.LookAheadDays)
	updateLogger.Debug().Time("start_date", now).Time("end_date", end).Int("lookahead_days", h.RuntimeConfig.Config.Schedule.LookAheadDays).Msg("Calculated date range")

	// Generate schedule
	updateLogger.Debug().Msg("Generating schedule")
	assignments, err := h.Scheduler.GenerateSchedule(now, end, time.Now())
	if err != nil {
		updateLogger.Error().Err(err).Msg("Failed to generate schedule")
		// Wrap error for context
		return fmt.Errorf("failed to generate schedule: %w", err)
	}
	updateLogger.Info().Int("assignments_generated", len(assignments)).Msg("Schedule generated successfully")

	// Sync with calendar
	updateLogger.Debug().Msg("Syncing schedule with calendar")
	if err := h.CalendarService.SyncSchedule(ctx, assignments); err != nil {
		updateLogger.Error().Err(err).Msg("Failed to sync schedule with calendar")
		// Wrap error for context
		return fmt.Errorf("failed to sync calendar: %w", err)
	}

	updateLogger.Info().
		Int("days", h.RuntimeConfig.Config.Schedule.LookAheadDays).
		Int("assignments", len(assignments)).
		Msg("Schedule update and sync completed successfully")
	return nil
}
