package scheduler

import (
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/logging" // Import logging
	"github.com/rs/zerolog"                               // Import zerolog
)

// ParentType represents which parent is assigned
type ParentType int

const (
	ParentTypeA ParentType = iota
	ParentTypeB
)

// String returns the string representation of the ParentType
func (p ParentType) String() string {
	switch p {
	case ParentTypeA:
		return "ParentA"
	case ParentTypeB:
		return "ParentB"
	default:
		return "Unknown"
	}
}

// Assignment represents a night routine assignment
type Assignment struct {
	ID                    int64
	Date                  time.Time
	Parent                string
	ParentType            ParentType
	GoogleCalendarEventID string
	DecisionReason        fairness.DecisionReason
	UpdatedAt             time.Time
}

// Scheduler handles the night routine scheduling logic
type Scheduler struct {
	config  *config.Config
	tracker fairness.TrackerInterface
	logger  zerolog.Logger // Add logger field
}

// New creates a new Scheduler instance
func New(cfg *config.Config, tracker fairness.TrackerInterface) *Scheduler {
	return &Scheduler{
		config:  cfg,
		tracker: tracker,
		logger:  logging.GetLogger("scheduler"), // Initialize logger
	}
}

// GenerateSchedule creates a schedule for the specified date range
// This updated version respects overridden assignments as fixed points
func (s *Scheduler) GenerateSchedule(start, end time.Time) ([]*Assignment, error) {
	genLogger := s.logger.With().
		Time("start_date", start).
		Time("end_date", end).
		Logger()
	genLogger.Info().Msg("Generating schedule")

	var schedule []*Assignment
	current := start

	// Get all existing assignments in the date range, including overrides
	genLogger.Debug().Msg("Fetching existing assignments in range")
	existingAssignments, err := s.tracker.GetAssignmentsInRange(start, end)
	if err != nil {
		genLogger.Error().Err(err).Msg("Failed to get existing assignments")
		return nil, fmt.Errorf("failed to get existing assignments: %w", err)
	}
	genLogger.Debug().Int("count", len(existingAssignments)).Msg("Fetched existing assignments")

	// Map overridden assignments by date for easy lookup
	assignmentByDateOverridden := make(map[string]*fairness.Assignment)
	overrideCount := 0
	for _, a := range existingAssignments {
		if !a.Override {
			continue
		}
		dateStr := a.Date.Format("2006-01-02")
		assignmentByDateOverridden[dateStr] = a
		overrideCount++
	}
	genLogger.Debug().Int("override_count", overrideCount).Msg("Mapped overridden assignments")

	// Process each day in the range
	genLogger.Debug().Msg("Processing days in range")
	for !current.After(end) {
		dateStr := current.Format("2006-01-02")
		dayLogger := genLogger.With().Str("date", dateStr).Logger()

		// Check if there's an existing assignment overridden for this date
		if existing, ok := assignmentByDateOverridden[dateStr]; ok {
			dayLogger.Info().Int64("assignment_id", existing.ID).Str("parent", existing.Parent).Msg("Using existing overridden assignment")
			// Convert to scheduler assignment
			parentType := ParentTypeB
			if existing.Parent == s.config.Parents.ParentA {
				parentType = ParentTypeA
			}
			assignment := &Assignment{
				ID:                    existing.ID,
				Date:                  existing.Date,
				Parent:                existing.Parent,
				ParentType:            parentType,
				GoogleCalendarEventID: existing.GoogleCalendarEventID,
				DecisionReason:        fairness.DecisionReasonOverride,
				UpdatedAt:             existing.UpdatedAt,
			}
			schedule = append(schedule, assignment)
		} else {
			dayLogger.Debug().Msg("No override found, assigning parent for date")
			// No overridden assignment, create a new one or update existing one
			assignment, err := s.assignForDate(current)
			if err != nil {
				dayLogger.Error().Err(err).Msg("Failed to assign parent for date")
				// Wrap error with date context
				return nil, fmt.Errorf("failed to assign for date %v: %w", current.Format("2006-01-02"), err)
			}
			dayLogger.Info().Int64("assignment_id", assignment.ID).Str("parent", assignment.Parent).Msg("Assigned parent for date")
			schedule = append(schedule, assignment)
		}

		current = current.AddDate(0, 0, 1)
	}

	genLogger.Info().Int("total_assignments", len(schedule)).Msg("Schedule generation complete")
	return schedule, nil
}

