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
func (t *Tracker) RecordAssignment(parent string, date time.Time) (*Assignment, error) {
	// Check if there's already an assignment for this date
	existingAssignment, err := t.GetAssignmentByDate(date)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}

	// If there's already an assignment, update it
	if existingAssignment != nil {
		// Only update if the parent has changed
		if existingAssignment.Parent != parent {
			_, err := t.db.Exec(`
			UPDATE assignments
			SET parent_name = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
			`, parent, existingAssignment.ID)

			if err != nil {
				return nil, fmt.Errorf("failed to update assignment: %w", err)
			}

			// Refresh the assignment
			return t.GetAssignmentByID(existingAssignment.ID)
		}

		// Parent hasn't changed, return the existing assignment
		return existingAssignment, nil
	}

	// No existing assignment, create a new one
	return t.RecordAssignmentWithDetails(parent, date, false, "")
}

// RecordAssignmentWithOverride records a new assignment with override flag
func (t *Tracker) RecordAssignmentWithOverride(parent string, date time.Time, override bool) (*Assignment, error) {
	return t.RecordAssignmentWithDetails(parent, date, override, "")
}

// RecordAssignmentWithDetails records an assignment with all available details
func (t *Tracker) RecordAssignmentWithDetails(parent string, date time.Time, override bool, googleCalendarEventID string) (*Assignment, error) {
	// Check if there's already an assignment for this date
	existingAssignment, err := t.GetAssignmentByDate(date)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}

	// If there's already an assignment, update it
	if existingAssignment != nil {
		// Update the assignment
		_, err := t.db.Exec(`
		UPDATE assignments
		SET parent_name = ?, override = ?, google_calendar_event_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		`, parent, override, googleCalendarEventID, existingAssignment.ID)

		if err != nil {
			return nil, fmt.Errorf("failed to update assignment: %w", err)
		}

		// Refresh the assignment
		return t.GetAssignmentByID(existingAssignment.ID)
	}

	// No existing assignment, create a new one
	result, err := t.db.Exec(`
	INSERT INTO assignments (parent_name, assignment_date, override, google_calendar_event_id) 
	VALUES (?, ?, ?, ?)
	`, parent, date.Format("2006-01-02"), override, googleCalendarEventID)

	if err != nil {
		return nil, fmt.Errorf("failed to record assignment: %w", err)
	}

	// Get the last inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Get the full assignment record
	assignment, err := t.GetAssignmentByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment after insert: %w", err)
	}

	return assignment, nil
}

// GetAssignmentByID retrieves an assignment by its ID
func (t *Tracker) GetAssignmentByID(id int64) (*Assignment, error) {
	row := t.db.QueryRow(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
	FROM assignments 
	WHERE id = ?
	`, id)

	var a Assignment
	var dateStr string
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&a.ID,
		&a.Parent,
		&dateStr,
		&a.Override,
		&a.GoogleCalendarEventID,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("assignment not found: %d", id)
		}
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}
	a.Date = date

	createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		a.CreatedAt = createdAt
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err == nil {
		a.UpdatedAt = updatedAt
	}

	return &a, nil
}

// UpdateAssignmentGoogleCalendarEventID updates an assignment with its Google Calendar event ID
func (t *Tracker) UpdateAssignmentGoogleCalendarEventID(id int64, googleCalendarEventID string) error {
	_, err := t.db.Exec(`
	UPDATE assignments 
	SET google_calendar_event_id = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`, googleCalendarEventID, id)

	if err != nil {
		return fmt.Errorf("failed to update assignment with Google Calendar event ID: %w", err)
	}

	return nil
}

// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
func (t *Tracker) UpdateAssignmentParent(id int64, parent string, override bool) error {
	_, err := t.db.Exec(`
	UPDATE assignments
	SET parent_name = ?, override = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`, parent, override, id)

	if err != nil {
		return fmt.Errorf("failed to update assignment parent: %w", err)
	}

	return nil
}

// GetLastAssignmentsUntil returns the last n assignments up to a specific date
func (t *Tracker) GetLastAssignmentsUntil(n int, until time.Time) ([]*Assignment, error) {
	untilStr := until.Format("2006-01-02")
	rows, err := t.db.Query(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
	FROM assignments 
	WHERE assignment_date < ?
	ORDER BY assignment_date DESC 
	LIMIT ?
	`, untilStr, n)
	if err != nil {
		return nil, fmt.Errorf("failed to query assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		var a Assignment
		var dateStr string
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&a.ID,
			&a.Parent,
			&dateStr,
			&a.Override,
			&a.GoogleCalendarEventID,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}
		a.Date = date

		createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err == nil {
			a.CreatedAt = createdAt
		}

		updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err == nil {
			a.UpdatedAt = updatedAt
		}

		assignments = append(assignments, &a)
	}

	return assignments, nil
}

