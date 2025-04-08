# Assignment Decision Tracking Implementation Plan

This document outlines the plan for adding a feature to track which part of the scheduler algorithm made the decision on parent assignment, store this information in the database, and display it in the calendar view.

## Overview

Currently, the scheduler algorithm in `internal/scheduler/scheduler.go` makes decisions based on several rules:

1. **Unavailability Check**: If one parent is unavailable on a specific day of the week
2. **Total Assignments**: The parent with fewer total assignments gets priority
3. **Recent Assignments**: The parent with fewer assignments in the last 30 days gets priority
4. **Consecutive Limit**: Avoid more than two consecutive assignments to the same parent
5. **Alternating Pattern**: Default to alternating when all else is equal
6. **Override**: Manual overrides set by users

The system doesn't currently track which of these rules was the deciding factor for each assignment. This plan outlines how to add this tracking and display it in the UI.

## Implementation Steps

### 1. Database Migration

Create a new migration file `internal/database/migrations/sqlite/000009_add_decision_reason.up.sql`:

```sql
-- Add decision_reason column to the assignments table
ALTER TABLE assignments ADD COLUMN decision_reason TEXT;
```

And the corresponding down migration `internal/database/migrations/sqlite/000009_add_decision_reason.down.sql`:

```sql
-- Remove decision_reason column from the assignments table
ALTER TABLE assignments DROP COLUMN decision_reason;
```

### 2. Define Decision Reason Type

Create a string enumeration for the decision reason in `internal/scheduler/scheduler.go`:

```go
// DecisionReason represents the reason for a parent assignment decision
type DecisionReason string

const (
    // Decision reason constants
    DecisionReasonUnavailability   DecisionReason = "Unavailability"
    DecisionReasonTotalCount       DecisionReason = "Total Count"
    DecisionReasonRecentCount      DecisionReason = "Recent Count"
    DecisionReasonConsecutiveLimit DecisionReason = "Consecutive Limit"
    DecisionReasonAlternating      DecisionReason = "Alternating"
    DecisionReasonOverride         DecisionReason = "Override"
)

// String returns the string representation of the DecisionReason
func (d DecisionReason) String() string {
    return string(d)
}
```

### 3. Update Data Models

Update the `Assignment` struct in `internal/fairness/tracker.go`:

```go
// Assignment represents a night routine assignment
type Assignment struct {
    ID                    int64
    Parent                string
    Date                  time.Time
    Override              bool
    GoogleCalendarEventID string
    DecisionReason        string  // Stored as string in the database
    CreatedAt             time.Time
    UpdatedAt             time.Time
}
```

Update the `Assignment` struct in `internal/scheduler/scheduler.go`:

```go
// Assignment represents a night routine assignment
type Assignment struct {
    ID                    int64
    Date                  time.Time
    Parent                string
    ParentType            ParentType
    GoogleCalendarEventID string
    DecisionReason        DecisionReason  // Using the enum type
    UpdatedAt             time.Time
}
```

### 4. Clean Up and Update TrackerInterface

Simplify the `TrackerInterface` in `internal/fairness/interface.go` to have only one way to record assignments:

```go
// TrackerInterface defines the operations for tracking fairness
type TrackerInterface interface {
    // RecordAssignment records a new assignment with all details
    RecordAssignment(parent string, date time.Time, override bool, googleCalendarEventID string, decisionReason string) (*Assignment, error)

    // GetLastAssignmentsUntil returns the last n assignments up to a specific date
    GetLastAssignmentsUntil(n int, until time.Time) ([]*Assignment, error)

    // GetParentStatsUntil returns statistics for each parent up to a specific date
    GetParentStatsUntil(until time.Time) (map[string]Stats, error)

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
}
```

### 5. Clean Up and Update Tracker Implementation

Replace the multiple assignment recording methods in `internal/fairness/tracker.go` with a single implementation:

