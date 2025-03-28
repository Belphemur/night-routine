package calendar

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/belphemur/night-routine/internal/database"
)

// SetupNotificationChannel sets up a notification channel for calendar changes
func (s *Service) SetupNotificationChannel(ctx context.Context) error {
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

	// Delete any expired notification channels
	if err := s.tokenStore.DeleteExpiredNotificationChannels(); err != nil {
		return fmt.Errorf("failed to delete expired notification channels: %w", err)
	}

	// Check if we already have an active notification channel
	activeChannels, err := s.tokenStore.GetActiveNotificationChannels()
	if err != nil {
		return fmt.Errorf("failed to get active notification channels: %w", err)
	}

	// If we have an active channel for this calendar, stop here
	for _, channel := range activeChannels {
		if channel.CalendarID == s.calendarID {
			// We already have an active channel for this calendar
			return nil
		}
	}

	// Create a new notification channel
	// The channel ID should be unique
	channelID := fmt.Sprintf("night-routine-%d", time.Now().UnixNano())

	// The address where Google will send notifications
	// This should be a publicly accessible URL
	address := fmt.Sprintf("%s/api/webhook/calendar", s.config.App.Url)

	// Create the channel
	channel := &calendar.Channel{
		Id:      channelID,
		Type:    "web_hook",
		Address: address,
		Params: map[string]string{
			"ttl": "2592000", // 30 days in seconds
		},
	}

	// Watch the calendar
	createdChannel, err := s.srv.Events.Watch(s.calendarID, channel).Do()
	if err != nil {
		return fmt.Errorf("failed to watch calendar: %w", err)
	}

	// Calculate expiration time
	expiration := time.Now().Add(30 * 24 * time.Hour) // 30 days
	if createdChannel.Expiration > 0 {
		expiration = time.Unix(createdChannel.Expiration/1000, 0)
	}

	// Save the notification channel
	notificationChannel := &database.NotificationChannel{
		ID:         createdChannel.Id,
		ResourceID: createdChannel.ResourceId,
		CalendarID: s.calendarID,
		Expiration: expiration,
	}

	if err := s.tokenStore.SaveNotificationChannel(notificationChannel); err != nil {
		// Try to stop the channel if we couldn't save it
		_ = s.StopNotificationChannel(ctx, createdChannel.Id, createdChannel.ResourceId)
		return fmt.Errorf("failed to save notification channel: %w", err)
	}

	return nil
}

// StopNotificationChannel stops a notification channel
func (s *Service) StopNotificationChannel(ctx context.Context, channelID, resourceID string) error {
	// Get latest token in case it was refreshed
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return fmt.Errorf("no valid token available")
	}

	// Stop the channel
	channel := &calendar.Channel{
		Id:         channelID,
		ResourceId: resourceID,
	}

	err = s.srv.Channels.Stop(channel).Do()
	if err != nil {
		return fmt.Errorf("failed to stop notification channel: %w", err)
	}

	// Delete the notification channel from the database
	if err := s.tokenStore.DeleteNotificationChannel(channelID); err != nil {
		return fmt.Errorf("failed to delete notification channel from database: %w", err)
	}

	return nil
}

// StopAllNotificationChannels stops all active notification channels
func (s *Service) StopAllNotificationChannels(ctx context.Context) error {
	// Get latest token in case it was refreshed
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		return fmt.Errorf("no valid token available")
	}

	// Get all active notification channels
	activeChannels, err := s.tokenStore.GetActiveNotificationChannels()
	if err != nil {
		return fmt.Errorf("failed to get active notification channels: %w", err)
	}

	// Stop each channel
	for _, channel := range activeChannels {
		err := s.StopNotificationChannel(ctx, channel.ID, channel.ResourceID)
		if err != nil {
			// Log the error but continue with other channels
			fmt.Printf("Warning: Failed to stop notification channel %s: %v\n", channel.ID, err)
		}
	}

	return nil
}
