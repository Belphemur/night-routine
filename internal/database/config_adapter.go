package database

import (
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/constants"
	"golang.org/x/oauth2"
)

// ConfigAdapter adapts ConfigStore to the config.ConfigStoreInterface, adding the
// static OAuth2 config (which lives in file/env, not the database) so that
// handlers only need a single ConfigStoreInterface dependency — no RuntimeConfig.
type ConfigAdapter struct {
	store       *ConfigStore
	oauthConfig *oauth2.Config
}

// NewConfigAdapter creates a new config adapter.
// oauthConfig carries the static OAuth2 credentials (from environment variables /
// file config) that cannot be stored in the database.
func NewConfigAdapter(store *ConfigStore, oauthConfig *oauth2.Config) *ConfigAdapter {
	return &ConfigAdapter{store: store, oauthConfig: oauthConfig}
}

// GetParents implements config.ConfigStoreInterface
func (a *ConfigAdapter) GetParents() (parentA, parentB string, err error) {
	return a.store.GetParents()
}

// GetAvailability implements config.ConfigStoreInterface
func (a *ConfigAdapter) GetAvailability(parent string) ([]string, error) {
	return a.store.GetAvailability(parent)
}

// GetSchedule implements config.ConfigStoreInterface
func (a *ConfigAdapter) GetSchedule() (updateFrequency string, lookAheadDays, pastEventThresholdDays int, statsOrder constants.StatsOrder, err error) {
	return a.store.GetSchedule()
}

// GetOAuthConfig implements config.ConfigStoreInterface.
// Returns the static OAuth2 configuration (client ID, secret, redirect URL, scopes)
// that was set at application startup from environment variables and the config file.
func (a *ConfigAdapter) GetOAuthConfig() *oauth2.Config {
	return a.oauthConfig
}

// LoadRuntimeConfig is a convenience function that loads runtime config using a ConfigStore.
// The resulting RuntimeConfig is only used for one-time initialisation (e.g. the Scheduler);
// live reads should go through ConfigStoreInterface directly.
func LoadRuntimeConfig(fileConfig *config.Config, store *ConfigStore) (*config.RuntimeConfig, error) {
	adapter := NewConfigAdapter(store, fileConfig.OAuth)
	loader := config.NewDatabaseConfigLoader(adapter)
	return config.LoadRuntimeConfig(fileConfig, loader)
}
