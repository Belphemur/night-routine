package database

import (
	"net/url"
	"strconv"
	"strings"
)

// buildConnectionString generates a SQLite connection string from options
func (opts *SQLiteOptions) buildConnectionString() string {
	params := url.Values{}

	// Core options
	if opts.Journal != "" {
		params.Set("_journal_mode", string(opts.Journal))
	}
	if opts.ForeignKeys {
		params.Set("_foreign_keys", "true")
	}
	if opts.BusyTimeout > 0 {
		params.Set("_busy_timeout", strconv.Itoa(opts.BusyTimeout))
	}
	if opts.CacheSize != 0 {
		params.Set("_cache_size", strconv.Itoa(opts.CacheSize))
	}
	if opts.Synchronous != "" {
		params.Set("_synchronous", string(opts.Synchronous))
	}
	if opts.Cache != "" {
		params.Set("cache", string(opts.Cache))
	}
	if opts.Immutable {
		params.Set("immutable", "true")
	}

	// Transaction & Locking options
	if opts.LockingMode != "" {
		params.Set("_locking_mode", string(opts.LockingMode))
	}
	if opts.TxLock != "" {
		params.Set("_txlock", opts.TxLock)
	}
	if opts.MutexLocking != "" {
		params.Set("_mutex", opts.MutexLocking)
	}

	// Advanced options
	if opts.AutoVacuum != "" {
		params.Set("_auto_vacuum", opts.AutoVacuum)
	}
	if opts.CaseSensitiveLike {
		params.Set("_case_sensitive_like", "true")
	}
	if opts.DeferForeignKeys {
		params.Set("_defer_foreign_keys", "true")
	}
	if opts.IgnoreCheckConstraints {
		params.Set("_ignore_check_constraints", "true")
	}
	if opts.QueryOnly {
		params.Set("_query_only", "true")
	}
	if opts.RecursiveTriggers {
		params.Set("_recursive_triggers", "true")
	}
	if opts.SecureDelete != "" {
		params.Set("_secure_delete", opts.SecureDelete)
	}
	if opts.WritableSchema {
		params.Set("_writable_schema", "true")
	}

	// Authentication options
	if opts.Auth {
		params.Set("_auth", "")
		if opts.AuthUser != "" {
			params.Set("_auth_user", opts.AuthUser)
		}
		if opts.AuthPass != "" {
			params.Set("_auth_pass", opts.AuthPass)
		}
		if opts.AuthCrypt != "" {
			params.Set("_auth_crypt", opts.AuthCrypt)
		}
		if opts.AuthSalt != "" {
			params.Set("_auth_salt", opts.AuthSalt)
		}
	}

	// Mode
	if opts.Mode != "" {
		params.Set("mode", opts.Mode)
	}

	// Build the final connection string
	connStr := opts.Path
	if !strings.HasPrefix(connStr, "file:") {
		connStr = "file:" + connStr
	}
	if encoded := params.Encode(); encoded != "" {
		connStr += "?" + encoded
	}

	return connStr
}
