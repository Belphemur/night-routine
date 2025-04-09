package calendar

import (
	"context"

	"github.com/belphemur/night-routine/internal/fairness/scheduler"
)

// CalendarService defines the interface for calendar operations
type CalendarService interface {
	// Initialize sets up the authenticated calendar service if a valid token is available
	Initialize(ctx context.Context) error

	// IsInitialized returns whether the service has been initialized with a valid token
	IsInitialized() bool

	// SyncSchedule synchronizes the schedule with Google Calendar
	SyncSchedule(ctx context.Context, assignments []*scheduler.Assignment) error

	// SetupNotificationChannel sets up a notification channel for calendar changes
	SetupNotificationChannel(ctx context.Context) error

	// StopNotificationChannel stops a notification channel
	StopNotificationChannel(ctx context.Context, channelID, resourceID string) error

	// StopAllNotificationChannels stops all active notification channels
	StopAllNotificationChannels(ctx context.Context) error

	// VerifyNotificationChannel checks if a notification channel is still active with Google Calendar
	VerifyNotificationChannel(ctx context.Context, channelID, resourceID string) (bool, error)
}

// Ensure Service implements CalendarService
var _ CalendarService = (*Service)(nil)
