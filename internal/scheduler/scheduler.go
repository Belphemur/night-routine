package scheduler

import (
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
)

// Assignment represents a night routine assignment
type Assignment struct {
	Date   time.Time
	Parent string
}

// Scheduler handles the night routine scheduling logic
type Scheduler struct {
	config  *config.Config
	tracker *fairness.Tracker
}

// New creates a new Scheduler instance
func New(cfg *config.Config, tracker *fairness.Tracker) *Scheduler {
	return &Scheduler{
		config:  cfg,
		tracker: tracker,
	}
}

// GenerateSchedule creates a schedule for the specified date range
func (s *Scheduler) GenerateSchedule(start, end time.Time) ([]Assignment, error) {
	var schedule []Assignment
	current := start

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

	for !current.After(end) {
		assignment, err := s.assignForDate(current, lastAssignments, stats)
		if err != nil {
			return nil, fmt.Errorf("failed to assign for date %v: %w", current, err)
		}
		schedule = append(schedule, assignment)
		current = current.AddDate(0, 0, 1)
	}

	return schedule, nil
}

// assignForDate determines who should do the night routine on a specific date
func (s *Scheduler) assignForDate(date time.Time, lastAssignments []fairness.Assignment, stats map[string]fairness.Stats) (Assignment, error) {
	dayOfWeek := date.Format("Monday")

	// Check unavailability
	parentAUnavailable := contains(s.config.Availability.ParentAUnavailable, dayOfWeek)
	parentBUnavailable := contains(s.config.Availability.ParentBUnavailable, dayOfWeek)

	if parentAUnavailable && parentBUnavailable {
		return Assignment{}, fmt.Errorf("both parents unavailable on %s", dayOfWeek)
	}

	// If one parent is unavailable, assign to the other
	if parentAUnavailable {
		return Assignment{Date: date, Parent: s.config.Parents.ParentB}, nil
	}
	if parentBUnavailable {
		return Assignment{Date: date, Parent: s.config.Parents.ParentA}, nil
	}

	// Determine next parent based on fairness rules
	nextParent := s.determineNextParent(lastAssignments, stats)
	return Assignment{Date: date, Parent: nextParent}, nil
}

// determineNextParent applies fairness rules to select the next parent
func (s *Scheduler) determineNextParent(lastAssignments []fairness.Assignment, stats map[string]fairness.Stats) string {
	if len(lastAssignments) == 0 {
		// First assignment ever, check total stats
		if stats[s.config.Parents.ParentA].TotalAssignments <= stats[s.config.Parents.ParentB].TotalAssignments {
			return s.config.Parents.ParentA
		}
		return s.config.Parents.ParentB
	}

	// Check consecutive assignments
	consecutiveCount := 1
	lastParent := lastAssignments[0].Parent
	for i := 1; i < len(lastAssignments) && lastAssignments[i].Parent == lastParent; i++ {
		consecutiveCount++
	}

	if consecutiveCount >= 2 {
		// Switch after two consecutive assignments
		if lastParent == s.config.Parents.ParentA {
			return s.config.Parents.ParentB
		}
		return s.config.Parents.ParentA
	}

	// Balance monthly assignments
	statsA := stats[s.config.Parents.ParentA]
	statsB := stats[s.config.Parents.ParentB]
	if statsA.Last30Days < statsB.Last30Days {
		return s.config.Parents.ParentA
	} else if statsB.Last30Days < statsA.Last30Days {
		return s.config.Parents.ParentB
	}

	// If everything is balanced, alternate from last assignment
	if lastParent == s.config.Parents.ParentA {
		return s.config.Parents.ParentB
	}
	return s.config.Parents.ParentA
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
