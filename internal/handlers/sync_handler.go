package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/scheduler"
)

// SyncHandler manages manual synchronization functionality
type SyncHandler struct {
	*BaseHandler
	Scheduler *scheduler.Scheduler
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(baseHandler *BaseHandler, scheduler *scheduler.Scheduler) *SyncHandler {
	return &SyncHandler{
		BaseHandler: baseHandler,
		Scheduler:   scheduler,
	}
}

// RegisterRoutes registers sync related routes
func (h *SyncHandler) RegisterRoutes() {
	http.HandleFunc("/sync", h.handleManualSync)
}

// handleManualSync handles manual synchronization requests
func (h *SyncHandler) handleManualSync(w http.ResponseWriter, r *http.Request) {
	token, err := h.TokenStore.GetToken()
	if err != nil || token == nil || !token.Valid() {
		http.Redirect(w, r, "/?error=authentication_required", http.StatusSeeOther)
		return
	}

	calendarID, err := h.TokenStore.GetSelectedCalendar()
	if err != nil || calendarID == "" {
		http.Redirect(w, r, "/?error=calendar_selection_required", http.StatusSeeOther)
		return
	}

	// Create a new calendar service
	calService, err := h.createCalendarService(r.Context())
	if err != nil {
		log.Printf("Failed to create calendar service: %v", err)
		http.Redirect(w, r, "/?error=sync_error", http.StatusSeeOther)
		return
	}

	// Run the schedule update
	if err := h.updateSchedule(r.Context(), calService); err != nil {
		log.Printf("Failed to update schedule: %v", err)
		http.Redirect(w, r, "/?error=sync_error", http.StatusSeeOther)
		return
	}

	// Redirect back to home with success message
	http.Redirect(w, r, "/?success=sync_complete", http.StatusSeeOther)
}

// createCalendarService creates a new Google Calendar service
func (h *SyncHandler) createCalendarService(ctx context.Context) (*calendar.Service, error) {
	calendarService, err := calendar.New(ctx, h.Config, h.TokenStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return calendarService, nil
}

// updateSchedule generates and syncs a new schedule
func (h *SyncHandler) updateSchedule(ctx context.Context, calSvc *calendar.Service) error {
	// Calculate date range
	now := time.Now()
	end := now.AddDate(0, 0, h.Config.Schedule.LookAheadDays)

	// Generate schedule
	assignments, err := h.Scheduler.GenerateSchedule(now, end)
	if err != nil {
		return fmt.Errorf("failed to generate schedule: %w", err)
	}

	// Sync with calendar
	if err := calSvc.SyncSchedule(ctx, assignments); err != nil {
		return fmt.Errorf("failed to sync calendar: %w", err)
	}

	// Record assignments
	for _, a := range assignments {
		if err := h.Tracker.RecordAssignment(a.Parent, a.Date); err != nil {
			return fmt.Errorf("failed to record assignment: %w", err)
		}
	}

	log.Printf("Manual sync: Updated schedule for %d days with %d assignments",
		h.Config.Schedule.LookAheadDays, len(assignments))
	return nil
}
