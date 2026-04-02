package scheduler

import (
	"testing"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	_ "modernc.org/sqlite" // Register modernc sqlite driver
)

// testConfigStore implements config.ConfigStoreInterface for scheduler tests.
type testConfigStore struct {
	parentA            string
	parentB            string
	parentAUnavailable []string
	parentBUnavailable []string
}

func (s *testConfigStore) GetParents() (string, string, error) {
	return s.parentA, s.parentB, nil
}

func (s *testConfigStore) GetAvailability(parent string) ([]string, error) {
	if parent == "parent_a" {
		return s.parentAUnavailable, nil
	}
	return s.parentBUnavailable, nil
}

func (s *testConfigStore) GetSchedule() (string, int, int, constants.StatsOrder, error) {
	return "weekly", 7, 5, constants.StatsOrderDesc, nil
}

func (s *testConfigStore) GetOAuthConfig() *oauth2.Config {
	return nil
}

// newTestConfigStore creates a testConfigStore with the given parent names and availability.
func newTestConfigStore(parentA, parentB string, parentAUnavailable, parentBUnavailable []string) *testConfigStore {
	return &testConfigStore{
		parentA:            parentA,
		parentB:            parentB,
		parentAUnavailable: parentAUnavailable,
		parentBUnavailable: parentBUnavailable,
	}
}

// setupTestDB creates a new in-memory database for testing
func setupTestDB(t *testing.T) (*database.DB, func()) {
	// Create a new in-memory database with shared cache and foreign keys enabled
	opts := database.SQLiteOptions{
		Path:        ":memory:",           // Use ":memory:" for in-memory database path
		Mode:        "memory",             // Explicitly set mode to memory
		Cache:       database.CacheShared, // Use shared cache
		ForeignKeys: true,                 // Enable foreign keys via PRAGMA
		// Use other defaults from NewDefaultOptions if needed, or keep minimal
		Journal:     database.JournalMemory, // Memory journal is suitable for in-memory DB
		BusyTimeout: 5000,                   // Default busy timeout
	}
	db, err := database.New(opts)
	assert.NoError(t, err)

	// Run migrations
	err = db.MigrateDatabase()
	assert.NoError(t, err)

	// Return the database and a cleanup function
	return db, func() {
		err := db.Close()
		assert.NoError(t, err)
	}
}
