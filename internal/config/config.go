package config

import (
	"fmt"
	"net/url"
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
	App          ApplicationConfig  `toml:"app"`
	OAuth        *oauth2.Config     // Replaced with Google OAuth2 Config
}

// ApplicationConfig holds the application configuration
type ApplicationConfig struct {
	Port      int    `toml:"port"`       // Port to listen on
	AppUrl    string `toml:"app_url"`    // Application URL for internal use (OAuth, etc.)
	PublicUrl string `toml:"public_url"` // Public URL for external access (webhooks)
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
	UpdateFrequency        string `toml:"update_frequency"`
	CalendarID             string `toml:"calendar_id"`
	LookAheadDays          int    `toml:"look_ahead_days"`
	PastEventThresholdDays int    `toml:"past_event_threshold_days"`
}

// ServiceConfig holds the service configuration
type ServiceConfig struct {
	StateFile           string `toml:"state_file"`
	LogLevel            string `toml:"log_level"`              // New field for log level
	ManualSyncOnStartup bool   `toml:"manual_sync_on_startup"` // Perform a sync on startup if token exists
}

// Load reads the configuration file and environment variables
func Load(path string) (*Config, error) {
	var cfg Config
	// Set defaults before decoding
	cfg.Service.ManualSyncOnStartup = true  // Default to true
	cfg.Schedule.PastEventThresholdDays = 5 // Default to 5 days

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}

	// Ensure the state file path is absolute
	if !filepath.IsAbs(cfg.Service.StateFile) {
		configDir := filepath.Dir(path)
		cfg.Service.StateFile = filepath.Join(configDir, "..", cfg.Service.StateFile)
	}

	// Set default port and allow override from environment
	if cfg.App.Port == 0 {
		cfg.App.Port = 8888 // Default port
	}
	if port := os.Getenv("PORT"); port != "" {
		portNum := 0
		if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
			return nil, fmt.Errorf("PORT must be a valid number: %v", err)
		}
		cfg.App.Port = portNum
	}

	// Load OAuth config essentials from environment (needed for validation)
	cfg.OAuth = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		// RedirectURL, Scopes, Endpoint will be set after validation and potential AppUrl default
	}

	// Set default LogLevel before validation if not provided
	if cfg.Service.LogLevel == "" {
		cfg.Service.LogLevel = "info"
	}

	// Validate the configuration loaded so far (including URLs from TOML)
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	// Now set the final OAuth details using the validated AppUrl
	cfg.OAuth.RedirectURL = cfg.App.AppUrl + "/oauth/callback"
	cfg.OAuth.Scopes = []string{
		calendar.CalendarEventsScope,
		calendar.CalendarCalendarlistReadonlyScope,
	}
	cfg.OAuth.Endpoint = google.Endpoint

	return &cfg, nil
}

// validate checks if the configuration is valid
func validate(cfg *Config) error {
	// Validate parent names
	if cfg.Parents.ParentA == "" || cfg.Parents.ParentB == "" {
		return fmt.Errorf("both parent names are required")
	}

	if cfg.Parents.ParentA == cfg.Parents.ParentB {
		return fmt.Errorf("parent names must be different")
	}

	// Validate schedule configuration
	switch cfg.Schedule.UpdateFrequency {
	case "daily", "weekly", "monthly":
		// Valid frequencies
	default:
		return fmt.Errorf("invalid update frequency: %s", cfg.Schedule.UpdateFrequency)
	}

	if cfg.Schedule.LookAheadDays < 1 {
		return fmt.Errorf("look ahead days must be positive")
	}

	// Validate application URLs
	if cfg.App.AppUrl == "" {
		return fmt.Errorf("app_url is required in [app] configuration")
	}
	if _, err := url.ParseRequestURI(cfg.App.AppUrl); err != nil {
		return fmt.Errorf("invalid app_url '%s': %w", cfg.App.AppUrl, err)
	}

	if cfg.App.PublicUrl == "" {
		return fmt.Errorf("public_url is required in [app] configuration")
	}
	if _, err := url.ParseRequestURI(cfg.App.PublicUrl); err != nil {
		return fmt.Errorf("invalid public_url '%s': %w", cfg.App.PublicUrl, err)
	}

	// Validate OAuth configuration
	if cfg.OAuth.ClientID == "" {
		return fmt.Errorf("GOOGLE_OAUTH_CLIENT_ID environment variable is required")
	}
	if cfg.OAuth.ClientSecret == "" {
		return fmt.Errorf("GOOGLE_OAUTH_CLIENT_SECRET environment variable is required")
	}

	return nil
}
