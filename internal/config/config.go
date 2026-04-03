package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	ktoml "github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	koanf "github.com/knadh/koanf/v2"

	"github.com/belphemur/night-routine/internal/constants"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// OAuthCredentials holds raw Google OAuth2 credentials.
// These are used to construct the *oauth2.Config after loading and validation.
// Set via NR_OAUTH__CLIENT_ID / NR_OAUTH__CLIENT_SECRET (preferred)
// or the legacy GOOGLE_OAUTH_CLIENT_ID / GOOGLE_OAUTH_CLIENT_SECRET env vars.
type OAuthCredentials struct {
	ClientID     string `koanf:"client_id"`
	ClientSecret string `koanf:"client_secret"`
}

// Config holds the application configuration.
type Config struct {
	Parents      ParentsConfig      `toml:"parents"      koanf:"parents"`
	Availability AvailabilityConfig `toml:"availability" koanf:"availability"`
	Schedule     ScheduleConfig     `toml:"schedule"     koanf:"schedule"`
	Service      ServiceConfig      `toml:"service"      koanf:"service"`
	App          ApplicationConfig  `toml:"app"          koanf:"app"`
	// Credentials holds the raw OAuth2 client ID and secret loaded from environment variables.
	Credentials OAuthCredentials `koanf:"oauth"`
	// OAuth is the fully constructed Google OAuth2 config, built after loading and validation.
	// It is not sourced from config files.
	OAuth *oauth2.Config
}

// ApplicationConfig holds the application server settings.
type ApplicationConfig struct {
	Port      int    `toml:"port"       koanf:"port"`       // Port to listen on
	AppUrl    string `toml:"app_url"    koanf:"app_url"`    // Application URL for internal use (OAuth, etc.)
	PublicUrl string `toml:"public_url" koanf:"public_url"` // Public URL for external access (webhooks)
}

// ParentsConfig holds the parent names.
type ParentsConfig struct {
	ParentA string `toml:"parent_a" koanf:"parent_a"`
	ParentB string `toml:"parent_b" koanf:"parent_b"`
}

// AvailabilityConfig holds the unavailability schedule for each parent.
type AvailabilityConfig struct {
	ParentAUnavailable []string `toml:"parent_a_unavailable" koanf:"parent_a_unavailable"`
	ParentBUnavailable []string `toml:"parent_b_unavailable" koanf:"parent_b_unavailable"`
}

// ScheduleConfig holds the scheduling parameters.
type ScheduleConfig struct {
	UpdateFrequency        string               `toml:"update_frequency"          koanf:"update_frequency"`
	CalendarID             string               `toml:"calendar_id"               koanf:"calendar_id"`
	LookAheadDays          int                  `toml:"look_ahead_days"           koanf:"look_ahead_days"`
	PastEventThresholdDays int                  `toml:"past_event_threshold_days" koanf:"past_event_threshold_days"`
	StatsOrder             constants.StatsOrder `toml:"stats_order"               koanf:"stats_order"`
}

// ServiceConfig holds the service configuration.
type ServiceConfig struct {
	StateFile           string `toml:"state_file"             koanf:"state_file"`
	LogLevel            string `toml:"log_level"              koanf:"log_level"`
	ManualSyncOnStartup bool   `toml:"manual_sync_on_startup" koanf:"manual_sync_on_startup"` // Perform a sync on startup if token exists
}

