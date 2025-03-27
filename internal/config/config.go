package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration
type Config struct {
	Parents      ParentsConfig      `toml:"parents"`
	Availability AvailabilityConfig `toml:"availability"`
	Schedule     ScheduleConfig     `toml:"schedule"`
	Service      ServiceConfig      `toml:"service"`
	OAuth        *OAuthConfig       // From environment
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

// OAuthConfig holds the Google OAuth configuration from environment
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Load reads the configuration file and environment variables
func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}

	// Ensure the state file path is absolute
	if !filepath.IsAbs(cfg.Service.StateFile) {
		configDir := filepath.Dir(path)
		cfg.Service.StateFile = filepath.Join(configDir, "..", cfg.Service.StateFile)
	}

	// Load OAuth config from environment
	cfg.OAuth = &OAuthConfig{
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate checks if the configuration is valid
func validate(cfg *Config) error {
	if cfg.Parents.ParentA == "" || cfg.Parents.ParentB == "" {
		return fmt.Errorf("both parent names are required")
	}

	if cfg.Parents.ParentA == cfg.Parents.ParentB {
		return fmt.Errorf("parent names must be different")
	}

	switch cfg.Schedule.UpdateFrequency {
	case "daily", "weekly", "monthly":
	// Valid frequencies
	default:
		return fmt.Errorf("invalid update frequency: %s", cfg.Schedule.UpdateFrequency)
	}

	if cfg.Schedule.LookAheadDays < 1 {
		return fmt.Errorf("look ahead days must be positive")
	}

	// Validate OAuth configuration
	if cfg.OAuth.ClientID == "" {
		return fmt.Errorf("GOOGLE_OAUTH_CLIENT_ID environment variable is required")
	}
	if cfg.OAuth.ClientSecret == "" {
		return fmt.Errorf("GOOGLE_OAUTH_CLIENT_SECRET environment variable is required")
	}
	if cfg.OAuth.RedirectURL == "" {
		return fmt.Errorf("GOOGLE_OAUTH_REDIRECT_URL environment variable is required")
	}

	return nil
}
