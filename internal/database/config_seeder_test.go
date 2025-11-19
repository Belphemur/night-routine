package database

import (
	"os"
	"testing"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestSeeder(t *testing.T) (*ConfigSeeder, *ConfigStore, func()) {
	// Create a temporary database file
	dbPath := "test_config_seeder.db"

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

	// Create seeder
	seeder := NewConfigSeeder(store)

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}

	return seeder, store, cleanup
}

func createTestConfig() *config.Config {
	return &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "Alice",
			ParentB: "Bob",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{"Monday", "Wednesday"},
			ParentBUnavailable: []string{"Friday"},
		},
		Schedule: config.ScheduleConfig{
			UpdateFrequency:        "weekly",
			LookAheadDays:          30,
			PastEventThresholdDays: 5,
		},
	}
}

func TestConfigSeeder_InitialSeeding(t *testing.T) {
	seeder, store, cleanup := setupTestSeeder(t)
	defer cleanup()

	cfg := createTestConfig()

	// Verify no configuration exists
	hasConfig, err := store.HasConfiguration()
	require.NoError(t, err)
	assert.False(t, hasConfig, "Database should be empty initially")

	// Seed configuration
	err = seeder.SeedFromConfig(cfg)
	require.NoError(t, err, "Seeding should succeed")

	// Verify configuration was seeded
	hasConfig, err = store.HasConfiguration()
	require.NoError(t, err)
	assert.True(t, hasConfig, "Configuration should exist after seeding")

	// Verify parents
	parentA, parentB, err := store.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "Alice", parentA)
	assert.Equal(t, "Bob", parentB)

	// Verify availability
	unavailableA, err := store.GetAvailability("parent_a")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Monday", "Wednesday"}, unavailableA)

	unavailableB, err := store.GetAvailability("parent_b")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Friday"}, unavailableB)

	// Verify schedule
	freq, lookAhead, threshold, err := store.GetSchedule()
	require.NoError(t, err)
	assert.Equal(t, "weekly", freq)
	assert.Equal(t, 30, lookAhead)
	assert.Equal(t, 5, threshold)
}

func TestConfigSeeder_MigrationScenario(t *testing.T) {
	seeder, store, cleanup := setupTestSeeder(t)
	defer cleanup()

	// Simulate an upgrade scenario where user had old version with TOML config
	// and is now upgrading to version with DB config tables

	cfg := createTestConfig()

	// First run after upgrade - should migrate TOML to DB
	err := seeder.SeedFromConfig(cfg)
	require.NoError(t, err, "Migration should succeed")

	// Verify configuration was migrated
	parentA, parentB, err := store.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "Alice", parentA)
	assert.Equal(t, "Bob", parentB)
}

func TestConfigSeeder_SkipIfAlreadySeeded(t *testing.T) {
	seeder, store, cleanup := setupTestSeeder(t)
	defer cleanup()

	cfg := createTestConfig()

	// First seeding
	err := seeder.SeedFromConfig(cfg)
	require.NoError(t, err)

	// Manually update configuration in DB
	err = store.SaveParents("Charlie", "Diana")
	require.NoError(t, err)

	// Create new config with different values
	newCfg := &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "NewParentA",
			ParentB: "NewParentB",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{},
			ParentBUnavailable: []string{},
		},
		Schedule: config.ScheduleConfig{
			UpdateFrequency:        "daily",
			LookAheadDays:          7,
			PastEventThresholdDays: 1,
		},
	}

	// Attempt to seed again
	err = seeder.SeedFromConfig(newCfg)
	require.NoError(t, err)

	// Verify DB values were NOT overwritten (Charlie and Diana should remain)
	parentA, parentB, err := store.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "Charlie", parentA, "DB values should not be overwritten")
	assert.Equal(t, "Diana", parentB, "DB values should not be overwritten")
}