// GetAssignmentByDate retrieves an assignment for a specific date
func (t *Tracker) GetAssignmentByDate(date time.Time) (*Assignment, error) {
	dateStr := date.Format("2006-01-02")
	row := t.db.QueryRow(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
	FROM assignments 
	WHERE assignment_date = ?
	ORDER BY id DESC
	LIMIT 1
	`, dateStr)

	var a Assignment
	var rowDateStr string
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&a.ID,
		&a.Parent,
		&rowDateStr,
		&a.Override,
		&a.GoogleCalendarEventID,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No assignment found, which is ok
		}
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	assignmentDate, err := time.Parse("2006-01-02", rowDateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}
	a.Date = assignmentDate

	createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		a.CreatedAt = createdAt
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err == nil {
		a.UpdatedAt = updatedAt
	}

	return &a, nil
}

// GetAssignmentByGoogleCalendarEventID retrieves an assignment by its Google Calendar event ID
func (t *Tracker) GetAssignmentByGoogleCalendarEventID(eventID string) (*Assignment, error) {
	if eventID == "" {
		return nil, nil
	}

	row := t.db.QueryRow(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
	FROM assignments
	WHERE google_calendar_event_id = ?
	`, eventID)

	var a Assignment
	var dateStr string
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&a.ID,
		&a.Parent,
		&dateStr,
		&a.Override,
		&a.GoogleCalendarEventID,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No assignment found, which is ok
		}
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}
	a.Date = date

	createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		a.CreatedAt = createdAt
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err == nil {
		a.UpdatedAt = updatedAt
	}

	return &a, nil
}

// GetAssignmentsInRange retrieves all assignments in a date range
func (t *Tracker) GetAssignmentsInRange(start, end time.Time) ([]*Assignment, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	rows, err := t.db.Query(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
	FROM assignments
	WHERE assignment_date >= ? AND assignment_date <= ?
	ORDER BY assignment_date ASC
	`, startStr, endStr)

	if err != nil {
		return nil, fmt.Errorf("failed to query assignments in range: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		var a Assignment
		var dateStr string
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&a.ID,
			&a.Parent,
			&dateStr,
			&a.Override,
			&a.GoogleCalendarEventID,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}
		a.Date = date

		createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err == nil {
			a.CreatedAt = createdAt
		}

		updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err == nil {
			a.UpdatedAt = updatedAt
		}

		assignments = append(assignments, &a)
	}

	return assignments, nil
}

// GetParentStatsUntil returns statistics for each parent up to a specific date
func (t *Tracker) GetParentStatsUntil(until time.Time) (map[string]Stats, error) {
	untilStr := until.Format("2006-01-02")
	thirtyDaysBeforeUntil := until.AddDate(0, 0, -30).Format("2006-01-02")

	rows, err := t.db.Query(`
	SELECT 
	parent_name,
	COUNT(*) as total_assignments,
	SUM(CASE WHEN assignment_date >= ? AND assignment_date < ? THEN 1 ELSE 0 END) as last_30_days
	FROM assignments
	WHERE assignment_date < ?
	GROUP BY parent_name
	`, thirtyDaysBeforeUntil, untilStr, untilStr)
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
	ID                    int64
	Parent                string
	Date                  time.Time
	Override              bool
	GoogleCalendarEventID string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// Stats represents statistics for a parent
type Stats struct {
	TotalAssignments int
	Last30Days       int
}
