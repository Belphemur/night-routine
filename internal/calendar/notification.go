package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/belphemur/night-routine/internal/database"
)

// SetupNotificationChannel sets up a notification channel for calendar changes
func (s *Service) SetupNotificationChannel(ctx context.Context) error {
	s.logger.Info().Msg("Setting up notification channel...")
	// Get latest token in case it was refreshed
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get valid token for notification setup")
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		s.logger.Error().Msg("No valid token available for notification setup")
		return fmt.Errorf("no valid token available")
	}

	// Get latest calendar ID in case it was changed
	calendarID, err := s.tokenStore.GetSelectedCalendar()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get selected calendar ID for notification setup")
		return fmt.Errorf("failed to get calendar ID: %w", err)
	}
	if calendarID != "" && calendarID != s.calendarID {
		s.logger.Info().Str("old_calendar_id", s.calendarID).Str("new_calendar_id", calendarID).Msg("Calendar ID changed, updating service for notification setup")
		s.calendarID = calendarID
	} else if calendarID == "" {
		s.logger.Warn().Msg("No calendar ID selected, cannot set up notification channel")
		return fmt.Errorf("no calendar ID selected")
	}
	logger := s.logger.With().Str("calendar_id", s.calendarID).Logger() // Logger with calendar ID context

	// Delete any expired notification channels
	logger.Debug().Msg("Deleting expired notification channels")
	if err := s.tokenStore.DeleteExpiredNotificationChannels(); err != nil {
		// Log warning but continue, maybe we can still set up a new one
		logger.Warn().Err(err).Msg("Failed to delete expired notification channels")
		// return fmt.Errorf("failed to delete expired notification channels: %w", err) // Decide if this is fatal
	}

	// Check if we already have an active notification channel for this calendar
	logger.Debug().Msg("Checking for existing active notification channels")
	activeChannels, err := s.tokenStore.GetActiveNotificationChannels()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get active notification channels from store")
		return fmt.Errorf("failed to get active notification channels: %w", err)
	}

	// If we have an active channel for this calendar, verify it with Google
	for _, channel := range activeChannels {
		if channel.CalendarID == s.calendarID {
			channelLogger := logger.With().
				Str("channel_id", channel.ID).
				Str("resource_id", channel.ResourceID).
				Time("expiration", channel.Expiration).
				Logger()

			channelLogger.Info().Msg("Found potentially active notification channel, verifying with Google Calendar...")

			// Verify the channel is actually active with Google
			isActive, verifyErr := s.VerifyNotificationChannel(ctx, channel.ID, channel.ResourceID)

			if verifyErr != nil {
				channelLogger.Warn().Err(verifyErr).Msg("Failed to verify channel status with Google Calendar")
				// Continue to create a new channel when verification fails
			} else if isActive {
				channelLogger.Info().Msg("Verified active notification channel with Google Calendar")
				// We have an active channel that Google confirms is working
				return nil
			} else {
				channelLogger.Warn().Msg("Channel exists in our DB but is not active with Google Calendar, will create a new one")

				// Stop and delete the inactive channel
				channelLogger.Debug().Msg("Removing inactive channel from database")
				if err := s.tokenStore.DeleteNotificationChannel(channel.ID); err != nil {
					channelLogger.Warn().Err(err).Msg("Failed to delete inactive channel from database")
					// Non-fatal, continue
				}
			}

			// Either verification failed or channel is inactive, continue to create a new one
			break
		}
	}
	logger.Info().Msg("No active notification channel found for this calendar, creating a new one")

	// Create a new notification channel
	// The channel ID should be unique
	channelID := fmt.Sprintf("night-routine-%d", time.Now().UnixNano())
	logger = logger.With().Str("new_channel_id", channelID).Logger() // Add new channel ID to context

	// The address where Google will send notifications
	// This should be a publicly accessible URL
	address := fmt.Sprintf("%s/api/webhook/calendar", s.config.App.PublicUrl)
	logger.Debug().Str("webhook_address", address).Msg("Generated webhook address")

	// Create the channel object for Google API
	channel := &calendar.Channel{
		Id:      channelID,
		Type:    "web_hook",
		Address: address,
		Params: map[string]string{
			"ttl": "2592000", // 30 days in seconds
		},
	}

	// Watch the calendar
	logger.Info().Msg("Sending watch request to Google Calendar API")
	createdChannel, err := s.srv.Events.Watch(s.calendarID, channel).Do()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to watch calendar via Google API")
		return fmt.Errorf("failed to watch calendar: %w", err)
	}
	logger.Info().Str("created_channel_id", createdChannel.Id).Str("resource_id", createdChannel.ResourceId).Int64("expires_ms", createdChannel.Expiration).Msg("Successfully created watch channel with Google")

	// Calculate expiration time
	expiration := time.Now().Add(30 * 24 * time.Hour) // Default 30 days
	if createdChannel.Expiration > 0 {
		expiration = time.Unix(createdChannel.Expiration/1000, 0)
	}
	logger.Debug().Time("expiration_time", expiration).Msg("Calculated channel expiration time")

	// Save the notification channel details to our database
	notificationChannel := &database.NotificationChannel{
		ID:         createdChannel.Id,
		ResourceID: createdChannel.ResourceId,
		CalendarID: s.calendarID,
		Expiration: expiration,
	}

	logger.Debug().Msg("Saving notification channel details to database")
	if err := s.tokenStore.SaveNotificationChannel(notificationChannel); err != nil {
		logger.Error().Err(err).Msg("Failed to save notification channel details to database")
		// Try to stop the channel we just created if we couldn't save it
		logger.Warn().Msg("Attempting to stop the newly created Google channel due to DB save failure")
		stopErr := s.StopNotificationChannel(ctx, createdChannel.Id, createdChannel.ResourceId)
		if stopErr != nil {
			logger.Error().Err(stopErr).Msg("Failed to stop the Google channel after DB save failure")
		} else {
			logger.Info().Msg("Successfully stopped the Google channel after DB save failure")
		}
		return fmt.Errorf("failed to save notification channel: %w", err)
	}

	logger.Info().Msg("Notification channel setup completed successfully")
	return nil
}