```go
// RecordAssignment records a new assignment with all details
func (t *Tracker) RecordAssignment(parent string, date time.Time, override bool, googleCalendarEventID string, decisionReason string) (*Assignment, error) {
    recordLogger := t.logger.With().
        Str("date", date.Format(dateFormat)).
        Str("parent", parent).
        Bool("override", override).
        Str("event_id", googleCalendarEventID).
        Str("decision_reason", decisionReason).
        Logger()
    recordLogger.Debug().Msg("Recording assignment details")

    // Check if there's already an assignment for this date
    recordLogger.Debug().Msg("Checking for existing assignment on this date")
    existingAssignment, err := t.GetAssignmentByDate(date)
    if err != nil {
        // Error already logged in GetAssignmentByDate
        return nil, fmt.Errorf("failed to check existing assignment: %w", err)
    }

    // If there's already an assignment, update it
    if existingAssignment != nil {
        recordLogger = recordLogger.With().Int64("assignment_id", existingAssignment.ID).Logger()
        recordLogger.Debug().Msg("Existing assignment found, updating details")

        // Only update if the parent has changed and it wasn't an override
        if existingAssignment.Parent != parent && !existingAssignment.Override {
            recordLogger.Debug().Str("old_parent", existingAssignment.Parent).Str("new_parent", parent).Msg("Updating existing assignment parent (non-override)")
            _, err := t.db.Exec(`
            UPDATE assignments
            SET parent_name = ?, override = ?, google_calendar_event_id = ?, decision_reason = ?
            WHERE id = ?
            `, parent, override, googleCalendarEventID, decisionReason, existingAssignment.ID)

            if err != nil {
                recordLogger.Debug().Err(err).Msg("Failed to update assignment")
                return nil, fmt.Errorf("failed to update assignment: %w", err)
            }
            recordLogger.Debug().Msg("Assignment updated successfully")
            // Refresh the assignment
            return t.GetAssignmentByID(existingAssignment.ID)
        } else if existingAssignment.Override {
            recordLogger.Debug().Str("existing_parent", existingAssignment.Parent).Msg("Existing assignment is an override, not changing parent")
            return existingAssignment, nil
        } else {
            recordLogger.Debug().Str("parent", existingAssignment.Parent).Msg("Parent has not changed, returning existing assignment")
            return existingAssignment, nil
        }
    }

    // No existing assignment, create a new one
    recordLogger.Debug().Msg("No existing assignment found, creating new one")
    result, err := t.db.Exec(`
    INSERT INTO assignments (parent_name, assignment_date, override, google_calendar_event_id, decision_reason)
    VALUES (?, ?, ?, ?, ?)
    `, parent, date.Format(dateFormat), override, googleCalendarEventID, decisionReason)

    if err != nil {
        recordLogger.Debug().Err(err).Msg("Failed to insert new assignment")
        return nil, fmt.Errorf("failed to record assignment: %w", err)
    }

    // Get the last inserted ID
    id, err := result.LastInsertId()
    if err != nil {
        recordLogger.Debug().Err(err).Msg("Failed to get last insert ID after insert")
        return nil, fmt.Errorf("failed to get last insert ID: %w", err)
    }
    recordLogger = recordLogger.With().Int64("assignment_id", id).Logger()
    recordLogger.Debug().Msg("New assignment inserted successfully")

    // Get the full assignment record
    assignment, err := t.GetAssignmentByID(id)
    if err != nil {
        // Error logged in GetAssignmentByID
        return nil, fmt.Errorf("failed to get assignment after insert: %w", err)
    }

    return assignment, nil
}
```

### 6. Update scanAssignment Function

Update the `scanAssignment` function in `internal/fairness/tracker.go` to include the new field:

```go
// scanAssignment scans a row into an Assignment struct
func (t *Tracker) scanAssignment(scanner interface {
    Scan(dest ...interface{}) error
}) (*Assignment, error) {
    var a Assignment
    var dateStr string
    var createdAt, updatedAt time.Time
    var googleEventID sql.NullString
    var decisionReason sql.NullString // New field

    err := scanner.Scan(
        &a.ID,
        &a.Parent,
        &dateStr,
        &a.Override,
        &googleEventID,
        &decisionReason, // New field
        &createdAt,
        &updatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, fmt.Errorf("failed to scan assignment: %w", err)
    }

    if googleEventID.Valid {
        a.GoogleCalendarEventID = googleEventID.String
    }

    if decisionReason.Valid {
        a.DecisionReason = decisionReason.String
    }

    date, err := time.Parse(dateFormat, dateStr)
    if err != nil {
        return nil, fmt.Errorf("failed to parse date: %w", err)
    }
    a.Date = date

    a.CreatedAt = createdAt
    a.UpdatedAt = updatedAt

    return &a, nil
}
```

### 7. Update SQL Queries

Update all SQL queries in `internal/fairness/tracker.go` to include the new column:

```sql
-- Example for GetAssignmentByID
SELECT id, parent_name, assignment_date, override, google_calendar_event_id, decision_reason, created_at, updated_at
FROM assignments
WHERE id = ?
```

### 8. Update Scheduler Logic

Update the `determineParentForDate` function in `internal/scheduler/scheduler.go`:

```go
// determineParentForDate determines who should do the night routine on a specific date
func (s *Scheduler) determineParentForDate(date time.Time, lastAssignments []*fairness.Assignment, stats map[string]fairness.Stats) (string, DecisionReason, error) {
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
        return s.config.Parents.ParentB, DecisionReasonUnavailability, nil
    }
    if parentBUnavailable {
        determineLogger.Info().Str("assigned_parent", s.config.Parents.ParentA).Msg("Parent B unavailable, assigning Parent A")
        return s.config.Parents.ParentA, DecisionReasonUnavailability, nil
    }

    // Determine next parent based on fairness rules
    determineLogger.Debug().Msg("Both parents available, determining next parent based on fairness")
    parent, reason := s.determineNextParent(lastAssignments, stats)
    determineLogger.Info().Str("assigned_parent", parent).Str("reason", string(reason)).Msg("Determined next parent based on fairness rules")
    return parent, reason, nil
}
```

Update the `determineNextParent` function:

