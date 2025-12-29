package database

import (
	"fmt"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/rs/zerolog"
)

// ConfigSeeder handles seeding configuration from TOML to database
type ConfigSeeder struct {
	store  *ConfigStore
	logger zerolog.Logger
}

// NewConfigSeeder creates a new config seeder
func NewConfigSeeder(store *ConfigStore) *ConfigSeeder {
	return &ConfigSeeder{
		store:  store,
		logger: logging.GetLogger("config-seeder"),
	}
}

// SeedFromConfig seeds the database with configuration from TOML file
// This is called on every startup and handles both initial setup and migration:
// - On initial setup: Seeds all config from TOML
// - On upgrade: Migrates existing TOML config to new DB tables
// - On normal startup: Skips if DB config already exists
func (s *ConfigSeeder) SeedFromConfig(cfg *config.Config) error {
	s.logger.Info().Msg("Checking if configuration needs seeding/migration")

	// Check if configuration already exists
	hasConfig, err := s.store.HasConfiguration()
	if err != nil {
		return fmt.Errorf("failed to check existing configuration: %w", err)
	}

	if hasConfig {
		s.logger.Info().Msg("Configuration already exists in database, skipping seeding")
		return nil
	}

	s.logger.Info().Msg("No configuration found in database, migrating from TOML config file")

	// Seed parent configuration
	if err := s.seedParents(cfg); err != nil {
		return fmt.Errorf("failed to seed parent configuration: %w", err)
	}

	// Seed availability configuration
	if err := s.seedAvailability(cfg); err != nil {
		return fmt.Errorf("failed to seed availability configuration: %w", err)
	}

	// Seed schedule configuration
	if err := s.seedSchedule(cfg); err != nil {
		return fmt.Errorf("failed to seed schedule configuration: %w", err)
	}

	s.logger.Info().Msg("Configuration migration from TOML completed successfully")
	return nil
}

// seedParents seeds parent names from config
func (s *ConfigSeeder) seedParents(cfg *config.Config) error {
	s.logger.Debug().
		Str("parent_a", cfg.Parents.ParentA).
		Str("parent_b", cfg.Parents.ParentB).
		Msg("Seeding parent configuration")

	if err := s.store.SaveParents(cfg.Parents.ParentA, cfg.Parents.ParentB); err != nil {
		return err
	}

	s.logger.Info().Msg("Parent configuration seeded successfully")
	return nil
}

// seedAvailability seeds availability configuration from config
func (s *ConfigSeeder) seedAvailability(cfg *config.Config) error {
	s.logger.Debug().Msg("Seeding availability configuration")

	// Seed parent A availability
	s.logger.Debug().
		Str("parent", "parent_a").
		Int("unavailable_days", len(cfg.Availability.ParentAUnavailable)).
		Msg("Seeding parent A availability")

	if err := s.store.SaveAvailability("parent_a", cfg.Availability.ParentAUnavailable); err != nil {
		return fmt.Errorf("failed to seed parent A availability: %w", err)
	}

	// Seed parent B availability
	s.logger.Debug().
		Str("parent", "parent_b").
		Int("unavailable_days", len(cfg.Availability.ParentBUnavailable)).
		Msg("Seeding parent B availability")

	if err := s.store.SaveAvailability("parent_b", cfg.Availability.ParentBUnavailable); err != nil {
		return fmt.Errorf("failed to seed parent B availability: %w", err)
	}

	s.logger.Info().Msg("Availability configuration seeded successfully")
	return nil
}

// seedSchedule seeds schedule configuration from config
func (s *ConfigSeeder) seedSchedule(cfg *config.Config) error {
	s.logger.Debug().
		Str("update_frequency", cfg.Schedule.UpdateFrequency).
		Int("look_ahead_days", cfg.Schedule.LookAheadDays).
		Int("past_event_threshold_days", cfg.Schedule.PastEventThresholdDays).
		Str("stats_order", cfg.Schedule.StatsOrder.String()).
		Msg("Seeding schedule configuration")

	if err := s.store.SaveSchedule(
		cfg.Schedule.UpdateFrequency,
		cfg.Schedule.LookAheadDays,
		cfg.Schedule.PastEventThresholdDays,
		cfg.Schedule.StatsOrder,
	); err != nil {
		return err
	}

	s.logger.Info().Msg("Schedule configuration seeded successfully")
	return nil
}
