package scheduler

import (
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
)

// Assignment represents a night routine assignment
type Assignment struct {
	ID                    int64
	Date                  time.Time
	Parent                string
	GoogleCalendarEventID string
}

// Scheduler handles the night routine scheduling logic
type Scheduler struct {
	config  *config.Config
	tracker fairness.TrackerInterface
}

// New creates a new Scheduler instance
func New(cfg *config.Config, tracker fairness.TrackerInterface) *Scheduler {
	return &Scheduler{
		config:  cfg,
		tracker: tracker,
	}
}

// GenerateSchedule creates a schedule for the specified date range
// This updated version respects overridden assignments as fixed points
func (s *Scheduler) GenerateSchedule(start, end time.Time) ([]*Assignment, error) {
	var schedule []*Assignment
	current := start

	// Get all existing assignments in the date range, including overrides
	existingAssignments, err := s.tracker.GetAssignmentsInRange(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing assignments: %w", err)
	}

	// Map overridden assignments by date for easy lookup
	assignmentByDateOverridden := make(map[string]*fairness.Assignment)
	for _, a := range existingAssignments {
		if !a.Override {
			// Only keep overridden assignments
			continue
		}
		dateStr := a.Date.Format("2006-01-02")
		assignmentByDateOverridden[dateStr] = a
	}

	// Process each day in the range
	for !current.After(end) {
		dateStr := current.Format("2006-01-02")

		// Check if there's an existing assignment overridden for this date
		if existing, ok := assignmentByDateOverridden[dateStr]; ok {
			// Convert to scheduler assignment
			assignment := &Assignment{
				ID:                    existing.ID,
				Date:                  existing.Date,
				Parent:                existing.Parent,
				GoogleCalendarEventID: existing.GoogleCalendarEventID,
			}
			schedule = append(schedule, assignment)
		} else {
			// No overridden assignment, create a new one or update existing one
			assignment, err := s.assignForDate(current)
			if err != nil {
				return nil, fmt.Errorf("failed to assign for date %v: %w", current, err)
			}
			schedule = append(schedule, assignment)
		}

		current = current.AddDate(0, 0, 1)
	}

	return schedule, nil
}

// assignForDate determines who should do the night routine on a specific date and records it
func (s *Scheduler) assignForDate(date time.Time) (*Assignment, error) {
	// Get last assignments up to the given date to ensure fairness, including overridden ones
	lastAssignments, err := s.tracker.GetLastAssignmentsUntil(5, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get last assignments: %w", err)
	}

	// Get parent stats for balanced distribution up to the given date
	stats, err := s.tracker.GetParentStatsUntil(date)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent stats: %w", err)
	}

	// Determine the next parent to assign based on fairness rules
	parent, err := s.determineParentForDate(date, lastAssignments, stats)
	if err != nil {
		return nil, err
	}

	// Record the assignment in the database
	trackerAssignment, err := s.tracker.RecordAssignment(parent, date)
	if err != nil {
		return nil, fmt.Errorf("failed to record assignment: %w", err)
	}

	// Convert to scheduler assignment
	return &Assignment{
		ID:                    trackerAssignment.ID,
		Date:                  trackerAssignment.Date,
		Parent:                trackerAssignment.Parent,
		GoogleCalendarEventID: trackerAssignment.GoogleCalendarEventID,
	}, nil
}

// UpdateGoogleCalendarEventID updates the assignment with the Google Calendar event ID
func (s *Scheduler) UpdateGoogleCalendarEventID(assignment *Assignment, eventID string) error {
	err := s.tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, eventID)
	if err != nil {
		return fmt.Errorf("failed to update assignment with Google Calendar event ID: %w", err)
	}

	// Update the assignment object
	assignment.GoogleCalendarEventID = eventID
	return nil
}

// GetAssignmentByGoogleCalendarEventID finds an assignment by its Google Calendar event ID
func (s *Scheduler) GetAssignmentByGoogleCalendarEventID(eventID string) (*Assignment, error) {
	assignment, err := s.tracker.GetAssignmentByGoogleCalendarEventID(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment by Google Calendar event ID: %w", err)
	}

	if assignment == nil {
		return nil, nil
	}

	return &Assignment{
		ID:                    assignment.ID,
		Date:                  assignment.Date,
		Parent:                assignment.Parent,
		GoogleCalendarEventID: assignment.GoogleCalendarEventID,
	}, nil
}

// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
func (s *Scheduler) UpdateAssignmentParent(id int64, parent string, override bool) error {
	err := s.tracker.UpdateAssignmentParent(id, parent, override)
	if err != nil {
		return fmt.Errorf("failed to update assignment parent: %w", err)
	}

	return nil
}

// determineParentForDate determines who should do the night routine on a specific date
func (s *Scheduler) determineParentForDate(date time.Time, lastAssignments []*fairness.Assignment, stats map[string]fairness.Stats) (string, error) {
	dayOfWeek := date.Format("Monday")

	// Check unavailability
	parentAUnavailable := contains(s.config.Availability.ParentAUnavailable, dayOfWeek)
	parentBUnavailable := contains(s.config.Availability.ParentBUnavailable, dayOfWeek)

	if parentAUnavailable && parentBUnavailable {
		return "", fmt.Errorf("both parents unavailable on %s", dayOfWeek)
	}

	// If one parent is unavailable, assign to the other
	if parentAUnavailable {
		return s.config.Parents.ParentB, nil
	}
	if parentBUnavailable {
		return s.config.Parents.ParentA, nil
	}

	// Determine next parent based on fairness rules
	return s.determineNextParent(lastAssignments, stats), nil
}

// contains checks if a string slice contains a specific value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// determineNextParent applies fairness rules to select the next parent
func (s *Scheduler) determineNextParent(lastAssignments []*fairness.Assignment, stats map[string]fairness.Stats) string {
	if len(lastAssignments) == 0 {
		// First assignment ever, assign to the parent with fewer total assignments
		if stats[s.config.Parents.ParentA].TotalAssignments <= stats[s.config.Parents.ParentB].TotalAssignments {
			return s.config.Parents.ParentA
		}
		return s.config.Parents.ParentB
	}

	// Prioritize the parent with fewer total assignments
	statsA := stats[s.config.Parents.ParentA]
	statsB := stats[s.config.Parents.ParentB]

	if statsA.TotalAssignments < statsB.TotalAssignments {
		return s.config.Parents.ParentA
	} else if statsB.TotalAssignments < statsA.TotalAssignments {
		return s.config.Parents.ParentB
	}

	// If total assignments are equal, prioritize the parent with fewer recent assignments (last 30 days)
	if statsA.Last30Days < statsB.Last30Days {
		return s.config.Parents.ParentA
	} else if statsB.Last30Days < statsA.Last30Days {
		return s.config.Parents.ParentB
	}

	// Avoid more than two consecutive assignments
	lastParent := lastAssignments[0].Parent
	consecutiveCount := 1
	for i := 1; i < len(lastAssignments) && lastAssignments[i].Parent == lastParent; i++ {
		consecutiveCount++
	}

	if consecutiveCount >= 2 {
		// Force switch to the other parent
		if lastParent == s.config.Parents.ParentA {
			return s.config.Parents.ParentB
		}
		return s.config.Parents.ParentA
	}

	// Default to alternating
	if lastParent == s.config.Parents.ParentB {
		return s.config.Parents.ParentA
	}
	return s.config.Parents.ParentB
}
