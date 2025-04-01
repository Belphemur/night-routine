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
		UseWAL:      true,
		ForeignKeys: true,
		BusyTimeout: 5000,
		CacheSize:   2000,
		Synchronous: SynchronousNormal,
		Cache:       CachePrivate,
		Immutable:   true,
		Mode:        "rwc",
	}

	result := opts.buildConnectionString()
	assert.Contains(t, result, "_journal_mode=WAL")
	assert.Contains(t, result, "_foreign_keys=true")
	assert.Contains(t, result, "_busy_timeout=5000")
	assert.Contains(t, result, "_cache_size=2000")
	assert.Contains(t, result, "_synchronous=NORMAL")
	assert.Contains(t, result, "cache=private")
	assert.Contains(t, result, "immutable=true")
	assert.Contains(t, result, "mode=rwc")
}

func TestBuildConnectionString_TransactionOptions(t *testing.T) {
	opts := SQLiteOptions{
		Path:         "test.db",
		LockingMode:  LockingExclusive,
		TxLock:       "immediate",
		MutexLocking: "full",
	}

	result := opts.buildConnectionString()
	assert.Contains(t, result, "_locking_mode=EXCLUSIVE")
	assert.Contains(t, result, "_txlock=immediate")
	assert.Contains(t, result, "_mutex=full")
}

func TestBuildConnectionString_AdvancedOptions(t *testing.T) {
	opts := SQLiteOptions{
		Path:                   "test.db",
		AutoVacuum:             "full",
		CaseSensitiveLike:      true,
		DeferForeignKeys:       true,
		IgnoreCheckConstraints: true,
		QueryOnly:              true,
		RecursiveTriggers:      true,
		SecureDelete:           "FAST",
		WritableSchema:         true,
	}

	result := opts.buildConnectionString()
	assert.Contains(t, result, "_auto_vacuum=full")
	assert.Contains(t, result, "_case_sensitive_like=true")
	assert.Contains(t, result, "_defer_foreign_keys=true")
	assert.Contains(t, result, "_ignore_check_constraints=true")
	assert.Contains(t, result, "_query_only=true")
	assert.Contains(t, result, "_recursive_triggers=true")
	assert.Contains(t, result, "_secure_delete=FAST")
	assert.Contains(t, result, "_writable_schema=true")
}

func TestBuildConnectionString_AuthOptions(t *testing.T) {
	opts := SQLiteOptions{
		Path:      "test.db",
		Auth:      true,
		AuthUser:  "admin",
		AuthPass:  "password123",
		AuthCrypt: "SHA256",
		AuthSalt:  "salt123",
	}

	result := opts.buildConnectionString()
	assert.Contains(t, result, "_auth=")
	assert.Contains(t, result, "_auth_user=admin")
	assert.Contains(t, result, "_auth_pass=password123")
	assert.Contains(t, result, "_auth_crypt=SHA256")
	assert.Contains(t, result, "_auth_salt=salt123")
}

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
		UseWAL:      true,
		ForeignKeys: true,
		BusyTimeout: 10000,
		LockingMode: LockingExclusive,
		QueryOnly:   true,
		Auth:        true,
		AuthUser:    "admin",
		Mode:        "rwc",
	}

	result := opts.buildConnectionString()
	assert.Contains(t, result, "_journal_mode=WAL")
	assert.Contains(t, result, "_foreign_keys=true")
	assert.Contains(t, result, "_busy_timeout=10000")
	assert.Contains(t, result, "_locking_mode=EXCLUSIVE")
	assert.Contains(t, result, "_query_only=true")
	assert.Contains(t, result, "_auth=")
	assert.Contains(t, result, "_auth_user=admin")
	assert.Contains(t, result, "mode=rwc")
}
