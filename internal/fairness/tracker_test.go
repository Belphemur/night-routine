package fairness

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/database"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) (*database.DB, func()) {
	// Create a new in-memory database with shared cache and foreign keys enabled
	opts := database.SQLiteOptions{
		Path:        ":memory:",           // Use ":memory:" for in-memory database path
		Mode:        "memory",             // Explicitly set mode to memory
		Cache:       database.CacheShared, // Use shared cache
		ForeignKeys: true,                 // Enable foreign keys via PRAGMA
		// Use other defaults from NewDefaultOptions if needed, or keep minimal
		Journal:     database.JournalMemory, // Memory journal is suitable for in-memory DB
		BusyTimeout: 5000,                   // Default busy timeout
	}
	db, err := database.New(opts)
	assert.NoError(t, err)

	// Run migrations
	err = db.MigrateDatabase()
	assert.NoError(t, err)

	// Return the database and a cleanup function
	return db, func() {
		err := db.Close()
		assert.NoError(t, err)
	}
}

// TestRecordAssignment tests the RecordAssignment method
func TestRecordAssignment(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Test recording a new assignment
	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	assignment, err := tracker.RecordAssignment("Alice", date, false, "Total Count")
	assert.NoError(t, err)
	assert.NotNil(t, assignment)
	assert.Equal(t, "Alice", assignment.Parent)
	assert.Equal(t, date.Format("2006-01-02"), assignment.Date.Format("2006-01-02"))
	assert.False(t, assignment.Override)

	// Test recording another assignment for the same date (should update)
	assignment2, err := tracker.RecordAssignment("Bob", date, false, "Alternating")
	assert.NoError(t, err)
	assert.NotNil(t, assignment2)
	assert.Equal(t, "Bob", assignment2.Parent)
	assert.Equal(t, assignment.ID, assignment2.ID) // Should be the same assignment (updated)
}

// TestGetLastAssignmentsUntil tests the GetLastAssignmentsUntil method
func TestGetLastAssignmentsUntil(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Create some test assignments
	dates := []time.Time{
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
	}

	parents := []string{"Alice", "Bob", "Alice"}

	for i, date := range dates {
		_, err := tracker.RecordAssignment(parents[i], date, false, "Alternating")
		assert.NoError(t, err)
	}

	// Test getting last 2 assignments until January 4th
	until := time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)
	assignments, err := tracker.GetLastAssignmentsUntil(2, until)
	assert.NoError(t, err)
	assert.Len(t, assignments, 2)
	assert.Equal(t, "Alice", assignments[0].Parent) // Most recent first
	assert.Equal(t, "Bob", assignments[1].Parent)
}

// TestGetParentStatsUntil tests the GetParentStatsUntil method
func TestGetParentStatsUntil(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	now := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	// Create assignments spanning more than 30 days
	assignments := []struct {
		parent string
		date   time.Time
	}{
		{"Alice", now.AddDate(0, 0, -40)}, // Old
		{"Alice", now.AddDate(0, 0, -20)}, // Within 30 days
		{"Alice", now.AddDate(0, 0, -10)}, // Within 30 days
		{"Bob", now.AddDate(0, 0, -35)},   // Old
		{"Bob", now.AddDate(0, 0, -15)},   // Within 30 days
	}

	for _, a := range assignments {
		_, err := tracker.RecordAssignment(a.parent, a.date, false, "Total Count")
		assert.NoError(t, err)
	}

	// Get stats until now
	stats, err := tracker.GetParentStatsUntil(now)
	assert.NoError(t, err)

	// Check Alice's stats
	aliceStats := stats["Alice"]
	assert.Equal(t, 3, aliceStats.TotalAssignments)
	assert.Equal(t, 2, aliceStats.Last30Days)

	// Check Bob's stats
	bobStats := stats["Bob"]
	assert.Equal(t, 2, bobStats.TotalAssignments)
	assert.Equal(t, 1, bobStats.Last30Days)
}

// TestGetAssignmentByDate tests the GetAssignmentByDate method
func TestGetAssignmentByDate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test getting non-existent assignment
	assignment, err := tracker.GetAssignmentByDate(date)
	assert.NoError(t, err)
	assert.Nil(t, assignment)

	// Create an assignment
	created, err := tracker.RecordAssignment("Alice", date, false, "Total Count")
	assert.NoError(t, err)

	// Get the assignment
	assignment, err = tracker.GetAssignmentByDate(date)
	assert.NoError(t, err)
	assert.NotNil(t, assignment)
	assert.Equal(t, created.ID, assignment.ID)
	assert.Equal(t, "Alice", assignment.Parent)
}

