package handlers

import (
	"context"
	"encoding/json"
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
	http.HandleFunc("/api/sync", h.handleAPISync)
}

// SyncRequest represents the JSON request body for sync
type SyncRequest struct {
	// StartDate is the start date for sync in YYYY-MM-DD format (user's local date)
	StartDate string `json:"start_date"`
}

// SyncResponse represents the JSON response for sync
type SyncResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleAPISync handles AJAX sync requests
func (h *SyncHandler) handleAPISync(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleAPISync").Logger()
	handlerLogger.Info().Msg("Handling API sync request")

	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		handlerLogger.Warn().Str("method", r.Method).Msg("Invalid method for API sync")
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(SyncResponse{
			Success: false,
			Error:   "Method not allowed",
		}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode JSON response")
		}
		return
	}

	// Parse the request body
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerLogger.Warn().Err(err).Msg("Failed to parse request body")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(SyncResponse{
			Success: false,
			Error:   "Invalid request body",
		}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode JSON response")
		}
		return
	}

	// Validate and parse the start date
	var startDate time.Time
	if req.StartDate != "" {
		// Parse the date string in UTC to ensure consistent timezone handling
		// The client sends their local date as YYYY-MM-DD, we interpret it as the start of that day in UTC
		parsed, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			handlerLogger.Warn().Err(err).Str("start_date", req.StartDate).Msg("Invalid start date format")
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(SyncResponse{
				Success: false,
				Error:   "Invalid start date format. Expected YYYY-MM-DD",
			}); err != nil {
				handlerLogger.Error().Err(err).Msg("Failed to encode JSON response")
			}
			return
		}
		// Ensure the parsed date is in UTC (time.Parse with this format defaults to UTC)
		startDate = parsed.UTC()
		handlerLogger.Debug().Time("start_date", startDate).Msg("Using provided start date (interpreted as UTC)")
	} else {
		startDate = time.Now().UTC()
		handlerLogger.Debug().Time("start_date", startDate).Msg("Using current UTC time as start date")
	}

	// Validate authentication and calendar
	if err := h.validateSyncPrerequisites(r); err != nil {
		handlerLogger.Warn().Err(err).Msg("Sync prerequisites not met")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(SyncResponse{
			Success: false,
			Error:   err.Error(),
		}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode JSON response")
		}
		return
	}

	// Run the schedule update with the provided start date
	handlerLogger.Info().Time("start_date", startDate).Msg("Starting schedule update process")
	if err := h.updateScheduleWithDate(r.Context(), startDate); err != nil {
		handlerLogger.Error().Err(err).Msg("Schedule update process failed")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(SyncResponse{
			Success: false,
			Error:   "Sync failed. Please try again.",
		}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode JSON response")
		}
		return
	}

	handlerLogger.Info().Msg("API sync completed successfully")
	if err := json.NewEncoder(w).Encode(SyncResponse{
		Success: true,
		Message: "Schedule synced successfully",
	}); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

// validateSyncPrerequisites checks if sync can proceed (auth, calendar, etc.)
func (h *SyncHandler) validateSyncPrerequisites(r *http.Request) error {
	// Check if we have a token
	hasToken, err := h.TokenManager.HasToken()
	if err != nil {
		return fmt.Errorf("failed to check authentication status: %w", err)
	}
	if !hasToken {
		return fmt.Errorf("authentication required: no token found")
	}

	// Verify token is valid
	token, err := h.TokenManager.GetValidToken(r.Context())
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}
	if token == nil {
		return fmt.Errorf("authentication required: token is invalid")
	}

	// Check if a calendar is selected
	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil {
		return fmt.Errorf("failed to get selected calendar: %w", err)
	}
	if calendarID == "" {
		return fmt.Errorf("calendar selection required: no calendar selected")
	}

	// Initialize calendar service if needed
	if !h.CalendarService.IsInitialized() {
		if err := h.CalendarService.Initialize(r.Context()); err != nil {
			return fmt.Errorf("failed to initialize calendar service: %w", err)
		}
	}

	return nil
}

