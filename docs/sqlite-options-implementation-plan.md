# SQLite Options Implementation Plan

## Overview

This document outlines the plan for implementing a new SQLite options configuration system in the database package. The goal is to provide a type-safe, well-documented way to configure SQLite database connections with all available options.

## Implementation Details

### New Types

```go
// SQLiteOptions contains all configuration options for SQLite connection
type SQLiteOptions struct {
    // Path to the SQLite database file
    Path string

    // Core Options
    Mode          string  // ro, rw, rwc, memory
    UseWAL        bool    // _journal_mode=WAL
    ForeignKeys   bool    // _foreign_keys=true
    BusyTimeout   int     // _busy_timeout (milliseconds)
    CacheSize     int     // _cache_size (in KB, negative for number of pages)
    Synchronous   string  // _synchronous: OFF, NORMAL, FULL, EXTRA
    Cache         string  // shared, private
    Immutable     bool    // immutable=true/false

    // Transaction & Locking
    LockingMode   string  // _locking_mode: NORMAL, EXCLUSIVE
    TxLock        string  // _txlock: immediate, deferred, exclusive
    MutexLocking  string  // _mutex: no, full

    // Advanced Options
    AutoVacuum           string  // _auto_vacuum: none, full, incremental
    CaseSensitiveLike    bool    // _case_sensitive_like
    DeferForeignKeys     bool    // _defer_foreign_keys
    IgnoreCheckConstraints bool  // _ignore_check_constraints
    QueryOnly           bool    // _query_only
    RecursiveTriggers   bool    // _recursive_triggers
    SecureDelete        string  // _secure_delete: boolean or "FAST"
    WritableSchema      bool    // _writable_schema

    // Authentication
    Auth        bool    // _auth
    AuthUser    string  // _auth_user
    AuthPass    string  // _auth_pass
    AuthCrypt   string  // _auth_crypt: SHA1, SSHA1, SHA256, etc.
    AuthSalt    string  // _auth_salt
}
```

### Constructor and Methods

```go
// NewDefaultOptions creates SQLiteOptions with recommended defaults
func NewDefaultOptions(path string) SQLiteOptions {
    return SQLiteOptions{
        Path:         path,
        Mode:         "rwc",
        UseWAL:      true,
        ForeignKeys:  true,
        BusyTimeout:  5000,
        CacheSize:    2000,
        Synchronous:  "NORMAL",
        Cache:        "private",
    }
}

// NewWithOptions creates a new database connection with the specified options
func NewWithOptions(opts SQLiteOptions) (*DB, error)
```

## Connection String Generation

The options will be converted to a connection string using the following format:

```
file:path/to/database.db?_journal_mode=WAL&_foreign_keys=on&...
```

Each option will be properly escaped and validated before being added to the connection string.

## Usage Examples

### Basic Usage with Defaults

```go
opts := database.NewDefaultOptions("./data.db")
db, err := database.NewWithOptions(opts)
```

### Custom Configuration

```go
opts := database.SQLiteOptions{
    Path:         "./data.db",
    Mode:         "rwc",
    UseWAL:       true,
    ForeignKeys:  true,
    BusyTimeout:  10000,
    CacheSize:    5000,
    Synchronous:  "FULL",
    Cache:        "private",
    LockingMode:  "NORMAL",
}
db, err := database.NewWithOptions(opts)
```

### Memory Database with Custom Settings

```go
opts := database.SQLiteOptions{
    Path:         ":memory:",
    Mode:         "memory",
    UseWAL:       false,  // WAL not needed for memory databases
    ForeignKeys:  true,
    CacheSize:    1000,
}
db, err := database.NewWithOptions(opts)
```

## Implementation Plan

1. Add the SQLiteOptions struct to internal/database/database.go
2. Implement NewDefaultOptions constructor
3. Implement connection string generation with proper escaping
4. Add NewWithOptions constructor that validates options and creates connection
5. Update documentation and add examples
6. Add tests for all new functionality

## Migration Guide

Existing code using the current New() constructor will continue to work unchanged:

```go
// Old code - still supported
db, err := database.New("./data.db")

// New code - with options
opts := database.NewDefaultOptions("./data.db")
opts.CacheSize = 5000
db, err := database.NewWithOptions(opts)
```

## Available Options Reference

| Option       | Key            | Values                                      | Description                                    |
| ------------ | -------------- | ------------------------------------------- | ---------------------------------------------- |
| Journal Mode | \_journal_mode | WAL, DELETE, TRUNCATE, PERSIST, MEMORY, OFF | Write-Ahead Logging configuration              |
| Foreign Keys | \_foreign_keys | boolean                                     | Enable/disable foreign key constraints         |
| Busy Timeout | \_busy_timeout | int (ms)                                    | How long to wait for locks                     |
| Cache Size   | \_cache_size   | int                                         | Maximum number of disk pages to hold in memory |
| Synchronous  | \_synchronous  | OFF, NORMAL, FULL, EXTRA                    | Disk synchronization mode                      |

[...full options table as provided...]

## Testing Plan

1. Test connection string generation
2. Test option validation
3. Test default options
4. Test each option's effect on database behavior
5. Test error cases and invalid configurations
