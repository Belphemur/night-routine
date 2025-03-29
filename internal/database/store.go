package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/logging"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

// NotificationChannel represents a Google Calendar notification channel
type NotificationChannel struct {
	ID         string
	ResourceID string
	CalendarID string
	Expiration time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TokenStore handles OAuth token storage in SQLite
type TokenStore struct {
	db     *sql.DB
	logger zerolog.Logger
}

// NewTokenStore creates a new token store
func NewTokenStore(db *DB) (*TokenStore, error) {
	logger := logging.GetLogger("token-store")
	return &TokenStore{db: db.Conn(), logger: logger}, nil
}

// SaveToken implements the TokenSaver interface
func (s *TokenStore) SaveToken(token *oauth2.Token) error {
	s.logger.Debug().Msg("Saving OAuth token") // Changed to Debug
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to marshal token to JSON") // Changed to Debug
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	s.logger.Debug().Msg("Executing query to save token")
	_, err = s.db.Exec(`
	INSERT OR REPLACE INTO oauth_tokens (id, token_data)
	VALUES (1, ?)`, tokenJSON)
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to execute save token query") // Changed to Debug
		return fmt.Errorf("failed to save token: %w", err)
	}

	s.logger.Debug().Msg("OAuth token saved successfully") // Changed to Debug
	return nil
}

// GetToken retrieves the saved OAuth token
func (s *TokenStore) GetToken() (*oauth2.Token, error) {
	s.logger.Debug().Msg("Retrieving OAuth token")
	var tokenJSON []byte
	err := s.db.QueryRow(`
	SELECT token_data FROM oauth_tokens WHERE id = 1
	`).Scan(&tokenJSON)
	if err == sql.ErrNoRows {
		s.logger.Debug().Msg("No OAuth token found in store") // Changed to Debug
		return nil, nil
	}
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to retrieve token from database") // Changed to Debug
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	var token oauth2.Token
	s.logger.Debug().Msg("Unmarshalling token data")
	if err := json.Unmarshal(tokenJSON, &token); err != nil {
		s.logger.Debug().Err(err).Msg("Failed to unmarshal token JSON") // Changed to Debug
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	s.logger.Debug().Msg("OAuth token retrieved successfully") // Changed to Debug
	return &token, nil
}

// ClearToken removes the saved OAuth token
func (s *TokenStore) ClearToken() error {
	s.logger.Debug().Msg("Clearing OAuth token") // Changed to Debug
	_, err := s.db.Exec(`DELETE FROM oauth_tokens WHERE id = 1`)
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to execute clear token query") // Changed to Debug
		return fmt.Errorf("failed to clear token: %w", err)
	}
	s.logger.Debug().Msg("OAuth token cleared successfully") // Changed to Debug
	return nil
}

// SaveSelectedCalendar saves the selected calendar ID
func (s *TokenStore) SaveSelectedCalendar(calendarID string) error {
	saveLogger := s.logger.With().Str("calendar_id", calendarID).Logger()
	saveLogger.Debug().Msg("Saving selected calendar ID") // Changed to Debug
	_, err := s.db.Exec(`
	INSERT OR REPLACE INTO calendar_settings (id, calendar_id)
	VALUES (1, ?)`, calendarID)
	if err != nil {
		saveLogger.Debug().Err(err).Msg("Failed to execute save calendar ID query") // Changed to Debug
		return fmt.Errorf("failed to save calendar ID: %w", err)
	}
	saveLogger.Debug().Msg("Selected calendar ID saved successfully") // Changed to Debug
	return nil
}

// GetSelectedCalendar retrieves the saved calendar ID
func (s *TokenStore) GetSelectedCalendar() (string, error) {
	s.logger.Debug().Msg("Retrieving selected calendar ID")
	var calendarID string
	err := s.db.QueryRow(`
	SELECT calendar_id FROM calendar_settings WHERE id = 1
	`).Scan(&calendarID)
	if err == sql.ErrNoRows {
		s.logger.Debug().Msg("No selected calendar ID found") // Changed to Debug
		return "", nil
	}
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to retrieve selected calendar ID") // Changed to Debug
		return "", fmt.Errorf("failed to retrieve calendar ID: %w", err)
	}
	s.logger.Debug().Str("calendar_id", calendarID).Msg("Selected calendar ID retrieved successfully") // Changed to Debug
	return calendarID, nil
}

// SaveNotificationChannel saves a notification channel
func (s *TokenStore) SaveNotificationChannel(channel *NotificationChannel) error {
	saveLogger := s.logger.With().
		Str("channel_id", channel.ID).
		Str("resource_id", channel.ResourceID).
		Str("calendar_id", channel.CalendarID).
		Time("expiration", channel.Expiration).
		Logger()
	saveLogger.Debug().Msg("Saving notification channel") // Changed to Debug
	_, err := s.db.Exec(`
	INSERT OR REPLACE INTO notification_channels (id, resource_id, calendar_id, expiration)
	VALUES (?, ?, ?, ?)`,
		channel.ID, channel.ResourceID, channel.CalendarID, channel.Expiration.Format(time.RFC3339))
	if err != nil {
		saveLogger.Debug().Err(err).Msg("Failed to execute save notification channel query") // Changed to Debug
		return fmt.Errorf("failed to save notification channel: %w", err)
	}
	saveLogger.Debug().Msg("Notification channel saved successfully") // Changed to Debug
	return nil
}

