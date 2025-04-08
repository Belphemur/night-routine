package fairness

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/rs/zerolog"
)

const (
	// dateFormat is the format used for dates in the database
	dateFormat = "2006-01-02"
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

// RecordAssignment records a new assignment with all details
func (t *Tracker) RecordAssignment(parent string, date time.Time, override bool, googleCalendarEventID string, decisionReason string) (*Assignment, error) {
	recordLogger := t.logger.With().
		Str("date", date.Format(dateFormat)).
		Str("parent", parent).
		Bool("override", override).
		Str("event_id", googleCalendarEventID).
		Str("decision_reason", decisionReason).
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

		// Only update if the parent has changed and it wasn't an override
		if existingAssignment.Parent != parent && !existingAssignment.Override {
			recordLogger.Debug().Str("old_parent", existingAssignment.Parent).Str("new_parent", parent).Msg("Updating existing assignment parent (non-override)")
			_, err := t.db.Exec(`
			UPDATE assignments
			SET parent_name = ?, override = ?, google_calendar_event_id = ?, decision_reason = ?
			WHERE id = ?
			`, parent, override, googleCalendarEventID, decisionReason, existingAssignment.ID)

			if err != nil {
				recordLogger.Debug().Err(err).Msg("Failed to update assignment")
				return nil, fmt.Errorf("failed to update assignment: %w", err)
			}
			recordLogger.Debug().Msg("Assignment updated successfully")
			// Refresh the assignment
			return t.GetAssignmentByID(existingAssignment.ID)
		} else if existingAssignment.Override {
			recordLogger.Debug().Str("existing_parent", existingAssignment.Parent).Msg("Existing assignment is an override, not changing parent")
			return existingAssignment, nil
		} else {
			recordLogger.Debug().Str("parent", existingAssignment.Parent).Msg("Parent has not changed, returning existing assignment")
			return existingAssignment, nil
		}
	}

	// No existing assignment, create a new one
	recordLogger.Debug().Msg("No existing assignment found, creating new one")
	result, err := t.db.Exec(`
	INSERT INTO assignments (parent_name, assignment_date, override, google_calendar_event_id, decision_reason)
	VALUES (?, ?, ?, ?, ?)
	`, parent, date.Format(dateFormat), override, googleCalendarEventID, decisionReason)

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

// No deprecated methods here - we've consolidated to a single RecordAssignment method

// scanAssignment scans a row into an Assignment struct
func (t *Tracker) scanAssignment(scanner interface {
	Scan(dest ...interface{}) error
}) (*Assignment, error) {
	var a Assignment
	var dateStr string
	var createdAt, updatedAt time.Time
	var googleEventID sql.NullString
	var decisionReason sql.NullString // New field

	err := scanner.Scan(
		&a.ID,
		&a.Parent,
		&dateStr,
		&a.Override,
		&googleEventID,
		&decisionReason, // New field
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	if googleEventID.Valid {
		a.GoogleCalendarEventID = googleEventID.String
	}

	if decisionReason.Valid {
		a.DecisionReason = decisionReason.String
	}

	date, err := time.Parse(dateFormat, dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}
	a.Date = date

	a.CreatedAt = createdAt
	a.UpdatedAt = updatedAt

	return &a, nil
}

// GetAssignmentByID retrieves an assignment by its ID
func (t *Tracker) GetAssignmentByID(id int64) (*Assignment, error) {
	getLogger := t.logger.With().Int64("assignment_id", id).Logger()
	getLogger.Debug().Msg("Getting assignment by ID")
	row := t.db.QueryRow(`
		SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
		FROM assignments
		WHERE id = ?
	`, id)

	a, err := t.scanAssignment(row)
	if err != nil {
		getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	getLogger.Debug().Msg("Assignment retrieved successfully")
	return a, nil
}

// UpdateAssignmentGoogleCalendarEventID updates an assignment with its Google Calendar event ID
func (t *Tracker) UpdateAssignmentGoogleCalendarEventID(id int64, googleCalendarEventID string) error {
	updateLogger := t.logger.With().Int64("assignment_id", id).Str("event_id", googleCalendarEventID).Logger()
	updateLogger.Debug().Msg("Updating assignment Google Calendar Event ID")
	_, err := t.db.Exec(`
	UPDATE assignments
	SET google_calendar_event_id = ?
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
	SET parent_name = ?, override = ?
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
	untilStr := until.Format(dateFormat)
	rows, err := t.db.Query(`
SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
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
		a, err := t.scanAssignment(rows)
		if err != nil {
			getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		assignments = append(assignments, a)
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
	getLogger := t.logger.With().Str("date", date.Format(dateFormat)).Logger()
	getLogger.Debug().Msg("Getting assignment by date")
	dateStr := date.Format(dateFormat)
	row := t.db.QueryRow(`
		SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
		FROM assignments
		WHERE assignment_date = ?
		ORDER BY id DESC
		LIMIT 1
	`, dateStr)

	a, err := t.scanAssignment(row)
	if err != nil {
		getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	if a != nil {
		getLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment retrieved successfully")
	} else {
		getLogger.Debug().Msg("No assignment found for this date")
	}
	return a, nil
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
		SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
		FROM assignments
		WHERE google_calendar_event_id = ?
	`, eventID)

	a, err := t.scanAssignment(row)
	if err != nil {
		getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}

	if a != nil {
		if a.GoogleCalendarEventID == "" {
			getLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment found by event ID, but event ID column was NULL")
		}
		getLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment retrieved successfully")
	} else {
		getLogger.Debug().Msg("No assignment found for this event ID")
	}
	return a, nil
}

// GetAssignmentsInRange retrieves all assignments in a date range
func (t *Tracker) GetAssignmentsInRange(start, end time.Time) ([]*Assignment, error) {
	getLogger := t.logger.With().Time("start_date", start).Time("end_date", end).Logger()
	getLogger.Debug().Msg("Getting assignments in date range")
	startStr := start.Format(dateFormat)
	endStr := end.Format(dateFormat)

	rows, err := t.db.Query(`
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
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
		a, err := t.scanAssignment(rows)
		if err != nil {
			getLogger.Debug().Err(err).Msg("Failed to scan assignment row")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		assignments = append(assignments, a)
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
	untilStr := until.Format(dateFormat)
	thirtyDaysBeforeUntil := until.AddDate(0, 0, -30).Format(dateFormat)

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

	date, err := time.Parse(dateFormat, dateStr)
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
	DecisionReason        string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// Stats represents statistics for a parent
type Stats struct {
	TotalAssignments int
	Last30Days       int
}
