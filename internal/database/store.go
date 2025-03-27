package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
)

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

func initTokenTable(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS oauth_tokens (
id INTEGER PRIMARY KEY,
token_data BLOB NOT NULL,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS calendar_settings (
id INTEGER PRIMARY KEY,
calendar_id TEXT NOT NULL,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`)

	return err
}
