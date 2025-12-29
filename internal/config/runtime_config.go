package config

import (
	"fmt"

	"github.com/belphemur/night-routine/internal/constants"
)

// RuntimeConfig holds configuration loaded from database at runtime
// This allows UI-configurable settings to be updated without restarting the app
type RuntimeConfig struct {
	// Complete merged configuration
	Config *Config
}

// ConfigLoader interface for loading configuration from database
type ConfigLoader interface {
	GetParents() (parentA, parentB string, err error)
	GetAvailability() (parentAUnavailable, parentBUnavailable []string, err error)
	GetSchedule() (updateFrequency string, lookAheadDays, pastEventThresholdDays int, statsOrder constants.StatsOrder, err error)
}

// LoadRuntimeConfig loads runtime configuration from database
// This merges file-based config (app settings) with database config (UI-configurable settings)
func LoadRuntimeConfig(fileConfig *Config, loader ConfigLoader) (*RuntimeConfig, error) {
	// Create a new config with a copy of the file config
	mergedConfig := &Config{
		App:     fileConfig.App,
		Service: fileConfig.Service,
		OAuth:   fileConfig.OAuth,
	}

	// Load parent configuration from database
	parentA, parentB, err := loader.GetParents()
	if err != nil {
		return nil, fmt.Errorf("failed to load parent configuration: %w", err)
	}
	mergedConfig.Parents = ParentsConfig{
		ParentA: parentA,
		ParentB: parentB,
	}

	// Load availability configuration from database
	parentAUnavailable, parentBUnavailable, err := loader.GetAvailability()
	if err != nil {
		return nil, fmt.Errorf("failed to load availability configuration: %w", err)
	}
	mergedConfig.Availability = AvailabilityConfig{
		ParentAUnavailable: parentAUnavailable,
		ParentBUnavailable: parentBUnavailable,
	}

	// Load schedule configuration from database
	updateFrequency, lookAheadDays, pastEventThresholdDays, statsOrder, err := loader.GetSchedule()
	if err != nil {
		return nil, fmt.Errorf("failed to load schedule configuration: %w", err)
	}
	mergedConfig.Schedule = ScheduleConfig{
		UpdateFrequency:        updateFrequency,
		CalendarID:             fileConfig.Schedule.CalendarID, // CalendarID stays in token store
		LookAheadDays:          lookAheadDays,
		PastEventThresholdDays: pastEventThresholdDays,
		StatsOrder:             statsOrder,
	}

	return &RuntimeConfig{Config: mergedConfig}, nil
}

// DatabaseConfigLoader adapts ConfigStore to ConfigLoader interface
type DatabaseConfigLoader struct {
	store ConfigStoreInterface
}

// ConfigStoreInterface defines the interface for configuration storage
type ConfigStoreInterface interface {
	GetParents() (parentA, parentB string, err error)
	GetAvailability(parent string) ([]string, error)
	GetSchedule() (updateFrequency string, lookAheadDays, pastEventThresholdDays int, statsOrder constants.StatsOrder, err error)
}

// NewDatabaseConfigLoader creates a new database config loader
func NewDatabaseConfigLoader(store ConfigStoreInterface) *DatabaseConfigLoader {
	return &DatabaseConfigLoader{store: store}
}

// GetParents loads parent configuration
func (l *DatabaseConfigLoader) GetParents() (parentA, parentB string, err error) {
	return l.store.GetParents()
}

// GetAvailability loads availability configuration
func (l *DatabaseConfigLoader) GetAvailability() (parentAUnavailable, parentBUnavailable []string, err error) {
	parentAUnavailable, err = l.store.GetAvailability("parent_a")
	if err != nil {
		return nil, nil, err
	}

	parentBUnavailable, err = l.store.GetAvailability("parent_b")
	if err != nil {
		return nil, nil, err
	}

	return parentAUnavailable, parentBUnavailable, nil
}

// GetSchedule loads schedule configuration
func (l *DatabaseConfigLoader) GetSchedule() (updateFrequency string, lookAheadDays, pastEventThresholdDays int, statsOrder constants.StatsOrder, err error) {
	return l.store.GetSchedule()
}