// StopNotificationChannel stops a notification channel
func (s *Service) StopNotificationChannel(ctx context.Context, channelID, resourceID string) error {
	logger := s.logger.With().Str("channel_id", channelID).Str("resource_id", resourceID).Logger()
	logger.Info().Msg("Stopping notification channel...")

	// Get latest token in case it was refreshed
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get valid token for stopping notification")
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		logger.Error().Msg("No valid token available for stopping notification")
		return fmt.Errorf("no valid token available")
	}

	// Stop the channel via Google API
	channel := &calendar.Channel{
		Id:         channelID,
		ResourceId: resourceID,
	}

	logger.Debug().Msg("Sending stop channel request to Google API")
	err = s.srv.Channels.Stop(channel).Do()
	if err != nil {
		// Log error but continue to attempt DB deletion
		logger.Error().Err(err).Msg("Failed to stop notification channel via Google API")
		// Return error immediately? Or try DB delete first? Let's try DB delete.
		// return fmt.Errorf("failed to stop notification channel: %w", err)
	} else {
		logger.Info().Msg("Successfully stopped notification channel via Google API")
	}

	// Delete the notification channel from the database regardless of Google API result
	logger.Debug().Msg("Deleting notification channel from database")
	if err := s.tokenStore.DeleteNotificationChannel(channelID); err != nil {
		logger.Error().Err(err).Msg("Failed to delete notification channel from database")
		return fmt.Errorf("failed to delete notification channel from database: %w", err)
	}
	logger.Info().Msg("Successfully deleted notification channel from database")

	// If Google API stop failed but DB delete succeeded, return the Google API error
	if err != nil {
		return fmt.Errorf("failed to stop notification channel via Google API: %w (DB record deleted)", err)
	}

	logger.Info().Msg("Notification channel stopped and deleted successfully")
	return nil
}

