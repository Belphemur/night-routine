package fairness

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDecisionReasonTracking tests the decision reason tracking functionality
func TestDecisionReasonTracking(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Test recording assignments with different decision reasons
	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		parent         string
		decisionReason string
	}{
		{"Alice", "Total Count"},
		{"Bob", "Recent Count"},
		{"Alice", "Consecutive Limit"},
		{"Bob", "Alternating"},
		{"Alice", "Unavailability"},
		{"Bob", "Override"},
	}

	// Record assignments with different decision reasons
	for i, tc := range testCases {
		testDate := date.AddDate(0, 0, i) // Use a different date for each test case
		assignment, err := tracker.RecordAssignment(tc.parent, testDate, false, tc.decisionReason)
		assert.NoError(t, err)
		assert.Equal(t, tc.parent, assignment.Parent)
		assert.Equal(t, tc.decisionReason, assignment.DecisionReason)
	}

	// Test retrieving assignments and verifying decision reasons
	for i, tc := range testCases {
		testDate := date.AddDate(0, 0, i)
		assignment, err := tracker.GetAssignmentByDate(testDate)
		assert.NoError(t, err)
		assert.NotNil(t, assignment)
		assert.Equal(t, tc.parent, assignment.Parent)
		assert.Equal(t, tc.decisionReason, assignment.DecisionReason)
	}
}

// TestDecisionReasonWithOverride tests that decision reasons are preserved with overrides
func TestDecisionReasonWithOverride(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create initial assignment with a decision reason
	assignment, err := tracker.RecordAssignment("Alice", date, false, "Total Count")
	assert.NoError(t, err)
	assert.Equal(t, "Total Count", assignment.DecisionReason)

	// Override the assignment with a different decision reason
	updatedAssignment, err := tracker.RecordAssignment("Bob", date, true, "Override")
	assert.NoError(t, err)
	assert.Equal(t, "Bob", updatedAssignment.Parent)
	assert.Equal(t, "Override", updatedAssignment.DecisionReason)
	assert.True(t, updatedAssignment.Override)
	// Should still be marked as override
}

// TestDecisionReasonInRange tests retrieving assignments with decision reasons in a date range
func TestDecisionReasonInRange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create a series of assignments with different decision reasons
	decisionReasons := []string{"Total Count", "Recent Count", "Consecutive Limit", "Alternating", "Unavailability"}
	for i, reason := range decisionReasons {
		date := startDate.AddDate(0, 0, i)
		parent := "Alice"
		if i%2 == 1 {
			parent = "Bob"
		}
		_, err := tracker.RecordAssignment(parent, date, false, reason)
		assert.NoError(t, err)
	}

	// Get assignments in range
	rangeStart := startDate
	rangeEnd := startDate.AddDate(0, 0, len(decisionReasons)-1)
	assignments, err := tracker.GetAssignmentsInRange(rangeStart, rangeEnd)
	assert.NoError(t, err)
	assert.Len(t, assignments, len(decisionReasons))

	// Verify decision reasons are preserved
	for i, assignment := range assignments {
		assert.Equal(t, decisionReasons[i], assignment.DecisionReason)
	}
}

// TestDecisionReasonWithGoogleCalendarID tests that decision reasons work with Google Calendar IDs
func TestDecisionReasonWithGoogleCalendarID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	eventID := "google_event_123"

	// Create assignment with decision reason
	assignment, err := tracker.RecordAssignment("Alice", date, false, "Total Count")
	assert.NoError(t, err)

	// Set Google Calendar event ID separately
	err = tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, eventID)
	assert.NoError(t, err)

	// Get updated assignment
	assignment, err = tracker.GetAssignmentByID(assignment.ID)
	assert.NoError(t, err)
	assert.Equal(t, eventID, assignment.GoogleCalendarEventID)
	assert.Equal(t, "Total Count", assignment.DecisionReason)

	// Get assignment by Google Calendar event ID
	retrievedAssignment, err := tracker.GetAssignmentByGoogleCalendarEventID(eventID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedAssignment)
	assert.Equal(t, "Total Count", retrievedAssignment.DecisionReason)

	// Update Google Calendar event ID and verify decision reason is preserved
	newEventID := "google_event_456"
	err = tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, newEventID)
	assert.NoError(t, err)

	updatedAssignment, err := tracker.GetAssignmentByID(assignment.ID)
	assert.NoError(t, err)
	assert.Equal(t, newEventID, updatedAssignment.GoogleCalendarEventID)
	assert.Equal(t, "Total Count", updatedAssignment.DecisionReason)
}