// assignForDate determines who should do the night routine on a specific date and records it
func (s *Scheduler) assignForDate(date time.Time) (*Assignment, error) {
	assignLogger := s.logger.With().Str("date", date.Format("2006-01-02")).Logger()
	assignLogger.Debug().Msg("Assigning parent for date")

	// Get last assignments up to the given date to ensure fairness, including overridden ones
	assignLogger.Debug().Msg("Fetching last assignments")
	lastAssignments, err := s.tracker.GetLastAssignmentsUntil(5, date) // Use a constant for lookback?
	if err != nil {
		assignLogger.Error().Err(err).Msg("Failed to get last assignments")
		return nil, fmt.Errorf("failed to get last assignments: %w", err)
	}
	assignLogger.Debug().Int("count", len(lastAssignments)).Msg("Fetched last assignments")

	// Get parent stats for balanced distribution up to the given date
	assignLogger.Debug().Msg("Fetching parent stats")
	stats, err := s.tracker.GetParentStatsUntil(date)
	if err != nil {
		assignLogger.Error().Err(err).Msg("Failed to get parent stats")
		return nil, fmt.Errorf("failed to get parent stats: %w", err)
	}
	assignLogger.Debug().Interface("stats", stats).Msg("Fetched parent stats")

	// Determine the next parent to assign based on fairness rules
	assignLogger.Debug().Msg("Determining parent based on fairness rules")
	parent, decisionReason, err := s.determineParentForDate(date, lastAssignments, stats)
	if err != nil {
		assignLogger.Error().Err(err).Msg("Failed to determine parent for date")
		return nil, err // Error already has context
	}
	assignLogger.Info().Str("parent", parent).Str("decision_reason", string(decisionReason)).Msg("Determined parent for assignment")

	// Record the assignment in the database
	assignLogger.Debug().Msg("Recording assignment in tracker")
	trackerAssignment, err := s.tracker.RecordAssignment(parent, date, false, decisionReason)
	if err != nil {
		assignLogger.Error().Err(err).Msg("Failed to record assignment")
		return nil, fmt.Errorf("failed to record assignment: %w", err)
	}
	assignLogger.Info().Int64("assignment_id", trackerAssignment.ID).Msg("Assignment recorded successfully")

	// Convert to scheduler assignment
	parentType := ParentTypeB
	if trackerAssignment.Parent == s.config.Parents.ParentA {
		parentType = ParentTypeA
	}
	return &Assignment{
		ID:                    trackerAssignment.ID,
		Date:                  trackerAssignment.Date,
		Parent:                trackerAssignment.Parent,
		ParentType:            parentType,
		GoogleCalendarEventID: trackerAssignment.GoogleCalendarEventID,
		DecisionReason:        trackerAssignment.DecisionReason,
		UpdatedAt:             trackerAssignment.UpdatedAt,
	}, nil
}

// UpdateGoogleCalendarEventID updates the assignment with the Google Calendar event ID
func (s *Scheduler) UpdateGoogleCalendarEventID(assignment *Assignment, eventID string) error {
	updateLogger := s.logger.With().
		Int64("assignment_id", assignment.ID).
		Str("date", assignment.Date.Format("2006-01-02")).
		Str("parent", assignment.Parent).
		Str("event_id", eventID).
		Logger()
	updateLogger.Info().Msg("Updating assignment with Google Calendar Event ID")

	err := s.tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, eventID)
	if err != nil {
		updateLogger.Error().Err(err).Msg("Failed to update assignment event ID in tracker")
		return fmt.Errorf("failed to update assignment with Google Calendar event ID: %w", err)
	}

	// Update the assignment object in memory
	assignment.GoogleCalendarEventID = eventID
	updateLogger.Info().Msg("Assignment event ID updated successfully")
	return nil
}

