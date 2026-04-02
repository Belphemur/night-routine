package database

import (
	"os"
	"testing"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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

	adapter := NewConfigAdapter(store, nil)

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

	adapter := NewConfigAdapter(store, nil)
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

func TestConfigAdapter_GetOAuthConfig(t *testing.T) {
	_, store, cleanup := setupTestConfigAdapter(t)
	defer cleanup()

	// Without OAuth config
	adapterWithNil := NewConfigAdapter(store, nil)
	assert.Nil(t, adapterWithNil.GetOAuthConfig())

	// With a real OAuth config passed in
	oauthCfg := &oauth2.Config{ClientID: "test-client-id"}
	adapterWithCfg := NewConfigAdapter(store, oauthCfg)
	got := adapterWithCfg.GetOAuthConfig()
	require.NotNil(t, got)
	assert.Equal(t, "test-client-id", got.ClientID)
}