// Load reads the configuration from the given TOML file path, then layers
// environment variable overrides on top. Configuration sources are applied in
// order — later sources take precedence over earlier ones:
//
//  1. Built-in defaults
//  2. TOML file (path)
//  3. Legacy env vars: PORT, GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET
//  4. NR_* env vars (highest precedence) — covers every setting
//
// NR_* env var naming convention: NR_SECTION__FIELD (double underscore
// separates the section from the field name). Examples:
//
//	NR_APP__PORT=9090
//	NR_SERVICE__LOG_LEVEL=debug
//	NR_PARENTS__PARENT_A=Alice
//	NR_OAUTH__CLIENT_ID=...
//	NR_AVAILABILITY__PARENT_A_UNAVAILABLE=Monday,Wednesday
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	// 1. Built-in defaults.
	defaults := map[string]any{
		"app.port":                           8888,
		"service.log_level":                  "info",
		"service.manual_sync_on_startup":     true,
		"schedule.past_event_threshold_days": 5,
		"schedule.stats_order":               string(constants.StatsOrderDesc),
	}
	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return nil, fmt.Errorf("failed to load config defaults: %w", err)
	}

	// 2. TOML file.
	if err := k.Load(file.Provider(path), ktoml.Parser()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("toml: %w", err)
	}

	// 3. Legacy env vars (lower precedence than NR_*).
	if portStr := os.Getenv("PORT"); portStr != "" {
		portNum := 0
		if _, err := fmt.Sscanf(portStr, "%d", &portNum); err != nil {
			return nil, fmt.Errorf("PORT must be a valid number: %v", err)
		}
		if err := k.Load(confmap.Provider(map[string]any{"app.port": portNum}, "."), nil); err != nil {
			return nil, fmt.Errorf("failed to apply PORT env var: %w", err)
		}
	}
	if id := os.Getenv("GOOGLE_OAUTH_CLIENT_ID"); id != "" {
		if err := k.Load(confmap.Provider(map[string]any{"oauth.client_id": id}, "."), nil); err != nil {
			return nil, fmt.Errorf("failed to apply GOOGLE_OAUTH_CLIENT_ID env var: %w", err)
		}
	}
	if secret := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"); secret != "" {
		if err := k.Load(confmap.Provider(map[string]any{"oauth.client_secret": secret}, "."), nil); err != nil {
			return nil, fmt.Errorf("failed to apply GOOGLE_OAUTH_CLIENT_SECRET env var: %w", err)
		}
	}

	// 4. NR_* env vars (highest precedence).
	// NR_SECTION__FIELD_NAME → section.field_name
	// e.g. NR_APP__PORT → app.port, NR_OAUTH__CLIENT_ID → oauth.client_id
	if err := k.Load(env.Provider(".", env.Opt{
		Prefix: "NR_",
		TransformFunc: func(s string, v string) (string, any) {
			s = strings.TrimPrefix(s, "NR_")
			s = strings.ToLower(s)
			return strings.ReplaceAll(s, "__", "."), v
		},
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load NR_ env vars: %w", err)
	}

	// Unmarshal into struct using koanf tags. WeaklyTypedInput handles
	// string→int and string→bool coercions (e.g. env vars are always strings).
	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "koanf",
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				commaSeparatedStringToSliceHook(),
			),
			WeaklyTypedInput: true,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate state_file before path resolution to catch missing values early.
	if cfg.Service.StateFile == "" {
		return nil, fmt.Errorf("service.state_file is required (set NR_SERVICE__STATE_FILE or service.state_file in TOML)")
	}

	// Resolve relative state file paths against the config file's parent directory.
	if !filepath.IsAbs(cfg.Service.StateFile) {
		configDir := filepath.Dir(path)
		cfg.Service.StateFile = filepath.Join(configDir, "..", cfg.Service.StateFile)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	cfg.OAuth = &oauth2.Config{
		ClientID:     cfg.Credentials.ClientID,
		ClientSecret: cfg.Credentials.ClientSecret,
		RedirectURL:  strings.TrimSuffix(cfg.App.AppUrl, "/") + "/oauth/callback",
		Scopes: []string{
			calendar.CalendarEventsScope,
			calendar.CalendarCalendarlistReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}

	return &cfg, nil
}

// commaSeparatedStringToSliceHook returns a DecodeHookFunc that converts a
// comma-separated string into a []string. Whitespace around each element is
// trimmed. An empty string results in an empty slice (not a one-element slice
// containing ""), which is important for availability fields set via env vars.
func commaSeparatedStringToSliceHook() mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if from.Kind() != reflect.String || to != reflect.TypeFor[[]string]() {
			return data, nil
		}
		s := data.(string)
		if s == "" {
			return []string{}, nil
		}
		parts := strings.Split(s, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result, nil
	}
}

// validate checks that all required fields are present and valid.
func validate(cfg *Config) error {
	if cfg.Parents.ParentA == "" || cfg.Parents.ParentB == "" {
		return fmt.Errorf("both parent names are required")
	}

	if cfg.Parents.ParentA == cfg.Parents.ParentB {
		return fmt.Errorf("parent names must be different")
	}

	switch cfg.Schedule.UpdateFrequency {
	case "daily", "weekly", "monthly":
		// valid
	default:
		return fmt.Errorf("invalid update frequency: %s", cfg.Schedule.UpdateFrequency)
	}

	if cfg.Schedule.LookAheadDays < 1 {
		return fmt.Errorf("look ahead days must be positive")
	}

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

	if cfg.Credentials.ClientID == "" {
		return fmt.Errorf("OAuth client ID is required (set NR_OAUTH__CLIENT_ID or GOOGLE_OAUTH_CLIENT_ID environment variable)")
	}
	if cfg.Credentials.ClientSecret == "" {
		return fmt.Errorf("OAuth client secret is required (set NR_OAUTH__CLIENT_SECRET or GOOGLE_OAUTH_CLIENT_SECRET environment variable)")
	}

	return nil
}
