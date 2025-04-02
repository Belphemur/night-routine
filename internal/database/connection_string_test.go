package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildConnectionString_PathHandling(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "plain path",
			path:     "test.db",
			expected: "file:test.db",
		},
		{
			name:     "path with file prefix",
			path:     "file:test.db",
			expected: "file:test.db",
		},
		{
			name:     "memory database",
			path:     ":memory:",
			expected: "file::memory:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := SQLiteOptions{Path: tt.path}
			result := opts.buildConnectionString()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildConnectionString_CoreOptions(t *testing.T) {
	opts := SQLiteOptions{
		Path:        "test.db",
		Journal:     JournalWAL,
		ForeignKeys: true,
		BusyTimeout: 5000,
		CacheSize:   2000,
		Synchronous: SynchronousNormal,
		Cache:       CachePrivate,
		Immutable:   true,
		Mode:        "rwc",
	}

	result := opts.buildConnectionString()
	// Only check for parameters supported via URI
	assert.Contains(t, result, "cache=private")
	assert.Contains(t, result, "immutable=1") // Should be =1 now
	assert.Contains(t, result, "mode=rwc")
	// Assert that removed parameters are NOT present
	assert.NotContains(t, result, "_journal_mode")
	assert.NotContains(t, result, "_foreign_keys")
	assert.NotContains(t, result, "_busy_timeout")
	assert.NotContains(t, result, "_cache_size")
	assert.NotContains(t, result, "_synchronous")
}

// Removed TestBuildConnectionString_TransactionOptions as these are set via PRAGMA now

// Removed TestBuildConnectionString_AdvancedOptions as these are set via PRAGMA now

// Removed TestBuildConnectionString_AuthOptions as these are likely set via PRAGMA or other means

func TestBuildConnectionString_EmptyOptions(t *testing.T) {
	opts := SQLiteOptions{
		Path: "test.db",
	}

	result := opts.buildConnectionString()
	assert.Equal(t, "file:test.db", result)
}

func TestBuildConnectionString_ComplexCombination(t *testing.T) {
	opts := SQLiteOptions{
		Path:        "complex.db",
		Journal:     JournalWAL,
		ForeignKeys: true,
		BusyTimeout: 10000,
		LockingMode: LockingExclusive,
		QueryOnly:   true,
		Auth:        true,
		AuthUser:    "admin",
		Mode:        "rwc",
	}

	result := opts.buildConnectionString()
	// Only check for parameters supported via URI
	assert.Contains(t, result, "mode=rwc")
	// Assert that removed parameters are NOT present
	assert.NotContains(t, result, "_journal_mode", "ComplexCombination")
	assert.NotContains(t, result, "_foreign_keys", "ComplexCombination")
	assert.NotContains(t, result, "_busy_timeout", "ComplexCombination")
	assert.NotContains(t, result, "_locking_mode", "ComplexCombination")
	assert.NotContains(t, result, "_query_only", "ComplexCombination")
	assert.NotContains(t, result, "_auth", "ComplexCombination") // Ensure _auth and _auth_user are not present
	assert.NotContains(t, result, "_auth_user", "ComplexCombination")
}
