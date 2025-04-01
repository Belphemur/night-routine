package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDBClose(t *testing.T) {
	// Use a temporary file for testing
	dbPath := "test_close.db"
	defer os.Remove(dbPath)

	// Create a new database connection
	db, err := New(dbPath)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Close the connection
	err = db.Close()
	assert.NoError(t, err)

	// Verify connection is closed by trying to ping
	err = db.conn.Ping()
	assert.Error(t, err) // Should error because connection is closed
}

func TestNewWithOptions(t *testing.T) {
	dbPath := "test_options.db"
	defer os.Remove(dbPath)

	opts := NewDefaultOptions(dbPath)
	opts.CacheSize = 5000
	opts.BusyTimeout = 10000

	db, err := NewWithOptions(opts)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	defer db.Close()

	// Test connection works
	err = db.conn.Ping()
	assert.NoError(t, err)
}
