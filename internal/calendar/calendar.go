package calendar

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/handlers"
	"github.com/belphemur/night-routine/internal/scheduler"
)

// Service handles Google Calendar operations
type Service struct {
	calendarID string
	srv        *calendar.Service
}

// New creates a new calendar service
func New(ctx context.Context, cfg *config.Config, tokenStore *handlers.TokenStore) (*Service, error) {
	creds, err := os.ReadFile(cfg.Google.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	oauthConfig, err := google.ConfigFromJSON(creds, calendar.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
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
	}, nil
}

// SyncSchedule synchronizes the schedule with Google Calendar
func (s *Service) SyncSchedule(ctx context.Context, assignments []scheduler.Assignment) error {
	for _, assignment := range assignments {
		event := &calendar.Event{
			Summary: fmt.Sprintf("Night Routine - %s", assignment.Parent),
			Start: &calendar.EventDateTime{
				Date: assignment.Date.Format("2006-01-02"),
			},
			End: &calendar.EventDateTime{
				Date: assignment.Date.Format("2006-01-02"),
			},
			Description: fmt.Sprintf("Night routine duty assigned to %s", assignment.Parent),
		}

		_, err := s.srv.Events.Insert(s.calendarID, event).Do()
		if err != nil {
			return fmt.Errorf("failed to create event for %v: %w", assignment.Date, err)
		}
	}

	return nil
}
