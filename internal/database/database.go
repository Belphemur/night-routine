package database

import (
	"database/sql"
	"embed"
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
// has a non-default value. It returns an error immediately if any PRAGMA fails.
func applyPragmas(conn *sql.DB, opts SQLiteOptions, logger zerolog.Logger) error {
	pragmas := []struct {
		Name  string
		Value interface{}
	}{
		{"journal_mode", opts.Journal},
		{"busy_timeout", opts.BusyTimeout},
		{"foreign_keys", opts.ForeignKeys},
		{"synchronous", opts.Synchronous},
		{"cache_size", opts.CacheSize},
		{"locking_mode", opts.LockingMode},
		// {"txlock", opts.TxLock}, // Covered by locking_mode generally
		{"auto_vacuum", opts.AutoVacuum},
		{"case_sensitive_like", opts.CaseSensitiveLike},
		{"defer_foreign_keys", opts.DeferForeignKeys},
		{"ignore_check_constraints", opts.IgnoreCheckConstraints},
		{"query_only", opts.QueryOnly},
		{"recursive_triggers", opts.RecursiveTriggers},
		{"secure_delete", opts.SecureDelete},
		{"writable_schema", opts.WritableSchema},
		// Add other PRAGMAs here if needed
	}

	for _, p := range pragmas {
		var sqlValueStr string // Store the value formatted for SQL query
		skip := false

		// Determine the SQL value and whether to skip based on type and value
		switch v := p.Value.(type) {
		case JournalMode:
			if v == "" {
				skip = true
			} else {
				// String enums can usually be embedded directly
				sqlValueStr = string(v)
			}
		case int:
			// Allow setting 0 explicitly for some pragmas like foreign_keys, busy_timeout
			if v == 0 && !(p.Name == "foreign_keys" || p.Name == "busy_timeout") {
				skip = true
			} else {
				// Integers can be converted to string
				sqlValueStr = fmt.Sprintf("%d", v)
			}
		case bool:
			// Apply boolean pragmas only if they are true (or explicitly false for foreign_keys)
			if !v && p.Name != "foreign_keys" && p.Name != "defer_foreign_keys" {
				skip = true
			} else {
				if v {
					sqlValueStr = "1"
				} else {
					sqlValueStr = "0"
				}
			}
		case SynchronousMode:
			if v == "" {
				skip = true
			} else {
				switch v {
				case SynchronousOff:
					sqlValueStr = "0"
				case SynchronousNormal:
					sqlValueStr = "1"
				case SynchronousFull:
					sqlValueStr = "2"
				case SynchronousExtra:
					sqlValueStr = "3"
				default:
					logger.Warn().Str("pragma", p.Name).Str("value", string(v)).Msg("Unknown synchronous mode value, skipping PRAGMA.")
					skip = true
				}
			}
		case LockingMode:
			if v == "" {
				skip = true
			} else {
				// String enums can usually be embedded directly
				sqlValueStr = string(v)
			}
		case string:
			if v == "" {
				skip = true
			} else {
				// Handle potential string values that might need quoting?
				// For now, assume simple string values like 'FAST' or 'incremental'
				// If values could contain spaces or special chars, quoting might be needed.
				// Example: sqlValueStr = "'" + strings.ReplaceAll(v, "'", "''") + "'"
				sqlValueStr = v // Embed directly for now
			}
		default:
			logger.Warn().Str("pragma", p.Name).Interface("value", p.Value).Msg("Unsupported PRAGMA type, skipping.")
			skip = true
		}

		if skip {
			continue
		}

		// Build the full query string with the value embedded
		query := fmt.Sprintf("PRAGMA %s = %s;", p.Name, sqlValueStr)
		logger.Debug().Str("pragma", p.Name).Str("value", sqlValueStr).Str("query", query).Msg("Applying PRAGMA")
		_, err := conn.Exec(query) // Execute the full query string
		if err != nil {
			// Attempt to query the value if setting failed, maybe it's read-only or needs specific context
			var currentValue interface{}
			queryErr := conn.QueryRow(fmt.Sprintf("PRAGMA %s;", p.Name)).Scan(&currentValue)
			logCtx := logger.Error().Err(err).Str("pragma", p.Name).Str("attempted_value", sqlValueStr).Str("query", query)
			if queryErr == nil {
				logCtx = logCtx.Interface("current_value", currentValue)
			}
			logCtx.Msg("Failed to apply PRAGMA")
			return fmt.Errorf("failed to apply PRAGMA %s=%s: %w", p.Name, sqlValueStr, err)
		}
	}
	return nil
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
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