// handleManualSync handles manual synchronization requests (GET for backwards compatibility)
func (h *SyncHandler) handleManualSync(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleManualSync").Logger()
	handlerLogger.Info().Msg("Handling manual sync request")

	// Check if we have a token
	handlerLogger.Debug().Msg("Checking token existence")
	hasToken, err := h.TokenManager.HasToken()
	if err != nil {
		// Log the error before redirecting
		handlerLogger.Error().Err(err).Msg("Failed to check token existence")
		http.Redirect(w, r, "/?error="+ErrCodeAuthRequired, http.StatusSeeOther)
		return
	}
	if !hasToken {
		handlerLogger.Warn().Msg("No token found, redirecting for authentication")
		http.Redirect(w, r, "/?error="+ErrCodeAuthRequired, http.StatusSeeOther)
		return
	}
	handlerLogger.Debug().Msg("Token exists")

	// Verify token is valid
	handlerLogger.Debug().Msg("Validating token")
	token, err := h.TokenManager.GetValidToken(r.Context())
	if err != nil {
		handlerLogger.Warn().Err(err).Msg("Failed to validate token, redirecting for authentication")
		http.Redirect(w, r, "/?error="+ErrCodeAuthRequired, http.StatusSeeOther)
		return
	}
	if token == nil { // Should not happen if GetValidToken doesn't return error, but check anyway
		handlerLogger.Error().Msg("Token is nil after validation without error, redirecting for authentication")
		http.Redirect(w, r, "/?error="+ErrCodeAuthRequired, http.StatusSeeOther)
		return
	}
	handlerLogger.Debug().Msg("Token is valid")

	// Check if a calendar is selected
	handlerLogger.Debug().Msg("Checking for selected calendar")
	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get selected calendar from store")
		http.Redirect(w, r, "/?error="+ErrCodeSyncFailed, http.StatusSeeOther) // Generic sync error
		return
	}
	if calendarID == "" {
		handlerLogger.Warn().Msg("No calendar selected, redirecting")
		http.Redirect(w, r, "/?error="+ErrCodeCalendarSelectionRequired, http.StatusSeeOther)
		return
	}
	handlerLogger.Debug().Str("calendar_id", calendarID).Msg("Calendar is selected")

	// Check if calendar service is initialized, initialize if not
	if !h.CalendarService.IsInitialized() {
		handlerLogger.Info().Msg("Calendar service not initialized, attempting initialization")
		if err := h.CalendarService.Initialize(r.Context()); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to initialize calendar service during manual sync")
			http.Redirect(w, r, "/?error="+ErrCodeSyncFailed, http.StatusSeeOther)
			return
		}
		handlerLogger.Info().Msg("Calendar service initialized successfully")
	}

	// Run the schedule update
	handlerLogger.Info().Msg("Starting schedule update process")
	if err := h.updateSchedule(r.Context()); err != nil {
		// Error is already logged within updateSchedule
		handlerLogger.Error().Err(err).Msg("Schedule update process failed")
		http.Redirect(w, r, "/?error="+ErrCodeSyncFailed, http.StatusSeeOther)
		return
	}
	handlerLogger.Info().Msg("Schedule update process completed successfully")

	// Redirect back to home with success message
	http.Redirect(w, r, "/?success="+SuccessCodeSyncComplete, http.StatusSeeOther)
}

// updateSchedule generates and syncs a new schedule using current time
func (h *SyncHandler) updateSchedule(ctx context.Context) error {
	return h.updateScheduleWithDate(ctx, time.Now())
}

// updateScheduleWithDate generates and syncs a new schedule starting from the specified date
func (h *SyncHandler) updateScheduleWithDate(ctx context.Context, startDate time.Time) error {
	updateLogger := h.logger.With().Str("operation", "updateSchedule").Logger()
	updateLogger.Info().Time("start_date", startDate).Msg("Starting schedule generation and sync")

	// Calculate date range
	end := startDate.AddDate(0, 0, h.RuntimeConfig.Config.Schedule.LookAheadDays)
	updateLogger.Debug().Time("start_date", startDate).Time("end_date", end).Int("lookahead_days", h.RuntimeConfig.Config.Schedule.LookAheadDays).Msg("Calculated date range")

	// Generate schedule - use startDate as both the start and currentTime
	updateLogger.Debug().Msg("Generating schedule")
	assignments, err := h.Scheduler.GenerateSchedule(startDate, end, startDate)
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