// GetAssignmentByGoogleCalendarEventID finds an assignment by its Google Calendar event ID
func (s *Scheduler) GetAssignmentByGoogleCalendarEventID(eventID string) (*Assignment, error) {
	getLogger := s.logger.With().Str("event_id", eventID).Logger()
	getLogger.Debug().Msg("Getting assignment by Google Calendar Event ID")

	assignment, err := s.tracker.GetAssignmentByGoogleCalendarEventID(eventID)
	if err != nil {
		getLogger.Error().Err(err).Msg("Failed to get assignment by event ID from tracker")
		return nil, fmt.Errorf("failed to get assignment by Google Calendar event ID: %w", err)
	}

	if assignment == nil {
		getLogger.Warn().Msg("No assignment found for the given event ID")
		return nil, nil
	}

	getLogger.Info().Int64("assignment_id", assignment.ID).Msg("Found assignment by event ID")
	parentType := ParentTypeB
	if assignment.Parent == s.config.Parents.ParentA {
		parentType = ParentTypeA
	}
	return &Assignment{
		ID:                    assignment.ID,
		Date:                  assignment.Date,
		Parent:                assignment.Parent,
		ParentType:            parentType,
		GoogleCalendarEventID: assignment.GoogleCalendarEventID,
		DecisionReason:        assignment.DecisionReason,
		UpdatedAt:             assignment.UpdatedAt, // Include UpdatedAt
	}, nil
}

// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
func (s *Scheduler) UpdateAssignmentParent(id int64, parent string, override bool) error {
	updateLogger := s.logger.With().
		Int64("assignment_id", id).
		Str("new_parent", parent).
		Bool("override", override).
		Logger()
	updateLogger.Info().Msg("Updating assignment parent")

	err := s.tracker.UpdateAssignmentParent(id, parent, override)
	if err != nil {
		updateLogger.Error().Err(err).Msg("Failed to update assignment parent in tracker")
		return fmt.Errorf("failed to update assignment parent: %w", err)
	}

	updateLogger.Info().Msg("Assignment parent updated successfully")
	return nil
}