// GetNotificationChannelByID retrieves a notification channel by its ID
func (s *TokenStore) GetNotificationChannelByID(id string) (*NotificationChannel, error) {
	getLogger := s.logger.With().Str("channel_id", id).Logger()
	getLogger.Debug().Msg("Retrieving notification channel by ID")
	if id == "" {
		getLogger.Debug().Msg("Empty channel ID provided") // Changed to Debug
		return nil, nil
	}

	var channel NotificationChannel
	var expirationStr, createdAtStr, updatedAtStr string

	err := s.db.QueryRow(`
	SELECT id, resource_id, calendar_id, expiration, created_at, updated_at
	FROM notification_channels
	WHERE id = ?`, id).Scan(
		&channel.ID,
		&channel.ResourceID,
		&channel.CalendarID,
		&expirationStr,
		&createdAtStr,
		&updatedAtStr,
	)

	if err == sql.ErrNoRows {
		getLogger.Debug().Msg("Notification channel not found") // Changed to Debug
		return nil, nil
	}
	if err != nil {
		getLogger.Debug().Err(err).Msg("Failed to retrieve notification channel") // Changed to Debug
		return nil, fmt.Errorf("failed to retrieve notification channel: %w", err)
	}

	expiration, err := time.Parse(time.RFC3339, expirationStr)
	if err != nil {
		getLogger.Debug().Err(err).Str("expiration_string", expirationStr).Msg("Failed to parse expiration date") // Changed to Debug
		return nil, fmt.Errorf("failed to parse expiration date: %w", err)
	}
	channel.Expiration = expiration

	createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		channel.CreatedAt = createdAt
	} else {
		getLogger.Debug().Err(err).Str("timestamp_string", createdAtStr).Msg("Failed to parse created_at timestamp") // Changed to Debug
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err == nil {
		channel.UpdatedAt = updatedAt
	} else {
		getLogger.Debug().Err(err).Str("timestamp_string", updatedAtStr).Msg("Failed to parse updated_at timestamp") // Changed to Debug
	}

	getLogger.Debug().Msg("Notification channel retrieved successfully")
	return &channel, nil
}

// GetActiveNotificationChannels retrieves all active notification channels
func (s *TokenStore) GetActiveNotificationChannels() ([]*NotificationChannel, error) {
	s.logger.Debug().Msg("Retrieving active notification channels")
	rows, err := s.db.Query(`
	SELECT id, resource_id, calendar_id, expiration, created_at, updated_at
	FROM notification_channels
	WHERE expiration > datetime('now')
	ORDER BY expiration ASC`)
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to query active notification channels") // Changed to Debug
		return nil, fmt.Errorf("failed to query notification channels: %w", err)
	}
	defer rows.Close()

	var channels []*NotificationChannel
	for rows.Next() {
		var channel NotificationChannel
		var expirationStr, createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&channel.ID,
			&channel.ResourceID,
			&channel.CalendarID,
			&expirationStr,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			s.logger.Debug().Err(err).Msg("Failed to scan notification channel row") // Changed to Debug
			return nil, fmt.Errorf("failed to scan notification channel: %w", err)
		}

		expiration, err := time.Parse(time.RFC3339, expirationStr)
		if err != nil {
			s.logger.Debug().Err(err).Str("expiration_string", expirationStr).Str("channel_id", channel.ID).Msg("Failed to parse expiration date for channel") // Changed to Debug
			return nil, fmt.Errorf("failed to parse expiration date: %w", err)
		}
		channel.Expiration = expiration

		createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err == nil {
			channel.CreatedAt = createdAt
		} else {
			s.logger.Debug().Err(err).Str("timestamp_string", createdAtStr).Str("channel_id", channel.ID).Msg("Failed to parse created_at timestamp") // Changed to Debug
		}

		updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err == nil {
			channel.UpdatedAt = updatedAt
		} else {
			s.logger.Debug().Err(err).Str("timestamp_string", updatedAtStr).Str("channel_id", channel.ID).Msg("Failed to parse updated_at timestamp") // Changed to Debug
		}

		channels = append(channels, &channel)
	}
	if err := rows.Err(); err != nil {
		s.logger.Debug().Err(err).Msg("Error iterating active notification channel rows") // Changed to Debug
		return nil, fmt.Errorf("failed during row iteration: %w", err)
	}

	s.logger.Debug().Int("count", len(channels)).Msg("Active notification channels retrieved successfully")
	return channels, nil
}

// DeleteNotificationChannel deletes a notification channel by its ID
func (s *TokenStore) DeleteNotificationChannel(id string) error {
	deleteLogger := s.logger.With().Str("channel_id", id).Logger()
	deleteLogger.Debug().Msg("Deleting notification channel") // Changed to Debug
	result, err := s.db.Exec(`DELETE FROM notification_channels WHERE id = ?`, id)
	if err != nil {
		deleteLogger.Debug().Err(err).Msg("Failed to execute delete notification channel query") // Changed to Debug
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	deleteLogger.Debug().Int64("rows_affected", rowsAffected).Msg("Notification channel deleted successfully") // Changed to Debug
	return nil
}

// DeleteExpiredNotificationChannels deletes all expired notification channels
func (s *TokenStore) DeleteExpiredNotificationChannels() error {
	s.logger.Debug().Msg("Deleting expired notification channels") // Changed to Debug
	result, err := s.db.Exec(`DELETE FROM notification_channels WHERE expiration <= datetime('now')`)
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to execute delete expired notification channels query") // Changed to Debug
		return fmt.Errorf("failed to delete expired notification channels: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	s.logger.Debug().Int64("rows_affected", rowsAffected).Msg("Expired notification channels deleted successfully") // Changed to Debug
	return nil
}
