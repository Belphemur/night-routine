package calendar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/logging" // Import logging package
	"github.com/belphemur/night-routine/internal/scheduler"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/rs/zerolog" // Import zerolog
)

// Service handles Google Calendar operations
type Service struct {
	calendarID   string
	srv          *calendar.Service
	config       *config.Config
	tokenStore   *database.TokenStore
	tokenManager *token.TokenManager
	scheduler    *scheduler.Scheduler
	initialized  bool
	logger       zerolog.Logger // Add logger field
}

// New creates a new calendar service. It doesn't require a valid token to initialize.
// The service will return errors for operations that require authentication until Initialize is called.
func New(cfg *config.Config, tokenStore *database.TokenStore, scheduler *scheduler.Scheduler, tokenManager *token.TokenManager) *Service {
	return &Service{
		calendarID:   cfg.Schedule.CalendarID, // Default calendar ID from config
		config:       cfg,
		tokenStore:   tokenStore,
		tokenManager: tokenManager,
		scheduler:    scheduler,
		initialized:  false,
		logger:       logging.GetLogger("calendar"), // Initialize logger
	}
}

// Initialize sets up the authenticated calendar service if a valid token is available
func (s *Service) Initialize(ctx context.Context) error {
	s.logger.Info().Msg("Attempting to initialize calendar service...")
	// Check if we have a token
	hasToken, err := s.tokenManager.HasToken()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to check token availability")
		return fmt.Errorf("failed to check token availability: %w", err)
	}

	if !hasToken {
		s.logger.Warn().Msg("No token available for initialization")
		return fmt.Errorf("no token available - please authenticate via web interface first")
	}
	s.logger.Debug().Msg("Token found")

	// Get and validate token
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get valid token")
		return fmt.Errorf("failed to get valid token: %w", err)
	}
	s.logger.Debug().Msg("Valid token obtained")

	// Create authenticated client
	client := s.config.OAuth.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create Google Calendar service client")
		return fmt.Errorf("failed to create calendar service: %w", err)
	}
	s.logger.Debug().Msg("Google Calendar service client created")

	// Get calendar ID from store
	calendarID, err := s.tokenStore.GetSelectedCalendar()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get selected calendar ID from store")
		return fmt.Errorf("failed to get selected calendar: %w", err)
	}
	if calendarID != "" {
		s.logger.Info().Str("calendar_id", calendarID).Msg("Using selected calendar ID from store")
		s.calendarID = calendarID
	} else {
		s.logger.Info().Str("calendar_id", s.calendarID).Msg("Using default calendar ID from config")
	}

	// Update service with authenticated client
	s.srv = srv
	s.initialized = true
	s.logger.Info().Str("calendar_id", s.calendarID).Msg("Calendar service initialized successfully")

	return nil
}

// IsInitialized returns whether the service has been initialized with a valid token
func (s *Service) IsInitialized() bool {
	return s.initialized
}

