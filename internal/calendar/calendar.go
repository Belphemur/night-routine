package calendar

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

// Service handles Google Calendar operations
type Service struct {
	calendarID   string
	srv          *calendar.Service
	oauthConfig  *oauth2.Config
	appUrl       string
	publicUrl    string
	tokenStore   *database.TokenStore
	tokenManager *token.TokenManager
	scheduler    *scheduler.Scheduler
	initialized  bool
	logger       zerolog.Logger
}

// New creates a new calendar service. It doesn't require a valid token to initialize.
// The service will return errors for operations that require authentication until Initialize is called.
// oauthConfig, appUrl, and publicUrl are static values from file/env configuration.
func New(oauthConfig *oauth2.Config, appUrl string, publicUrl string, tokenStore *database.TokenStore, scheduler *scheduler.Scheduler, tokenManager *token.TokenManager) *Service {
	return &Service{
		oauthConfig:  oauthConfig,
		appUrl:       appUrl,
		publicUrl:    publicUrl,
		tokenStore:   tokenStore,
		tokenManager: tokenManager,
		scheduler:    scheduler,
		initialized:  false,
		logger:       logging.GetLogger("calendar"),
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
	client := s.oauthConfig.Client(ctx, token)
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

	// Map events created by our app by assignment ID and date for easy lookup.
	eventsByAssignmentID := make(map[int64][]*calendar.Event)
	eventsByDate := make(map[string][]*calendar.Event)
	ourEventCount := 0
	for _, event := range events.Items {
		if !eventBelongsToApp(event, s.appUrl) {
			continue
		}

		ourEventCount++
		if eventDate := eventStartDate(event); eventDate != "" {
			eventsByDate[eventDate] = append(eventsByDate[eventDate], event)
		}

		assignmentID, ok, err := eventAssignmentID(event)
		if err != nil {
			s.logger.Warn().Err(err).Str("event_id", event.Id).Msg("Failed to parse assignmentId from event properties")
			continue
		}
		if !ok {
			continue
		}

		eventsByAssignmentID[assignmentID] = append(eventsByAssignmentID[assignmentID], event)
	}
	s.logger.Debug().
		Int("our_event_count", ourEventCount).
		Int("assignments_with_events", len(eventsByAssignmentID)).
		Int("dates_with_events", len(eventsByDate)).
		Msg("Mapped existing events created by this app")

	// Track assignments we've already processed to avoid duplicates
	processedAssignments := make(map[int64]bool)
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
		// Skip if we've already handled this assignment ID - thread-safe check
		mu.Lock()
		if processedAssignments[assignment.ID] {
			mu.Unlock()
			continue
		}
		processedAssignments[assignment.ID] = true
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

			startDateStr := a.Date.Format("2006-01-02")
			// For all-day events, the end date is the day after the start date.
			endDateStr := a.Date.AddDate(0, 0, 1).Format("2006-01-02")

			privateData := map[string]string{
				"updatedAt":     a.UpdatedAt.Format(time.RFC3339),
				"assignmentId":  fmt.Sprintf("%d", a.ID),
				"parent":        a.Parent,
				"caregiverType": a.CaregiverType.String(),
				"app":           constants.NightRoutineIdentifier,
			}
			if a.CaregiverType == fairness.CaregiverTypeBabysitter {
				privateData["babysitterName"] = a.Parent
			}

			// Check if we already have a Google Calendar event ID for this assignment
			if a.GoogleCalendarEventID != "" {
				goroutineLogger.Debug().Str("event_id", a.GoogleCalendarEventID).Msg("Assignment has existing event ID, attempting update")
				event, err := s.srv.Events.Get(s.calendarID, a.GoogleCalendarEventID).Do()
				if err == nil {
					if eventBelongsToApp(event, s.appUrl) {
						goroutineLogger.Debug().Str("event_id", event.Id).Msg("Existing managed event found by ID, updating")
						populateManagedEvent(event, a, privateData, startDateStr, endDateStr, s.appUrl)

						_, err = s.srv.Events.Update(s.calendarID, event.Id, event).Do()
						if err == nil {
							goroutineLogger.Info().Str("event_id", event.Id).Msg("Successfully updated existing event")
							return
						}
						goroutineLogger.Warn().Err(err).Str("event_id", event.Id).Msg("Failed to update existing event, will attempt relink or recreate")
					} else {
						goroutineLogger.Warn().Str("event_id", event.Id).Msg("Stored event ID points to an event not managed by Night Routine, will relink or recreate")
					}
				} else if isGoogleAPINotFound(err) {
					goroutineLogger.Info().Str("event_id", a.GoogleCalendarEventID).Msg("Stored event ID no longer exists in Google Calendar, will relink or recreate")
				} else {
					goroutineLogger.Warn().Err(err).Str("event_id", a.GoogleCalendarEventID).Msg("Failed to get existing event by ID, will attempt relink or recreate")
				}
			}

			var assignmentEvents []*calendar.Event
			var dateEvents []*calendar.Event
			mu.Lock()
			assignmentEvents = append(assignmentEvents, eventsByAssignmentID[a.ID]...)
			dateEvents = append(dateEvents, eventsByDate[startDateStr]...)
			mu.Unlock()

			reusableEvent, duplicateEvents := selectReusableManagedEvent(assignmentEvents, dateEvents)
			if reusableEvent != nil {
				goroutineLogger.Debug().
					Str("event_id", reusableEvent.Id).
					Int("duplicate_count", len(duplicateEvents)).
					Msg("Found existing managed event to relink")
				populateManagedEvent(reusableEvent, a, privateData, startDateStr, endDateStr, s.appUrl)

				_, err := s.srv.Events.Update(s.calendarID, reusableEvent.Id, reusableEvent).Do()
				if err == nil {
					if a.GoogleCalendarEventID != reusableEvent.Id {
						if err := s.scheduler.UpdateGoogleCalendarEventID(a, reusableEvent.Id); err != nil {
							goroutineLogger.Error().Err(err).Str("event_id", reusableEvent.Id).Msg("Failed to relink assignment in DB to existing managed event")
						} else {
							goroutineLogger.Info().Str("event_id", reusableEvent.Id).Msg("Relinked assignment in DB to existing managed event")
						}
					}

					for _, duplicateEvent := range duplicateEvents {
						goroutineLogger.Debug().Str("event_id", duplicateEvent.Id).Msg("Deleting duplicate managed event")
						err := s.srv.Events.Delete(s.calendarID, duplicateEvent.Id).Do()
						if err != nil {
							goroutineLogger.Error().Err(err).Str("event_id", duplicateEvent.Id).Msg("Failed to delete duplicate managed event")
							errChan <- fmt.Errorf("failed to delete duplicate managed event %s for %v: %w", duplicateEvent.Id, a.Date, err)
						} else {
							goroutineLogger.Info().Str("event_id", duplicateEvent.Id).Msg("Successfully deleted duplicate managed event")
						}
					}
					return
				}

				goroutineLogger.Warn().Err(err).Str("event_id", reusableEvent.Id).Msg("Failed to update relink candidate, will recreate")
				duplicateEvents = append([]*calendar.Event{reusableEvent}, duplicateEvents...)
			}

			if len(duplicateEvents) > 0 {
				goroutineLogger.Debug().Int("count", len(duplicateEvents)).Msg("Deleting existing managed events before recreation")
				for _, existingEvent := range duplicateEvents {
					goroutineLogger.Debug().Str("event_id", existingEvent.Id).Msg("Deleting event")
					err := s.srv.Events.Delete(s.calendarID, existingEvent.Id).Do()
					if err != nil {
						if isGoogleAPINotFound(err) {
							goroutineLogger.Info().Str("event_id", existingEvent.Id).Msg("Managed event already missing during delete, continuing with recreation")
							continue
						}
						goroutineLogger.Error().Err(err).Str("event_id", existingEvent.Id).Msg("Failed to delete existing event")
						errChan <- fmt.Errorf("failed to delete existing event %s for %v: %w", existingEvent.Id, a.Date, err)
					} else {
						goroutineLogger.Info().Str("event_id", existingEvent.Id).Msg("Successfully deleted existing event")
					}
				}
			}

			// Create new event with our identifier
			goroutineLogger.Debug().Msg("Creating new calendar event")
			event := &calendar.Event{
				Start: &calendar.EventDateTime{
					Date: startDateStr,
				},
				End: &calendar.EventDateTime{
					Date: endDateStr,
				},
				Location:     "Home",
				Transparency: "transparent",
				Source: &calendar.EventSource{
					Title: constants.NightRoutineIdentifier,
					Url:   s.appUrl,
				},
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: privateData,
				},
			}
			populateManagedEvent(event, a, privateData, startDateStr, endDateStr, s.appUrl)

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
	var allErrors []error // Slice to hold all errors
	for err := range errChan {
		if err != nil {
			allErrors = append(allErrors, err) // Collect all non-nil errors
			s.logger.Error().Err(err).Msg("Error occurred during concurrent assignment processing")
		}
	}

	if len(allErrors) > 0 {
		joinedErr := errors.Join(allErrors...) // Join all collected errors
		s.logger.Error().Err(joinedErr).Int("error_count", len(allErrors)).Msg("Errors occurred during sync, returning joined error")
		return joinedErr // Return the joined error
	}

	s.logger.Info().Int("assignments_count", len(assignments)).Msg("Schedule sync completed successfully")
	return nil
}