```go
// determineNextParent applies fairness rules to select the next parent
func (s *Scheduler) determineNextParent(lastAssignments []*fairness.Assignment, stats map[string]fairness.Stats) (string, DecisionReason) {
    fairnessLogger := s.logger.With().Interface("stats", stats).Logger()
    fairnessLogger.Debug().Msg("Applying fairness rules to determine next parent")

    parentA := s.config.Parents.ParentA
    parentB := s.config.Parents.ParentB

    if len(lastAssignments) == 0 {
        fairnessLogger.Info().Msg("No previous assignments, assigning based on total counts")
        // First assignment ever, assign to the parent with fewer total assignments
        if stats[parentA].TotalAssignments <= stats[parentB].TotalAssignments {
            fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (fewer/equal total)")
            return parentA, DecisionReasonTotalCount
        }
        fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (fewer total)")
        return parentB, DecisionReasonTotalCount
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
        return parentA, DecisionReasonTotalCount
    } else if statsB.TotalAssignments < statsA.TotalAssignments {
        fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (fewer total)")
        return parentB, DecisionReasonTotalCount
    }

    // If total assignments are equal, prioritize the parent with fewer recent assignments (last 30 days)
    fairnessLogger.Debug().
        Int("parent_a_last30", statsA.Last30Days).
        Int("parent_b_last30", statsB.Last30Days).
        Msg("Total assignments equal, comparing last 30 days")
    if statsA.Last30Days < statsB.Last30Days {
        fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (fewer last 30 days)")
        return parentA, DecisionReasonRecentCount
    } else if statsB.Last30Days < statsA.Last30Days {
        fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (fewer last 30 days)")
        return parentB, DecisionReasonRecentCount
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
            return parentB, DecisionReasonConsecutiveLimit
        }
        fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (forced switch)")
        return parentA, DecisionReasonConsecutiveLimit
    }

    // Default to alternating if all else is equal
    fairnessLogger.Info().Msg("All fairness factors equal or within limits, defaulting to alternating")
    if lastParent == parentB {
        fairnessLogger.Debug().Str("assigned_parent", parentA).Msg("Assigning Parent A (alternating)")
        return parentA, DecisionReasonAlternating
    }
    fairnessLogger.Debug().Str("assigned_parent", parentB).Msg("Assigning Parent B (alternating)")
    return parentB, DecisionReasonAlternating
}
```

Update the `assignForDate` method:

```go
// assignForDate determines who should do the night routine on a specific date and records it
func (s *Scheduler) assignForDate(date time.Time) (*Assignment, error) {
    assignLogger := s.logger.With().Str("date", date.Format("2006-01-02")).Logger()
    assignLogger.Debug().Msg("Assigning parent for date")

    // Get last assignments up to the given date to ensure fairness, including overridden ones
    assignLogger.Debug().Msg("Fetching last assignments")
    lastAssignments, err := s.tracker.GetLastAssignmentsUntil(5, date)
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
    trackerAssignment, err := s.tracker.RecordAssignment(parent, date, false, "", string(decisionReason))
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
        DecisionReason:        DecisionReason(trackerAssignment.DecisionReason),
        UpdatedAt:             trackerAssignment.UpdatedAt,
    }, nil
}
```

Update the `GenerateSchedule` method for overrides:

```go
// In the GenerateSchedule method where it handles overrides
if existing, ok := assignmentByDateOverridden[dateStr]; ok {
    // ...
    assignment := &Assignment{
        ID:                    existing.ID,
        Date:                  existing.Date,
        Parent:                existing.Parent,
        ParentType:            parentType,
        GoogleCalendarEventID: existing.GoogleCalendarEventID,
        DecisionReason:        DecisionReasonOverride,
        UpdatedAt:             existing.UpdatedAt,
    }
    // ...
}
```

### 9. UI Updates

Update the calendar cell in `internal/handlers/templates/home.html`:

```html
{{if .Assignment}}
<span class="assignment">{{.Assignment.Parent}}</span>
{{if .Assignment.DecisionReason}}
<span class="decision-reason">{{.Assignment.DecisionReason}}</span>
{{end}} {{end}}
```

Add CSS styling for the decision reason:

```css
.calendar td .decision-reason {
  font-size: 0.8em;
  font-style: italic;
  display: block;
  margin-top: 2px;
  color: #777; /* Muted color */
}

/* Ensure decision reason inherits parent cell styling for different parent types */
.calendar td.ParentA .decision-reason {
  color: rgba(25, 118, 210, 0.7); /* Muted blue */
}

.calendar td.ParentB .decision-reason {
  color: rgba(245, 124, 0, 0.7); /* Muted orange */
}
```

## Implementation Sequence

1. Create the database migration
2. Update the data models and interfaces
3. Clean up the fairness tracker to have only one way to record assignments
4. Implement the decision tracking in the scheduler
5. Update the UI to display the decision reason
6. Test the changes to ensure they work as expected

## Considerations

- **Backward Compatibility**: Existing assignments won't have a decision reason. We may want to handle this gracefully in the UI.
- **Testing**: We should update tests to verify that decision reasons are correctly tracked.
- **User Experience**: The decision reason will be displayed as a subtitle under the parent name, in a smaller font and slightly muted color.