func TestConfigSeeder_EmptyAvailability(t *testing.T) {
	seeder, store, cleanup := setupTestSeeder(t)
	defer cleanup()

	cfg := &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "Alice",
			ParentB: "Bob",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{},
			ParentBUnavailable: []string{},
		},
		Schedule: config.ScheduleConfig{
			UpdateFrequency:        "weekly",
			LookAheadDays:          30,
			PastEventThresholdDays: 5,
		},
	}

	// Seed configuration
	err := seeder.SeedFromConfig(cfg)
	require.NoError(t, err)

	// Verify empty availability lists
	unavailableA, err := store.GetAvailability("parent_a")
	require.NoError(t, err)
	assert.Empty(t, unavailableA)

	unavailableB, err := store.GetAvailability("parent_b")
	require.NoError(t, err)
	assert.Empty(t, unavailableB)
}

func TestConfigSeeder_AllFrequencyTypes(t *testing.T) {
	tests := []struct {
		name      string
		frequency string
	}{
		{"Daily frequency", "daily"},
		{"Weekly frequency", "weekly"},
		{"Monthly frequency", "monthly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seeder, store, cleanup := setupTestSeeder(t)
			defer cleanup()

			cfg := createTestConfig()
			cfg.Schedule.UpdateFrequency = tt.frequency

			err := seeder.SeedFromConfig(cfg)
			require.NoError(t, err)

			freq, _, _, err := store.GetSchedule()
			require.NoError(t, err)
			assert.Equal(t, tt.frequency, freq)
		})
	}
}

func TestConfigSeeder_PreservesDatabaseState(t *testing.T) {
	seeder, store, cleanup := setupTestSeeder(t)
	defer cleanup()

	// Seed initial configuration
	cfg := createTestConfig()
	err := seeder.SeedFromConfig(cfg)
	require.NoError(t, err)

	// User updates configuration via UI (simulated)
	err = store.SaveParents("UpdatedA", "UpdatedB")
	require.NoError(t, err)

	err = store.SaveAvailability("parent_a", []string{"Saturday", "Sunday"})
	require.NoError(t, err)

	err = store.SaveSchedule("daily", 14, 7)
	require.NoError(t, err)

	// Application restarts and tries to seed again
	err = seeder.SeedFromConfig(cfg)
	require.NoError(t, err)

	// Verify user's updates are preserved
	parentA, parentB, err := store.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "UpdatedA", parentA, "User updates should be preserved")
	assert.Equal(t, "UpdatedB", parentB, "User updates should be preserved")

	unavailableA, err := store.GetAvailability("parent_a")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Saturday", "Sunday"}, unavailableA, "User updates should be preserved")

	freq, lookAhead, threshold, err := store.GetSchedule()
	require.NoError(t, err)
	assert.Equal(t, "daily", freq, "User updates should be preserved")
	assert.Equal(t, 14, lookAhead, "User updates should be preserved")
	assert.Equal(t, 7, threshold, "User updates should be preserved")
}

func TestConfigSeeder_SeedFromConfig_ParentsSeedError(t *testing.T) {
seeder, _, cleanup := setupTestSeeder(t)
defer cleanup()

// Create invalid config with same parent names
cfg := &config.Config{
Parents: config.ParentsConfig{
ParentA: "SameName",
ParentB: "SameName", // This will fail validation
},
Availability: config.AvailabilityConfig{
ParentAUnavailable: []string{},
ParentBUnavailable: []string{},
},
Schedule: config.ScheduleConfig{
UpdateFrequency:        "weekly",
LookAheadDays:          30,
PastEventThresholdDays: 5,
},
}

// Seed should fail
err := seeder.SeedFromConfig(cfg)
assert.Error(t, err)
assert.Contains(t, err.Error(), "failed to seed parent configuration")
}

func TestConfigSeeder_SeedFromConfig_ScheduleSeedError(t *testing.T) {
seeder, _, cleanup := setupTestSeeder(t)
defer cleanup()

// Create config with invalid schedule
cfg := &config.Config{
Parents: config.ParentsConfig{
ParentA: "Alice",
ParentB: "Bob",
},
Availability: config.AvailabilityConfig{
ParentAUnavailable: []string{},
ParentBUnavailable: []string{},
},
Schedule: config.ScheduleConfig{
UpdateFrequency:        "invalid", // Invalid frequency
LookAheadDays:          30,
PastEventThresholdDays: 5,
},
}

// Seed should fail on schedule
err := seeder.SeedFromConfig(cfg)
assert.Error(t, err)
assert.Contains(t, err.Error(), "failed to seed schedule configuration")
}