// displayName returns the name to show in calendar events.
// For all caregiver types, parent_name holds the correct display name.
func displayName(assignment *scheduler.Assignment) string {
	return assignment.Parent
}

func formatEventSummary(assignment *scheduler.Assignment) string {
	return fmt.Sprintf("[%s] 🌃👶Routine", displayName(assignment))
}

// formatEventDescription formats the event description string.
func formatEventDescription(assignment *scheduler.Assignment) string {
	name := displayName(assignment)
	if assignment.CaregiverType == fairness.CaregiverTypeBabysitter {
		return fmt.Sprintf("Night routine handled by babysitter %s. Reason: %s [%s]",
			name, assignment.DecisionReason.String(), constants.NightRoutineIdentifier)
	}
	return fmt.Sprintf("Night routine duty assigned to %s. Reason: %s [%s]",
		name, assignment.DecisionReason.String(), constants.NightRoutineIdentifier)
}

// setNoReminders disables all reminders for an event.
func setNoReminders(event *calendar.Event) {
	event.Reminders = &calendar.EventReminders{
		UseDefault:      false,
		Overrides:       []*calendar.EventReminder{},
		ForceSendFields: []string{"UseDefault", "Overrides"},
	}
}

func populateManagedEvent(event *calendar.Event, assignment *scheduler.Assignment, privateData map[string]string, startDateStr string, endDateStr string, appURL string) {
	event.Summary = formatEventSummary(assignment)
	event.Description = formatEventDescription(assignment)
	if event.Start == nil {
		event.Start = &calendar.EventDateTime{}
	}
	event.Start.Date = startDateStr
	if event.End == nil {
		event.End = &calendar.EventDateTime{}
	}
	event.End.Date = endDateStr
	if event.Source == nil {
		event.Source = &calendar.EventSource{}
	}
	event.Source.Title = constants.NightRoutineIdentifier
	event.Source.Url = appURL
	if event.ExtendedProperties == nil {
		event.ExtendedProperties = &calendar.EventExtendedProperties{}
	}
	event.ExtendedProperties.Private = privateData
	setNoReminders(event)
}

