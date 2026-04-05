package viewhelpers

import "time"

// DisplayAssignment is a presentation-layer DTO for calendar assignments.
// It decouples the UI from internal scheduler types so that templates and
// JSON serialisation never depend on the fairness or scheduler packages.
type DisplayAssignment struct {
	ID             int64
	Date           time.Time
	Parent         string // Display name of the assigned caregiver
	ParentType     string // "ParentA", "ParentB", or "Babysitter"
	CaregiverType  string // "parent" or "babysitter"
	DecisionReason string // e.g. "Total Count", "Alternating", "Override"
}
