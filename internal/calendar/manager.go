package calendar

import (
	"context"
	"fmt"

	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/signals"
	"github.com/belphemur/night-routine/internal/token"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Manager handles calendar-related operations such as listing and selection
type Manager struct {
	tokenStore   *database.TokenStore
	tokenManager *token.TokenManager
	config       *oauth2.Config
}

// NewManager creates a new calendar manager
func NewManager(tokenStore *database.TokenStore, tokenManager *token.TokenManager, oauthConfig *oauth2.Config) *Manager {
	return &Manager{
		tokenStore:   tokenStore,
		tokenManager: tokenManager,
		config:       oauthConfig,
	}
}

// GetCalendarList fetches available calendars for the authenticated user
func (m *Manager) GetCalendarList(ctx context.Context) (*calendar.CalendarList, error) {
	// Get valid token
	token, err := m.tokenManager.GetValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Create authenticated client
	client := m.config.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Fetch calendar list
	calendars, err := srv.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendars: %w", err)
	}

	return calendars, nil
}

// SelectCalendar saves the selected calendar ID and emits a signal
func (m *Manager) SelectCalendar(ctx context.Context, calendarID string) error {
	if calendarID == "" {
		return fmt.Errorf("calendar ID cannot be empty")
	}

	// Save selected calendar
	if err := m.tokenStore.SaveSelectedCalendar(calendarID); err != nil {
		return fmt.Errorf("failed to save calendar selection: %w", err)
	}

	// Emit calendar selection signal
	signals.EmitCalendarSelected(ctx, calendarID)

	return nil
}

// SelectCalendarWithName saves the selected calendar ID and name, and emits a signal
func (m *Manager) SelectCalendarWithName(ctx context.Context, calendarID string, calendarName string) error {
	if calendarID == "" {
		return fmt.Errorf("calendar ID cannot be empty")
	}

	// Save selected calendar with name
	if err := m.tokenStore.SaveSelectedCalendarWithName(calendarID, calendarName); err != nil {
		return fmt.Errorf("failed to save calendar selection: %w", err)
	}

	// Emit calendar selection signal
	signals.EmitCalendarSelected(ctx, calendarID)

	return nil
}

// GetSelectedCalendar returns the currently selected calendar ID
func (m *Manager) GetSelectedCalendar() (string, error) {
	return m.tokenStore.GetSelectedCalendar()
}

// GetSelectedCalendarWithName returns the currently selected calendar ID and name
func (m *Manager) GetSelectedCalendarWithName() (string, string, error) {
	return m.tokenStore.GetSelectedCalendarWithName()
}