// StopAllNotificationChannels stops all active notification channels
func (s *Service) StopAllNotificationChannels(ctx context.Context) error {
	s.logger.Info().Msg("Stopping all active notification channels...")
	// Get latest token in case it was refreshed
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get valid token for stopping all notifications")
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		s.logger.Error().Msg("No valid token available for stopping all notifications")
		return fmt.Errorf("no valid token available")
	}

	// Get all active notification channels
	s.logger.Debug().Msg("Fetching active notification channels from database")
	activeChannels, err := s.tokenStore.GetActiveNotificationChannels()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get active notification channels from database")
		return fmt.Errorf("failed to get active notification channels: %w", err)
	}
	s.logger.Info().Int("channel_count", len(activeChannels)).Msg("Found active channels to stop")

	if len(activeChannels) == 0 {
		s.logger.Info().Msg("No active channels found to stop.")
		return nil
	}

	var firstErr error
	// Stop each channel
	for _, channel := range activeChannels {
		stopErr := s.StopNotificationChannel(ctx, channel.ID, channel.ResourceID)
		if stopErr != nil {
			// Log the error but continue with other channels
			s.logger.Warn().Err(stopErr).Str("channel_id", channel.ID).Msg("Failed to stop notification channel during StopAll operation")
			if firstErr == nil {
				firstErr = stopErr // Keep track of the first error
			}
		}
	}

	if firstErr != nil {
		s.logger.Error().Err(firstErr).Msg("Errors occurred while stopping all notification channels")
		return fmt.Errorf("one or more errors occurred while stopping notification channels: %w", firstErr)
	}

	s.logger.Info().Msg("Successfully stopped all active notification channels")
	return nil
}

// VerifyNotificationChannel checks if a notification channel is still active with Google Calendar
func (s *Service) VerifyNotificationChannel(ctx context.Context, channelID, resourceID string) (bool, error) {
	logger := s.logger.With().Str("channel_id", channelID).Str("resource_id", resourceID).Logger()
	logger.Debug().Msg("Verifying notification channel with Google Calendar API")

	// Get latest token in case it was refreshed
	token, err := s.tokenManager.GetValidToken(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get valid token for channel verification")
		return false, fmt.Errorf("failed to get token: %w", err)
	}
	if token == nil {
		logger.Error().Msg("No valid token available for channel verification")
		return false, fmt.Errorf("no valid token available")
	}

	// Get channel details using the Google Calendar API
	// Unfortunately, the API doesn't provide a direct method to check channel status
	// We need to use an indirect approach - try to list events with the channel's watchFilter

	// Set a unique identifier to add to the request
	verificationTag := fmt.Sprintf("verify-channel-%d", time.Now().UnixNano())

	// List events with a filter that includes this channel's resource ID
	// We include a unique tag to make this a unique request
	// We limit to 1 event just to minimize data transfer
	listCall := s.srv.Events.List(s.calendarID).
		MaxResults(1).
		ShowDeleted(false).
		SingleEvents(true)

	// Add resource ID and verification tag as custom headers
	listCall.Header().Add("X-Goog-Channel-ID", channelID)
	listCall.Header().Add("X-Goog-Resource-ID", resourceID)
	listCall.Header().Add("X-Verification-Tag", verificationTag)

	// Execute the request
	_, err = listCall.Do()

	// If we get a 404 Not Found error with a specific message about the channel,
	// this indicates the channel is no longer active
	if err != nil {
		logger.Warn().Err(err).Msg("Error when verifying channel")
		// Check error message for indications that the channel doesn't exist
		errStr := err.Error()
		if strings.Contains(errStr, "Channel not found") ||
			strings.Contains(errStr, "Channel ID not found") ||
			strings.Contains(errStr, "Resource ID not found") {
			logger.Info().Msg("Channel verification failed - channel not active with Google Calendar")
			return false, nil
		}
		// For other errors, we can't determine the channel state
		return false, fmt.Errorf("failed to verify channel: %w", err)
	}

	// If we reach here with no error, the channel is likely active
	logger.Info().Msg("Channel verification passed - channel appears to be active with Google Calendar")
	return true, nil
}