// TestAssignmentWithOverride tests override functionality
func TestAssignmentWithOverride(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create initial assignment
	assignment, err := tracker.RecordAssignment("Alice", date, false, "Total Count")
	assert.NoError(t, err)
	assert.False(t, assignment.Override)
	assert.Equal(t, DecisionReason("Total Count"), assignment.DecisionReason)

	// Override the assignment
	err = tracker.UpdateAssignmentParent(assignment.ID, "Bob", true)
	assert.NoError(t, err)

	// Verify the override
	updated, err := tracker.GetAssignmentByDate(date)
	assert.NoError(t, err)
	assert.True(t, updated.Override)
	assert.Equal(t, "Bob", updated.Parent)
	assert.Equal(t, DecisionReasonOverride, updated.DecisionReason, "Decision reason should be set to Override when overriding")

	// With our simplified method, overrides can be changed
	assignment, err = tracker.RecordAssignment("Alice", date, false, "Total Count")
	assert.NoError(t, err)
	assert.Equal(t, "Alice", assignment.Parent) // Should now be Alice (overrides can be changed)
	assert.False(t, assignment.Override)        // Override flag is updated
	assert.Equal(t, DecisionReason("Total Count"), assignment.DecisionReason, "Decision reason should be updated when override is removed")
}

// TestUpdateAssignmentParentWithOverride tests the UpdateAssignmentParent method with override
func TestUpdateAssignmentParentWithOverride(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create initial assignment with a specific decision reason
	initialReason := DecisionReason("Alternating")
	assignment, err := tracker.RecordAssignment("Alice", date, false, initialReason)
	assert.NoError(t, err)
	assert.Equal(t, initialReason, assignment.DecisionReason)

	// Test case 1: Update with override=true
	err = tracker.UpdateAssignmentParent(assignment.ID, "Bob", true)
	assert.NoError(t, err)

	// Verify decision reason is set to Override
	updated, err := tracker.GetAssignmentByDate(date)
	assert.NoError(t, err)
	assert.Equal(t, "Bob", updated.Parent)
	assert.True(t, updated.Override)
	assert.Equal(t, DecisionReasonOverride, updated.DecisionReason, "Decision reason should be set to Override when override=true")

	// Test case 2: Update with override=false
	err = tracker.UpdateAssignmentParent(updated.ID, "Charlie", false)
	assert.NoError(t, err)

	// Verify decision reason is not changed when override=false
	updated2, err := tracker.GetAssignmentByDate(date)
	assert.NoError(t, err)
	assert.Equal(t, "Charlie", updated2.Parent)
	assert.False(t, updated2.Override)
	assert.Equal(t, DecisionReasonOverride, updated2.DecisionReason, "Decision reason should not be changed when override=false")

	// Test case 3: Set override=true again with a different parent
	err = tracker.UpdateAssignmentParent(updated2.ID, "David", true)
	assert.NoError(t, err)

	// Verify decision reason is set to Override again
	updated3, err := tracker.GetAssignmentByDate(date)
	assert.NoError(t, err)
	assert.Equal(t, "David", updated3.Parent)
	assert.True(t, updated3.Override)
	assert.Equal(t, DecisionReasonOverride, updated3.DecisionReason, "Decision reason should be set to Override when override=true")
}

// TestGetAssignmentsInRange tests the GetAssignmentsInRange method
func TestGetAssignmentsInRange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Create assignments
	assignments := []struct {
		parent string
		date   time.Time
	}{
		{"Alice", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"Bob", time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)},
		{"Alice", time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)},
		{"Bob", time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)},
	}

	for _, a := range assignments {
		_, err := tracker.RecordAssignment(a.parent, a.date, false, "Alternating")
		assert.NoError(t, err)
	}

	// Test getting assignments in range
	start := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)
	rangeAssignments, err := tracker.GetAssignmentsInRange(start, end)
	assert.NoError(t, err)
	assert.Len(t, rangeAssignments, 2)
	assert.Equal(t, "Bob", rangeAssignments[0].Parent)
	assert.Equal(t, "Alice", rangeAssignments[1].Parent)
}

// TestGoogleCalendarIntegration tests the Google Calendar related methods
func TestGoogleCalendarIntegration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	eventID := "google_event_123"

	// Create assignment
	assignment, err := tracker.RecordAssignment("Alice", date, false, "Override")
	assert.NoError(t, err)

	// Update with Google Calendar event ID
	err = tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, eventID)
	assert.NoError(t, err)

	// Retrieve updated assignment
	assignment, err = tracker.GetAssignmentByID(assignment.ID)
	assert.NoError(t, err)
	assert.Equal(t, eventID, assignment.GoogleCalendarEventID)

	// Get assignment by event ID
	found, err := tracker.GetAssignmentByGoogleCalendarEventID(eventID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, assignment.ID, found.ID)

	// Update event ID
	newEventID := "google_event_456"
	err = tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, newEventID)
	assert.NoError(t, err)

	// Verify update
	updated, err := tracker.GetAssignmentByID(assignment.ID)
	assert.NoError(t, err)
	assert.Equal(t, newEventID, updated.GoogleCalendarEventID)
}
