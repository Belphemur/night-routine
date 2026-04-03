package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/rs/zerolog"
)

// WebhookHandler handles incoming webhook notifications
type WebhookHandler struct {
	*BaseHandler
	CalendarService calendar.CalendarService
	Scheduler       Scheduler.SchedulerInterface
	TokenManager    *token.TokenManager
	// ConfigStore is used to read schedule configuration live from the database,
	// so that settings changes (e.g. PastEventThresholdDays, LookAheadDays) take
	// effect immediately without requiring an application restart.
	ConfigStore config.ConfigStoreInterface
	logger      zerolog.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(baseHandler *BaseHandler, calendarService calendar.CalendarService, scheduler Scheduler.SchedulerInterface, tokenManager *token.TokenManager, configStore config.ConfigStoreInterface) *WebhookHandler {
	return &WebhookHandler{
		BaseHandler:     baseHandler,
		CalendarService: calendarService,
		Scheduler:       scheduler,
		TokenManager:    tokenManager,
		ConfigStore:     configStore,
		logger:          logging.GetLogger("webhook"),
	}
}

// RegisterRoutes registers webhook related routes
func (h *WebhookHandler) RegisterRoutes() {
	http.HandleFunc("/api/webhook/calendar", h.handleCalendarWebhook)
}

// handleCalendarWebhook processes incoming calendar notifications
func (h *WebhookHandler) handleCalendarWebhook(w http.ResponseWriter, r *http.Request) {
	// Add request context to logger
	requestLogger := h.logger.With().
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Str("channel_id", r.Header.Get("X-Goog-Channel-ID")).
		Str("resource_id", r.Header.Get("X-Goog-Resource-ID")).
		Str("resource_state", r.Header.Get("X-Goog-Resource-State")).
		Logger()
	requestLogger.Info().Msg("Received calendar webhook notification")

	// Validate the request
	channelID := r.Header.Get("X-Goog-Channel-ID")
	resourceID := r.Header.Get("X-Goog-Resource-ID")
	resourceState := r.Header.Get("X-Goog-Resource-State")

	// Verify the channel ID and resource ID
	channel, err := h.TokenStore.GetNotificationChannelByID(channelID)
	if err != nil {
		requestLogger.Error().Err(err).Msg("Error retrieving notification channel from store")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	// Check channel before accessing ResourceID
	expectedResourceID := ""
	if channel != nil {
		expectedResourceID = channel.ResourceID
	}
	if channel == nil || channel.ResourceID != resourceID {
		requestLogger.Warn().
			Bool("channel_found", channel != nil).
			Str("expected_resource_id", expectedResourceID).
			Msg("Invalid notification channel ID or resource ID mismatch")
		http.Error(w, "Invalid notification channel", http.StatusBadRequest)
		return
	}
	requestLogger.Debug().Msg("Notification channel validated")

	// Check if the channel is close to expiration (within 7 days)
	if time.Until(channel.Expiration) < 7*24*time.Hour {
		requestLogger.Info().Time("expiration", channel.Expiration).Msg("Notification channel is close to expiration, attempting refresh")
		// Refresh the notification channel
		if err := h.CalendarService.SetupNotificationChannel(r.Context()); err != nil {
			requestLogger.Warn().Err(err).Msg("Failed to refresh notification channel")
			// Continue processing the current notification even if refresh fails
		} else {
			requestLogger.Info().Msg("Successfully refreshed notification channel")
		}
	}

	// Process the notification
	if resourceState == "sync" {
		requestLogger.Info().Msg("Received sync notification, acknowledging")
		// This is just a sync message, acknowledge it
		w.WriteHeader(http.StatusOK)
		return
	}

	// This is an actual change notification
	requestLogger.Info().Msg("Processing event change notification")
	if err := h.processEventChanges(r.Context(), channel.CalendarID); err != nil {
		requestLogger.Error().Err(err).Msg("Error processing event changes")
		http.Error(w, "Failed to process event changes", http.StatusInternalServerError)
		return
	}

	requestLogger.Info().Msg("Event changes processed successfully")
	w.WriteHeader(http.StatusOK)
}

// processEventChanges fetches recent changes and updates assignments
func (h *WebhookHandler) processEventChanges(ctx context.Context, calendarID string) error {
	procLogger := h.logger.With().Str("calendar_id", calendarID).Logger()
	procLogger.Info().Msg("Processing event changes")

	// Get a valid token using TokenManager
	token, err := h.TokenManager.GetValidToken(ctx)
	if err != nil {
		procLogger.Error().Err(err).Msg("Failed to get valid token for processing changes")
		return fmt.Errorf("failed to get valid token: %w", err)
	}
	procLogger.Debug().Msg("Valid token obtained")

	// Create a calendar client using the OAuth config from the config store
	client := h.ConfigStore.GetOAuthConfig().Client(ctx, token)
	calendarSvc, err := gcalendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		procLogger.Error().Err(err).Msg("Failed to create Google Calendar service client")
		return fmt.Errorf("failed to create calendar service: %w", err)
	}
	procLogger.Debug().Msg("Google Calendar service client created")

	// Get events that were recently updated
	// Look back slightly further to avoid race conditions with notification delivery
	timeMin := time.Now().Add(-2 * time.Minute).Format(time.RFC3339)
	procLogger.Debug().Str("updated_min", timeMin).Msg("Fetching recently updated events")
	events, err := calendarSvc.Events.List(calendarID).
		UpdatedMin(timeMin).
		SingleEvents(true).
		OrderBy("updated").
		Do()
	if err != nil {
		procLogger.Error().Err(err).Msg("Failed to list updated events from Google Calendar")
		return fmt.Errorf("failed to list updated events: %w", err)
	}
	procLogger.Info().Int("event_count", len(events.Items)).Msg("Fetched updated events")

	if len(events.Items) == 0 {
		procLogger.Info().Msg("No recently updated events found")
		return nil
	}

	return h.processEvents(ctx, events.Items, procLogger)
}

// processEvents processes a batch of calendar events and updates assignments accordingly
func (h *WebhookHandler) processEvents(ctx context.Context, events []*gcalendar.Event, procLogger zerolog.Logger) error {
	var processingErrors []error
	parentA, parentB, err := h.ConfigStore.GetParents()
	if err != nil {
		procLogger.Warn().Err(err).Msg("Failed to get parent names from config store, falling back to summary-only parsing")
		parentA = ""
		parentB = ""
	}

	// Read the past-event threshold live from the database so that UI setting
	// changes take effect immediately without requiring an application restart.
	_, _, thresholdDays, _, err := h.ConfigStore.GetSchedule()
	if err != nil {
		procLogger.Error().Err(err).Msg("Failed to get schedule configuration for past event threshold")
		return fmt.Errorf("failed to get schedule configuration: %w", err)
	}
	procLogger.Debug().Int("threshold_days", thresholdDays).Msg("Using past event threshold from live config")

	for _, event := range events {
		eventLogger := procLogger.With().Str("event_id", event.Id).Logger()
		eventLogger.Debug().Msg("Processing event")

		if event.Status == "cancelled" {
			eventLogger.Info().Msg("Event was cancelled, skipping processing for parent update")
			continue // Don't process cancelled events for parent changes
		}

		if event.ExtendedProperties == nil || event.ExtendedProperties.Private == nil {
			eventLogger.Debug().Msg("Event has no extended private properties, skipping")
			continue
		}

		if val, ok := event.ExtendedProperties.Private["app"]; !ok || val != constants.NightRoutineIdentifier {
			eventLogger.Debug().Msg("Event is not managed by Night Routine app, skipping")
			continue
		}
		eventLogger.Debug().Msg("Event identified as managed by Night Routine")

		assignee, ok := parseManagedEventAssignee(event.Summary, parentA, parentB)
		if !ok {
			eventLogger.Warn().Str("summary", event.Summary).Msg("Could not parse managed assignee from event summary, skipping")
			continue
		}
		eventLogger = eventLogger.With().
			Str("event_assignee", assignee.Name).
			Str("event_caregiver_type", assignee.CaregiverType.String()).
			Logger()
		eventLogger.Debug().Msg("Extracted managed assignee from event summary")

		// Find the assignment by Google Calendar event ID
		assignment, err := h.Scheduler.GetAssignmentByGoogleCalendarEventID(event.Id)
		if err != nil {
			eventLogger.Error().Err(err).Msg("Error finding assignment by event ID")
			processingErrors = append(processingErrors, err) // Collect error
			continue
		}

		// If assignment not found, log and skip
		if assignment == nil {
			eventLogger.Warn().Msg("No matching assignment found for this event ID, skipping update")
			continue
		}
		eventLogger = eventLogger.With().
			Int64("assignment_id", assignment.ID).
			Str("assignment_parent", assignment.Parent).
			Str("assignment_date", assignment.Date.Format("2006-01-02")).
			Logger()
		eventLogger.Debug().Msg("Found matching assignment")

		// If parent name hasn't changed in the summary, skip
		if assignment.CaregiverType == assignee.CaregiverType {
			if assignee.CaregiverType == fairness.CaregiverTypeBabysitter {
				if assignment.Parent == assignee.Name {
					eventLogger.Debug().Msg("Event summary babysitter matches assignment babysitter, no update needed")
					continue
				}
			} else if assignment.Parent == assignee.Name {
				eventLogger.Debug().Msg("Event summary parent matches assignment parent, no update needed")
				continue
			}
		}

		if assignment.CaregiverType != fairness.CaregiverTypeBabysitter && assignment.Parent == assignee.Name {
			eventLogger.Debug().Msg("Event summary parent matches assignment parent, no update needed")
			continue
		}

		// Check if the private property already reflects the change (maybe updated by another process)
		if currentAssigneeProp, ok := event.ExtendedProperties.Private["parent"]; ok {
			currentTypeProp := event.ExtendedProperties.Private["caregiverType"]
			if currentAssigneeProp == assignee.Name && currentTypeProp == assignee.CaregiverType.String() {
				eventLogger.Debug().Msg("Event private properties already match summary assignee, skipping update")
				continue
			}
		}

		// Check if the assignment is within the configurable past event threshold
		now := time.Now()
		thresholdDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -thresholdDays)

		// Ensure assignment date is compared in the same timezone/location
		// We use the Year/Month/Day from the assignment date to construct a new date in the local timezone
		// This avoids issues where DB returns UTC time which shifts the day when converted to Local
		y, m, d := assignment.Date.Date()
		assignmentDate := time.Date(y, m, d, 0, 0, 0, 0, now.Location())

		if assignmentDate.Before(thresholdDate) {
			eventLogger.Warn().
				Int("threshold_days", thresholdDays).
				Str("threshold_date", thresholdDate.Format("2006-01-02")).
				Str("assignment_date", assignmentDate.Format("2006-01-02")).
				Msg("Rejecting override attempt for past assignment outside threshold")
			continue
		}
		eventLogger.Debug().
			Int("threshold_days", thresholdDays).
			Msg("Assignment date is within threshold, proceeding with update")

		if assignee.CaregiverType == fairness.CaregiverTypeBabysitter {
			eventLogger.Info().Msg("Updating assignment to babysitter due to event change (override)")
			if err := h.Scheduler.UpdateAssignmentToBabysitter(assignment.ID, assignee.Name, true); err != nil {
				eventLogger.Error().Err(err).Msg("Error updating assignment to babysitter in database")
				processingErrors = append(processingErrors, err)
				continue
			}
		} else {
			eventLogger.Info().Msg("Updating assignment parent due to event change (override)")
			if err := h.Scheduler.UpdateAssignmentParent(assignment.ID, assignee.Name, true); err != nil {
				eventLogger.Error().Err(err).Msg("Error updating assignment parent in database")
				processingErrors = append(processingErrors, err)
				continue
			}
		}

		eventLogger.Info().Msg("Successfully updated assignment in database")

		// Recalculate the schedule for future days starting from the modified assignment's date
		eventLogger.Info().Msg("Recalculating schedule due to override")
		if err := h.recalculateSchedule(ctx, assignment.Date); err != nil {
			eventLogger.Error().Err(err).Msg("Error recalculating schedule after override")
			processingErrors = append(processingErrors, err) // Collect error
			continue
		}
		eventLogger.Info().Msg("Successfully recalculated schedule")
	}

	// Join multiple errors if they occurred
	if len(processingErrors) > 0 {
		combinedErr := errors.Join(processingErrors...) // Use errors.Join
		procLogger.Error().Err(combinedErr).Int("error_count", len(processingErrors)).Msg("Errors occurred while processing event changes")
		return combinedErr // Return the combined error to trigger rollback
	}

	procLogger.Info().Msg("Finished processing event changes")
	return nil // Success - transaction will be committed
}

