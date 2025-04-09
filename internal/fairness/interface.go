package fairness

import "time"

// TrackerInterface defines the operations for tracking fairness
type TrackerInterface interface {
	// RecordAssignment records a new assignment with all details
	RecordAssignment(parent string, date time.Time, override bool, decisionReason DecisionReason) (*Assignment, error)

	// GetLastAssignmentsUntil returns the last n assignments up to a specific date
	GetLastAssignmentsUntil(n int, until time.Time) ([]*Assignment, error)

	// GetParentStatsUntil returns statistics for each parent up to a specific date
	GetParentStatsUntil(until time.Time) (map[string]Stats, error)

	// GetAssignmentByID retrieves an assignment by its ID
	GetAssignmentByID(id int64) (*Assignment, error)

	// GetAssignmentByDate retrieves an assignment for a specific date
	GetAssignmentByDate(date time.Time) (*Assignment, error)

	// UpdateAssignmentGoogleCalendarEventID updates an assignment with Google Calendar event ID
	UpdateAssignmentGoogleCalendarEventID(id int64, googleCalendarEventID string) error

	// GetAssignmentByGoogleCalendarEventID retrieves an assignment by its Google Calendar event ID
	GetAssignmentByGoogleCalendarEventID(eventID string) (*Assignment, error)

	// GetAssignmentsInRange retrieves all assignments in a date range
	GetAssignmentsInRange(start, end time.Time) ([]*Assignment, error)

	// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
	UpdateAssignmentParent(id int64, parent string, override bool) error

	// GetLastAssignmentDate returns the date of the last assignment in the database
	GetLastAssignmentDate() (time.Time, error)
}

// Ensure Tracker implements the TrackerInterface
var _ TrackerInterface = (*Tracker)(nil)
