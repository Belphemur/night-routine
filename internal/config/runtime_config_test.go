package config

import (
	"testing"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConfigLoader implements ConfigLoader for testing
type MockConfigLoader struct {
	parents      func() (string, string, error)
	availability func() ([]string, []string, error)
	schedule     func() (string, int, int, constants.StatsOrder, error)
}

func (m *MockConfigLoader) GetParents() (string, string, error) {
	if m.parents != nil {
		return m.parents()
	}
	return "MockParentA", "MockParentB", nil
}

func (m *MockConfigLoader) GetAvailability() ([]string, []string, error) {
	if m.availability != nil {
		return m.availability()
	}
	return []string{"Monday"}, []string{"Friday"}, nil
}

func (m *MockConfigLoader) GetSchedule() (string, int, int, constants.StatsOrder, error) {
	if m.schedule != nil {
		return m.schedule()
	}
	return "weekly", 30, 5, constants.StatsOrderDesc, nil
}

// MockConfigStore implements ConfigStoreInterface for testing
type MockConfigStore struct {
	parents      func() (string, string, error)
	availability func(string) ([]string, error)
	schedule     func() (string, int, int, constants.StatsOrder, error)
}

func (m *MockConfigStore) GetParents() (string, string, error) {
	if m.parents != nil {
		return m.parents()
	}
	return "StoreParentA", "StoreParentB", nil
}

func (m *MockConfigStore) GetAvailability(parent string) ([]string, error) {
	if m.availability != nil {
		return m.availability(parent)
	}
	if parent == "parent_a" {
		return []string{"Tuesday"}, nil
	}
	return []string{"Thursday"}, nil
}

func (m *MockConfigStore) GetSchedule() (string, int, int, constants.StatsOrder, error) {
	if m.schedule != nil {
		return m.schedule()
	}
	return "daily", 14, 3, constants.StatsOrderDesc, nil
}

func TestLoadRuntimeConfig_Success(t *testing.T) {
	fileConfig := &Config{
		App: ApplicationConfig{
			Port:      8080,
			AppUrl:    "http://localhost:8080",
			PublicUrl: "http://localhost:8080",
		},
		Service: ServiceConfig{
			StateFile:           "data/state.db",
			LogLevel:            "info",
			ManualSyncOnStartup: true,
		},
		Schedule: ScheduleConfig{
			CalendarID: "test-calendar-id",
		},
	}

	loader := &MockConfigLoader{}

	runtimeCfg, err := LoadRuntimeConfig(fileConfig, loader)
	require.NoError(t, err)
	require.NotNil(t, runtimeCfg)
	require.NotNil(t, runtimeCfg.Config)

	// Check app settings from file
	assert.Equal(t, 8080, runtimeCfg.Config.App.Port)
	assert.Equal(t, "http://localhost:8080", runtimeCfg.Config.App.AppUrl)
	assert.Equal(t, "info", runtimeCfg.Config.Service.LogLevel)

	// Check runtime settings from loader
	assert.Equal(t, "MockParentA", runtimeCfg.Config.Parents.ParentA)
	assert.Equal(t, "MockParentB", runtimeCfg.Config.Parents.ParentB)
	assert.Equal(t, []string{"Monday"}, runtimeCfg.Config.Availability.ParentAUnavailable)
	assert.Equal(t, []string{"Friday"}, runtimeCfg.Config.Availability.ParentBUnavailable)
	assert.Equal(t, "weekly", runtimeCfg.Config.Schedule.UpdateFrequency)
	assert.Equal(t, 30, runtimeCfg.Config.Schedule.LookAheadDays)
	assert.Equal(t, 5, runtimeCfg.Config.Schedule.PastEventThresholdDays)
	assert.Equal(t, constants.StatsOrderDesc, runtimeCfg.Config.Schedule.StatsOrder)

	// Check calendar ID preserved from file
	assert.Equal(t, "test-calendar-id", runtimeCfg.Config.Schedule.CalendarID)
}

func TestLoadRuntimeConfig_ParentsError(t *testing.T) {
	fileConfig := &Config{}
	loader := &MockConfigLoader{
		parents: func() (string, string, error) {
			return "", "", assert.AnError
		},
	}

	_, err := LoadRuntimeConfig(fileConfig, loader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load parent configuration")
}

func TestLoadRuntimeConfig_AvailabilityError(t *testing.T) {
	fileConfig := &Config{}
	loader := &MockConfigLoader{
		availability: func() ([]string, []string, error) {
			return nil, nil, assert.AnError
		},
	}

	_, err := LoadRuntimeConfig(fileConfig, loader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load availability configuration")
}

func TestLoadRuntimeConfig_ScheduleError(t *testing.T) {
	fileConfig := &Config{}
	loader := &MockConfigLoader{
		schedule: func() (string, int, int, constants.StatsOrder, error) {
			return "", 0, 0, "", assert.AnError
		},
	}

	_, err := LoadRuntimeConfig(fileConfig, loader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load schedule configuration")
}

func TestDatabaseConfigLoader_GetParents(t *testing.T) {
	store := &MockConfigStore{}
	loader := NewDatabaseConfigLoader(store)

	parentA, parentB, err := loader.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "StoreParentA", parentA)
	assert.Equal(t, "StoreParentB", parentB)
}

func TestDatabaseConfigLoader_GetAvailability(t *testing.T) {
	store := &MockConfigStore{}
	loader := NewDatabaseConfigLoader(store)

	parentADays, parentBDays, err := loader.GetAvailability()
	require.NoError(t, err)
	assert.Equal(t, []string{"Tuesday"}, parentADays)
	assert.Equal(t, []string{"Thursday"}, parentBDays)
}

func TestDatabaseConfigLoader_GetAvailability_ParentAError(t *testing.T) {
	store := &MockConfigStore{
		availability: func(parent string) ([]string, error) {
			if parent == "parent_a" {
				return nil, assert.AnError
			}
			return []string{"Thursday"}, nil
		},
	}
	loader := NewDatabaseConfigLoader(store)

	_, _, err := loader.GetAvailability()
	assert.Error(t, err)
}

func TestDatabaseConfigLoader_GetAvailability_ParentBError(t *testing.T) {
	store := &MockConfigStore{
		availability: func(parent string) ([]string, error) {
			if parent == "parent_a" {
				return []string{"Tuesday"}, nil
			}
			return nil, assert.AnError
		},
	}
	loader := NewDatabaseConfigLoader(store)

	_, _, err := loader.GetAvailability()
	assert.Error(t, err)
}

func TestDatabaseConfigLoader_GetSchedule(t *testing.T) {
	store := &MockConfigStore{}
	loader := NewDatabaseConfigLoader(store)

	freq, lookAhead, threshold, statsOrder, err := loader.GetSchedule()
	require.NoError(t, err)
	assert.Equal(t, "daily", freq)
	assert.Equal(t, 14, lookAhead)
	assert.Equal(t, 3, threshold)
	assert.Equal(t, constants.StatsOrderDesc, statsOrder)
}
