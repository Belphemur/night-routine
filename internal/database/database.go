package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"

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

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	logger := logging.GetLogger("database").With().Str("db_path", dbPath).Logger()
	logger.Info().Msg("Opening database connection")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to open database")
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ping the database to ensure connection is valid
	if err := conn.Ping(); err != nil {
		logger.Error().Err(err).Msg("Failed to ping database after open")
		conn.Close() // Close the connection if ping fails
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Info().Msg("Database connection opened successfully")

	return &DB{conn: conn, logger: logger, dbPath: dbPath}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	db.logger.Info().Msg("Closing database connection")
	err := db.conn.Close()
	if err != nil {
		db.logger.Error().Err(err).Msg("Failed to close database connection")
		return err
	}
	db.logger.Info().Msg("Database connection closed successfully")
	return nil
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
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

// InitSchema is kept for backward compatibility but delegates to MigrateDatabase
// Deprecated: Use MigrateDatabase instead
func (db *DB) InitSchema() error {
	db.logger.Warn().Msg("InitSchema is deprecated, use MigrateDatabase")
	return db.MigrateDatabase()
}
