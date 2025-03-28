package fairness

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/rs/zerolog"
)

// Tracker maintains the state of night routine assignments
type Tracker struct {
	db     *sql.DB
	logger zerolog.Logger
}

// New creates a new Tracker instance
func New(db *database.DB) (*Tracker, error) {
	return &Tracker{
		db:     db.Conn(),
		logger: logging.GetLogger("fairness-tracker"),
	}, nil
}

// RecordAssignment records a new assignment in the state database
// This is a simplified version, prefer RecordAssignmentWithDetails for full control
func (t *Tracker) RecordAssignment(parent string, date time.Time) (*Assignment, error) {
	t.logger.Debug().Msg("RecordAssignment called (simplified version)")
	// Check if there's already an assignment for this date
	existingAssignment, err := t.GetAssignmentByDate(date)
	if err != nil {
		// Error already logged in GetAssignmentByDate
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}

	// If there's already an assignment, update it only if parent changes
	if existingAssignment != nil {
		updateLogger := t.logger.With().Int64("assignment_id", existingAssignment.ID).Str("date", date.Format("2006-01-02")).Logger()
		// Only update if the parent has changed and it wasn't an override
		if existingAssignment.Parent != parent && !existingAssignment.Override {
			updateLogger.Debug().Str("old_parent", existingAssignment.Parent).Str("new_parent", parent).Msg("Updating existing assignment parent (non-override)")
			_, err := t.db.Exec(`
			UPDATE assignments
			SET parent_name = ?, updated_at = CURRENT_TIMESTAMP, override = ?
			WHERE id = ?
			`, parent, false, existingAssignment.ID)

			if err != nil {
				updateLogger.Debug().Err(err).Msg("Failed to update assignment")
				return nil, fmt.Errorf("failed to update assignment: %w", err)
			}
			updateLogger.Debug().Msg("Assignment updated successfully")
			// Refresh the assignment
			return t.GetAssignmentByID(existingAssignment.ID)
		} else if existingAssignment.Override {
			updateLogger.Debug().Str("existing_parent", existingAssignment.Parent).Msg("Existing assignment is an override, not changing parent")
			return existingAssignment, nil
		} else {
			updateLogger.Debug().Str("parent", existingAssignment.Parent).Msg("Parent has not changed, returning existing assignment")
			return existingAssignment, nil
		}
	}

	// No existing assignment, create a new one with default details
	t.logger.Debug().Str("date", date.Format("2006-01-02")).Str("parent", parent).Msg("No existing assignment found, creating new one")
	return t.RecordAssignmentWithDetails(parent, date, false, "")
}

// RecordAssignmentWithOverride records a new assignment with override flag
// Deprecated: Use RecordAssignmentWithDetails instead
func (t *Tracker) RecordAssignmentWithOverride(parent string, date time.Time, override bool) (*Assignment, error) {
	t.logger.Debug().Msg("RecordAssignmentWithOverride called (deprecated)")
	return t.RecordAssignmentWithDetails(parent, date, override, "")
}

// RecordAssignmentWithDetails records an assignment with all available details, handling insert or update.
func (t *Tracker) RecordAssignmentWithDetails(parent string, date time.Time, override bool, googleCalendarEventID string) (*Assignment, error) {
	recordLogger := t.logger.With().
		Str("date", date.Format("2006-01-02")).
		Str("parent", parent).
		Bool("override", override).
		Str("event_id", googleCalendarEventID).
		Logger()
	recordLogger.Debug().Msg("Recording assignment details")

	// Check if there's already an assignment for this date
	recordLogger.Debug().Msg("Checking for existing assignment on this date")
	existingAssignment, err := t.GetAssignmentByDate(date)
	if err != nil {
		// Error already logged in GetAssignmentByDate
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}

	// If there's already an assignment, update it
	if existingAssignment != nil {
		recordLogger = recordLogger.With().Int64("assignment_id", existingAssignment.ID).Logger()
		recordLogger.Debug().Msg("Existing assignment found, updating details")
		// Update the assignment
		_, err := t.db.Exec(`
		UPDATE assignments
		SET parent_name = ?, override = ?, google_calendar_event_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		`, parent, override, googleCalendarEventID, existingAssignment.ID)

		if err != nil {
			recordLogger.Debug().Err(err).Msg("Failed to update assignment")
			return nil, fmt.Errorf("failed to update assignment: %w", err)
		}
		recordLogger.Debug().Msg("Assignment updated successfully")
		// Refresh the assignment
		return t.GetAssignmentByID(existingAssignment.ID)
	}

	// No existing assignment, create a new one
	recordLogger.Debug().Msg("No existing assignment found, creating new one")
	result, err := t.db.Exec(`
	INSERT INTO assignments (parent_name, assignment_date, override, google_calendar_event_id)
	VALUES (?, ?, ?, ?)
	`, parent, date.Format("2006-01-02"), override, googleCalendarEventID)

	if err != nil {
		recordLogger.Debug().Err(err).Msg("Failed to insert new assignment")
		return nil, fmt.Errorf("failed to record assignment: %w", err)
	}

	// Get the last inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		recordLogger.Debug().Err(err).Msg("Failed to get last insert ID after insert")
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}
	recordLogger = recordLogger.With().Int64("assignment_id", id).Logger()
	recordLogger.Debug().Msg("New assignment inserted successfully")

	// Get the full assignment record
	assignment, err := t.GetAssignmentByID(id)
	if err != nil {
		// Error logged in GetAssignmentByID
		return nil, fmt.Errorf("failed to get assignment after insert: %w", err)
	}

	return assignment, nil
}

