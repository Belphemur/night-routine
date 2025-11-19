package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/rs/zerolog"
)

// ConfigParents represents parent configuration
type ConfigParents struct {
	ID        int64
	ParentA   string
	ParentB   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConfigSchedule represents schedule configuration
type ConfigSchedule struct {
	ID                     int64
	UpdateFrequency        string
	LookAheadDays          int
	PastEventThresholdDays int
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// ConfigStore handles configuration storage in SQLite
type ConfigStore struct {
	db     *sql.DB
	logger zerolog.Logger
}

// NewConfigStore creates a new config store
func NewConfigStore(db *DB) (*ConfigStore, error) {
	logger := logging.GetLogger("config-store")
	return &ConfigStore{db: db.Conn(), logger: logger}, nil
}

// GetParents retrieves parent configuration
func (s *ConfigStore) GetParents() (parentA, parentB string, err error) {
	s.logger.Debug().Msg("Retrieving parent configuration")
	err = s.db.QueryRow(`
		SELECT parent_a, parent_b
		FROM config_parents
		WHERE id = 1
	`).Scan(&parentA, &parentB)

	if err == sql.ErrNoRows {
		s.logger.Debug().Msg("No parent configuration found in database")
		return "", "", fmt.Errorf("no parent configuration found")
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to retrieve parent configuration")
		return "", "", fmt.Errorf("failed to retrieve parent configuration: %w", err)
	}

	s.logger.Debug().Str("parent_a", parentA).Str("parent_b", parentB).Msg("Parent configuration retrieved")
	return parentA, parentB, nil
}

// GetParentsFull retrieves full parent configuration with metadata
func (s *ConfigStore) GetParentsFull() (*ConfigParents, error) {
	s.logger.Debug().Msg("Retrieving full parent configuration")
	var config ConfigParents
	err := s.db.QueryRow(`
		SELECT id, parent_a, parent_b, created_at, updated_at
		FROM config_parents
		WHERE id = 1
	`).Scan(&config.ID, &config.ParentA, &config.ParentB, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		s.logger.Debug().Msg("No parent configuration found in database")
		return nil, nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to retrieve parent configuration")
		return nil, fmt.Errorf("failed to retrieve parent configuration: %w", err)
	}

	s.logger.Debug().Str("parent_a", config.ParentA).Str("parent_b", config.ParentB).Msg("Full parent configuration retrieved")
	return &config, nil
}

// SaveParents saves or updates parent configuration
func (s *ConfigStore) SaveParents(parentA, parentB string) error {
	if parentA == "" || parentB == "" {
		return fmt.Errorf("parent names cannot be empty")
	}
	if parentA == parentB {
		return fmt.Errorf("parent names must be different")
	}

	s.logger.Debug().Str("parent_a", parentA).Str("parent_b", parentB).Msg("Saving parent configuration")
	_, err := s.db.Exec(`
		INSERT INTO config_parents (id, parent_a, parent_b, updated_at)
		VALUES (1, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			parent_a = excluded.parent_a,
			parent_b = excluded.parent_b,
			updated_at = CURRENT_TIMESTAMP
	`, parentA, parentB)

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to save parent configuration")
		return fmt.Errorf("failed to save parent configuration: %w", err)
	}

	s.logger.Info().Msg("Parent configuration saved successfully")
	return nil
}

// GetAvailability retrieves unavailable days for a parent
func (s *ConfigStore) GetAvailability(parent string) ([]string, error) {
	if parent != "parent_a" && parent != "parent_b" {
		return nil, fmt.Errorf("invalid parent identifier: %s", parent)
	}

	s.logger.Debug().Str("parent", parent).Msg("Retrieving availability configuration")
	rows, err := s.db.Query(`
		SELECT unavailable_day
		FROM config_availability
		WHERE parent = ?
		ORDER BY unavailable_day
	`, parent)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to query availability")
		return nil, fmt.Errorf("failed to retrieve availability: %w", err)
	}
	defer rows.Close()

	var days []string
	for rows.Next() {
		var day string
		if err := rows.Scan(&day); err != nil {
			s.logger.Error().Err(err).Msg("Failed to scan availability row")
			return nil, fmt.Errorf("failed to scan availability: %w", err)
		}
		days = append(days, day)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error().Err(err).Msg("Error iterating availability rows")
		return nil, fmt.Errorf("error iterating availability: %w", err)
	}

	s.logger.Debug().Str("parent", parent).Int("count", len(days)).Msg("Availability retrieved")
	return days, nil
}

