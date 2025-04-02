package database

import (
	"net/url"
	"strings"
)

// buildConnectionString generates a SQLite connection string from options.
// It only includes parameters that are directly supported by the ncruces/go-sqlite3
// driver via the DSN URI. Other options must be set via PRAGMA commands after connection.
func (opts *SQLiteOptions) buildConnectionString() string {
	params := url.Values{}

	// Options supported directly via DSN URI parameters
	if opts.Cache != "" {
		params.Set("cache", string(opts.Cache))
	}
	if opts.Immutable {
		params.Set("immutable", "1") // Use "1" for true as per docs
	}
	// NOTE: Other options like Journal, ForeignKeys, BusyTimeout, CacheSize, Synchronous,
	// LockingMode, TxLock, MutexLocking, AutoVacuum, etc., are NOT set via DSN parameters.
	// They must be set using PRAGMA commands after the connection is established.
	// Authentication options are also not handled via DSN parameters.

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
