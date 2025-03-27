# Refactoring Plan: Scheduler.GenerateSchedule

## Overview

This document outlines the plan for refactoring the `Scheduler.GenerateSchedule` method to respect overridden events as fixed points in time when generating the schedule. This is part of the Calendar Entry Overrides feature, which allows users to manually change parent assignments by editing events in Google Calendar.

## Current Implementation

Currently, the `GenerateSchedule` method creates a schedule for a specified date range without considering whether any assignments have been manually overridden. It treats all days in the range as if they need a new assignment, only checking if there's an existing assignment for each date.

```go
// Current GenerateSchedule implementation (simplified)
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
```

## Desired Behavior

The refactored `GenerateSchedule` method should:

1. Respect overridden assignments as fixed points in time
2. Only generate new assignments for dates that don't have existing assignments
3. Ensure the fairness rules take into account the overridden assignments when determining future assignments

## Implementation Plan

### 1. Add Index on assignment_date

First, we need to add an index on the `assignment_date` column to improve query performance when retrieving assignments by date range:

```sql
-- Create a new migration file: 000006_add_index_assignment_date.up.sql
CREATE INDEX IF NOT EXISTS idx_assignments_date ON assignments(assignment_date);

-- Create a new migration file: 000006_add_index_assignment_date.down.sql
DROP INDEX IF EXISTS idx_assignments_date;
```

### 2. Update RecordAssignment Method

Next, we need to update the `RecordAssignment` method in the `Tracker` to check if there's already an assignment at the specified date. If there is, it should update the parent; otherwise, it should create a new assignment:

```go
// RecordAssignment records a new assignment or updates an existing one
func (t *Tracker) RecordAssignment(parent string, date time.Time) (*Assignment, error) {
    // Check if there's already an assignment for this date
    existingAssignment, err := t.GetAssignmentByDate(date)
    if err != nil {
        return nil, fmt.Errorf("failed to check existing assignment: %w", err)
    }

    // If there's already an assignment, update it
    if existingAssignment != nil {
        // Only update if the parent has changed
        if existingAssignment.Parent != parent {
            _, err := t.db.Exec(`
            UPDATE assignments
            SET parent_name = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
            `, parent, existingAssignment.ID)

            if err != nil {
                return nil, fmt.Errorf("failed to update assignment: %w", err)
            }

            // Refresh the assignment
            return t.GetAssignmentByID(existingAssignment.ID)
        }

        // Parent hasn't changed, return the existing assignment
        return existingAssignment, nil
    }

    // No existing assignment, create a new one
    result, err := t.db.Exec(`
    INSERT INTO assignments (parent_name, assignment_date, override, google_calendar_event_id)
    VALUES (?, ?, ?, ?)
    `, parent, date.Format("2006-01-02"), false, "")

    if err != nil {
        return nil, fmt.Errorf("failed to record assignment: %w", err)
    }

    // Get the last inserted ID
    id, err := result.LastInsertId()
    if err != nil {
        return nil, fmt.Errorf("failed to get last insert ID: %w", err)
    }

    // Get the full assignment record
    return t.GetAssignmentByID(id)
}
```

### 3. Add Method to Get All Assignments in a Date Range

Next, we need to add a method to the `Tracker` to retrieve all assignments in a date range:

```go
// GetAssignmentsInRange retrieves all assignments in a date range
func (t *Tracker) GetAssignmentsInRange(start, end time.Time) ([]*Assignment, error) {
    startStr := start.Format("2006-01-02")
    endStr := end.Format("2006-01-02")

    rows, err := t.db.Query(`
    SELECT id, parent_name, assignment_date, override, google_calendar_event_id, created_at, updated_at
    FROM assignments
    WHERE assignment_date >= ? AND assignment_date <= ?
    ORDER BY assignment_date ASC
    `, startStr, endStr)

    if err != nil {
        return nil, fmt.Errorf("failed to query assignments in range: %w", err)
    }
    defer rows.Close()

    var assignments []*Assignment
    for rows.Next() {
        var a Assignment
        var dateStr string
        var createdAtStr, updatedAtStr string

        if err := rows.Scan(
            &a.ID,
            &a.Parent,
            &dateStr,
            &a.Override,
            &a.GoogleCalendarEventID,
            &createdAtStr,
            &updatedAtStr,
        ); err != nil {
            return nil, fmt.Errorf("failed to scan row: %w", err)
        }

        date, err := time.Parse("2006-01-02", dateStr)
        if err != nil {
            return nil, fmt.Errorf("failed to parse date: %w", err)
        }
        a.Date = date

        createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
        if err == nil {
            a.CreatedAt = createdAt
        }

        updatedAt, err := time.Parse("2006-01-02 15:04:05", updatedAtStr)
        if err == nil {
            a.UpdatedAt = updatedAt
        }

        assignments = append(assignments, &a)
    }

    return assignments, nil
}
```

### 4. Refactor GenerateSchedule Method

Next, we'll refactor the `GenerateSchedule` method to respect existing assignments, particularly those that have been overridden:

```go
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

    // Map overriden assignments by date for easy lookup
    assignmentByDateOverridden := make(map[string]*Assignment)
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

        // Check if there's an existing assignment overriden for this date
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
            // No overriden assignment, create a new one or update existing one
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
```

### 5. Update assignForDate Method

The `assignForDate` method also needs to be updated to take into account overridden assignments when determining fairness:

```go
// assignForDate determines who should do the night routine on a specific date and records it
func (s *Scheduler) assignForDate(date time.Time) (*Assignment, error) {

    // Get last assignments to ensure fairness, including overridden ones
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
```

## Testing Plan

To ensure the refactored `GenerateSchedule` method works correctly, we should test the following scenarios:

1. **Basic Schedule Generation**: Generate a schedule for a date range with no existing assignments
2. **Respecting Existing Assignments**: Generate a schedule for a date range with some existing assignments
3. **Respecting Overridden Assignments**: Generate a schedule for a date range with some overridden assignments
4. **Fairness with Overrides**: Verify that the fairness rules take into account overridden assignments
5. **Performance Testing**: Verify that the index on `assignment_date` improves query performance for date range queries
6. **RecordAssignment Update Logic**: Verify that the `RecordAssignment` method correctly updates existing assignments instead of creating duplicates

## Implementation Steps

1. Create a new migration to add an index on the `assignment_date` column
2. Update the `RecordAssignment` method to check for existing assignments
3. Add the `GetAssignmentsInRange` method to the `Tracker`
4. Refactor the `GenerateSchedule` method to respect existing assignments
5. Update the `assignForDate` method to consider overridden assignments in fairness calculations
6. Write tests to verify the correct behavior
7. Update the webhook handler to use the refactored methods when recalculating the schedule after an override

## Impact Analysis

This refactoring will ensure that when a user manually overrides a parent assignment in Google Calendar, the system will:

1. Respect that override as a fixed point in time
2. Recalculate the schedule for future days taking into account the override
3. Maintain fairness in the distribution of assignments
4. Avoid creating duplicate assignments for the same date

The addition of an index on the `assignment_date` column will improve query performance when retrieving assignments by date range, which is particularly important for the `GetAssignmentsInRange` method that will be called frequently.

The update to the `RecordAssignment` method will ensure that we don't create duplicate assignments for the same date, which could happen if the method is called multiple times for the same date.

This will provide users with the flexibility to make manual adjustments to the schedule while still maintaining the overall fairness and balance of the system.
