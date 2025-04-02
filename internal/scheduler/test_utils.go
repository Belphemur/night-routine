package scheduler

import (
	"testing"

	"github.com/belphemur/night-routine/internal/database"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs"
	"github.com/stretchr/testify/assert"
)

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