func eventBelongsToApp(event *calendar.Event, appURL string) bool {
	if event == nil {
		return false
	}
	if event.ExtendedProperties != nil && event.ExtendedProperties.Private != nil {
		if appIdentifier, ok := event.ExtendedProperties.Private["app"]; ok && appIdentifier == constants.NightRoutineIdentifier {
			return true
		}
	}
	return event.Source != nil && event.Source.Url == appURL
}

func eventAssignmentID(event *calendar.Event) (int64, bool, error) {
	if event == nil || event.ExtendedProperties == nil || event.ExtendedProperties.Private == nil {
		return 0, false, nil
	}
	assignmentIDStr, ok := event.ExtendedProperties.Private["assignmentId"]
	if !ok || assignmentIDStr == "" {
		return 0, false, nil
	}
	assignmentID, err := strconv.ParseInt(assignmentIDStr, 10, 64)
	if err != nil {
		return 0, true, fmt.Errorf("failed to parse assignmentId %q: %w", assignmentIDStr, err)
	}
	return assignmentID, true, nil
}

func eventStartDate(event *calendar.Event) string {
	if event == nil || event.Start == nil {
		return ""
	}
	if event.Start.Date != "" {
		return event.Start.Date
	}
	if event.Start.DateTime == "" {
		return ""
	}
	startTime, err := time.Parse(time.RFC3339, event.Start.DateTime)
	if err != nil {
		return ""
	}
	return startTime.Format("2006-01-02")
}

func selectReusableManagedEvent(priorityEvents []*calendar.Event, fallbackEvents []*calendar.Event) (*calendar.Event, []*calendar.Event) {
	orderedEvents := make([]*calendar.Event, 0, len(priorityEvents)+len(fallbackEvents))
	seen := make(map[string]struct{}, len(priorityEvents)+len(fallbackEvents))

	appendUnique := func(events []*calendar.Event) {
		for _, event := range events {
			if event == nil || event.Id == "" {
				continue
			}
			if _, ok := seen[event.Id]; ok {
				continue
			}
			seen[event.Id] = struct{}{}
			orderedEvents = append(orderedEvents, event)
		}
	}

	appendUnique(priorityEvents)
	appendUnique(fallbackEvents)

	if len(orderedEvents) == 0 {
		return nil, nil
	}
	return orderedEvents[0], orderedEvents[1:]
}

func isGoogleAPINotFound(err error) bool {
	var googleAPIError *googleapi.Error
	return errors.As(err, &googleAPIError) && googleAPIError.Code == 404
}
