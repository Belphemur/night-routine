package fairness

import "time"

// TrackerInterface defines the operations for tracking fairness
type TrackerInterface interface {
	// RecordAssignment records a new assignment
	RecordAssignment(parent string, date time.Time) (*Assignment, error)

	// RecordAssignmentWithOverride records assignment with override flag
	RecordAssignmentWithOverride(parent string, date time.Time, override bool) (*Assignment, error)

	// RecordAssignmentWithDetails records assignment with all available details
	RecordAssignmentWithDetails(parent string, date time.Time, override bool, googleCalendarEventID string) (*Assignment, error)

	// GetLastAssignments returns the last n assignments
	GetLastAssignments(n int) ([]*Assignment, error)

	// GetParentStats returns statistics for each parent
	GetParentStats() (map[string]Stats, error)

	// GetAssignmentByID retrieves an assignment by its ID
	GetAssignmentByID(id int64) (*Assignment, error)

	// GetAssignmentByDate retrieves an assignment for a specific date
	GetAssignmentByDate(date time.Time) (*Assignment, error)

	// UpdateAssignmentGoogleCalendarEventID updates an assignment with Google Calendar event ID
	UpdateAssignmentGoogleCalendarEventID(id int64, googleCalendarEventID string) error
}

// Ensure both Tracker and MockTracker implement the TrackerInterface
var _ TrackerInterface = (*Tracker)(nil)
var _ TrackerInterface = (*MockTracker)(nil)