// GetAssignmentByID retrieves an assignment by its ID
func (t *Tracker) GetAssignmentByID(id int64) (*Assignment, error) {
	getLogger := t.logger.With().Int64("assignment_id", id).Logger()
	getLogger.Debug().Msg("Getting assignment by ID")
	row := t.db.QueryRow(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
	FROM assignments
	WHERE id = ?
	`, id)

	var a Assignment
	var dateStr string
	var createdAtStr, updatedAtStr string
	var googleEventID sql.NullString

	err := row.Scan(
		&a.ID,
		&a.Parent,
		&dateStr,
		&a.Override,
		&googleEventID,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			getLogger.Debug().Msg("Assignment not found")
			return nil, nil
		}
		getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	if googleEventID.Valid {
		a.GoogleCalendarEventID = googleEventID.String
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		getLogger.Debug().Err(err).Str("date_string", dateStr).Msg("Failed to parse assignment date")
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}
	a.Date = date

	createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		a.CreatedAt = createdAt
	} else {
		getLogger.Debug().Err(err).Str("timestamp_string", createdAtStr).Msg("Failed to parse created_at timestamp")
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err == nil {
		a.UpdatedAt = updatedAt
	} else {
		getLogger.Debug().Err(err).Str("timestamp_string", updatedAtStr).Msg("Failed to parse updated_at timestamp")
	}

	getLogger.Debug().Msg("Assignment retrieved successfully")
	return &a, nil
}

// UpdateAssignmentGoogleCalendarEventID updates an assignment with its Google Calendar event ID
func (t *Tracker) UpdateAssignmentGoogleCalendarEventID(id int64, googleCalendarEventID string) error {
	updateLogger := t.logger.With().Int64("assignment_id", id).Str("event_id", googleCalendarEventID).Logger()
	updateLogger.Debug().Msg("Updating assignment Google Calendar Event ID")
	_, err := t.db.Exec(`
	UPDATE assignments
	SET google_calendar_event_id = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`, googleCalendarEventID, id)

	if err != nil {
		updateLogger.Debug().Err(err).Msg("Failed to execute update query")
		return fmt.Errorf("failed to update assignment with Google Calendar event ID: %w", err)
	}

	updateLogger.Debug().Msg("Assignment event ID updated in DB")
	return nil
}

// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
func (t *Tracker) UpdateAssignmentParent(id int64, parent string, override bool) error {
	updateLogger := t.logger.With().Int64("assignment_id", id).Str("new_parent", parent).Bool("override", override).Logger()
	updateLogger.Debug().Msg("Updating assignment parent and override status")
	_, err := t.db.Exec(`
	UPDATE assignments
	SET parent_name = ?, override = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`, parent, override, id)

	if err != nil {
		updateLogger.Debug().Err(err).Msg("Failed to execute update query")
		return fmt.Errorf("failed to update assignment parent: %w", err)
	}

	updateLogger.Debug().Msg("Assignment parent/override updated in DB")
	return nil
}

// GetLastAssignmentsUntil returns the last n assignments up to a specific date
func (t *Tracker) GetLastAssignmentsUntil(n int, until time.Time) ([]*Assignment, error) {
	getLogger := t.logger.With().Int("limit", n).Time("until_date", until).Logger()
	getLogger.Debug().Msg("Getting last assignments until date")
	untilStr := until.Format("2006-01-02")
	rows, err := t.db.Query(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
	FROM assignments
	WHERE assignment_date < ?
	ORDER BY assignment_date DESC
	LIMIT ?
	`, untilStr, n)
	if err != nil {
		getLogger.Debug().Err(err).Msg("Failed to query last assignments")
		return nil, fmt.Errorf("failed to query assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		var a Assignment
		var dateStr string
		var createdAtStr, updatedAtStr string
		var googleEventID sql.NullString

		if err := rows.Scan(
			&a.ID,
			&a.Parent,
			&dateStr,
			&a.Override,
			&googleEventID,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if googleEventID.Valid {
			a.GoogleCalendarEventID = googleEventID.String
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			getLogger.Debug().Err(err).Str("date_string", dateStr).Int64("assignment_id", a.ID).Msg("Failed to parse assignment date")
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}
		a.Date = date

		createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err == nil {
			a.CreatedAt = createdAt
		} else {
			getLogger.Debug().Err(err).Str("timestamp_string", createdAtStr).Int64("assignment_id", a.ID).Msg("Failed to parse created_at timestamp")
		}

		updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err == nil {
			a.UpdatedAt = updatedAt
		} else {
			getLogger.Debug().Err(err).Str("timestamp_string", updatedAtStr).Int64("assignment_id", a.ID).Msg("Failed to parse updated_at timestamp")
		}

		assignments = append(assignments, &a)
	}
	if err := rows.Err(); err != nil {
		getLogger.Debug().Err(err).Msg("Error iterating assignment rows")
		return nil, fmt.Errorf("failed during row iteration: %w", err)
	}

	getLogger.Debug().Int("count", len(assignments)).Msg("Fetched last assignments successfully")
	return assignments, nil
}

// GetAssignmentByDate retrieves an assignment for a specific date
func (t *Tracker) GetAssignmentByDate(date time.Time) (*Assignment, error) {
	getLogger := t.logger.With().Str("date", date.Format("2006-01-02")).Logger()
	getLogger.Debug().Msg("Getting assignment by date")
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
	var googleEventID sql.NullString

	err := row.Scan(
		&a.ID,
		&a.Parent,
		&rowDateStr,
		&a.Override,
		&googleEventID,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			getLogger.Debug().Msg("No assignment found for this date")
			return nil, nil // No assignment found, which is ok
		}
		getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	if googleEventID.Valid {
		a.GoogleCalendarEventID = googleEventID.String
	}

	assignmentDate, err := time.Parse("2006-01-02", rowDateStr)
	if err != nil {
		getLogger.Debug().Err(err).Str("date_string", rowDateStr).Int64("assignment_id", a.ID).Msg("Failed to parse assignment date")
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}
	a.Date = assignmentDate

	createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		a.CreatedAt = createdAt
	} else {
		getLogger.Debug().Err(err).Str("timestamp_string", createdAtStr).Int64("assignment_id", a.ID).Msg("Failed to parse created_at timestamp")
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err == nil {
		a.UpdatedAt = updatedAt
	} else {
		getLogger.Debug().Err(err).Str("timestamp_string", updatedAtStr).Int64("assignment_id", a.ID).Msg("Failed to parse updated_at timestamp")
	}

	getLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment retrieved successfully")
	return &a, nil
}

// GetAssignmentByGoogleCalendarEventID retrieves an assignment by its Google Calendar event ID
func (t *Tracker) GetAssignmentByGoogleCalendarEventID(eventID string) (*Assignment, error) {
	getLogger := t.logger.With().Str("event_id", eventID).Logger()
	getLogger.Debug().Msg("Getting assignment by Google Calendar Event ID")
	if eventID == "" {
		getLogger.Debug().Msg("Empty event ID provided")
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
	var googleEventID sql.NullString

	err := row.Scan(
		&a.ID,
		&a.Parent,
		&dateStr,
		&a.Override,
		&googleEventID,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			getLogger.Debug().Msg("No assignment found for this event ID")
			return nil, nil // No assignment found, which is ok
		}
		getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	if googleEventID.Valid {
		a.GoogleCalendarEventID = googleEventID.String
	} else {
		getLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment found by event ID, but event ID column was NULL")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		getLogger.Debug().Err(err).Str("date_string", dateStr).Int64("assignment_id", a.ID).Msg("Failed to parse assignment date")
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}
	a.Date = date

	createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		a.CreatedAt = createdAt
	} else {
		getLogger.Debug().Err(err).Str("timestamp_string", createdAtStr).Int64("assignment_id", a.ID).Msg("Failed to parse created_at timestamp")
	}

	updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
	if err == nil {
		a.UpdatedAt = updatedAt
	} else {
		getLogger.Debug().Err(err).Str("timestamp_string", updatedAtStr).Int64("assignment_id", a.ID).Msg("Failed to parse updated_at timestamp")
	}

	getLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment retrieved successfully")
	return &a, nil
}

// GetAssignmentsInRange retrieves all assignments in a date range
func (t *Tracker) GetAssignmentsInRange(start, end time.Time) ([]*Assignment, error) {
	getLogger := t.logger.With().Time("start_date", start).Time("end_date", end).Logger()
	getLogger.Debug().Msg("Getting assignments in date range")
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	rows, err := t.db.Query(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
	FROM assignments
	WHERE assignment_date >= ? AND assignment_date <= ?
	ORDER BY assignment_date ASC
	`, startStr, endStr)

	if err != nil {
		getLogger.Debug().Err(err).Msg("Failed to query assignments in range")
		return nil, fmt.Errorf("failed to query assignments in range: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		var a Assignment
		var dateStr string
		var createdAtStr, updatedAtStr string
		var googleEventID sql.NullString

		if err := rows.Scan(
			&a.ID,
			&a.Parent,
			&dateStr,
			&a.Override,
			&googleEventID,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if googleEventID.Valid {
			a.GoogleCalendarEventID = googleEventID.String
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			getLogger.Debug().Err(err).Str("date_string", dateStr).Int64("assignment_id", a.ID).Msg("Failed to parse assignment date")
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}
		a.Date = date

		createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err == nil {
			a.CreatedAt = createdAt
		} else {
			getLogger.Debug().Err(err).Str("timestamp_string", createdAtStr).Int64("assignment_id", a.ID).Msg("Failed to parse created_at timestamp")
		}

		updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err == nil {
			a.UpdatedAt = updatedAt
		} else {
			getLogger.Debug().Err(err).Str("timestamp_string", updatedAtStr).Int64("assignment_id", a.ID).Msg("Failed to parse updated_at timestamp")
		}

		assignments = append(assignments, &a)
	}
	if err := rows.Err(); err != nil {
		getLogger.Debug().Err(err).Msg("Error iterating assignment rows")
		return nil, fmt.Errorf("failed during row iteration: %w", err)
	}

	getLogger.Debug().Int("count", len(assignments)).Msg("Fetched assignments in range successfully")
	return assignments, nil
}

// GetParentStatsUntil returns statistics for each parent up to a specific date
func (t *Tracker) GetParentStatsUntil(until time.Time) (map[string]Stats, error) {
	getLogger := t.logger.With().Time("until_date", until).Logger()
	getLogger.Debug().Msg("Getting parent stats until date")
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
		getLogger.Debug().Err(err).Msg("Failed to query parent stats")
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]Stats)
	for rows.Next() {
		var parentName string
		var s Stats
		if err := rows.Scan(&parentName, &s.TotalAssignments, &s.Last30Days); err != nil {
			getLogger.Debug().Err(err).Msg("Failed to scan parent stats row")
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats[parentName] = s
	}
	if err := rows.Err(); err != nil {
		getLogger.Debug().Err(err).Msg("Error iterating parent stats rows")
		return nil, fmt.Errorf("failed during row iteration: %w", err)
	}

	getLogger.Debug().Interface("stats", stats).Msg("Fetched parent stats successfully")
	return stats, nil
}

// GetLastAssignmentDate returns the date of the last assignment in the database
func (t *Tracker) GetLastAssignmentDate() (time.Time, error) {
	getLogger := t.logger
	getLogger.Debug().Msg("Getting last assignment date")
	row := t.db.QueryRow(`
	SELECT assignment_date
	FROM assignments
	ORDER BY assignment_date DESC
	LIMIT 1
	`)

	var dateStr string
	err := row.Scan(&dateStr)
	if err != nil {
		if err == sql.ErrNoRows {
			getLogger.Debug().Msg("No assignments found in database")
			return time.Time{}, nil
		}
		getLogger.Debug().Err(err).Msg("Failed to scan last assignment date")
		return time.Time{}, fmt.Errorf("failed to get last assignment date: %w", err)
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		getLogger.Debug().Err(err).Str("date_string", dateStr).Msg("Failed to parse last assignment date")
		return time.Time{}, fmt.Errorf("failed to parse date: %w", err)
	}

	getLogger.Debug().Time("last_date", date).Msg("Last assignment date retrieved successfully")
	return date, nil
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
