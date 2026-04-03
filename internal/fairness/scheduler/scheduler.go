package scheduler

import (
	"fmt"
	"slices"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/rs/zerolog"
)

// ParentType represents which parent is assigned
type ParentType int

const (
	ParentTypeA ParentType = iota
	ParentTypeB
	ParentTypeBabysitter
)

// String returns the string representation of the ParentType
func (p ParentType) String() string {
	switch p {
	case ParentTypeA:
		return "ParentA"
	case ParentTypeB:
		return "ParentB"
	case ParentTypeBabysitter:
		return "Babysitter"
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
	CaregiverType         fairness.CaregiverType
	GoogleCalendarEventID string
	DecisionReason        fairness.DecisionReason
	UpdatedAt             time.Time
}

// scheduleConfig holds configuration resolved once per GenerateSchedule call
// to avoid repeated config store queries for every day in the range.
type scheduleConfig struct {
	parentA            string
	parentB            string
	parentAUnavailable []string
	parentBUnavailable []string
}

// Scheduler handles the night routine scheduling logic
type Scheduler struct {
	configStore config.ConfigStoreInterface
	tracker     fairness.TrackerInterface
	logger      zerolog.Logger
}

// New creates a new Scheduler instance
func New(configStore config.ConfigStoreInterface, tracker fairness.TrackerInterface) *Scheduler {
	return &Scheduler{
		configStore: configStore,
		tracker:     tracker,
		logger:      logging.GetLogger("scheduler"),
	}
}

// getParents reads parent names from the config store.
func (s *Scheduler) getParents() (parentA, parentB string, err error) {
	return s.configStore.GetParents()
}

// resolveScheduleConfig fetches parents and availability once from the config
// store so that the per-day assignment loop does not repeat those queries.
func (s *Scheduler) resolveScheduleConfig() (*scheduleConfig, error) {
	parentA, parentB, err := s.configStore.GetParents()
	if err != nil {
		return nil, fmt.Errorf("failed to get parent names: %w", err)
	}
	parentADays, err := s.configStore.GetAvailability("parent_a")
	if err != nil {
		return nil, fmt.Errorf("failed to get parent_a availability: %w", err)
	}
	parentBDays, err := s.configStore.GetAvailability("parent_b")
	if err != nil {
		return nil, fmt.Errorf("failed to get parent_b availability: %w", err)
	}
	return &scheduleConfig{
		parentA:            parentA,
		parentB:            parentB,
		parentAUnavailable: parentADays,
		parentBUnavailable: parentBDays,
	}, nil
}

// GenerateSchedule creates a schedule for the specified date range, considering a current time.
// Assignments that are overridden or occurred before/on currentTime are considered fixed.
// When an override exists on or after the current day, all non-override days after that override are recalculated.
func (s *Scheduler) GenerateSchedule(start, end time.Time, currentTime time.Time) ([]*Assignment, error) {
	genLogger := s.logger.With().
		Time("start_date", start).
		Time("end_date", end).
		Time("current_time", currentTime).
		Logger()
	genLogger.Info().Msg("Generating schedule")

	// Resolve config once for the entire schedule generation to avoid
	// repeated config store queries for every day in the range.
	cfg, err := s.resolveScheduleConfig()
	if err != nil {
		genLogger.Error().Err(err).Msg("Failed to resolve schedule config")
		return nil, fmt.Errorf("failed to resolve schedule config: %w", err)
	}
	parentA := cfg.parentA

	var schedule []*Assignment
	current := start

	// Get all existing assignments in the date range
	genLogger.Debug().Msg("Fetching all existing assignments in range")
	existingAssignments, err := s.tracker.GetAssignmentsInRange(start, end)
	if err != nil {
		genLogger.Error().Err(err).Msg("Failed to get existing assignments")
		return nil, fmt.Errorf("failed to get existing assignments: %w", err)
	}
	genLogger.Debug().Int("count", len(existingAssignments)).Msg("Fetched existing assignments")

	// Use the local date string of currentTime for "today" comparisons.
	// time.Truncate(24h) truncates to UTC midnight which is wrong for servers in non-UTC
	// timezones: a server in UTC-4 at 20:00 local = 00:00 UTC next day, making
	// Truncate identify tomorrow as "today".  Date strings (formatted in the time's
	// own location) are always consistent with the DB which stores local date strings.

	// First pass: find the earliest override in the range.
	// Days after this override that are on or after currentDay need recalculation.
	var earliestOverrideStr string
	for _, a := range existingAssignments {
		if a.Override {
			assignmentDayStr := a.Date.Format("2006-01-02")
			if earliestOverrideStr == "" || assignmentDayStr < earliestOverrideStr {
				earliestOverrideStr = assignmentDayStr
			}
		}
	}
	if earliestOverrideStr != "" {
		genLogger.Debug().Str("earliest_override", earliestOverrideStr).Msg("Found earliest override in range")
	} else {
		genLogger.Debug().Msg("No override found in range")
	}

	// Second pass: map assignments that are fixed
	// Fixed assignments are:
	// 1. Assignments strictly before today AND strictly before the start date (truly past)
	// 2. Override assignments (always fixed - user explicitly set them)
	// NOT fixed (will be recalculated):
	// - Non-override assignments at the start date (the caller explicitly requested recalculation from here)
	// - Non-override assignments on or after currentDay that are after an override
	startDayStr := start.Format("2006-01-02")
	currentDayStr := currentTime.Format("2006-01-02")
	assignmentFixedInTime := make(map[string]*fairness.Assignment)
	fixedCount := 0
	for _, a := range existingAssignments {
		assignmentDayStr := a.Date.Format("2006-01-02")

		// Overrides are always fixed
		if a.Override {
			assignmentFixedInTime[assignmentDayStr] = a
			fixedCount++
			continue
		}

		// The start date is never fixed — the caller explicitly requested
		// recalculation from this point (e.g. after an unlock or babysitter removal).
		if assignmentDayStr == startDayStr {
			continue
		}

		// If there's an override and this non-override assignment is after it,
		// recalculate regardless of whether it's in the past or future.
		// This ensures that setting a babysitter on a recent past day correctly
		// shifts subsequent assignments even if they are also in the past.
		if earliestOverrideStr != "" && assignmentDayStr > earliestOverrideStr {
			continue // Not fixed, will be recalculated
		}

		// Past assignments (strictly before today's local date) are fixed - they already happened
		if assignmentDayStr < currentDayStr {
			assignmentFixedInTime[assignmentDayStr] = a
			fixedCount++
			continue
		}

		// Today's assignment not affected by an override: fix it
		if assignmentDayStr == currentDayStr {
			assignmentFixedInTime[assignmentDayStr] = a
			fixedCount++
		}
		// Future assignments (not override, not past, not today): recalculate
	}
	genLogger.Debug().Int("fixed_count", fixedCount).Msg("Mapped fixed assignments (overridden or past)")

	// Process each day in the range
	genLogger.Debug().Msg("Processing days in range")
	for !current.After(end) {
		dateStr := current.Format("2006-01-02")
		dayLogger := genLogger.With().Str("date", dateStr).Logger()

		// Check if there's a fixed assignment (overridden, past, or before override) for this date
		if fixedAssignment, ok := assignmentFixedInTime[dateStr]; ok {
			dayLogger.Info().Int64("assignment_id", fixedAssignment.ID).Str("parent", fixedAssignment.Parent).Str("reason", string(fixedAssignment.DecisionReason)).Bool("override", fixedAssignment.Override).Msg("Using fixed assignment")
			// Convert to scheduler assignment
			parentType := resolveParentType(fixedAssignment, parentA)
			assignment := &Assignment{
				ID:                    fixedAssignment.ID,
				Date:                  fixedAssignment.Date,
				Parent:                fixedAssignment.Parent,
				ParentType:            parentType,
				CaregiverType:         fixedAssignment.CaregiverType,
				GoogleCalendarEventID: fixedAssignment.GoogleCalendarEventID,
				DecisionReason:        fixedAssignment.DecisionReason, // Use the reason from the fixed assignment
				UpdatedAt:             fixedAssignment.UpdatedAt,
			}
			schedule = append(schedule, assignment)
		} else {
			dayLogger.Debug().Msg("No fixed assignment found for this date, assigning parent")
			// No fixed assignment, determine assignment based on fairness rules
			assignment, err := s.assignForDate(current, cfg)
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

// assignForDate determines who should do the night routine on a specific date and records it.
// It uses the pre-resolved scheduleConfig to avoid repeated config store queries.
func (s *Scheduler) assignForDate(date time.Time, cfg *scheduleConfig) (*Assignment, error) {
	assignLogger := s.logger.With().Str("date", date.Format("2006-01-02")).Logger()
	assignLogger.Debug().Msg("Assigning parent for date")

	parentAName := cfg.parentA
	parentBName := cfg.parentB

	// Fetch the last 7 assignments of all caregiver types. This single list is
	// used for everything: parent-only entries are derived via parentOnly() for
	// streaks and lastParent; the full list detects babysitter gaps and recent
	// unavailability. Fetching 7 ensures enough parent entries even when
	// babysitter nights are interspersed.
	assignLogger.Debug().Msg("Fetching last assignments")
	lastAssignments, err := s.tracker.GetLastAssignmentsUntil(7, date)
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
	parent, decisionReason, err := s.determineParentForDate(date, lastAssignments, stats, cfg)
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

	// Save assignment details for non-override decisions
	if trackerAssignment.CaregiverType != fairness.CaregiverTypeBabysitter && decisionReason != fairness.DecisionReasonOverride {
		assignLogger.Debug().Msg("Saving assignment details")
		statsA := stats[parentAName]
		statsB := stats[parentBName]

		err = s.tracker.SaveAssignmentDetails(trackerAssignment.ID, date, parentAName, statsA, parentBName, statsB)
		if err != nil {
			// Log error but don't fail the assignment
			assignLogger.Error().Err(err).Msg("Failed to save assignment details")
		} else {
			assignLogger.Debug().Msg("Assignment details saved successfully")
		}
	}

	// Convert to scheduler assignment
	parentType := resolveParentType(trackerAssignment, parentAName)
	return &Assignment{
		ID:                    trackerAssignment.ID,
		Date:                  trackerAssignment.Date,
		Parent:                trackerAssignment.Parent,
		ParentType:            parentType,
		CaregiverType:         trackerAssignment.CaregiverType,
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
	parentA, _, err := s.getParents()
	if err != nil {
		getLogger.Error().Err(err).Msg("Failed to get parent names")
		return nil, fmt.Errorf("failed to get parent names: %w", err)
	}
	parentType := resolveParentType(assignment, parentA)
	return &Assignment{
		ID:                    assignment.ID,
		Date:                  assignment.Date,
		Parent:                assignment.Parent,
		ParentType:            parentType,
		CaregiverType:         assignment.CaregiverType,
		GoogleCalendarEventID: assignment.GoogleCalendarEventID,
		DecisionReason:        assignment.DecisionReason,
		UpdatedAt:             assignment.UpdatedAt, // Include UpdatedAt
	}, nil
}

// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
// When override is true, it also sets the decision reason to Override
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

// UpdateAssignmentToBabysitter updates an assignment to a babysitter and sets override state.
func (s *Scheduler) UpdateAssignmentToBabysitter(id int64, babysitterName string, override bool) error {
	updateLogger := s.logger.With().
		Int64("assignment_id", id).
		Str("babysitter_name", babysitterName).
		Bool("override", override).
		Logger()
	updateLogger.Info().Msg("Updating assignment to babysitter")

	err := s.tracker.UpdateAssignmentToBabysitter(id, babysitterName, override)
	if err != nil {
		updateLogger.Error().Err(err).Msg("Failed to update assignment to babysitter in tracker")
		return fmt.Errorf("failed to update assignment to babysitter: %w", err)
	}

	updateLogger.Info().Msg("Assignment updated to babysitter successfully")
	return nil
}

func resolveParentType(a *fairness.Assignment, parentAName string) ParentType {
	if a.CaregiverType == fairness.CaregiverTypeBabysitter {
		return ParentTypeBabysitter
	}
	if a.Parent == parentAName {
		return ParentTypeA
	}
	return ParentTypeB
}

// determineParentForDate determines who should do the night routine on a specific date.
// It uses the pre-resolved scheduleConfig for parent names and availability.
// lastAssignments contains all caregiver types (parent + babysitter); parent-only
// entries are derived internally via parentOnly() when needed for streaks/stats.
func (s *Scheduler) determineParentForDate(date time.Time, lastAssignments []*fairness.Assignment, stats map[string]fairness.Stats, cfg *scheduleConfig) (string, fairness.DecisionReason, error) {
	determineLogger := s.logger.With().Str("date", date.Format("2006-01-02")).Logger()
	determineLogger.Debug().Msg("Determining parent for date considering unavailability")
	dayOfWeek := date.Format("Monday")

	parentA := cfg.parentA
	parentB := cfg.parentB

	parentAUnavailable := contains(cfg.parentAUnavailable, dayOfWeek)
	parentBUnavailable := contains(cfg.parentBUnavailable, dayOfWeek)
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
		determineLogger.Info().Str("assigned_parent", parentB).Msg("Parent A unavailable, assigning Parent B")
		return parentB, fairness.DecisionReasonUnavailability, nil
	}
	if parentBUnavailable {
		determineLogger.Info().Str("assigned_parent", parentA).Msg("Parent B unavailable, assigning Parent A")
		return parentA, fairness.DecisionReasonUnavailability, nil
	}

	// Determine next parent based on fairness rules
	determineLogger.Debug().Msg("Both parents available, determining next parent based on fairness")
	parent, reason := s.determineNextParent(date, parentA, parentB, lastAssignments, stats)
	determineLogger.Info().Str("assigned_parent", parent).Str("reason", string(reason)).Msg("Determined next parent based on fairness rules")
	return parent, reason, nil
}

// contains checks if a string slice contains a specific value
func contains(slice []string, value string) bool {
	return slices.Contains(slice, value)
}

// parentOnly returns a filtered slice containing only parent assignments,
// preserving the original reverse-chronological order. This allows the
// scheduler to work with a single all-types list while still extracting
// parent-only information for streak counting and lastParent detection.
func parentOnly(assignments []*fairness.Assignment) []*fairness.Assignment {
	var parents []*fairness.Assignment
	for _, a := range assignments {
		if a.CaregiverType == fairness.CaregiverTypeParent {
			parents = append(parents, a)
		}
	}
	return parents
}

// otherParentOf returns the other parent given the current parent.
func otherParentOf(current, parentA, parentB string) string {
	if current == parentA {
		return parentB
	}
	return parentA
}

// determineNextParent applies fairness rules to select the next parent.
//
// Decision cascade (first match wins):
//  1. No prior parent assignments → parent with fewer (or equal) total assignments (TotalCount)
//  2. TotalCount — parent with fewer total assignments.
//  3. ConsecutiveLimit — when totals are tied and the same parent has 2+
//     consecutive assignments, force a switch.
//  4. RecentCount — parent with fewer last-30-day assignments.
//  5. Alternating — default: alternate from the last parent.
//
// lastAssignments contains all caregiver types (parent + babysitter) in reverse
// chronological order. Parent-only entries are derived via parentOnly() for
// streak counting and lastParent detection; babysitter nights are excluded from
// these calculations but preserved in the full list for context.
func (s *Scheduler) determineNextParent(date time.Time, parentA, parentB string, lastAssignments []*fairness.Assignment, stats map[string]fairness.Stats) (string, fairness.DecisionReason) {
	fairnessLogger := s.logger.With().Interface("stats", stats).Logger()
	fairnessLogger.Debug().Msg("Applying fairness rules to determine next parent")

	// Derive parent-only entries for streaks and lastParent.
	parents := parentOnly(lastAssignments)

	// ── 1. No prior parent assignments ───────────────────────────────────
	if len(parents) == 0 {
		fairnessLogger.Info().Msg("No previous assignments, assigning based on total counts")
		if stats[parentA].TotalAssignments <= stats[parentB].TotalAssignments {
			fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (fewer/equal total)")
			return parentA, fairness.DecisionReasonTotalCount
		}
		fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (fewer total)")
		return parentB, fairness.DecisionReasonTotalCount
	}

	lastParent := parents[0].Parent
	other := otherParentOf(lastParent, parentA, parentB)

	statsA := stats[parentA]
	statsB := stats[parentB]

	// ── 2. TotalCount ───────────────────────────────────────────────────
	fairnessLogger.Debug().
		Int("parent_a_total", statsA.TotalAssignments).
		Int("parent_b_total", statsB.TotalAssignments).
		Str("last_parent", lastParent).
		Msg("Comparing total assignments")

	if statsA.TotalAssignments != statsB.TotalAssignments {
		fewerParent := parentA
		if statsB.TotalAssignments < statsA.TotalAssignments {
			fewerParent = parentB
		}

		fairnessLogger.Debug().Str("assigned_parent", fewerParent).Msg("Assigning parent with fewer total")
		return fewerParent, fairness.DecisionReasonTotalCount
	}

	// ── 3. ConsecutiveLimit (totals tied, 2+ streak) ─────────────────────
	consecutiveCount := 1
	for i := 1; i < len(parents) && parents[i].Parent == lastParent; i++ {
		consecutiveCount++
	}
	fairnessLogger.Debug().Str("last_parent", lastParent).Int("consecutive_count", consecutiveCount).Msg("Checking consecutive assignments")

	if consecutiveCount >= 2 {
		fairnessLogger.Info().Msg("Forcing switch due to consecutive assignments limit")
		fairnessLogger.Debug().Str("assigned_parent", other).Msg("Assigning other parent (forced switch)")
		return other, fairness.DecisionReasonConsecutiveLimit
	}

	// ── 4. RecentCount ──────────────────────────────────────────────────
	fairnessLogger.Debug().
		Int("parent_a_last30", statsA.Last30Days).
		Int("parent_b_last30", statsB.Last30Days).
		Msg("Total assignments equal, comparing last 30 days")

	if statsA.Last30Days != statsB.Last30Days {
		fewerRecentParent := parentA
		if statsB.Last30Days < statsA.Last30Days {
			fewerRecentParent = parentB
		}

		fairnessLogger.Debug().Str("assigned_parent", fewerRecentParent).Msg("Assigning parent with fewer recent")
		return fewerRecentParent, fairness.DecisionReasonRecentCount
	}

	// ── 5. Alternating ───────────────────────────────────────────────────
	fairnessLogger.Info().Msg("All fairness factors equal or within limits, defaulting to alternating")
	fairnessLogger.Debug().Str("assigned_parent", other).Msg("Assigning other parent (alternating)")
	return other, fairness.DecisionReasonAlternating
}
