package calendar

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/scheduler"
	"github.com/belphemur/night-routine/internal/token"
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
	}
}

// Initialize sets up the authenticated calendar service if a valid token is available
func (s *Service) Initialize(ctx context.Context) error {
	// Check if we have a token
	hasToken, err := s.tokenManager.HasToken()
	if err != nil {
		return fmt.Errorf("failed to check token availability: %w", err)
	}

	if !hasToken {
		return fmt.Errorf("no token available - please authenticate via web interface first")
	}

	// Get and validate token
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	// Create authenticated client
	client := s.config.OAuth.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Get calendar ID from store
	calendarID, err := s.tokenStore.GetSelectedCalendar()
	if err != nil {
		return fmt.Errorf("failed to get selected calendar: %w", err)
	}
	if calendarID != "" {
		s.calendarID = calendarID
	}

	// Update service with authenticated client
	s.srv = srv
	s.initialized = true

	return nil
}

// IsInitialized returns whether the service has been initialized with a valid token
func (s *Service) IsInitialized() bool {
	return s.initialized
}

// SyncSchedule synchronizes the schedule with Google Calendar
func (s *Service) SyncSchedule(ctx context.Context, assignments []*scheduler.Assignment) error {
	if !s.initialized || s.srv == nil {
		return fmt.Errorf("calendar service not initialized - authentication required")
	}

	// Get latest token in case it was refreshed
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return fmt.Errorf("no valid token available")
	}

	// Get latest calendar ID in case it was changed
	calendarID, err := s.tokenStore.GetSelectedCalendar()
	if err != nil {
		return fmt.Errorf("failed to get calendar ID: %w", err)
	}
	if calendarID != "" {
		s.calendarID = calendarID
	}

	// Unique identifier for events created by this application
	const nightRoutineIdentifier = "night-routine-app-event"

	// If no assignments, nothing to sync
	if len(assignments) == 0 {
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

	// Fetch all events in the date range at once
	timeMin := firstDate.Add(-24 * time.Hour).Format(time.RFC3339)
	timeMax := lastDate.Add(24 * time.Hour).Format(time.RFC3339) // Add a day to include last date fully

	events, err := s.srv.Events.List(s.calendarID).
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return fmt.Errorf("failed to list events for date range: %w", err)
	}

	// Map events created by our app by date for easy lookup
	eventsByDate := make(map[string][]*calendar.Event)
	for _, event := range events.Items {
		if event.Source == nil || event.Source.Title != nightRoutineIdentifier {
			continue
		}
		// Extract date from the event
		var eventDate string
		if event.Start.Date != "" {
			eventDate = event.Start.Date
		} else if event.Start.DateTime != "" {
			// Parse datetime if date is not available directly
			t, err := time.Parse(time.RFC3339, event.Start.DateTime)
			if err == nil {
				eventDate = t.Format("2006-01-02")
			}
		}
		if eventDate != "" {
			eventsByDate[eventDate] = append(eventsByDate[eventDate], event)
		}
	}

	// Track dates we've already processed to avoid duplicates
	processedDates := make(map[string]bool)

	// Process assignments
	for _, assignment := range assignments {
		dateStr := assignment.Date.Format("2006-01-02")

		// Skip if we've already handled this date
		if processedDates[dateStr] {
			continue
		}
		processedDates[dateStr] = true

		// Check if we already have a Google Calendar event ID for this assignment
		if assignment.GoogleCalendarEventID != "" {
			// Try to update the existing event
			event, err := s.srv.Events.Get(s.calendarID, assignment.GoogleCalendarEventID).Do()
			if err == nil {
				// Event exists, update it
				event.Summary = fmt.Sprintf("[%s] 🌃👶Routine", assignment.Parent)
				event.Description = fmt.Sprintf("Night routine duty assigned to %s [%s]",
					assignment.Parent, nightRoutineIdentifier)

				_, err = s.srv.Events.Update(s.calendarID, event.Id, event).Do()
				if err == nil {
					// Successfully updated, continue to next assignment
					continue
				}
				// If update fails, we'll fall through to create a new event
			}
			// If get fails or update fails, we'll fall through to create a new event
		}

		// Delete any existing events on this date (if we couldn't update)
		for _, existingEvent := range eventsByDate[dateStr] {
			// Skip if this is the event we just tried to update
			if existingEvent.Id == assignment.GoogleCalendarEventID {
				continue
			}

			err := s.srv.Events.Delete(s.calendarID, existingEvent.Id).Do()
			if err != nil {
				return fmt.Errorf("failed to delete existing event for %v: %w", assignment.Date, err)
			}
		}

		// Create new event with our identifier
		event := &calendar.Event{
			Summary: fmt.Sprintf("[%s] 🌃👶Routine", assignment.Parent),
			Start: &calendar.EventDateTime{
				Date: dateStr,
			},
			End: &calendar.EventDateTime{
				Date: dateStr,
			},
			Description: fmt.Sprintf("Night routine duty assigned to %s [%s]",
				assignment.Parent, nightRoutineIdentifier),
			Location:     "Home",
			Transparency: "transparent",
			Source: &calendar.EventSource{
				Title: nightRoutineIdentifier,
				Url:   s.config.App.Url,
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
			return fmt.Errorf("failed to create event for %v: %w", assignment.Date, err)
		}

		// Update the assignment with the Google Calendar event ID
		if err := s.scheduler.UpdateGoogleCalendarEventID(assignment, createdEvent.Id); err != nil {
			// Log error but continue; this isn't fatal
			fmt.Printf("Warning: Failed to update assignment with Google Calendar event ID: %v\n", err)
		}
	}

	return nil
}
