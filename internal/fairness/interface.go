package fairness

import "time"

// TrackerInterface defines the interface for tracking night routine assignments
type TrackerInterface interface {
	RecordAssignment(parent string, date time.Time) error
	GetLastAssignments(n int) ([]Assignment, error)
	GetParentStats() (map[string]Stats, error)
}

// Ensure both Tracker and MockTracker implement the TrackerInterface
var _ TrackerInterface = (*Tracker)(nil)
var _ TrackerInterface = (*MockTracker)(nil)
