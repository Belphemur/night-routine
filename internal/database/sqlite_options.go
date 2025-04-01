package database

// SynchronousMode represents the available synchronous settings for SQLite
type SynchronousMode string

const (
	SynchronousOff    SynchronousMode = "OFF"
	SynchronousNormal SynchronousMode = "NORMAL"
	SynchronousFull   SynchronousMode = "FULL"
	SynchronousExtra  SynchronousMode = "EXTRA"
)

// JournalMode represents the available journal modes for SQLite
type JournalMode string

const (
	JournalDelete   JournalMode = "DELETE"
	JournalTruncate JournalMode = "TRUNCATE"
	JournalPersist  JournalMode = "PERSIST"
	JournalMemory   JournalMode = "MEMORY"
	JournalWAL      JournalMode = "WAL"
	JournalOff      JournalMode = "OFF"
)

// LockingMode represents the available locking modes for SQLite
type LockingMode string

const (
	LockingNormal    LockingMode = "NORMAL"
	LockingExclusive LockingMode = "EXCLUSIVE"
)

// CacheMode represents the available cache modes for SQLite
type CacheMode string

const (
	CacheShared  CacheMode = "shared"
	CachePrivate CacheMode = "private"
)

// SQLiteOptions contains configuration options for SQLite connection
type SQLiteOptions struct {
	// Path to the SQLite database file
	Path string

	// Core Options
	Mode        string          // ro, rw, rwc, memory
	Journal     JournalMode     // _journal_mode: DELETE, TRUNCATE, PERSIST, MEMORY, WAL, OFF
	ForeignKeys bool            // _foreign_keys=true
	BusyTimeout int             // _busy_timeout (milliseconds)
	CacheSize   int             // _cache_size (in KB, negative for number of pages)
	Synchronous SynchronousMode // _synchronous: OFF, NORMAL, FULL, EXTRA
	Cache       CacheMode       // shared, private
	Immutable   bool            // immutable=true/false

	// Transaction & Locking
	LockingMode  LockingMode // _locking_mode: NORMAL, EXCLUSIVE
	TxLock       string      // _txlock: immediate, deferred, exclusive
	MutexLocking string      // _mutex: no, full

	// Advanced Options
	AutoVacuum             string // _auto_vacuum: none, full, incremental
	CaseSensitiveLike      bool   // _case_sensitive_like
	DeferForeignKeys       bool   // _defer_foreign_keys
	IgnoreCheckConstraints bool   // _ignore_check_constraints
	QueryOnly              bool   // _query_only
	RecursiveTriggers      bool   // _recursive_triggers
	SecureDelete           string // _secure_delete: boolean or "FAST"
	WritableSchema         bool   // _writable_schema

	// Authentication
	Auth      bool   // _auth
	AuthUser  string // _auth_user
	AuthPass  string // _auth_pass
	AuthCrypt string // _auth_crypt: SHA1, SSHA1, SHA256, etc.
	AuthSalt  string // _auth_salt
}

// NewDefaultOptions creates SQLiteOptions with recommended defaults
func NewDefaultOptions(path string) SQLiteOptions {
	return SQLiteOptions{
		Path:        path,
		Mode:        "rwc",
		Journal:     JournalWAL, // WAL is recommended for better concurrency
		ForeignKeys: true,
		BusyTimeout: 5000,
		CacheSize:   2000,
		Synchronous: SynchronousNormal,
		Cache:       CachePrivate,
	}
}
