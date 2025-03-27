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
)

//go:embed migrations
var migrationsFS embed.FS

// DB manages the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// MigrateDatabase performs database migrations
func (db *DB) MigrateDatabase() error {
	// Create a new instance of the SQLite driver
	driver, err := sqlite3.WithInstance(db.conn, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Extract the sub-filesystem containing only the migrations
	subFS, err := fs.Sub(migrationsFS, "migrations/sqlite")
	if err != nil {
		return fmt.Errorf("failed to create sub-filesystem: %w", err)
	}

	// Create a new instance of the embed source driver
	sourceInstance, err := iofs.New(subFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create embedded file source: %w", err)
	}

	// Create a new instance of the migrator
	m, err := migrate.NewWithInstance("iofs", sourceInstance, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Run the migrations up
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// InitSchema is kept for backward compatibility but delegates to MigrateDatabase
// Deprecated: Use MigrateDatabase instead
func (db *DB) InitSchema() error {
	return db.MigrateDatabase()
}
