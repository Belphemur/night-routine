package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/scheduler"
	"github.com/belphemur/night-routine/internal/token"
)

// WebhookHandler handles incoming webhook notifications
type WebhookHandler struct {
	*BaseHandler
	CalendarService *calendar.Service
	Scheduler       *scheduler.Scheduler
	Config          *config.Config
	TokenManager    *token.TokenManager
}

// RegisterRoutes registers webhook related routes
func (h *WebhookHandler) RegisterRoutes() {
	http.HandleFunc("/api/webhook/calendar", h.handleCalendarWebhook)
}

// handleCalendarWebhook processes incoming calendar notifications
func (h *WebhookHandler) handleCalendarWebhook(w http.ResponseWriter, r *http.Request) {
	// Validate the request
	channelID := r.Header.Get("X-Goog-Channel-ID")
	resourceID := r.Header.Get("X-Goog-Resource-ID")

	// Verify the channel ID and resource ID
	channel, err := h.TokenStore.GetNotificationChannelByID(channelID)
	if err != nil || channel == nil || channel.ResourceID != resourceID {
		http.Error(w, "Invalid notification channel", http.StatusBadRequest)
		return
	}

	// Check if the channel is close to expiration (within 7 days)
	if time.Until(channel.Expiration) < 7*24*time.Hour {
		// Refresh the notification channel
		if err := h.CalendarService.SetupNotificationChannel(r.Context()); err != nil {
			log.Printf("Warning: Failed to refresh notification channel: %v", err)
		} else {
			log.Printf("Successfully refreshed notification channel that was close to expiration")
		}
	}

	// Process the notification
	if r.Header.Get("X-Goog-Resource-State") == "sync" {
		// This is just a sync message, acknowledge it
		w.WriteHeader(http.StatusOK)
		return
	}

	// This is an actual change notification
	// Process the event changes
	if err := h.processEventChanges(r.Context(), channel.CalendarID); err != nil {
		log.Printf("Error processing event changes: %v", err)
		http.Error(w, "Failed to process event changes", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// processEventChanges fetches recent changes and updates assignments
func (h *WebhookHandler) processEventChanges(ctx context.Context, calendarID string) error {
	// Get a valid token using TokenManager
	token, err := h.TokenManager.GetValidToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	// Create a calendar client
	client := h.Config.OAuth.Client(ctx, token)
	calendarSvc, err := gcalendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Get events that were recently updated
	timeMin := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	events, err := calendarSvc.Events.List(calendarID).
		UpdatedMin(timeMin).
		SingleEvents(true).
		OrderBy("updated").
		Do()
	if err != nil {
		return fmt.Errorf("failed to list updated events: %w", err)
	}

	// Process each event
	for _, event := range events.Items {

		if event.ExtendedProperties == nil || event.ExtendedProperties.Private == nil {
			continue
		}

		if val, ok := event.ExtendedProperties.Private["app"]; !ok || val != constants.NightRoutineIdentifier {
			log.Printf("Event %s is not from Night Routine app, skipping", event.Id)
			continue
		}
		// Extract the parent name from the event summary
		// Format: "[Parent] 🌃👶Routine"
		parentName := extractParentName(event.Summary)
		if parentName == "" {
			continue
		}

		// Find the assignment by Google Calendar event ID
		assignment, err := h.Scheduler.GetAssignmentByGoogleCalendarEventID(event.Id)
		if err != nil {
			log.Printf("Error finding assignment for event %s: %v", event.Id, err)
			continue
		}

		// If assignment not found or parent name hasn't changed, skip
		if assignment == nil || assignment.Parent == parentName {
			continue
		}

		if event.ExtendedProperties.Private["parent"] == parentName {
			log.Printf("Same parent name, skipping update for event %s and assignment %d", event.Id, assignment.ID)
			continue
		}

		// Check if the assignment is in the future or today
		today := time.Now().Truncate(24 * time.Hour)
		if assignment.Date.Before(today) {
			log.Printf("Rejecting override for past assignment on %s", assignment.Date.Format("2006-01-02"))
			continue
		}

		// Update the assignment with the new parent name and set override flag
		if err := h.Scheduler.UpdateAssignmentParent(assignment.ID, parentName, true); err != nil {
			log.Printf("Error updating assignment %d: %v", assignment.ID, err)
			continue
		}

		log.Printf("Updated assignment [%s] %d from %s to %s (override)",
			assignment.Date, assignment.ID, assignment.Parent, parentName)

		// Recalculate the schedule for future days
		if err := h.recalculateSchedule(ctx, assignment.Date); err != nil {
			log.Printf("Error recalculating schedule: %v", err)
		}
	}

	return nil
}

// recalculateSchedule regenerates the schedule from the given date
func (h *WebhookHandler) recalculateSchedule(ctx context.Context, fromDate time.Time) error {
	// Use the same look-ahead period as defined in the config
	lookAheadDays := h.Config.Schedule.LookAheadDays

	// Calculate the end date
	endDate := fromDate.AddDate(0, 0, lookAheadDays)

	// Generate a new schedule
	assignments, err := h.Scheduler.GenerateSchedule(fromDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to generate schedule: %w", err)
	}

	// Sync the new schedule with Google Calendar
	if err := h.CalendarService.SyncSchedule(ctx, assignments); err != nil {
		return fmt.Errorf("failed to sync schedule: %w", err)
	}

	return nil
}

// extractParentName extracts the parent name from the event summary
func extractParentName(summary string) string {
	// Format: "[Parent] 🌃👶Routine"
	if len(summary) < 3 || !strings.HasPrefix(summary, "[") {
		return ""
	}

	endBracket := strings.Index(summary, "]")
	if endBracket <= 1 {
		return ""
	}

	return summary[1:endBracket]
}
