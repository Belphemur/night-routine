package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// Config holds the application configuration
type Config struct {
	Parents      ParentsConfig      `toml:"parents"`
	Availability AvailabilityConfig `toml:"availability"`
	Schedule     ScheduleConfig     `toml:"schedule"`
	Service      ServiceConfig      `toml:"service"`
	App          *ApplicationConfig // From environment
	OAuth        *oauth2.Config     // Replaced with Google OAuth2 Config
}

// ApplicationConfig holds the application configuration from environment
type ApplicationConfig struct {
	Port int
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
	StateFile string `toml:"state_file"`
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

	// Load application config from environment
	portNum := 8888 // Default port
	if port := os.Getenv("PORT"); port != "" {
		if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
			return nil, fmt.Errorf("PORT must be a valid number: %v", err)
		}
	}
	cfg.App = &ApplicationConfig{
		Port: portNum,
	}

	// Load OAuth config from environment
	cfg.OAuth = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
		Scopes: []string{
			calendar.CalendarEventsScope,
			calendar.CalendarCalendarlistReadonlyScope,
		},
		Endpoint: google.Endpoint,
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
