package database

import (
	"os"
	"testing"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfigAdapter(t *testing.T) (*ConfigAdapter, *ConfigStore, func()) {
	// Create a temporary database file
	dbPath := "test_config_adapter.db"

	// Remove if exists
	os.Remove(dbPath)

	// Create database with test options
	opts := SQLiteOptions{
		Path:        dbPath,
		Mode:        "rwc",
		Cache:       CachePrivate,
		Journal:     JournalWAL,
		ForeignKeys: true,
		BusyTimeout: 5000,
		Synchronous: SynchronousNormal,
		CacheSize:   2000,
	}

	db, err := New(opts)
	require.NoError(t, err, "Failed to create test database")

	// Run migrations
	err = db.MigrateDatabase()
	require.NoError(t, err, "Failed to run migrations")

	// Create config store
	store, err := NewConfigStore(db)
	require.NoError(t, err, "Failed to create config store")

	// Seed test data
	err = store.SaveParents("AdapterParentA", "AdapterParentB")
	require.NoError(t, err)
	err = store.SaveAvailability("parent_a", []string{"Wednesday", "Friday"})
	require.NoError(t, err)
	err = store.SaveAvailability("parent_b", []string{"Monday", "Thursday"})
	require.NoError(t, err)
	err = store.SaveSchedule("monthly", 60, 10, constants.StatsOrderDesc)
	require.NoError(t, err)

	adapter := NewConfigAdapter(store)

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}

	return adapter, store, cleanup
}

func TestNewConfigAdapter(t *testing.T) {
	_, store, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	adapter := NewConfigAdapter(store)
	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.store)
}

func TestConfigAdapter_GetParents(t *testing.T) {
	adapter, _, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	parentA, parentB, err := adapter.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "AdapterParentA", parentA)
	assert.Equal(t, "AdapterParentB", parentB)
}

func TestConfigAdapter_GetAvailability_ParentA(t *testing.T) {
	adapter, _, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	days, err := adapter.GetAvailability("parent_a")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Wednesday", "Friday"}, days)
}

func TestConfigAdapter_GetAvailability_ParentB(t *testing.T) {
	adapter, _, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	days, err := adapter.GetAvailability("parent_b")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Monday", "Thursday"}, days)
}

func TestConfigAdapter_GetAvailability_InvalidParent(t *testing.T) {
	adapter, _, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	_, err := adapter.GetAvailability("parent_c")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parent identifier")
}

func TestConfigAdapter_GetSchedule(t *testing.T) {
	adapter, _, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	freq, lookAhead, threshold, statsOrder, err := adapter.GetSchedule()
	require.NoError(t, err)
	assert.Equal(t, "monthly", freq)
	assert.Equal(t, 60, lookAhead)
	assert.Equal(t, 10, threshold)
	assert.Equal(t, constants.StatsOrderDesc, statsOrder)
}

func TestLoadRuntimeConfig_WithAdapter(t *testing.T) {
	_, store, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	fileConfig := &config.Config{
		App: config.ApplicationConfig{
			Port:      9090,
			AppUrl:    "http://test:9090",
			PublicUrl: "http://test:9090",
		},
		Service: config.ServiceConfig{
			StateFile:           "test.db",
			LogLevel:            "debug",
			ManualSyncOnStartup: false,
		},
		Schedule: config.ScheduleConfig{
			CalendarID: "adapter-test-calendar",
		},
	}

	runtimeCfg, err := LoadRuntimeConfig(fileConfig, store)
	require.NoError(t, err)
	require.NotNil(t, runtimeCfg)
	require.NotNil(t, runtimeCfg.Config)

	// Check app settings from file are preserved
	assert.Equal(t, 9090, runtimeCfg.Config.App.Port)
	assert.Equal(t, "http://test:9090", runtimeCfg.Config.App.AppUrl)
	assert.Equal(t, "debug", runtimeCfg.Config.Service.LogLevel)

	// Check runtime settings from adapter/store
	assert.Equal(t, "AdapterParentA", runtimeCfg.Config.Parents.ParentA)
	assert.Equal(t, "AdapterParentB", runtimeCfg.Config.Parents.ParentB)
	assert.ElementsMatch(t, []string{"Wednesday", "Friday"}, runtimeCfg.Config.Availability.ParentAUnavailable)
	assert.ElementsMatch(t, []string{"Monday", "Thursday"}, runtimeCfg.Config.Availability.ParentBUnavailable)
	assert.Equal(t, "monthly", runtimeCfg.Config.Schedule.UpdateFrequency)
	assert.Equal(t, 60, runtimeCfg.Config.Schedule.LookAheadDays)
	assert.Equal(t, 10, runtimeCfg.Config.Schedule.PastEventThresholdDays)
	assert.Equal(t, constants.StatsOrderDesc, runtimeCfg.Config.Schedule.StatsOrder)

	// Check calendar ID is preserved from file
	assert.Equal(t, "adapter-test-calendar", runtimeCfg.Config.Schedule.CalendarID)
}

func TestLoadRuntimeConfig_DirectFunction(t *testing.T) {
	_, store, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	fileConfig := &config.Config{
		App: config.ApplicationConfig{
			Port: 8888,
		},
	}

	// Test the convenience function
	runtimeCfg, err := LoadRuntimeConfig(fileConfig, store)
	require.NoError(t, err)
	assert.NotNil(t, runtimeCfg)
	assert.Equal(t, "AdapterParentA", runtimeCfg.Config.Parents.ParentA)
}
