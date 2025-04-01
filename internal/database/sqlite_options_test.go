package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultOptions(t *testing.T) {
	opts := NewDefaultOptions("test.db")

	assert.Equal(t, "test.db", opts.Path)
	assert.Equal(t, "rwc", opts.Mode)
	assert.True(t, opts.UseWAL)
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
			name:     "default options",
			opts:     NewDefaultOptions("test.db"),
			expected: "file:test.db?_busy_timeout=5000&_cache_size=2000&_foreign_keys=true&_journal_mode=WAL&_synchronous=NORMAL&cache=private&mode=rwc",
		},
		{
			name: "memory database",
			opts: SQLiteOptions{
				Path:        ":memory:",
				Mode:        "memory",
				ForeignKeys: true,
				CacheSize:   1000,
			},
			expected: "file::memory:?_cache_size=1000&_foreign_keys=true&mode=memory",
		},
		{
			name: "all core options",
			opts: SQLiteOptions{
				Path:        "full.db",
				Mode:        "rwc",
				UseWAL:      true,
				ForeignKeys: true,
				BusyTimeout: 10000,
				CacheSize:   5000,
				Synchronous: SynchronousFull,
				Cache:       CachePrivate,
				Immutable:   true,
			},
			expected: "file:full.db?_busy_timeout=10000&_cache_size=5000&_foreign_keys=true&_journal_mode=WAL&_synchronous=FULL&cache=private&immutable=true&mode=rwc",
		},
		{
			name: "transaction and locking options",
			opts: SQLiteOptions{
				Path:         "locked.db",
				LockingMode:  LockingExclusive,
				TxLock:       "immediate",
				MutexLocking: "full",
			},
			expected: "file:locked.db?_locking_mode=EXCLUSIVE&_mutex=full&_txlock=immediate",
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
			expected: "file:advanced.db?_auto_vacuum=full&_case_sensitive_like=true&_defer_foreign_keys=true&_ignore_check_constraints=true&_query_only=true&_recursive_triggers=true&_secure_delete=FAST&_writable_schema=true",
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
			expected: "file:auth.db?_auth=&_auth_crypt=SHA256&_auth_pass=pass&_auth_salt=salt&_auth_user=user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.buildConnectionString()
			assert.Equal(t, tt.expected, result)
		})
	}
}