// SaveAvailability saves unavailable days for a parent
func (s *ConfigStore) SaveAvailability(parent string, unavailableDays []string) error {
	if parent != "parent_a" && parent != "parent_b" {
		return fmt.Errorf("invalid parent identifier: %s", parent)
	}

	s.logger.Debug().Str("parent", parent).Int("day_count", len(unavailableDays)).Msg("Saving availability configuration")

	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to begin transaction")
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Rollback is safe to call even after Commit
	}()

	// Delete existing availability for this parent
	_, err = tx.Exec(`DELETE FROM config_availability WHERE parent = ?`, parent)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to delete existing availability")
		return fmt.Errorf("failed to delete existing availability: %w", err)
	}

	// Insert new availability
	stmt, err := tx.Prepare(`INSERT INTO config_availability (parent, unavailable_day) VALUES (?, ?)`)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to prepare insert statement")
		return fmt.Errorf("failed to prepare insert: %w", err)
	}
	defer stmt.Close()

	// Validate day values
	for _, day := range unavailableDays {
		if !constants.IsValidDayOfWeek(day) {
			s.logger.Error().Str("day", day).Msg("Invalid day of week")
			return fmt.Errorf("invalid day of week: %s", day)
		}
		if _, err := stmt.Exec(parent, day); err != nil {
			s.logger.Error().Err(err).Str("day", day).Msg("Failed to insert availability")
			return fmt.Errorf("failed to insert availability for %s: %w", day, err)
		}
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error().Err(err).Msg("Failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info().Str("parent", parent).Msg("Availability configuration saved successfully")
	return nil
}

// GetSchedule retrieves schedule configuration
func (s *ConfigStore) GetSchedule() (updateFrequency string, lookAheadDays, pastEventThresholdDays int, err error) {
	s.logger.Debug().Msg("Retrieving schedule configuration")
	err = s.db.QueryRow(`
		SELECT update_frequency, look_ahead_days, past_event_threshold_days
		FROM config_schedule
		WHERE id = 1
	`).Scan(&updateFrequency, &lookAheadDays, &pastEventThresholdDays)

	if err == sql.ErrNoRows {
		s.logger.Debug().Msg("No schedule configuration found in database")
		return "", 0, 0, fmt.Errorf("no schedule configuration found")
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to retrieve schedule configuration")
		return "", 0, 0, fmt.Errorf("failed to retrieve schedule configuration: %w", err)
	}

	s.logger.Debug().
		Str("update_frequency", updateFrequency).
		Int("look_ahead_days", lookAheadDays).
		Int("past_event_threshold_days", pastEventThresholdDays).
		Msg("Schedule configuration retrieved")
	return updateFrequency, lookAheadDays, pastEventThresholdDays, nil
}

// GetScheduleFull retrieves full schedule configuration with metadata
func (s *ConfigStore) GetScheduleFull() (*ConfigSchedule, error) {
	s.logger.Debug().Msg("Retrieving full schedule configuration")
	var config ConfigSchedule
	err := s.db.QueryRow(`
		SELECT id, update_frequency, look_ahead_days, past_event_threshold_days, created_at, updated_at
		FROM config_schedule
		WHERE id = 1
	`).Scan(&config.ID, &config.UpdateFrequency, &config.LookAheadDays, &config.PastEventThresholdDays, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		s.logger.Debug().Msg("No schedule configuration found in database")
		return nil, nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to retrieve schedule configuration")
		return nil, fmt.Errorf("failed to retrieve schedule configuration: %w", err)
	}

	s.logger.Debug().
		Str("update_frequency", config.UpdateFrequency).
		Int("look_ahead_days", config.LookAheadDays).
		Int("past_event_threshold_days", config.PastEventThresholdDays).
		Msg("Full schedule configuration retrieved")
	return &config, nil
}

// SaveSchedule saves or updates schedule configuration
func (s *ConfigStore) SaveSchedule(updateFrequency string, lookAheadDays, pastEventThresholdDays int) error {
	// Validate inputs
	if updateFrequency != "daily" && updateFrequency != "weekly" && updateFrequency != "monthly" {
		return fmt.Errorf("invalid update frequency: %s", updateFrequency)
	}
	if lookAheadDays < 1 {
		return fmt.Errorf("look ahead days must be positive")
	}
	if pastEventThresholdDays < 0 {
		return fmt.Errorf("past event threshold days cannot be negative")
	}

	s.logger.Debug().
		Str("update_frequency", updateFrequency).
		Int("look_ahead_days", lookAheadDays).
		Int("past_event_threshold_days", pastEventThresholdDays).
		Msg("Saving schedule configuration")

	_, err := s.db.Exec(`
		INSERT INTO config_schedule (id, update_frequency, look_ahead_days, past_event_threshold_days, updated_at)
		VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			update_frequency = excluded.update_frequency,
			look_ahead_days = excluded.look_ahead_days,
			past_event_threshold_days = excluded.past_event_threshold_days,
			updated_at = CURRENT_TIMESTAMP
	`, updateFrequency, lookAheadDays, pastEventThresholdDays)

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to save schedule configuration")
		return fmt.Errorf("failed to save schedule configuration: %w", err)
	}

	s.logger.Info().Msg("Schedule configuration saved successfully")
	return nil
}

// HasConfiguration checks if any configuration exists in the database
func (s *ConfigStore) HasConfiguration() (bool, error) {
	s.logger.Debug().Msg("Checking if configuration exists")
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM config_parents WHERE id = 1`).Scan(&count)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to check configuration existence")
		return false, fmt.Errorf("failed to check configuration: %w", err)
	}

	exists := count > 0
	s.logger.Debug().Bool("exists", exists).Msg("Configuration existence checked")
	return exists, nil
}
