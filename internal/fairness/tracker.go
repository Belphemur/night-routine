package fairness

import (
	"context"
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
	// defaultQueryTimeout is the default timeout for database queries.
	defaultQueryTimeout = 30 * time.Second
)

// Tracker maintains the state of night routine assignments
type Tracker struct {
	db     *database.DB
	logger zerolog.Logger
}

// New creates a new Tracker instance
func New(db *database.DB) (*Tracker, error) {
	return &Tracker{
		db:     db,
		logger: logging.GetLogger("fairness-tracker"),
	}, nil
}

// RecordAssignment records a new assignment with all details
func (t *Tracker) RecordAssignment(parent string, date time.Time, override bool, decisionReason DecisionReason) (*Assignment, error) {
	recordLogger := t.logger.With().
		Str("date", date.Format(dateFormat)).
		Str("parent", parent).
		Bool("override", override).
		Str("decision_reason", decisionReason.String()).
		Logger()
	recordLogger.Debug().Msg("Recording assignment details")

	// Use proper UPSERT syntax with ON CONFLICT clause
	// This works because we have a unique index on assignment_date
	recordLogger.Debug().Msg("Using UPSERT with ON CONFLICT to create or update assignment")

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := t.db.Conn().ExecContext(ctx, `
	INSERT INTO assignments (parent_name, assignment_date, override, decision_reason)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(assignment_date) DO UPDATE SET 
		parent_name = excluded.parent_name,
		override = excluded.override,
		decision_reason = excluded.decision_reason
		`, parent, date.Format(dateFormat), override, decisionReason.String())

	if err != nil {
		if err == context.DeadlineExceeded {
			recordLogger.Error().Err(err).Msg("Database upsert for assignment timed out")
			return nil, fmt.Errorf("database upsert timed out: %w", err)
		}
		recordLogger.Error().Err(err).Msg("Failed to upsert assignment")
		return nil, fmt.Errorf("failed to record assignment: %w", err)
	}

	// Get the full assignment record
	assignment, err := t.GetAssignmentByDate(date)
	if err != nil {
		recordLogger.Debug().Err(err).Msg("Failed to get the upserted assignment")
		return nil, fmt.Errorf("failed to get assignment by date: %w", err)
	}
	recordLogger.Debug().Int64("assignment_id", assignment.ID).Msg("Assignment upserted successfully")
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
	var decisionReason sql.NullString

	err := scanner.Scan(
		&a.ID,
		&a.Parent,
		&dateStr,
		&a.Override,
		&googleEventID,
		&decisionReason,
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
		a.DecisionReason = DecisionReason(decisionReason.String)
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
	queryLogger := t.logger.With().Int64("assignment_id", id).Logger()
	queryLogger.Debug().Msg("Getting assignment by ID")

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	row := t.db.Conn().QueryRowContext(ctx, `
		SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
		FROM assignments
		WHERE id = ?
	`, id)

	a, err := t.scanAssignment(row)
	if err != nil {
		if err == context.DeadlineExceeded { // This check might be redundant if QueryRowContext handles it before scan
			queryLogger.Error().Err(err).Msg("Database query for assignment by ID timed out during scan")
			return nil, fmt.Errorf("database query timed out during scan: %w", err)
		}
		// sql.ErrNoRows is handled by scanAssignment
		queryLogger.Error().Err(err).Msg("Failed to scan assignment row")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}
	// QueryRowContext is expected to propagate context errors, so no additional check is needed here.

	queryLogger.Debug().Msg("Assignment retrieved successfully")
	return a, nil
}

// UpdateAssignmentGoogleCalendarEventID updates an assignment with its Google Calendar event ID
func (t *Tracker) UpdateAssignmentGoogleCalendarEventID(id int64, googleCalendarEventID string) error {
	updateLogger := t.logger.With().
		Int64("assignment_id", id).
		Str("google_calendar_event_id", googleCalendarEventID).
		Logger()
	updateLogger.Debug().Msg("Updating assignment Google Calendar Event ID")

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	_, err := t.db.Conn().ExecContext(ctx, `
	UPDATE assignments
	SET google_calendar_event_id = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`, googleCalendarEventID, id)

	if err != nil {
		if err == context.DeadlineExceeded {
			updateLogger.Error().Err(err).Msg("Database update timed out")
			return fmt.Errorf("database update timed out: %w", err)
		}
		updateLogger.Error().Err(err).Msg("Failed to execute update query")
		return fmt.Errorf("failed to update assignment: %w", err)
	}

	updateLogger.Debug().Msg("Assignment event ID updated in DB")
	return nil
}

// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
func (t *Tracker) UpdateAssignmentParent(id int64, parent string, override bool) error {
	updateLogger := t.logger.With().
		Int64("assignment_id", id).
		Str("new_parent", parent).
		Bool("override", override).
		Logger()
	updateLogger.Debug().Msg("Updating assignment parent")

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	query := `UPDATE assignments SET parent_name = ?, override = ?, updated_at = CURRENT_TIMESTAMP`
	args := []interface{}{parent, override}

	if override {
		// When overriding, also update the decision reason
		query += ", decision_reason = ?"
		args = append(args, DecisionReasonOverride)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	// Execute the query
	_, err := t.db.Conn().ExecContext(ctx, query, args...)
	if err != nil {
		if err == context.DeadlineExceeded {
			updateLogger.Error().Err(err).Msg("Database update timed out")
			return fmt.Errorf("database update timed out: %w", err)
		}
		updateLogger.Error().Err(err).Msg("Failed to execute update query")
		return fmt.Errorf("failed to update assignment: %w", err)
	}

	updateLogger.Debug().Msg("Assignment parent/override updated in DB")
	return nil
}

// UnlockAssignment removes the override flag from an assignment
func (t *Tracker) UnlockAssignment(id int64) error {
	updateLogger := t.logger.With().Int64("assignment_id", id).Logger()
	updateLogger.Debug().Msg("Unlocking assignment (removing override)")

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	return t.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Set override to false.
		result, err := tx.ExecContext(ctx, `
		UPDATE assignments
		SET override = 0, decision_reason = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		`, id)

		if err != nil {
			if err == context.DeadlineExceeded {
				updateLogger.Error().Err(err).Msg("Database update for unlocking assignment timed out")
				return fmt.Errorf("database update timed out: %w", err)
			}
			updateLogger.Error().Err(err).Msg("Failed to execute unlock query")
			return fmt.Errorf("failed to unlock assignment: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			updateLogger.Error().Err(err).Msg("Failed to get rows affected")
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			updateLogger.Warn().Msg("No assignment found to unlock")
			return fmt.Errorf("assignment not found")
		}

		return nil
	})
}

// GetLastAssignmentsUntil returns the last n assignments up to a specific date
func (t *Tracker) GetLastAssignmentsUntil(n int, until time.Time) ([]*Assignment, error) {
	queryLogger := t.logger.With().
		Int("limit", n).
		Str("until_date", until.Format(dateFormat)).
		Logger()
	queryLogger.Debug().Msg("Fetching last assignments")
	untilStr := until.Format(dateFormat)

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	rows, err := t.db.Conn().QueryContext(ctx, `
SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
FROM assignments
WHERE assignment_date < ?
ORDER BY assignment_date DESC
LIMIT ?
`, untilStr, n)
	if err != nil {
		if err == context.DeadlineExceeded {
			queryLogger.Error().Err(err).Msg("Database query for last assignments timed out")
			return nil, fmt.Errorf("database query timed out: %w", err)
		}
		queryLogger.Error().Err(err).Msg("Failed to query last assignments")
		return nil, fmt.Errorf("failed to query assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		a, err := t.scanAssignment(rows)
		if err != nil {
			queryLogger.Debug().Err(err).Msg("Failed to scan assignment row")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		queryLogger.Debug().Err(err).Msg("Error iterating assignment rows")
		return nil, fmt.Errorf("failed during row iteration: %w", err)
	}

	queryLogger.Debug().Int("count", len(assignments)).Msg("Fetched last assignments successfully")
	return assignments, nil
}

// GetAssignmentByDate retrieves an assignment for a specific date
func (t *Tracker) GetAssignmentByDate(date time.Time) (*Assignment, error) {
	queryLogger := t.logger.With().Str("date", date.Format(dateFormat)).Logger()
	queryLogger.Debug().Msg("Getting assignment by date")
	dateStr := date.Format(dateFormat)

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	row := t.db.Conn().QueryRowContext(ctx, `
		SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
		FROM assignments
		WHERE assignment_date = ?
		ORDER BY id DESC
		LIMIT 1
	`, dateStr)

	a, err := t.scanAssignment(row)
	if err != nil {
		// sql.ErrNoRows is handled by scanAssignment
		queryLogger.Error().Err(err).Msg("Failed to scan assignment row for GetAssignmentByDate")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}
	// Removed redundant context error check. Error handling is consolidated in the scanAssignment block.

	if a != nil {
		queryLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment retrieved successfully")
	} else {
		queryLogger.Debug().Msg("No assignment found for this date")
	}
	return a, nil
}

// GetAssignmentByGoogleCalendarEventID retrieves an assignment by its Google Calendar event ID
func (t *Tracker) GetAssignmentByGoogleCalendarEventID(eventID string) (*Assignment, error) {
	queryLogger := t.logger.With().Str("event_id", eventID).Logger()
	queryLogger.Debug().Msg("Getting assignment by Google Calendar Event ID")
	if eventID == "" {
		queryLogger.Debug().Msg("Empty event ID provided")
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	row := t.db.Conn().QueryRowContext(ctx, `
		SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
		FROM assignments
		WHERE google_calendar_event_id = ?
	`, eventID)

	a, err := t.scanAssignment(row)
	if err != nil {
		// sql.ErrNoRows is handled by scanAssignment
		queryLogger.Error().Err(err).Msg("Failed to scan assignment row for GetAssignmentByGoogleCalendarEventID")
		return nil, fmt.Errorf("failed to scan assignment: %w", err)
	}
	// Check for context error after QueryRowContext itself
	if err := ctx.Err(); err == context.DeadlineExceeded {
		queryLogger.Error().Err(err).Msg("Database query for assignment by Google Calendar Event ID timed out")
		return nil, fmt.Errorf("database query timed out: %w", err)
	}

	if a != nil {
		if a.GoogleCalendarEventID == "" {
			queryLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment found by event ID, but event ID column was NULL")
		}
		queryLogger.Debug().Int64("assignment_id", a.ID).Msg("Assignment retrieved successfully")
	} else {
		queryLogger.Debug().Msg("No assignment found for this event ID")
	}
	return a, nil
}

// GetAssignmentsInRange retrieves all assignments in a date range
func (t *Tracker) GetAssignmentsInRange(start, end time.Time) ([]*Assignment, error) {
	queryLogger := t.logger.With().
		Str("start_date", start.Format(dateFormat)).
		Str("end_date", end.Format(dateFormat)).
		Logger()
	queryLogger.Debug().Msg("Fetching assignments in range")
	startStr := start.Format(dateFormat)
	endStr := end.Format(dateFormat)

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	rows, err := t.db.Conn().QueryContext(ctx, `
	SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
	FROM assignments
	WHERE assignment_date >= ? AND assignment_date <= ?
	ORDER BY assignment_date ASC
	`, startStr, endStr)

	if err != nil {
		if err == context.DeadlineExceeded {
			queryLogger.Error().Err(err).Msg("Database query for assignments in range timed out")
			return nil, fmt.Errorf("database query timed out: %w", err)
		}
		queryLogger.Error().Err(err).Msg("Failed to query assignments in range")
		return nil, fmt.Errorf("failed to query assignments in range: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		a, err := t.scanAssignment(rows)
		if err != nil {
			queryLogger.Debug().Err(err).Msg("Failed to scan assignment row")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		queryLogger.Debug().Err(err).Msg("Error iterating assignment rows")
		return nil, fmt.Errorf("failed during row iteration: %w", err)
	}

	queryLogger.Debug().Int("count", len(assignments)).Msg("Fetched assignments in range successfully")
	return assignments, nil
}

// GetParentStatsUntil returns statistics for each parent up to a specific date
func (t *Tracker) GetParentStatsUntil(until time.Time) (map[string]Stats, error) {
	queryLogger := t.logger.With().Str("until_date", until.Format(dateFormat)).Logger()
	queryLogger.Debug().Msg("Fetching parent statistics")
	untilStr := until.Format(dateFormat)
	thirtyDaysBeforeUntil := until.AddDate(0, 0, -30).Format(dateFormat)

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	rows, err := t.db.Conn().QueryContext(ctx, `
	SELECT
	parent_name,
	COUNT(*) as total_assignments,
	SUM(CASE WHEN assignment_date >= ? AND assignment_date < ? THEN 1 ELSE 0 END) as last_30_days
	FROM assignments
	WHERE assignment_date < ?
	GROUP BY parent_name
	`, thirtyDaysBeforeUntil, untilStr, untilStr)
	if err != nil {
		if err == context.DeadlineExceeded {
			queryLogger.Error().Err(err).Msg("Database query for parent stats timed out")
			return nil, fmt.Errorf("database query timed out: %w", err)
		}
		queryLogger.Error().Err(err).Msg("Failed to query parent stats")
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]Stats)
	for rows.Next() {
		var parentName string
		var s Stats
		if err := rows.Scan(&parentName, &s.TotalAssignments, &s.Last30Days); err != nil {
			queryLogger.Debug().Err(err).Msg("Failed to scan parent stats row")
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats[parentName] = s
	}
	if err := rows.Err(); err != nil {
		queryLogger.Debug().Err(err).Msg("Error iterating parent stats rows")
		return nil, fmt.Errorf("failed during row iteration: %w", err)
	}

	queryLogger.Debug().Interface("stats", stats).Msg("Fetched parent stats successfully")
	return stats, nil
}

// GetLastAssignmentDate returns the date of the last assignment in the database
func (t *Tracker) GetLastAssignmentDate() (time.Time, error) {
	t.logger.Debug().Msg("Fetching last assignment date")

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	var dateStr string
	err := t.db.Conn().QueryRowContext(ctx, `
	SELECT assignment_date
	FROM assignments
	ORDER BY assignment_date DESC
	LIMIT 1
	`).Scan(&dateStr)
	if err != nil {
		if err == sql.ErrNoRows {
			t.logger.Debug().Msg("No assignments found in database")
			return time.Time{}, nil // Not a timeout error, but legitimate no rows
		}
		// For QueryRowContext, the context error might be returned by Scan if the query itself failed due to timeout
		if err == context.DeadlineExceeded || ctx.Err() == context.DeadlineExceeded {
			t.logger.Error().Err(err).Msg("Database query for last assignment date timed out")
			return time.Time{}, fmt.Errorf("database query timed out: %w", err)
		}
		t.logger.Error().Err(err).Msg("Failed to scan last assignment date")
		return time.Time{}, fmt.Errorf("failed to get last assignment date: %w", err)
	}
	// An additional check for ctx.Err() might be redundant if Scan already propagated it.
	// However, it's safer to check if the context was cancelled for other reasons.
	if err := ctx.Err(); err == context.DeadlineExceeded {
		t.logger.Error().Err(err).Msg("Database query for last assignment date timed out after scan attempt")
		return time.Time{}, fmt.Errorf("database query timed out: %w", err)
	}

	date, err := time.Parse(dateFormat, dateStr)
	if err != nil {
		t.logger.Debug().Err(err).Str("date_string", dateStr).Msg("Failed to parse last assignment date")
		return time.Time{}, fmt.Errorf("failed to parse date: %w", err)
	}

	t.logger.Debug().Time("last_date", date).Msg("Last assignment date retrieved successfully")
	return date, nil
}

// GetParentMonthlyStatsForLastNMonths fetches and aggregates assignment counts per parent per month for the last n months,
// relative to the given referenceTime.
func (t *Tracker) GetParentMonthlyStatsForLastNMonths(referenceTime time.Time, nMonths int) ([]MonthlyStatRow, error) {
	queryLogger := t.logger.With().
		Time("reference_time", referenceTime).
		Int("n_months", nMonths).
		Logger()
	queryLogger.Debug().Msg("Fetching parent monthly stats")

	// Calculate the start date: first day of the month, nMonths ago
	// e.g., if ref is Nov 15, 2023, and n=3, we want Aug, Sep, Oct, Nov (partial)
	// Actually "last n months" usually implies including current month.
	// Let's go back n months from the first of the current month.
	startOfCurrentMonth := time.Date(referenceTime.Year(), referenceTime.Month(), 1, 0, 0, 0, 0, referenceTime.Location())
	startDate := startOfCurrentMonth.AddDate(0, -nMonths+1, 0)

	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	// SQLite query to group by month and parent
	// strftime('%Y-%m', assignment_date) gives YYYY-MM
	query := `
		SELECT 
			strftime('%Y-%m', assignment_date) as month_str,
			parent_name,
			COUNT(*) as count
		FROM assignments
		WHERE assignment_date >= ? AND assignment_date <= ?
		GROUP BY month_str, parent_name
		ORDER BY month_str ASC, parent_name ASC
	`
	// Query up to the provided referenceTime
	rows, err := t.db.Conn().QueryContext(ctx, query, startDate.Format(dateFormat), referenceTime.Format(dateFormat))
	if err != nil {
		if err == context.DeadlineExceeded {
			queryLogger.Error().Err(err).Msg("Database query for parent monthly stats timed out")
			return nil, fmt.Errorf("database query timed out: %w", err)
		}
		queryLogger.Error().Err(err).Msg("Failed to query parent monthly stats")
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer rows.Close()

	var stats []MonthlyStatRow
	for rows.Next() {
		var row MonthlyStatRow
		if err := rows.Scan(&row.MonthYear, &row.ParentName, &row.Count); err != nil {
			queryLogger.Debug().Err(err).Msg("Failed to scan parent monthly stats row")
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats = append(stats, row)
	}
	if err := rows.Err(); err != nil {
		queryLogger.Debug().Err(err).Msg("Error iterating parent monthly stats rows")
		return nil, fmt.Errorf("failed during row iteration: %w", err)
	}

	queryLogger.Debug().Int("count", len(stats)).Msg("Fetched parent monthly stats successfully")
	return stats, nil
}

// Assignment represents a night routine assignment
type Assignment struct {
	ID                    int64
	Parent                string
	Date                  time.Time
	Override              bool
	GoogleCalendarEventID string
	DecisionReason        DecisionReason
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// Stats represents statistics for a parent
type Stats struct {
	TotalAssignments int
	Last30Days       int
}

// MonthlyStatRow holds a raw row from the monthly statistics query.
type MonthlyStatRow struct {
	ParentName string
	MonthYear  string // Format: "YYYY-MM"
	Count      int
}
