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
	// Create a new in-memory database
	db, err := database.New("file::memory:?cache=shared&_foreign_keys=on")
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
