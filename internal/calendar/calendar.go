package calendar

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
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
	config     *config.Config
	tokenStore *handlers.TokenStore
}

// New creates a new calendar service
func New(ctx context.Context, cfg *config.Config, tokenStore *handlers.TokenStore) (*Service, error) {
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
