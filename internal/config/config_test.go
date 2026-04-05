package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a temporary config file
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_config.toml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err, "Failed to write temp config file")
	return tmpFile
}

// Helper function to set environment variables for a test
func setEnvVars(t *testing.T, vars map[string]string) {
	t.Helper()
	originalValues := make(map[string]string)

	for key, value := range vars {
		originalValues[key] = os.Getenv(key)
		err := os.Setenv(key, value)
		require.NoError(t, err, "Failed to set env var %s", key)
	}

	// Cleanup function to restore original environment variables
	t.Cleanup(func() {
		for key, value := range originalValues {
			if value == "" {
				err := os.Unsetenv(key)
				require.NoError(t, err, "Failed to unset env var %s", key)
			} else {
				err := os.Setenv(key, value)
				require.NoError(t, err, "Failed to restore env var %s", key)
			}
		}
	})
}

func TestLoadConfig_Valid(t *testing.T) {
	validToml := `
[app]
port = 9090
app_url = "http://localhost:9090"
public_url = "https://example.com/public"

[parents]
parent_a = "Alice"
parent_b = "Bob"

[availability]
parent_a_unavailable = ["Mon"]
parent_b_unavailable = ["Tue"]

[schedule]
update_frequency = "daily"
calendar_id = "primary"
look_ahead_days = 14

[service]
state_file = "data/test.db"
log_level = "debug"
manual_sync_on_startup = false # Explicitly set to false to test override
`
	configFile := createTempConfigFile(t, validToml)
	setEnvVars(t, map[string]string{
		"GOOGLE_OAUTH_CLIENT_ID":     "test-client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET": "test-client-secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 9090, cfg.App.Port)
	assert.Equal(t, "http://localhost:9090", cfg.App.AppUrl)
	assert.Equal(t, "https://example.com/public", cfg.App.PublicUrl)
	assert.Equal(t, "Alice", cfg.Parents.ParentA)
	assert.Equal(t, "Bob", cfg.Parents.ParentB)
	assert.Equal(t, []string{"Mon"}, cfg.Availability.ParentAUnavailable)
	assert.Equal(t, []string{"Tue"}, cfg.Availability.ParentBUnavailable)
	assert.Equal(t, "daily", cfg.Schedule.UpdateFrequency)
	assert.Equal(t, "primary", cfg.Schedule.CalendarID)
	assert.Equal(t, 14, cfg.Schedule.LookAheadDays)
	assert.True(t, filepath.IsAbs(cfg.Service.StateFile), "State file path should be absolute")
	// Check if the cleaned absolute path ends with the expected relative path components
	expectedSuffix := filepath.Join("data", "test.db")
	assert.True(t, strings.HasSuffix(filepath.Clean(cfg.Service.StateFile), expectedSuffix), "Expected StateFile path '%s' to end with '%s'", cfg.Service.StateFile, expectedSuffix)
	assert.Equal(t, "debug", cfg.Service.LogLevel)
	assert.False(t, cfg.Service.ManualSyncOnStartup, "ManualSyncOnStartup should be false as set in TOML") // Check override

	require.NotNil(t, cfg.OAuth)
	assert.Equal(t, "test-client-id", cfg.OAuth.ClientID)
	assert.Equal(t, "test-client-secret", cfg.OAuth.ClientSecret)
	assert.Equal(t, "http://localhost:9090/oauth/callback", cfg.OAuth.RedirectURL)
}

func TestLoadConfig_DisabledFrequency(t *testing.T) {
	disabledToml := `
[app]
app_url = "http://localhost:8080"
public_url = "http://localhost:8080"

[parents]
parent_a = "Alice"
parent_b = "Bob"

[schedule]
update_frequency = "disabled"
look_ahead_days = 14

[service]
state_file = "data/test.db"
`
	configFile := createTempConfigFile(t, disabledToml)
	setEnvVars(t, map[string]string{
		"GOOGLE_OAUTH_CLIENT_ID":     "test-client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET": "test-client-secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "disabled", cfg.Schedule.UpdateFrequency)
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Provide required URLs, other fields will use defaults
	minimalToml := `
[app]
app_url = "http://required-app.com"
public_url = "http://required-public.com"

[parents]
parent_a = "Alice"
parent_b = "Bob"

[schedule]
update_frequency = "weekly"
look_ahead_days = 7

[service]
state_file = "state.db"
# log_level is missing, should default to "info"
# manual_sync_on_startup is missing, should default to true
# port is missing, should default to 8888
`
	configFile := createTempConfigFile(t, minimalToml)
	setEnvVars(t, map[string]string{
		"GOOGLE_OAUTH_CLIENT_ID":     "test-client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET": "test-client-secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check defaults for fields NOT provided in TOML
	assert.Equal(t, 8888, cfg.App.Port)                                                           // Default port
	assert.Equal(t, "info", cfg.Service.LogLevel)                                                 // Default log level
	assert.True(t, cfg.Service.ManualSyncOnStartup, "ManualSyncOnStartup should default to true") // Check new default
	assert.Equal(t, "", cfg.Schedule.CalendarID)                                                  // Default calendar ID is empty

	// Check values provided in TOML
	assert.Equal(t, "http://required-app.com", cfg.App.AppUrl)
	assert.Equal(t, "http://required-public.com", cfg.App.PublicUrl)
	assert.Equal(t, "", cfg.Schedule.CalendarID) // Default calendar ID is empty

	// Check values from file
	assert.Equal(t, "Alice", cfg.Parents.ParentA)
	assert.Equal(t, "Bob", cfg.Parents.ParentB)
	assert.Equal(t, "weekly", cfg.Schedule.UpdateFrequency)
	assert.Equal(t, 7, cfg.Schedule.LookAheadDays)
	assert.True(t, filepath.IsAbs(cfg.Service.StateFile), "State file path should be absolute")
	assert.Contains(t, cfg.Service.StateFile, "state.db")

	require.NotNil(t, cfg.OAuth)
	assert.Equal(t, "test-client-id", cfg.OAuth.ClientID)
	assert.Equal(t, "test-client-secret", cfg.OAuth.ClientSecret)
	assert.Equal(t, "http://required-app.com/oauth/callback", cfg.OAuth.RedirectURL) // Based on provided AppUrl
}

func TestLoadConfig_EnvVarOverrides(t *testing.T) {
	tomlContent := `
[app]
port = 9000 # Port in TOML
app_url = "http://config-app.com"
public_url = "http://config-public.com"

[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "weekly"
look_ahead_days = 7
[service]
state_file = "state.db"
# manual_sync_on_startup is missing, should default to true
`
	configFile := createTempConfigFile(t, tomlContent)
	setEnvVars(t, map[string]string{
		"PORT":                       "9999", // Override port via ENV
		"GOOGLE_OAUTH_CLIENT_ID":     "env-client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET": "env-client-secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 9999, cfg.App.Port, "Port should be overridden by ENV var")
	assert.Equal(t, "http://config-app.com", cfg.App.AppUrl) // URLs should come from TOML
	assert.Equal(t, "http://config-public.com", cfg.App.PublicUrl)
	assert.True(t, cfg.Service.ManualSyncOnStartup, "ManualSyncOnStartup should be true (default)") // Check default
	assert.Equal(t, "env-client-id", cfg.OAuth.ClientID)
	assert.Equal(t, "env-client-secret", cfg.OAuth.ClientSecret)
	assert.Equal(t, "http://config-app.com/oauth/callback", cfg.OAuth.RedirectURL) // Redirect uses AppUrl from TOML
}

func TestLoadConfig_InvalidToml(t *testing.T) {
	invalidToml := `
[app
port = 8080
`
	configFile := createTempConfigFile(t, invalidToml)
	_, err := Load(configFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "toml:") // Check for TOML parsing error
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := Load("nonexistent/config.toml")
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist) // Check for file not found error
}

func TestLoadConfig_ValidationErrors(t *testing.T) {
	setEnvVars(t, map[string]string{
		"GOOGLE_OAUTH_CLIENT_ID":     "test-id",
		"GOOGLE_OAUTH_CLIENT_SECRET": "test-secret",
	})

	testCases := []struct {
		name        string
		tomlContent string
		expectedErr string
	}{
		{
			name: "Missing Parent A",
			tomlContent: `
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_b = "B"
[schedule]
update_frequency = "weekly"
look_ahead_days = 7
[service]
state_file = "s.db"`,
			expectedErr: "both parent names are required",
		},
		{
			name: "Same Parent Names",
			tomlContent: `
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_a = "Same"
parent_b = "Same"
[schedule]
update_frequency = "weekly"
look_ahead_days = 7
[service]
state_file = "s.db"`,
			expectedErr: "parent names must be different",
		},
		{
			name: "Invalid Frequency",
			tomlContent: `
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "yearly"
look_ahead_days = 7
[service]
state_file = "s.db"`,
			expectedErr: "invalid update frequency: yearly",
		},
		{
			name: "Invalid Look Ahead Days",
			tomlContent: `
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "daily"
look_ahead_days = 0
[service]
state_file = "s.db"`,
			expectedErr: "look ahead days must be positive",
		},
		{
			name: "Missing App URL",
			tomlContent: `
[app]
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "daily"
look_ahead_days = 1
[service]
state_file = "s.db"`,
			expectedErr: "app_url is required",
		},
		{
			name: "Invalid App URL format",
			tomlContent: `
[app]
app_url = "not a url"
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "daily"
look_ahead_days = 1
[service]
state_file = "s.db"`,
			expectedErr: "invalid app_url 'not a url'",
		},
		{
			name: "Missing Public URL",
			tomlContent: `
[app]
app_url = "http://a.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "daily"
look_ahead_days = 1
[service]
state_file = "s.db"`,
			expectedErr: "public_url is required",
		},
		{
			name: "Invalid Public URL format",
			tomlContent: `
[app]
app_url = "http://a.com"
public_url = "http://app url with spaces.com" # Use URL with invalid characters
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "daily"
look_ahead_days = 1
[service]
state_file = "s.db"`,
			expectedErr: "invalid public_url 'http://app url with spaces.com'", // Update expected error
		},
		{
			name: "Missing State File",
			tomlContent: `
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "daily"
look_ahead_days = 1
[service]`,
			expectedErr: "service.state_file is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configFile := createTempConfigFile(t, tc.tomlContent)
			_, err := Load(configFile)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestLoadConfig_MissingOAuthEnvVars(t *testing.T) {
	validToml := `
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "daily"
look_ahead_days = 1
[service]
state_file = "s.db"
`
	configFile := createTempConfigFile(t, validToml)

	t.Run("Missing Client ID", func(t *testing.T) {
		setEnvVars(t, map[string]string{
			"GOOGLE_OAUTH_CLIENT_SECRET": "test-secret",
		})
		os.Unsetenv("GOOGLE_OAUTH_CLIENT_ID")
		os.Unsetenv("NR_OAUTH__CLIENT_ID")

		_, err := Load(configFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "NR_OAUTH__CLIENT_ID")
		assert.Contains(t, err.Error(), "GOOGLE_OAUTH_CLIENT_ID")
	})

	t.Run("Missing Client Secret", func(t *testing.T) {
		setEnvVars(t, map[string]string{
			"GOOGLE_OAUTH_CLIENT_ID": "test-id",
		})
		os.Unsetenv("GOOGLE_OAUTH_CLIENT_SECRET")
		os.Unsetenv("NR_OAUTH__CLIENT_SECRET")

		_, err := Load(configFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "NR_OAUTH__CLIENT_SECRET")
		assert.Contains(t, err.Error(), "GOOGLE_OAUTH_CLIENT_SECRET")
	})
}

func TestLoadConfig_NREnvVarOverrides(t *testing.T) {
	tomlContent := `
[app]
port = 9000
app_url = "http://config-app.com"
public_url = "http://config-public.com"

[parents]
parent_a = "TomlA"
parent_b = "TomlB"

[schedule]
update_frequency = "weekly"
look_ahead_days = 7

[service]
state_file = "state.db"
log_level = "warn"
`
	configFile := createTempConfigFile(t, tomlContent)
	setEnvVars(t, map[string]string{
		"NR_APP__PORT":            "7777",
		"NR_SERVICE__LOG_LEVEL":   "trace",
		"NR_PARENTS__PARENT_A":    "NRAlice",
		"NR_PARENTS__PARENT_B":    "NRBob",
		"NR_OAUTH__CLIENT_ID":     "nr-client-id",
		"NR_OAUTH__CLIENT_SECRET": "nr-client-secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)

	assert.Equal(t, 7777, cfg.App.Port, "NR_APP__PORT should override TOML port")
	assert.Equal(t, "trace", cfg.Service.LogLevel, "NR_SERVICE__LOG_LEVEL should override TOML log_level")
	assert.Equal(t, "NRAlice", cfg.Parents.ParentA, "NR_PARENTS__PARENT_A should override TOML parent_a")
	assert.Equal(t, "NRBob", cfg.Parents.ParentB, "NR_PARENTS__PARENT_B should override TOML parent_b")
	assert.Equal(t, "nr-client-id", cfg.OAuth.ClientID)
	assert.Equal(t, "nr-client-secret", cfg.OAuth.ClientSecret)
	// Non-overridden fields come from TOML
	assert.Equal(t, "http://config-app.com", cfg.App.AppUrl)
}

func TestLoadConfig_NREnvVarPrecedenceOverLegacy(t *testing.T) {
	tomlContent := `
[app]
app_url = "http://a.com"
public_url = "http://p.com"

[parents]
parent_a = "A"
parent_b = "B"

[schedule]
update_frequency = "weekly"
look_ahead_days = 7

[service]
state_file = "state.db"
`
	configFile := createTempConfigFile(t, tomlContent)
	// Set both legacy and NR_* — NR_* must win
	setEnvVars(t, map[string]string{
		"PORT":                       "6000",
		"NR_APP__PORT":               "6666",
		"GOOGLE_OAUTH_CLIENT_ID":     "legacy-id",
		"GOOGLE_OAUTH_CLIENT_SECRET": "legacy-secret",
		"NR_OAUTH__CLIENT_ID":        "nr-id",
		"NR_OAUTH__CLIENT_SECRET":    "nr-secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)

	assert.Equal(t, 6666, cfg.App.Port, "NR_APP__PORT must take precedence over PORT")
	assert.Equal(t, "nr-id", cfg.OAuth.ClientID, "NR_OAUTH__CLIENT_ID must take precedence over GOOGLE_OAUTH_CLIENT_ID")
	assert.Equal(t, "nr-secret", cfg.OAuth.ClientSecret, "NR_OAUTH__CLIENT_SECRET must take precedence over GOOGLE_OAUTH_CLIENT_SECRET")
}

func TestLoadConfig_NREnvVarAvailability(t *testing.T) {
	tomlContent := `
[app]
app_url = "http://a.com"
public_url = "http://p.com"

[parents]
parent_a = "A"
parent_b = "B"

[schedule]
update_frequency = "weekly"
look_ahead_days = 7

[service]
state_file = "state.db"
`
	configFile := createTempConfigFile(t, tomlContent)
	setEnvVars(t, map[string]string{
		"NR_AVAILABILITY__PARENT_A_UNAVAILABLE": "Monday, Wednesday",
		"NR_AVAILABILITY__PARENT_B_UNAVAILABLE": "Friday",
		"NR_OAUTH__CLIENT_ID":                   "id",
		"NR_OAUTH__CLIENT_SECRET":               "secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)

	assert.Equal(t, []string{"Monday", "Wednesday"}, cfg.Availability.ParentAUnavailable,
		"NR_AVAILABILITY__ should set comma-separated unavailable days")
	assert.Equal(t, []string{"Friday"}, cfg.Availability.ParentBUnavailable)
}

func TestLoadConfig_NREnvVarEmptyAvailability(t *testing.T) {
	tomlContent := `
[app]
app_url = "http://a.com"
public_url = "http://p.com"

[parents]
parent_a = "A"
parent_b = "B"

[availability]
parent_a_unavailable = ["Monday"]

[schedule]
update_frequency = "weekly"
look_ahead_days = 7

[service]
state_file = "state.db"
`
	configFile := createTempConfigFile(t, tomlContent)
	setEnvVars(t, map[string]string{
		// Override TOML's ["Monday"] with empty string → should result in []
		"NR_AVAILABILITY__PARENT_A_UNAVAILABLE": "",
		"NR_OAUTH__CLIENT_ID":                   "id",
		"NR_OAUTH__CLIENT_SECRET":               "secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)

	assert.Empty(t, cfg.Availability.ParentAUnavailable,
		"empty NR_ availability env var should result in an empty slice")
}

func TestLoadConfig_NROAuthOnly(t *testing.T) {
	tomlContent := `
[app]
app_url = "http://a.com"
public_url = "http://p.com"

[parents]
parent_a = "A"
parent_b = "B"

[schedule]
update_frequency = "weekly"
look_ahead_days = 7

[service]
state_file = "state.db"
`
	configFile := createTempConfigFile(t, tomlContent)
	// Provide credentials only via NR_*, no legacy vars
	os.Unsetenv("GOOGLE_OAUTH_CLIENT_ID")
	os.Unsetenv("GOOGLE_OAUTH_CLIENT_SECRET")
	setEnvVars(t, map[string]string{
		"NR_OAUTH__CLIENT_ID":     "only-nr-id",
		"NR_OAUTH__CLIENT_SECRET": "only-nr-secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)

	assert.Equal(t, "only-nr-id", cfg.OAuth.ClientID)
	assert.Equal(t, "only-nr-secret", cfg.OAuth.ClientSecret)
}

func TestLoadConfig_InvalidPortEnvVar(t *testing.T) {
	tomlContent := `
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "weekly"
look_ahead_days = 7
[service]
state_file = "state.db"
`
	configFile := createTempConfigFile(t, tomlContent)
	setEnvVars(t, map[string]string{
		"PORT":                       "not-a-number",
		"GOOGLE_OAUTH_CLIENT_ID":     "env-client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET": "env-client-secret",
	})

	_, err := Load(configFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PORT must be a valid number")
}

func TestLoadConfig_StateFileAbsPath(t *testing.T) {
	// Use standard multi-line TOML for clarity and correctness
	tomlContentRelative := `
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "weekly"
look_ahead_days = 7
[service]
state_file = "relative/path/state.db"
`
	configFileRelative := createTempConfigFile(t, tomlContentRelative)
	configDir := filepath.Dir(configFileRelative)

	setEnvVars(t, map[string]string{"GOOGLE_OAUTH_CLIENT_ID": "id", "GOOGLE_OAUTH_CLIENT_SECRET": "secret"})

	cfgRel, errRel := Load(configFileRelative)
	require.NoError(t, errRel)
	expectedRelPath := filepath.Join(configDir, "..", "relative/path/state.db")
	// Normalize paths for comparison as Join might produce different separators
	assert.Equal(t, filepath.Clean(expectedRelPath), filepath.Clean(cfgRel.Service.StateFile))
	assert.True(t, filepath.IsAbs(cfgRel.Service.StateFile))

	absPath := "/absolute/path/state.db"
	if os.PathSeparator == '\\' { // Handle Windows paths
		absPath = "C:\\absolute\\path\\state.db"
		// Create the directory structure if it doesn't exist (needed for IsAbs check on Windows sometimes)
		err := os.MkdirAll(filepath.Dir(absPath), 0755)
		require.NoError(t, err, "Failed to create directory structure for absolute path test")
	}

	// Use standard multi-line TOML for clarity and correctness
	tomlContentAbsolute := fmt.Sprintf(`
[app]
app_url = "http://a.com"
public_url = "http://p.com"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "weekly"
look_ahead_days = 7
[service]
state_file = %q
`, absPath)

	configFileAbsolute := createTempConfigFile(t, tomlContentAbsolute)

	cfgAbs, errAbs := Load(configFileAbsolute)
	require.NoError(t, errAbs)
	assert.Equal(t, absPath, cfgAbs.Service.StateFile)
	assert.True(t, filepath.IsAbs(cfgAbs.Service.StateFile))
}

func TestLoadConfig_TrailingSlashAppUrl(t *testing.T) {
	// app_url with a trailing slash must not produce a double-slash redirect URL
	tomlContent := `
[app]
app_url = "http://localhost:8888/"
public_url = "http://localhost:8888/"
[parents]
parent_a = "A"
parent_b = "B"
[schedule]
update_frequency = "weekly"
look_ahead_days = 7
[service]
state_file = "state.db"
`
	configFile := createTempConfigFile(t, tomlContent)
	setEnvVars(t, map[string]string{
		"NR_OAUTH__CLIENT_ID":     "id",
		"NR_OAUTH__CLIENT_SECRET": "secret",
	})

	cfg, err := Load(configFile)
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:8888/oauth/callback", cfg.OAuth.RedirectURL,
		"trailing slash in app_url must not produce a double-slash redirect URL")
}
