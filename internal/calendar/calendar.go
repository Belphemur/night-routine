package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/scheduler"
)

// Service handles Google Calendar operations
type Service struct {
	calendarID string
	srv        *calendar.Service
	config     *config.Config
	tokenStore *database.TokenStore
}

// New creates a new calendar service
func New(ctx context.Context, cfg *config.Config, tokenStore *database.TokenStore) (*Service, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		RedirectURL:  cfg.OAuth.RedirectURL,
		Scopes: []string{
			calendar.CalendarEventsScope,
		},
		Endpoint: google.Endpoint,
	}

	token, err := tokenStore.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return nil, fmt.Errorf("no token available - please authenticate via web interface first")
	}

	client := oauthConfig.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Get calendar ID from store
	calendarID, err := tokenStore.GetSelectedCalendar()
	if err != nil {
		return nil, fmt.Errorf("failed to get selected calendar: %w", err)
	}
	if calendarID == "" {
		calendarID = cfg.Schedule.CalendarID // Fallback to config
	}

	return &Service{
		calendarID: calendarID,
		srv:        srv,
		config:     cfg,
		tokenStore: tokenStore,
	}, nil
}

// SyncSchedule synchronizes the schedule with Google Calendar
func (s *Service) SyncSchedule(ctx context.Context, assignments []scheduler.Assignment) error {
	// Get latest token in case it was refreshed
	token, err := s.tokenStore.GetToken()
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
	timeMin := firstDate.Format(time.RFC3339)
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
		if strings.Contains(event.Description, nightRoutineIdentifier) {
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

		// Delete any existing events on this date
		for _, existingEvent := range eventsByDate[dateStr] {
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
		}

		_, err = s.srv.Events.Insert(s.calendarID, event).Do()
		if err != nil {
			return fmt.Errorf("failed to create event for %v: %w", assignment.Date, err)
		}
	}

	return nil
}
