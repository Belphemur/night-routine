package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/scheduler"
	"github.com/belphemur/night-routine/internal/token"
)

// SyncHandler manages manual synchronization functionality
type SyncHandler struct {
	*BaseHandler
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
	// Check if we have a valid token
	hasToken, err := h.TokenManager.HasToken()
	if err != nil || !hasToken {
		http.Redirect(w, r, "/?error=authentication_required", http.StatusSeeOther)
		return
	}

	// Verify token is valid
	token, err := h.TokenManager.GetValidToken(r.Context())
	if err != nil || token == nil {
		http.Redirect(w, r, "/?error=authentication_required", http.StatusSeeOther)
		return
	}

	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil || calendarID == "" {
		http.Redirect(w, r, "/?error=calendar_selection_required", http.StatusSeeOther)
		return
	}

	// Check if calendar service is initialized
	if !h.CalendarService.IsInitialized() {
		if err := h.CalendarService.Initialize(r.Context()); err != nil {
			log.Printf("Failed to initialize calendar service: %v", err)
			http.Redirect(w, r, "/?error=sync_error", http.StatusSeeOther)
			return
		}
	}

	// Run the schedule update
	if err := h.updateSchedule(r.Context()); err != nil {
		log.Printf("Failed to update schedule: %v", err)
		http.Redirect(w, r, "/?error=sync_error", http.StatusSeeOther)
		return
	}

	// Redirect back to home with success message
	http.Redirect(w, r, "/?success=sync_complete", http.StatusSeeOther)
}

// updateSchedule generates and syncs a new schedule
func (h *SyncHandler) updateSchedule(ctx context.Context) error {
	// Calculate date range
	now := time.Now()
	end := now.AddDate(0, 0, h.Config.Schedule.LookAheadDays)

	// Generate schedule
	assignments, err := h.Scheduler.GenerateSchedule(now, end)
	if err != nil {
		return fmt.Errorf("failed to generate schedule: %w", err)
	}

	// Sync with calendar
	if err := h.CalendarService.SyncSchedule(ctx, assignments); err != nil {
		return fmt.Errorf("failed to sync calendar: %w", err)
	}

	log.Printf("Manual sync: Updated schedule for %d days with %d assignments",
		h.Config.Schedule.LookAheadDays, len(assignments))
	return nil
}