// SyncSchedule synchronizes the schedule with Google Calendar
func (s *Service) SyncSchedule(ctx context.Context, assignments []*scheduler.Assignment) error {
	if !s.initialized || s.srv == nil {
		s.logger.Warn().Msg("SyncSchedule called but service is not initialized")
		return fmt.Errorf("calendar service not initialized - authentication required")
	}
	s.logger.Info().Int("assignments_count", len(assignments)).Msg("Starting schedule sync")

	// Get latest token in case it was refreshed
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get valid token during sync")
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		s.logger.Error().Msg("No valid token available during sync")
		return fmt.Errorf("no valid token available")
	}

	// Get latest calendar ID in case it was changed
	calendarID, err := s.tokenStore.GetSelectedCalendar()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get calendar ID during sync")
		return fmt.Errorf("failed to get calendar ID: %w", err)
	}
	if calendarID != "" && calendarID != s.calendarID {
		s.logger.Info().Str("old_calendar_id", s.calendarID).Str("new_calendar_id", calendarID).Msg("Calendar ID changed, updating service")
		s.calendarID = calendarID
	}

	// If no assignments, nothing to sync
	if len(assignments) == 0 {
		s.logger.Info().Msg("No assignments provided, skipping sync")
		return nil
	}

	// Find first and last date in assignments to define our date range for events
	firstDate := assignments[0].Date
	lastDate := assignments[0].Date

	for _, a := range assignments {
		if a.Date.Before(firstDate) {
			firstDate = a.Date
		}
		if a.Date.After(lastDate) {
			lastDate = a.Date
		}
	}
	s.logger.Debug().Time("first_date", firstDate).Time("last_date", lastDate).Msg("Determined assignment date range")

	// Fetch all events in the date range at once
	timeMin := firstDate.Add(-24 * time.Hour).Format(time.RFC3339)
	timeMax := lastDate.Add(24 * time.Hour).Format(time.RFC3339) // Add a day to include last date fully
	s.logger.Debug().Str("time_min", timeMin).Str("time_max", timeMax).Str("calendar_id", s.calendarID).Msg("Fetching existing events in range")

	events, err := s.srv.Events.List(s.calendarID).
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		s.logger.Error().Err(err).Str("calendar_id", s.calendarID).Msg("Failed to list events for date range")
		return fmt.Errorf("failed to list events for date range: %w", err)
	}
	s.logger.Debug().Int("event_count", len(events.Items)).Msg("Fetched existing events")

	// Map events created by our app by date for easy lookup
	eventsByDate := make(map[string][]*calendar.Event)
	ourEventCount := 0
	for _, event := range events.Items {
		if event.ExtendedProperties == nil || event.ExtendedProperties.Private == nil {
			continue
		}

		if val, ok := event.ExtendedProperties.Private["app"]; !ok || val != constants.NightRoutineIdentifier {
			continue
		}
		ourEventCount++
		// Extract date from the event
		var eventDate string
		if event.Start.Date != "" {
			eventDate = event.Start.Date
		} else if event.Start.DateTime != "" {
			// Parse datetime if date is not available directly
			t, err := time.Parse(time.RFC3339, event.Start.DateTime)
			if err == nil {
				eventDate = t.Format("2006-01-02")
			} else {
				s.logger.Warn().Err(err).Str("event_id", event.Id).Str("start_time", event.Start.DateTime).Msg("Failed to parse event start time")
			}
		}
		if eventDate != "" {
			eventsByDate[eventDate] = append(eventsByDate[eventDate], event)
		}
	}
	s.logger.Debug().Int("our_event_count", ourEventCount).Msg("Mapped existing events created by this app")

	// Track dates we've already processed to avoid duplicates
	processedDates := make(map[string]bool)
	var mu sync.Mutex // Mutex to protect the map

	// Use a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Channel for collecting errors from goroutines
	errChan := make(chan error, len(assignments))

	// Semaphore to limit concurrency to 2 at a time
	sem := make(chan struct{}, 2)
	s.logger.Debug().Int("concurrency_limit", 2).Msg("Starting concurrent assignment processing")

	// Process assignments concurrently
	for _, assignment := range assignments {
		dateStr := assignment.Date.Format("2006-01-02")

		// Skip if we've already handled this date - thread-safe check
		mu.Lock()
		if processedDates[dateStr] {
			mu.Unlock()
			continue
		}
		processedDates[dateStr] = true
		mu.Unlock()

		// Add to wait group
		wg.Add(1)

		// Launch goroutine for this assignment
		go func(a *scheduler.Assignment) {
			defer wg.Done()

			// Acquire semaphore slot (limits concurrency)
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore when done

			// Create a logger specific to this assignment processing goroutine
			goroutineLogger := s.logger.With().
				Int64("assignment_id", a.ID).
				Str("date", a.Date.Format("2006-01-02")).
				Str("parent", a.Parent).
				Logger()
			goroutineLogger.Debug().Msg("Processing assignment")

			dateStr := a.Date.Format("2006-01-02")

			privateData := map[string]string{
				"updatedAt":    a.UpdatedAt.Format(time.RFC3339),
				"assignmentId": fmt.Sprintf("%d", a.ID),
				"parent":       a.Parent,
				"app":          constants.NightRoutineIdentifier,
			}

			// Check if we already have a Google Calendar event ID for this assignment
			if a.GoogleCalendarEventID != "" {
				goroutineLogger.Debug().Str("event_id", a.GoogleCalendarEventID).Msg("Assignment has existing event ID, attempting update")
				// Try to update the existing event
				event, err := s.srv.Events.Get(s.calendarID, a.GoogleCalendarEventID).Do()
				if err == nil {
					// Event exists, update it
					goroutineLogger.Debug().Str("event_id", event.Id).Msg("Existing event found, updating")
					event.Summary = fmt.Sprintf("[%s] 🌃👶Routine", a.Parent)
					event.Description = fmt.Sprintf("Night routine duty assigned to %s [%s]",
						a.Parent, constants.NightRoutineIdentifier)
					event.Reminders = &calendar.EventReminders{
						UseDefault:      false,
						ForceSendFields: []string{"UseDefault"},
					}
					event.ExtendedProperties.Private = privateData

					_, err = s.srv.Events.Update(s.calendarID, event.Id, event).Do()
					if err == nil {
						goroutineLogger.Info().Str("event_id", event.Id).Msg("Successfully updated existing event")
						// Successfully updated, return from goroutine
						return
					}
					goroutineLogger.Warn().Err(err).Str("event_id", event.Id).Msg("Failed to update existing event, will attempt delete and create")
					// If update fails, we'll fall through to create a new event
				} else {
					goroutineLogger.Warn().Err(err).Str("event_id", a.GoogleCalendarEventID).Msg("Failed to get existing event by ID, will attempt delete and create")
				}
				// If get fails or update fails, we'll fall through to create a new event
			}

			// We need to safely access the shared eventsByDate map
			mu.Lock()
			dateEvents := eventsByDate[dateStr]
			mu.Unlock()

			// Delete any existing events on this date (if we couldn't update or had no ID)
			if len(dateEvents) > 0 {
				goroutineLogger.Debug().Int("count", len(dateEvents)).Msg("Deleting existing events for this date")
				for _, existingEvent := range dateEvents {
					// Skip if this is the event we just tried to update (and failed)
					if existingEvent.Id == a.GoogleCalendarEventID {
						continue
					}
					goroutineLogger.Debug().Str("event_id", existingEvent.Id).Msg("Deleting event")
					err := s.srv.Events.Delete(s.calendarID, existingEvent.Id).Do()
					if err != nil {
						// Log error but try to continue, maybe other deletes will work
						goroutineLogger.Error().Err(err).Str("event_id", existingEvent.Id).Msg("Failed to delete existing event")
						// Send error to channel but don't return immediately, attempt creation anyway
						errChan <- fmt.Errorf("failed to delete existing event %s for %v: %w", existingEvent.Id, a.Date, err)
						// return // Decide if deletion failure is fatal for this assignment
					} else {
						goroutineLogger.Info().Str("event_id", existingEvent.Id).Msg("Successfully deleted existing event")
					}
				}
			}

			// Create new event with our identifier
			goroutineLogger.Debug().Msg("Creating new calendar event")
			event := &calendar.Event{
				Summary: fmt.Sprintf("[%s] 🌃👶Routine", a.Parent),
				Start: &calendar.EventDateTime{
					Date: dateStr,
				},
				End: &calendar.EventDateTime{
					Date: dateStr,
				},
				Description: fmt.Sprintf("Night routine duty assigned to %s [%s]",
					a.Parent, constants.NightRoutineIdentifier),
				Location:     "Home",
				Transparency: "transparent",
				Source: &calendar.EventSource{
					Title: constants.NightRoutineIdentifier,
					Url:   s.config.App.Url,
				},
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: privateData,
				},
				Reminders: &calendar.EventReminders{
					UseDefault:      false,
					ForceSendFields: []string{"UseDefault"},
					Overrides: []*calendar.EventReminder{
						{
							Method:  "popup",
							Minutes: 4 * 60, // The day before at 8 PM
						},
					},
				},
			}

			// Create the event in Google Calendar
			createdEvent, err := s.srv.Events.Insert(s.calendarID, event).Do()
			if err != nil {
				goroutineLogger.Error().Err(err).Msg("Failed to create new event")
				errChan <- fmt.Errorf("failed to create event for %v: %w", a.Date, err)
				return
			}
			goroutineLogger.Info().Str("event_id", createdEvent.Id).Msg("Successfully created new event")

			// Update the assignment with the Google Calendar event ID
			if err := s.scheduler.UpdateGoogleCalendarEventID(a, createdEvent.Id); err != nil {
				// Log error but continue; this isn't fatal for the sync operation itself
				goroutineLogger.Error().Err(err).Str("event_id", createdEvent.Id).Msg("Failed to update assignment in DB with Google Calendar event ID")
				// Don't send to errChan as the calendar event was created
			} else {
				goroutineLogger.Debug().Str("event_id", createdEvent.Id).Msg("Successfully updated assignment in DB with event ID")
			}
		}(assignment)
	}

	// Wait for all goroutines to finish
	s.logger.Debug().Msg("Waiting for assignment processing goroutines to finish...")
	wg.Wait()
	close(errChan)
	s.logger.Debug().Msg("All assignment processing goroutines finished")

	// Check if any errors occurred
	var firstErr error
	for err := range errChan {
		if firstErr == nil {
			firstErr = err // Capture the first error
		}
		s.logger.Error().Err(err).Msg("Error occurred during concurrent assignment processing")
	}

	if firstErr != nil {
		s.logger.Error().Msg("Errors occurred during sync, returning first error")
		return firstErr // Return the first error encountered
	}

	s.logger.Info().Int("assignments_count", len(assignments)).Msg("Schedule sync completed successfully")
	return nil
}
