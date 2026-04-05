package fairness

import "time"

// TrackerInterface defines the operations for tracking fairness
type TrackerInterface interface {
	// RecordAssignment records a new assignment with all details
	RecordAssignment(parent string, date time.Time, override bool, decisionReason DecisionReason) (*Assignment, error)

	// RecordBabysitterAssignment records a named babysitter assignment for a date.
	RecordBabysitterAssignment(name string, date time.Time, override bool) (*Assignment, error)

	// GetLastAssignmentsUntil returns the last n assignments of all caregiver types up to a specific date.
	// Used to detect babysitter nights and gaps that break consecutive-assignment chains.
	// Parent-only entries can be derived from this list by filtering on CaregiverType.
	GetLastAssignmentsUntil(n int, until time.Time) ([]*Assignment, error)

	// GetParentStatsUntil returns statistics for each parent up to a specific date.
	// parentNames ensures that both configured parents appear in the result map
	// even if they have zero parent assignments so far, so that babysitter shift
	// counts are applied to both.
	GetParentStatsUntil(until time.Time, parentNames ...string) (map[string]Stats, error)

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

	// UpdateAssignmentToBabysitter sets an assignment to a named babysitter.
	UpdateAssignmentToBabysitter(id int64, babysitterName string, override bool) error

	UnlockAssignment(id int64) error

	// GetLastAssignmentDate returns the date of the last assignment in the database
	GetLastAssignmentDate() (time.Time, error)

	// GetParentMonthlyStatsForLastNMonths fetches and aggregates assignment counts per parent per month for the last n months,
	// relative to the given referenceTime.
	GetParentMonthlyStatsForLastNMonths(referenceTime time.Time, nMonths int) ([]MonthlyStatRow, error)

	// GetBabysitterMonthlyStatsForLastNMonths fetches babysitter assignment counts per babysitter per month.
	GetBabysitterMonthlyStatsForLastNMonths(referenceTime time.Time, nMonths int) ([]MonthlyStatRow, error)

	// SaveAssignmentDetails stores the fairness algorithm calculation details for an assignment
	SaveAssignmentDetails(assignmentID int64, calculationDate time.Time, parentAName string, statsA Stats, parentBName string, statsB Stats) error

	// GetAssignmentDetails retrieves the fairness algorithm calculation details for an assignment
	GetAssignmentDetails(assignmentID int64) (*AssignmentDetails, error)

	// SwapAssignments atomically swaps two assignments' parents within a single
	// database transaction. Both assignments are upserted with the new parent
	// and the given decision reason. Returns the updated assignment records.
	SwapAssignments(parentA string, dateA time.Time, parentB string, dateB time.Time, reason DecisionReason) (updatedA *Assignment, updatedB *Assignment, err error)
}

// Ensure Tracker implements the TrackerInterface
var _ TrackerInterface = (*Tracker)(nil)
