package database

import (
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/constants"
)

// ConfigAdapter adapts ConfigStore to the ConfigStoreInterface for runtime config
type ConfigAdapter struct {
	store *ConfigStore
}

// NewConfigAdapter creates a new config adapter
func NewConfigAdapter(store *ConfigStore) *ConfigAdapter {
	return &ConfigAdapter{store: store}
}

// GetParents implements ConfigStoreInterface
func (a *ConfigAdapter) GetParents() (parentA, parentB string, err error) {
	return a.store.GetParents()
}

// GetAvailability implements ConfigStoreInterface
func (a *ConfigAdapter) GetAvailability(parent string) ([]string, error) {
	return a.store.GetAvailability(parent)
}

// GetSchedule implements ConfigStoreInterface
func (a *ConfigAdapter) GetSchedule() (updateFrequency string, lookAheadDays, pastEventThresholdDays int, statsOrder constants.StatsOrder, err error) {
	return a.store.GetSchedule()
}

// LoadRuntimeConfig is a convenience function that loads runtime config using a ConfigStore
func LoadRuntimeConfig(fileConfig *config.Config, store *ConfigStore) (*config.RuntimeConfig, error) {
	adapter := NewConfigAdapter(store)
	loader := config.NewDatabaseConfigLoader(adapter)
	return config.LoadRuntimeConfig(fileConfig, loader)
}
