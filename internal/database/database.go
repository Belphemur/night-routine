package database

import (
	"context"
	"database/sql"
	"embed"
	"errors" // Import errors package for Join
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/ncruces/go-sqlite3/driver" // Register ncruces sqlite3 driver

	"github.com/belphemur/night-routine/internal/database/sqlite3"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/rs/zerolog"
)

//go:embed migrations
var migrationsFS embed.FS

// DB manages the database connection
type DB struct {
	conn   *sql.DB
	logger zerolog.Logger
	dbPath string // Store dbPath for logging
}

// Removed NewWithOptions as New now directly accepts SQLiteOptions

// New creates a new database connection using the provided options.
// It configures the connection using both DSN parameters (for supported options like mode, cache, immutable)
// and explicit PRAGMA commands executed after the connection is established for other settings.
func New(opts SQLiteOptions) (*DB, error) {
	// Build connection string with only URI-supported parameters
	connStr := opts.buildConnectionString()
	logger := logging.GetLogger("database").With().Str("db_path", opts.Path).Logger() // Use opts.Path for logging
	logger.Info().Str("connection_string", connStr).Msg("Opening database connection")
	conn, err := sql.Open("sqlite3", connStr)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to open database")
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply PRAGMAs not supported by DSN
	if err = applyPragmas(conn, opts, logger); err != nil {
		conn.Close()    // Close connection if PRAGMA application fails
		return nil, err // Return the specific PRAGMA error
	}

	// Ping the database to ensure connection and PRAGMAs are valid
	if err = conn.Ping(); err != nil {
		logger.Error().Err(err).Msg("Failed to ping database after open and applying PRAGMAs")
		conn.Close() // Close the connection if ping fails
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Info().Msg("Database connection opened and configured successfully")

	return &DB{conn: conn, logger: logger, dbPath: opts.Path}, nil // Store opts.Path
}

// applyPragmas executes PRAGMA commands based on SQLiteOptions after the connection is opened.
// It iterates through the options and applies the corresponding PRAGMA if the option
// has a non-default value. It attempts to apply all specified PRAGMAs and returns
// a combined error if one or more PRAGMA applications fail.
// pragmaConfig defines how a PRAGMA should be handled
type pragmaConfig struct {
	name        string                             // Name of the PRAGMA
	value       interface{}                        // Value to set
	allowZero   bool                               // Whether zero/false values should be applied
	formatValue func(v interface{}) (string, bool) // Custom value formatter, returns formatted value and whether to skip
}

func applyPragmas(conn *sql.DB, opts SQLiteOptions, logger zerolog.Logger) error {
	var errs []error // Slice to collect errors

	// Define formatters for specific types
	boolFormatter := func(v interface{}) (string, bool) {
		if b, ok := v.(bool); ok {
			if b {
				return "1", false
			}
			return "0", false
		}
		return "", true
	}

	syncFormatter := func(v interface{}) (string, bool) {
		if mode, ok := v.(SynchronousMode); ok && mode != "" {
			switch mode {
			case SynchronousOff:
				return "0", false
			case SynchronousNormal:
				return "1", false
			case SynchronousFull:
				return "2", false
			case SynchronousExtra:
				return "3", false
			}
		}
		return "", true
	}

	// Default string formatter for enums that ensures values are uppercase for SQLite
	enumFormatter := func(v interface{}) (string, bool) {
		switch val := v.(type) {
		case JournalMode:
			if val != "" {
				return string(val), false // JournalMode constants are already uppercase
			}
		case SynchronousMode:
			if val != "" {
				return string(val), false // SynchronousMode constants are already uppercase
			}
		case LockingMode:
			if val != "" {
				return string(val), false // LockingMode constants are already uppercase
			}
		case fmt.Stringer:
			if s := val.String(); s != "" {
				return s, false
			}
		case string:
			if val != "" {
				return val, false
			}
		}
		return "", true
	}

	pragmas := []pragmaConfig{
		{"journal_mode", opts.Journal, false, enumFormatter},
		{"busy_timeout", opts.BusyTimeout, true, nil},           // Always set busy_timeout
		{"foreign_keys", opts.ForeignKeys, true, boolFormatter}, // Always set foreign_keys
		{"synchronous", opts.Synchronous, false, syncFormatter},
		{"cache_size", opts.CacheSize, false, nil},
		{"locking_mode", opts.LockingMode, false, enumFormatter},
		{"auto_vacuum", opts.AutoVacuum, false, nil},
		{"case_sensitive_like", opts.CaseSensitiveLike, false, boolFormatter},
		{"defer_foreign_keys", opts.DeferForeignKeys, true, boolFormatter}, // Always set defer_foreign_keys
		{"ignore_check_constraints", opts.IgnoreCheckConstraints, false, boolFormatter},
		{"query_only", opts.QueryOnly, false, boolFormatter},
		{"recursive_triggers", opts.RecursiveTriggers, false, boolFormatter},
		{"secure_delete", opts.SecureDelete, false, nil},
		{"writable_schema", opts.WritableSchema, false, boolFormatter},
	}

	// Format and apply each PRAGMA
	for _, p := range pragmas {
		var sqlValueStr string

		switch v := p.value.(type) {
		case int:
			if v == 0 && !p.allowZero {
				continue
			}
			sqlValueStr = fmt.Sprintf("%d", v)

		case string:
			if v == "" {
				continue
			}
			sqlValueStr = v

		default:
			if p.formatValue != nil {
				var skipFormat bool
				sqlValueStr, skipFormat = p.formatValue(p.value)
				if skipFormat {
					continue
				}
			} else {
				// For any other type, skip if nil or non-zero check fails
				if p.value == nil || (!p.allowZero && isZero(p.value)) {
					continue
				}
				sqlValueStr = fmt.Sprint(p.value)
			}
		}

		// Build the full query string with the value embedded
		query := fmt.Sprintf("PRAGMA %s = %s;", p.name, sqlValueStr)
		logger.Debug().Str("pragma", p.name).Str("value", sqlValueStr).Str("query", query).Msg("Applying PRAGMA")
		_, err := conn.Exec(query) // Execute the full query string
		if err != nil {
			// Attempt to query the value if setting failed, maybe it's read-only or needs specific context
			var currentValue interface{}
			queryErr := conn.QueryRow(fmt.Sprintf("PRAGMA %s;", p.name)).Scan(&currentValue)
			logCtx := logger.Error().Err(err).Str("pragma", p.name).Str("attempted_value", sqlValueStr).Str("query", query)
			if queryErr == nil {
				logCtx = logCtx.Interface("current_value", currentValue)
			}
			logCtx.Msg("Failed to apply PRAGMA")
			// Collect the error instead of returning immediately
			errs = append(errs, fmt.Errorf("failed to apply PRAGMA %s=%s: %w", p.name, sqlValueStr, err))
		}
	}
	// Return a combined error if any PRAGMAs failed
	return errors.Join(errs...)
}

// isZero returns true if the value is the zero value for its type
func isZero(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return !val
	case int:
		return val == 0
	case string:
		return val == ""
	case fmt.Stringer:
		return val.String() == ""
	default:
		return v == nil
	}
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// beginTx starts a new database transaction with the given options (private method)
func (db *DB) beginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	db.logger.Debug().Msg("Starting database transaction")
	tx, err := db.conn.BeginTx(ctx, opts)
	if err != nil {
		db.logger.Error().Err(err).Msg("Failed to start database transaction")
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	db.logger.Debug().Msg("Database transaction started successfully")
	return tx, nil
}

// WithTransaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (db *DB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.beginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			db.logger.Error().Interface("panic", p).Msg("Panic occurred during transaction, rolling back")
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				db.logger.Error().Err(rollbackErr).Msg("Failed to rollback transaction during panic recovery")
			}
			panic(p) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		db.logger.Debug().Err(err).Msg("Transaction function returned error, rolling back")
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			db.logger.Error().Err(rollbackErr).Msg("Failed to rollback transaction")
			return fmt.Errorf("transaction failed: %w, rollback failed: %v", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		db.logger.Error().Err(err).Msg("Failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	db.logger.Debug().Msg("Transaction committed successfully")
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	db.logger.Info().Msg("Closing database connection")
	if err := db.conn.Close(); err != nil {
		db.logger.Error().Err(err).Msg("Failed to close database connection")
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	db.logger.Info().Msg("Database connection closed successfully")
	return nil
}

// MigrateDatabase performs database migrations
func (db *DB) MigrateDatabase() error {
	db.logger.Info().Msg("Starting database migration")
	// Create a new instance of the SQLite driver
	db.logger.Debug().Msg("Creating migration driver instance")
	driver, err := sqlite3.WithInstance(db.conn, &sqlite3.Config{})
	if err != nil {
		db.logger.Error().Err(err).Msg("Failed to create database driver for migration")
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Extract the sub-filesystem containing only the migrations
	db.logger.Debug().Msg("Extracting migration source filesystem")
	subFS, err := fs.Sub(migrationsFS, "migrations/sqlite")
	if err != nil {
		db.logger.Error().Err(err).Msg("Failed to create sub-filesystem for migrations")
		return fmt.Errorf("failed to create sub-filesystem: %w", err)
	}

	// Create a new instance of the embed source driver
	db.logger.Debug().Msg("Creating migration source instance")
	sourceInstance, err := iofs.New(subFS, ".")
	if err != nil {
		db.logger.Error().Err(err).Msg("Failed to create embedded file source for migration")
		return fmt.Errorf("failed to create embedded file source: %w", err)
	}

	// Create a new instance of the migrator
	db.logger.Debug().Msg("Creating migrator instance")
	m, err := migrate.NewWithInstance("iofs", sourceInstance, "sqlite3", driver)
	if err != nil {
		db.logger.Error().Err(err).Msg("Failed to create migrator instance")
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Get current migration version
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		db.logger.Error().Err(err).Msg("Failed to get current migration version")
		return fmt.Errorf("failed to get migration version: %w", err)
	}
	db.logger.Info().Uint("current_version", currentVersion).Bool("dirty", dirty).Msg("Current database migration version")

	// Run the migrations up
	db.logger.Info().Msg("Applying migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		db.logger.Error().Err(err).Msg("Failed to apply migrations")
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Check version again after migration
	newVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		db.logger.Error().Err(err).Msg("Failed to get migration version after applying migrations")
		// Don't return error here, migration might have partially succeeded
	}

	if err == migrate.ErrNoChange {
		db.logger.Info().Msg("No new migrations to apply")
	} else {
		db.logger.Info().Uint("previous_version", currentVersion).Uint("new_version", newVersion).Bool("dirty", dirty).Msg("Migrations applied successfully")
	}

	return nil
}
