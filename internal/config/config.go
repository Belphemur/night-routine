package config

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration
type Config struct {
	Parents      ParentsConfig      `toml:"parents"`
	Availability AvailabilityConfig `toml:"availability"`
	Schedule     ScheduleConfig     `toml:"schedule"`
	Service      ServiceConfig      `toml:"service"`
	Google       GoogleConfig       `toml:"google"`
}

// ParentsConfig holds the parent names
type ParentsConfig struct {
	ParentA string `toml:"parent_a"`
	ParentB string `toml:"parent_b"`
}

// AvailabilityConfig holds the unavailability schedule for each parent
type AvailabilityConfig struct {
	ParentAUnavailable []string `toml:"parent_a_unavailable"`
	ParentBUnavailable []string `toml:"parent_b_unavailable"`
}

// ScheduleConfig holds the scheduling parameters
type ScheduleConfig struct {
	UpdateFrequency string `toml:"update_frequency"`
	CalendarID      string `toml:"calendar_id"`
	LookAheadDays   int    `toml:"look_ahead_days"`
}

// ServiceConfig holds the service configuration
type ServiceConfig struct {
	Port      int    `toml:"port"`
	StateFile string `toml:"state_file"`
}

// GoogleConfig holds the Google Calendar API configuration
type GoogleConfig struct {
	CredentialsFile string `toml:"credentials_file"`
	TokenFile       string `toml:"token_file"`
}

// Load reads the configuration file and returns a Config struct
func Load(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validate checks if the configuration is valid
func validate(config *Config) error {
	if config.Parents.ParentA == "" || config.Parents.ParentB == "" {
		return fmt.Errorf("both parent names are required")
	}

	if config.Parents.ParentA == config.Parents.ParentB {
		return fmt.Errorf("parent names must be different")
	}

	// Validate update frequency
	switch config.Schedule.UpdateFrequency {
	case "daily", "weekly", "monthly":
	// Valid frequencies
	default:
		return fmt.Errorf("invalid update frequency: %s", config.Schedule.UpdateFrequency)
	}

	if config.Schedule.LookAheadDays < 1 {
		return fmt.Errorf("look ahead days must be positive")
	}

	// Validate days of week
	for _, day := range config.Availability.ParentAUnavailable {
		if _, err := time.Parse("Monday", day); err != nil {
			return fmt.Errorf("invalid day of week for ParentA: %s", day)
		}
	}

	for _, day := range config.Availability.ParentBUnavailable {
		if _, err := time.Parse("Monday", day); err != nil {
			return fmt.Errorf("invalid day of week for ParentB: %s", day)
		}
	}

	return nil
}
