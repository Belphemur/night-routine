package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultOptions(t *testing.T) {
	opts := NewDefaultOptions("test.db")

	assert.Equal(t, "test.db", opts.Path)
	assert.Equal(t, "rwc", opts.Mode)
	assert.Equal(t, JournalWAL, opts.Journal)
	assert.True(t, opts.ForeignKeys)
	assert.Equal(t, 5000, opts.BusyTimeout)
	assert.Equal(t, 2000, opts.CacheSize)
	assert.Equal(t, SynchronousNormal, opts.Synchronous)
	assert.Equal(t, CachePrivate, opts.Cache)
}

func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		opts     SQLiteOptions
		expected string
	}{
		{
			name: "default options",
			opts: NewDefaultOptions("test.db"),
			// Updated expected: Only URI params (mode, cache, immutable) should be present
			expected: "file:test.db?cache=private&mode=rwc",
		},
		{
			name: "memory database",
			opts: SQLiteOptions{
				Path:        ":memory:",
				Mode:        "memory",
				ForeignKeys: true,
				CacheSize:   1000,
			},
			// Updated expected: Only mode is set here via URI
			expected: "file::memory:?mode=memory",
		},
		{
			name: "all core options",
			opts: SQLiteOptions{
				Path:        "full.db",
				Mode:        "rwc",
				Journal:     JournalWAL,
				ForeignKeys: true,
				BusyTimeout: 10000,
				CacheSize:   5000,
				Synchronous: SynchronousFull,
				Cache:       CachePrivate,
				Immutable:   true,
			},
			// Updated expected: Only mode, cache, immutable are set via URI
			expected: "file:full.db?cache=private&immutable=1&mode=rwc", // immutable=1 for true
		},
		{
			name: "transaction and locking options",
			opts: SQLiteOptions{
				Path:         "locked.db",
				LockingMode:  LockingExclusive,
				TxLock:       "immediate",
				MutexLocking: "full",
			},
			// Updated expected: No URI params set for these options
			expected: "file:locked.db",
		},
		{
			name: "advanced options",
			opts: SQLiteOptions{
				Path:                   "advanced.db",
				AutoVacuum:             "full",
				CaseSensitiveLike:      true,
				DeferForeignKeys:       true,
				IgnoreCheckConstraints: true,
				QueryOnly:              true,
				RecursiveTriggers:      true,
				SecureDelete:           "FAST",
				WritableSchema:         true,
			},
			// Updated expected: No URI params set for these options
			expected: "file:advanced.db",
		},
		{
			name: "authentication options",
			opts: SQLiteOptions{
				Path:      "auth.db",
				Auth:      true,
				AuthUser:  "user",
				AuthPass:  "pass",
				AuthCrypt: "SHA256",
				AuthSalt:  "salt",
			},
			// Updated expected: No URI params set for auth options
			expected: "file:auth.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.buildConnectionString()
			assert.Equal(t, tt.expected, result)
		})
	}
}
