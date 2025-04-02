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
	db, err := New(NewDefaultOptions(dbPath)) // Use new signature with default options
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Close the connection
	err = db.Close()
	assert.NoError(t, err)

	// Verify connection is closed by trying to ping
	err = db.conn.Ping()
	assert.Error(t, err) // Should error because connection is closed
}

// Removed TestNewWithOptions as it's redundant now that New takes options
// and TestPragmaSettings covers detailed option verification.

// TestPragmaSettings verifies that options passed via NewWithOptions correctly
// set the corresponding SQLite PRAGMAs.
func TestPragmaSettings(t *testing.T) {
	dbPath := "test_pragma_settings.db"
	defer os.Remove(dbPath)

	// Define test cases
	testCases := []struct {
		name            string
		opts            SQLiteOptions
		expectedJournal string
		expectedBusy    int
		expectedCache   int
		expectedFK      int // 0 for false, 1 for true
		expectedSync    int // 0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA
	}{
		{
			name: "Default Options",
			opts: NewDefaultOptions(dbPath),
			// Reverted expected values to match options, assuming applyPragmas works
			expectedJournal: "wal",
			expectedBusy:    5000,
			expectedCache:   2000, // Positive KB input -> Positive KB output
			expectedFK:      1,
			expectedSync:    1,
		},
		{
			name: "Custom Options",
			opts: SQLiteOptions{
				Path:        dbPath,
				Journal:     JournalDelete,
				BusyTimeout: 12345,
				CacheSize:   -4000, // Set 4000 pages
				ForeignKeys: false,
				Synchronous: SynchronousFull,
			},
			// Reverted expected values
			expectedJournal: "delete",
			expectedBusy:    12345,
			expectedCache:   -4000, // Negative pages input -> Negative pages output
			expectedFK:      0,
			expectedSync:    2,
		},
		{
			name: "Custom Options KB Cache",
			opts: SQLiteOptions{
				Path:        dbPath,
				Journal:     JournalMemory,
				BusyTimeout: 999,
				CacheSize:   8000, // Set 8000 KB
				ForeignKeys: true,
				Synchronous: SynchronousOff,
			},
			// Reverted expected values
			expectedJournal: "memory",
			expectedBusy:    999,
			expectedCache:   8000, // Positive KB input -> Positive KB output
			expectedFK:      1,
			expectedSync:    0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Need to remove the db file between subtests if it exists
			os.Remove(dbPath)

			db, err := New(tc.opts) // Use new signature
			assert.NoError(t, err, "Failed to create DB connection")
			if err != nil {
				return // Stop test if connection failed
			}
			defer db.Close()

			// Verify journal_mode
			var journalMode string
			err = db.conn.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
			assert.NoError(t, err, "Failed to query journal_mode")
			assert.Equal(t, tc.expectedJournal, journalMode, "Unexpected journal_mode")

			// Verify busy_timeout
			var busyTimeout int
			err = db.conn.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout)
			assert.NoError(t, err, "Failed to query busy_timeout")
			assert.Equal(t, tc.expectedBusy, busyTimeout, "Unexpected busy_timeout")

			// Verify cache_size
			var cacheSize int
			err = db.conn.QueryRow("PRAGMA cache_size;").Scan(&cacheSize)
			assert.NoError(t, err, "Failed to query cache_size")
			// Note: PRAGMA cache_size returns pages if positive, negative KB if negative.
			// The test cases reflect this expectation based on the input opts.CacheSize.
			assert.Equal(t, tc.expectedCache, cacheSize, "Unexpected cache_size")

			// Verify foreign_keys
			var foreignKeys int
			err = db.conn.QueryRow("PRAGMA foreign_keys;").Scan(&foreignKeys)
			assert.NoError(t, err, "Failed to query foreign_keys")
			assert.Equal(t, tc.expectedFK, foreignKeys, "Unexpected foreign_keys setting")

			// Verify synchronous
			var synchronous int
			err = db.conn.QueryRow("PRAGMA synchronous;").Scan(&synchronous)
			assert.NoError(t, err, "Failed to query synchronous")
			assert.Equal(t, tc.expectedSync, synchronous, "Unexpected synchronous setting")
		})
	}
}
