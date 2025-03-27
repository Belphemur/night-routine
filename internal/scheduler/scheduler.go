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
func (s *Scheduler) GenerateSchedule(start, end time.Time) ([]*Assignment, error) {
	var schedule []*Assignment
	current := start

	for !current.After(end) {
		assignment, err := s.assignForDate(current)
		if err != nil {
			return nil, fmt.Errorf("failed to assign for date %v: %w", current, err)
		}
		schedule = append(schedule, assignment)
		current = current.AddDate(0, 0, 1)
	}

	return schedule, nil
}

// assignForDate determines who should do the night routine on a specific date and records it
func (s *Scheduler) assignForDate(date time.Time) (*Assignment, error) {
	// First check if there's already an assignment for this date
	existingAssignment, err := s.tracker.GetAssignmentByDate(date)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}

	// If there's already an assignment for this date, use it
	if existingAssignment != nil {
		return &Assignment{
			ID:                    existingAssignment.ID,
			Date:                  existingAssignment.Date,
			Parent:                existingAssignment.Parent,
			GoogleCalendarEventID: existingAssignment.GoogleCalendarEventID,
		}, nil
	}

	// Get last assignments to ensure fairness
	lastAssignments, err := s.tracker.GetLastAssignments(5)
	if err != nil {
		return nil, fmt.Errorf("failed to get last assignments: %w", err)
	}

	// Get parent stats for balanced distribution
	stats, err := s.tracker.GetParentStats()
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