// recalculateSchedule regenerates the schedule from the given date
func (h *WebhookHandler) recalculateSchedule(ctx context.Context, fromDate time.Time) error {
	return recalculateScheduleAndSync(
		ctx,
		h.logger,
		h.Tracker,
		h.Scheduler,
		h.CalendarService,
		h.ConfigStore,
		fromDate,
	)
}

type parsedManagedAssignee struct {
	Name          string
	CaregiverType fairness.CaregiverType
}

func parseManagedEventAssignee(summary, parentA, parentB string) (parsedManagedAssignee, bool) {
	trimmedSummary := strings.TrimSpace(summary)
	if trimmedSummary == "" {
		return parsedManagedAssignee{}, false
	}

	if strings.HasPrefix(trimmedSummary, "[") {
		endBracket := strings.Index(trimmedSummary, "]")
		if endBracket > 1 {
			name := strings.TrimSpace(trimmedSummary[1:endBracket])
			if name == "" {
				return parsedManagedAssignee{}, false
			}

			if parentA == "" && parentB == "" {
				return parsedManagedAssignee{Name: name, CaregiverType: fairness.CaregiverTypeParent}, true
			}

			if name == parentA || name == parentB {
				return parsedManagedAssignee{Name: name, CaregiverType: fairness.CaregiverTypeParent}, true
			}

			return parsedManagedAssignee{Name: name, CaregiverType: fairness.CaregiverTypeBabysitter}, true
		}
	}

	if before, ok := strings.CutSuffix(trimmedSummary, " - Babysitter"); ok {
		name := strings.TrimSpace(before)
		if name == "" {
			return parsedManagedAssignee{}, false
		}
		return parsedManagedAssignee{Name: name, CaregiverType: fairness.CaregiverTypeBabysitter}, true
	}

	return parsedManagedAssignee{}, false
}
