package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
	db *sql.DB
}

// NewTokenStore creates a new token store
func NewTokenStore(db *DB) (*TokenStore, error) {
	return &TokenStore{db: db.Conn()}, nil
}

// SaveToken implements the TokenSaver interface
func (s *TokenStore) SaveToken(token *oauth2.Token) error {
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	_, err = s.db.Exec(`
INSERT OR REPLACE INTO oauth_tokens (id, token_data)
VALUES (1, ?)`, tokenJSON)
	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// GetToken retrieves the saved OAuth token
func (s *TokenStore) GetToken() (*oauth2.Token, error) {
	var tokenJSON []byte
	err := s.db.QueryRow(`
SELECT token_data FROM oauth_tokens WHERE id = 1
`).Scan(&tokenJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(tokenJSON, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// ClearToken removes the saved OAuth token
func (s *TokenStore) ClearToken() error {
	_, err := s.db.Exec(`DELETE FROM oauth_tokens WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("failed to clear token: %w", err)
	}
	return nil
}

// SaveSelectedCalendar saves the selected calendar ID
func (s *TokenStore) SaveSelectedCalendar(calendarID string) error {
	_, err := s.db.Exec(`
INSERT OR REPLACE INTO calendar_settings (id, calendar_id)
VALUES (1, ?)`, calendarID)
	if err != nil {
		return fmt.Errorf("failed to save calendar ID: %w", err)
	}

	return nil
}

// GetSelectedCalendar retrieves the saved calendar ID
func (s *TokenStore) GetSelectedCalendar() (string, error) {
	var calendarID string
	err := s.db.QueryRow(`
SELECT calendar_id FROM calendar_settings WHERE id = 1
`).Scan(&calendarID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to retrieve calendar ID: %w", err)
	}

	return calendarID, nil
}

// SaveNotificationChannel saves a notification channel
func (s *TokenStore) SaveNotificationChannel(channel *NotificationChannel) error {
	_, err := s.db.Exec(`
INSERT OR REPLACE INTO notification_channels (id, resource_id, calendar_id, expiration)
VALUES (?, ?, ?, ?)`,
		channel.ID, channel.ResourceID, channel.CalendarID, channel.Expiration.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to save notification channel: %w", err)
	}

	return nil
}

// GetNotificationChannelByID retrieves a notification channel by its ID
func (s *TokenStore) GetNotificationChannelByID(id string) (*NotificationChannel, error) {
	if id == "" {
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
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve notification channel: %w", err)
	}

	expiration, err := time.Parse(time.RFC3339, expirationStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expiration date: %w", err)
	}
	channel.Expiration = expiration

	createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		channel.CreatedAt = createdAt
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err == nil {
		channel.UpdatedAt = updatedAt
	}

	return &channel, nil
}

// GetActiveNotificationChannels retrieves all active notification channels
func (s *TokenStore) GetActiveNotificationChannels() ([]*NotificationChannel, error) {
	rows, err := s.db.Query(`
SELECT id, resource_id, calendar_id, expiration, created_at, updated_at
FROM notification_channels
WHERE expiration > datetime('now')
ORDER BY expiration ASC`)
	if err != nil {
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
			return nil, fmt.Errorf("failed to scan notification channel: %w", err)
		}

		expiration, err := time.Parse(time.RFC3339, expirationStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expiration date: %w", err)
		}
		channel.Expiration = expiration

		createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err == nil {
			channel.CreatedAt = createdAt
		}

		updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err == nil {
			channel.UpdatedAt = updatedAt
		}

		channels = append(channels, &channel)
	}

	return channels, nil
}

// DeleteNotificationChannel deletes a notification channel by its ID
func (s *TokenStore) DeleteNotificationChannel(id string) error {
	_, err := s.db.Exec(`DELETE FROM notification_channels WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}
	return nil
}

// DeleteExpiredNotificationChannels deletes all expired notification channels
func (s *TokenStore) DeleteExpiredNotificationChannels() error {
	_, err := s.db.Exec(`DELETE FROM notification_channels WHERE expiration <= datetime('now')`)
	if err != nil {
		return fmt.Errorf("failed to delete expired notification channels: %w", err)
	}
	return nil
}
