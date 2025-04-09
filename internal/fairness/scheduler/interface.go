package scheduler

import (
	"time"
)

// SchedulerInterface defines the interface for the night routine scheduler
type SchedulerInterface interface {
	// GenerateSchedule creates a schedule for the specified date range
	GenerateSchedule(start, end time.Time) ([]*Assignment, error)

	// UpdateGoogleCalendarEventID updates the assignment with the Google Calendar event ID
	UpdateGoogleCalendarEventID(assignment *Assignment, eventID string) error

	// GetAssignmentByGoogleCalendarEventID finds an assignment by its Google Calendar event ID
	GetAssignmentByGoogleCalendarEventID(eventID string) (*Assignment, error)

	// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
	UpdateAssignmentParent(id int64, parent string, override bool) error
}

// Ensure Scheduler implements SchedulerInterface
var _ SchedulerInterface = (*Scheduler)(nil)
