package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfigStore(t *testing.T) (*ConfigStore, func()) {
	// Create a temporary database file
	dbPath := "test_config_store.db"

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

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}

	return store, cleanup
}

func TestConfigStore_SaveAndGetParents(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	// Save parent configuration
	err := store.SaveParents("Alice", "Bob")
	require.NoError(t, err)

	// Retrieve parent configuration
	parentA, parentB, err := store.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "Alice", parentA)
	assert.Equal(t, "Bob", parentB)

	// Update parent configuration
	err = store.SaveParents("Charlie", "Diana")
	require.NoError(t, err)

	// Verify update
	parentA, parentB, err = store.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "Charlie", parentA)
	assert.Equal(t, "Diana", parentB)
}

func TestConfigStore_SaveParents_Validation(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	tests := []struct {
		name    string
		parentA string
		parentB string
		wantErr bool
	}{
		{
			name:    "Empty parent A",
			parentA: "",
			parentB: "Bob",
			wantErr: true,
		},
		{
			name:    "Empty parent B",
			parentA: "Alice",
			parentB: "",
			wantErr: true,
		},
		{
			name:    "Same names",
			parentA: "Alice",
			parentB: "Alice",
			wantErr: true,
		},
		{
			name:    "Valid names",
			parentA: "Alice",
			parentB: "Bob",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.SaveParents(tt.parentA, tt.parentB)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigStore_SaveAndGetAvailability(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	// Save availability for parent A
	daysA := []string{"Monday", "Wednesday", "Friday"}
	err := store.SaveAvailability("parent_a", daysA)
	require.NoError(t, err)

	// Save availability for parent B
	daysB := []string{"Tuesday", "Thursday"}
	err = store.SaveAvailability("parent_b", daysB)
	require.NoError(t, err)

	// Retrieve availability for parent A
	retrievedA, err := store.GetAvailability("parent_a")
	require.NoError(t, err)
	assert.ElementsMatch(t, daysA, retrievedA)

	// Retrieve availability for parent B
	retrievedB, err := store.GetAvailability("parent_b")
	require.NoError(t, err)
	assert.ElementsMatch(t, daysB, retrievedB)

	// Update availability for parent A
	newDaysA := []string{"Saturday"}
	err = store.SaveAvailability("parent_a", newDaysA)
	require.NoError(t, err)

	// Verify update
	retrievedA, err = store.GetAvailability("parent_a")
	require.NoError(t, err)
	assert.ElementsMatch(t, newDaysA, retrievedA)

	// Verify parent B unchanged
	retrievedB, err = store.GetAvailability("parent_b")
	require.NoError(t, err)
	assert.ElementsMatch(t, daysB, retrievedB)
}

func TestConfigStore_SaveAvailability_EmptyList(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	// Save empty availability list
	err := store.SaveAvailability("parent_a", []string{})
	require.NoError(t, err)

	// Retrieve and verify empty
	days, err := store.GetAvailability("parent_a")
	require.NoError(t, err)
	assert.Empty(t, days)
}

func TestConfigStore_GetAvailability_InvalidParent(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	_, err := store.GetAvailability("parent_c")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parent identifier")
}

func TestConfigStore_SaveAndGetSchedule(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	// Save schedule configuration
	err := store.SaveSchedule("weekly", 30, 5)
	require.NoError(t, err)

	// Retrieve schedule configuration
	freq, lookAhead, threshold, err := store.GetSchedule()
	require.NoError(t, err)
	assert.Equal(t, "weekly", freq)
	assert.Equal(t, 30, lookAhead)
	assert.Equal(t, 5, threshold)

	// Update schedule configuration
	err = store.SaveSchedule("daily", 7, 3)
	require.NoError(t, err)

	// Verify update
	freq, lookAhead, threshold, err = store.GetSchedule()
	require.NoError(t, err)
	assert.Equal(t, "daily", freq)
	assert.Equal(t, 7, lookAhead)
	assert.Equal(t, 3, threshold)
}

func TestConfigStore_SaveSchedule_Validation(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	tests := []struct {
		name        string
		frequency   string
		lookAhead   int
		threshold   int
		wantErr     bool
		errContains string
	}{
		{
			name:        "Invalid frequency",
			frequency:   "biweekly",
			lookAhead:   30,
			threshold:   5,
			wantErr:     true,
			errContains: "invalid update frequency",
		},
		{
			name:        "Zero look ahead",
			frequency:   "weekly",
			lookAhead:   0,
			threshold:   5,
			wantErr:     true,
			errContains: "must be positive",
		},
		{
			name:        "Negative look ahead",
			frequency:   "weekly",
			lookAhead:   -1,
			threshold:   5,
			wantErr:     true,
			errContains: "must be positive",
		},
		{
			name:        "Negative threshold",
			frequency:   "weekly",
			lookAhead:   30,
			threshold:   -1,
			wantErr:     true,
			errContains: "cannot be negative",
		},
		{
			name:      "Valid daily",
			frequency: "daily",
			lookAhead: 7,
			threshold: 0,
			wantErr:   false,
		},
		{
			name:      "Valid monthly",
			frequency: "monthly",
			lookAhead: 90,
			threshold: 10,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.SaveSchedule(tt.frequency, tt.lookAhead, tt.threshold)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigStore_HasConfiguration(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	// Initially should have no configuration
	hasConfig, err := store.HasConfiguration()
	require.NoError(t, err)
	assert.False(t, hasConfig)

	// Save parent configuration
	err = store.SaveParents("Alice", "Bob")
	require.NoError(t, err)

	// Now should have configuration
	hasConfig, err = store.HasConfiguration()
	require.NoError(t, err)
	assert.True(t, hasConfig)
}

func TestConfigStore_GetParentsFull(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	// Save parent configuration
	err := store.SaveParents("Alice", "Bob")
	require.NoError(t, err)

	// Get full configuration
	config, err := store.GetParentsFull()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, int64(1), config.ID)
	assert.Equal(t, "Alice", config.ParentA)
	assert.Equal(t, "Bob", config.ParentB)
	assert.False(t, config.CreatedAt.IsZero())
	assert.False(t, config.UpdatedAt.IsZero())
}

func TestConfigStore_GetScheduleFull(t *testing.T) {
	store, cleanup := setupTestConfigStore(t)
	defer cleanup()

	// Save schedule configuration
	err := store.SaveSchedule("weekly", 30, 5)
	require.NoError(t, err)

	// Get full configuration
	config, err := store.GetScheduleFull()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, int64(1), config.ID)
	assert.Equal(t, "weekly", config.UpdateFrequency)
	assert.Equal(t, 30, config.LookAheadDays)
	assert.Equal(t, 5, config.PastEventThresholdDays)
	assert.False(t, config.CreatedAt.IsZero())
	assert.False(t, config.UpdatedAt.IsZero())
}

func TestConfigStore_GetParents_NoData(t *testing.T) {
store, cleanup := setupTestConfigStore(t)
defer cleanup()

// Try to get parents before any are saved
_, _, err := store.GetParents()
assert.Error(t, err)
assert.Contains(t, err.Error(), "no parent configuration found")
}

func TestConfigStore_GetParentsFull_NoData(t *testing.T) {
store, cleanup := setupTestConfigStore(t)
defer cleanup()

// Try to get full parents before any are saved
config, err := store.GetParentsFull()
assert.NoError(t, err)
assert.Nil(t, config)
}

func TestConfigStore_GetSchedule_NoData(t *testing.T) {
store, cleanup := setupTestConfigStore(t)
defer cleanup()

// Try to get schedule before any is saved
_, _, _, err := store.GetSchedule()
assert.Error(t, err)
assert.Contains(t, err.Error(), "no schedule configuration found")
}

func TestConfigStore_GetScheduleFull_NoData(t *testing.T) {
store, cleanup := setupTestConfigStore(t)
defer cleanup()

// Try to get full schedule before any is saved
config, err := store.GetScheduleFull()
assert.NoError(t, err)
assert.Nil(t, config)
}

func TestConfigStore_SaveAvailability_Transaction(t *testing.T) {
store, cleanup := setupTestConfigStore(t)
defer cleanup()

// Save initial availability
err := store.SaveAvailability("parent_a", []string{"Monday", "Wednesday"})
require.NoError(t, err)

// Update with different days
err = store.SaveAvailability("parent_a", []string{"Friday"})
require.NoError(t, err)

// Verify only new days exist
days, err := store.GetAvailability("parent_a")
require.NoError(t, err)
assert.Len(t, days, 1)
assert.Equal(t, []string{"Friday"}, days)
}

func TestConfigStore_GetAvailability_QueryError(t *testing.T) {
store, cleanup := setupTestConfigStore(t)
defer cleanup()

// Test with empty database - should return empty list
days, err := store.GetAvailability("parent_a")
require.NoError(t, err)
assert.Empty(t, days)
}
