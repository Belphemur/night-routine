package fairness

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/database"
)

// Tracker maintains the state of night routine assignments
type Tracker struct {
	db *sql.DB
}

// New creates a new Tracker instance
func New(db *database.DB) (*Tracker, error) {
	return &Tracker{db: db.Conn()}, nil
}

// RecordAssignment records a new assignment in the state database
func (t *Tracker) RecordAssignment(parent string, date time.Time) error {
	_, err := t.db.Exec(`
INSERT INTO assignments (parent_name, assignment_date) 
VALUES (?, ?)
`, parent, date.Format("2006-01-02"))

	if err != nil {
		return fmt.Errorf("failed to record assignment: %w", err)
	}

	return nil
}

// GetLastAssignments returns the last n assignments
func (t *Tracker) GetLastAssignments(n int) ([]Assignment, error) {
	rows, err := t.db.Query(`
SELECT parent_name, assignment_date 
FROM assignments 
ORDER BY assignment_date DESC 
LIMIT ?
`, n)
	if err != nil {
		return nil, fmt.Errorf("failed to query assignments: %w", err)
	}
	defer rows.Close()

	var assignments []Assignment
	for rows.Next() {
		var a Assignment
		var dateStr string
		if err := rows.Scan(&a.Parent, &dateStr); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}
		a.Date = date
		assignments = append(assignments, a)
	}

	return assignments, nil
}

// GetParentStats returns statistics for each parent
func (t *Tracker) GetParentStats() (map[string]Stats, error) {
	rows, err := t.db.Query(`
SELECT 
parent_name,
COUNT(*) as total_assignments,
SUM(CASE WHEN assignment_date >= date('now', '-30 days') THEN 1 ELSE 0 END) as last_30_days
FROM assignments
GROUP BY parent_name
`)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]Stats)
	for rows.Next() {
		var parentName string
		var s Stats
		if err := rows.Scan(&parentName, &s.TotalAssignments, &s.Last30Days); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats[parentName] = s
	}

	return stats, nil
}

// Assignment represents a night routine assignment
type Assignment struct {
	Parent string
	Date   time.Time
}

// Stats represents statistics for a parent
type Stats struct {
	TotalAssignments int
	Last30Days       int
}