// determineParentForDate determines who should do the night routine on a specific date
func (s *Scheduler) determineParentForDate(date time.Time, lastAssignments []*fairness.Assignment, stats map[string]fairness.Stats) (string, fairness.DecisionReason, error) {
	determineLogger := s.logger.With().Str("date", date.Format("2006-01-02")).Logger()
	determineLogger.Debug().Msg("Determining parent for date considering unavailability")
	dayOfWeek := date.Format("Monday")

	// Check unavailability
	parentAUnavailable := contains(s.config.Availability.ParentAUnavailable, dayOfWeek)
	parentBUnavailable := contains(s.config.Availability.ParentBUnavailable, dayOfWeek)
	determineLogger.Debug().
		Str("day_of_week", dayOfWeek).
		Bool("parent_a_unavailable", parentAUnavailable).
		Bool("parent_b_unavailable", parentBUnavailable).
		Msg("Checked parent unavailability")

	if parentAUnavailable && parentBUnavailable {
		err := fmt.Errorf("both parents unavailable on %s", dayOfWeek)
		determineLogger.Error().Err(err).Msg("Cannot assign parent")
		return "", "", err
	}

	// If one parent is unavailable, assign to the other
	if parentAUnavailable {
		determineLogger.Info().Str("assigned_parent", s.config.Parents.ParentB).Msg("Parent A unavailable, assigning Parent B")
		return s.config.Parents.ParentB, fairness.DecisionReasonUnavailability, nil
	}
	if parentBUnavailable {
		determineLogger.Info().Str("assigned_parent", s.config.Parents.ParentA).Msg("Parent B unavailable, assigning Parent A")
		return s.config.Parents.ParentA, fairness.DecisionReasonUnavailability, nil
	}

	// Determine next parent based on fairness rules
	determineLogger.Debug().Msg("Both parents available, determining next parent based on fairness")
	parent, reason := s.determineNextParent(lastAssignments, stats)
	determineLogger.Info().Str("assigned_parent", parent).Str("reason", string(reason)).Msg("Determined next parent based on fairness rules")
	return parent, reason, nil
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
func (s *Scheduler) determineNextParent(lastAssignments []*fairness.Assignment, stats map[string]fairness.Stats) (string, fairness.DecisionReason) {
	fairnessLogger := s.logger.With().Interface("stats", stats).Logger() // Add stats context
	fairnessLogger.Debug().Msg("Applying fairness rules to determine next parent")

	parentA := s.config.Parents.ParentA
	parentB := s.config.Parents.ParentB

	if len(lastAssignments) == 0 {
		fairnessLogger.Info().Msg("No previous assignments, assigning based on total counts")
		// First assignment ever, assign to the parent with fewer total assignments
		if stats[parentA].TotalAssignments <= stats[parentB].TotalAssignments {
			fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (fewer/equal total)")
			return parentA, fairness.DecisionReasonTotalCount
		}
		fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (fewer total)")
		return parentB, fairness.DecisionReasonTotalCount
	}

	// Prioritize the parent with fewer total assignments
	statsA := stats[parentA]
	statsB := stats[parentB]
	fairnessLogger.Debug().
		Int("parent_a_total", statsA.TotalAssignments).
		Int("parent_b_total", statsB.TotalAssignments).
		Msg("Comparing total assignments")

	if statsA.TotalAssignments < statsB.TotalAssignments {
		fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (fewer total)")
		return parentA, fairness.DecisionReasonTotalCount
	} else if statsB.TotalAssignments < statsA.TotalAssignments {
		fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (fewer total)")
		return parentB, fairness.DecisionReasonTotalCount
	}

	// If total assignments are equal, prioritize the parent with fewer recent assignments (last 30 days)
	fairnessLogger.Debug().
		Int("parent_a_last30", statsA.Last30Days).
		Int("parent_b_last30", statsB.Last30Days).
		Msg("Total assignments equal, comparing last 30 days")
	if statsA.Last30Days < statsB.Last30Days {
		fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (fewer last 30 days)")
		return parentA, fairness.DecisionReasonRecentCount
	} else if statsB.Last30Days < statsA.Last30Days {
		fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (fewer last 30 days)")
		return parentB, fairness.DecisionReasonRecentCount
	}

	// Avoid more than two consecutive assignments
	lastParent := lastAssignments[0].Parent
	consecutiveCount := 1
	for i := 1; i < len(lastAssignments) && lastAssignments[i].Parent == lastParent; i++ {
		consecutiveCount++
	}
	fairnessLogger.Debug().Str("last_parent", lastParent).Int("consecutive_count", consecutiveCount).Msg("Checking consecutive assignments")

	if consecutiveCount >= 2 {
		fairnessLogger.Info().Msg("Forcing switch due to consecutive assignments limit")
		// Force switch to the other parent
		if lastParent == parentA {
			fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (forced switch)")
			return parentB, fairness.DecisionReasonConsecutiveLimit
		}
		fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (forced switch)")
		return parentA, fairness.DecisionReasonConsecutiveLimit
	}

	// Default to alternating if all else is equal
	fairnessLogger.Info().Msg("All fairness factors equal or within limits, defaulting to alternating")
	if lastParent == parentB {
		fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (alternating)")
		return parentA, fairness.DecisionReasonAlternating
	}
	fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (alternating)")
	return parentB, fairness.DecisionReasonAlternating
}
