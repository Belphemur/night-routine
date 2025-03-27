package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

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

// InitSchema initializes the database schema
func (db *DB) InitSchema() error {
	// Initialize assignments table
	_, err := db.conn.Exec(`
CREATE TABLE IF NOT EXISTS assignments (
id INTEGER PRIMARY KEY AUTOINCREMENT,
parent_name TEXT NOT NULL,
assignment_date TEXT NOT NULL,
created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`)
	if err != nil {
		return fmt.Errorf("failed to initialize assignments table: %w", err)
	}

	// Initialize oauth_tokens and calendar_settings tables
	_, err = db.conn.Exec(`
CREATE TABLE IF NOT EXISTS oauth_tokens (
id INTEGER PRIMARY KEY,
token_data BLOB NOT NULL,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS calendar_settings (
id INTEGER PRIMARY KEY,
calendar_id TEXT NOT NULL,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`)
	if err != nil {
		return fmt.Errorf("failed to initialize token tables: %w", err)
	}

	return nil
}
