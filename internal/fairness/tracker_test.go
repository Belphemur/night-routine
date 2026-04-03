package fairness

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/database"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite" // Register modernc sqlite driver
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

func TestGetParentMonthlyStatsForLastNMonths(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Define a fixed "now" for consistent testing.
	testReferenceTime := time.Date(2025, time.May, 15, 10, 0, 0, 0, time.UTC)

	// Helper to create dates relative to testReferenceTime
	monthsAgo := func(m int) time.Time {
		return testReferenceTime.AddDate(0, -m, 0)
	}
	daysAgo := func(d int) time.Time {
		return testReferenceTime.AddDate(0, 0, -d)
	}

	t.Run("No assignments", func(t *testing.T) {
		stats, err := tracker.GetParentMonthlyStatsForLastNMonths(testReferenceTime, 12)
		assert.NoError(t, err)
		assert.Empty(t, stats)
	})

	// --- Setup data for subsequent tests ---
	// Parent A:
	// - 2 assignments 1 month ago (current month - 1)
	// - 1 assignment 3 months ago
	// - 1 assignment 13 months ago (should be excluded for 12 month lookback)
	_, err = tracker.RecordAssignment("Parent A", monthsAgo(1).AddDate(0, 0, -1), false, "Test") // e.g., April 2025
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Parent A", monthsAgo(1).AddDate(0, 0, -2), false, "Test") // e.g., April 2025
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Parent A", monthsAgo(3), false, "Test") // e.g., February 2025
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Parent A", monthsAgo(13), false, "Test") // e.g., April 2024 (too old)
	assert.NoError(t, err)

	// Parent B:
	// - 1 assignment this month (current month)
	// - 3 assignments 11 months ago (just within 12 month lookback)
	_, err = tracker.RecordAssignment("Parent B", daysAgo(5), false, "Test") // e.g., May 2025
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Parent B", monthsAgo(11).AddDate(0, 0, -1), false, "Test") // e.g., June 2024
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Parent B", monthsAgo(11).AddDate(0, 0, -2), false, "Test") // e.g., June 2024
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Parent B", monthsAgo(11).AddDate(0, 0, -3), false, "Test") // e.g., June 2024
	assert.NoError(t, err)

	// Parent C:
	// - 1 assignment 12 months ago (should be excluded as start of range is Nth month ago, not N+1)
	//   The logic `now.AddDate(0, -(nMonths - 1), 0)` means for 12 months, it goes back 11 months
	//   to the *start* of that month. So, 12 full months ago is outside.
	//   Example: If now is May 2025, 12 months lookback includes May 2024.
	//   `monthsAgo(11)` is June 2024. `monthsAgo(12)` is May 2024.
	//   The first day of range is `time.Date(startDateRange.Year(), startDateRange.Month(), 1, ...)`
	//   If nMonths = 12, startDateRange = now - 11 months.
	//   If now = May 15, 2025, startDateRange = June 15, 2024. firstDayOfRange = June 1, 2024.
	//   So, data from May 2024 should be excluded.
	_, err = tracker.RecordAssignment("Parent C", monthsAgo(12), false, "Test") // e.g., May 2024 (should be included if logic is inclusive of 12th month)
	assert.NoError(t, err)
	// Let's add one for Parent C that *is* included (11 months ago)
	_, err = tracker.RecordAssignment("Parent C", monthsAgo(11).AddDate(0, 0, -5), false, "Test") // e.g. June 2024
	assert.NoError(t, err)

	t.Run("With assignments within 12 months", func(t *testing.T) {
		stats, err := tracker.GetParentMonthlyStatsForLastNMonths(testReferenceTime, 12)
		assert.NoError(t, err)
		// Expected:
		// Parent A: monthsAgo(1) -> 2, monthsAgo(3) -> 1
		// Parent B: daysAgo(5) (current month) -> 1, monthsAgo(11) -> 3
		// Parent C: monthsAgo(11) -> 1 (monthsAgo(12) should be included by the query logic)

		// Create a map for easier assertion
		resultsMap := make(map[string]map[string]int) // Parent -> MonthYear -> Count
		for _, s := range stats {
			if _, ok := resultsMap[s.ParentName]; !ok {
				resultsMap[s.ParentName] = make(map[string]int)
			}
			resultsMap[s.ParentName][s.MonthYear] = s.Count
		}

		// Assertions for Parent A
		month1AgoStr := monthsAgo(1).Format("2006-01")
		month3AgoStr := monthsAgo(3).Format("2006-01")
		assert.Equal(t, 2, resultsMap["Parent A"][month1AgoStr], "Parent A, 1 month ago")
		assert.Equal(t, 1, resultsMap["Parent A"][month3AgoStr], "Parent A, 3 months ago")
		_, thirteenMonthsAgoExists := resultsMap["Parent A"][monthsAgo(13).Format("2006-01")]
		assert.False(t, thirteenMonthsAgoExists, "Parent A, 13 months ago should not exist")

		// Assertions for Parent B
		currentMonthStr := testReferenceTime.Format("2006-01") // May 2025
		month11AgoStr := monthsAgo(11).Format("2006-01")       // June 2024
		assert.Equal(t, 1, resultsMap["Parent B"][currentMonthStr], "Parent B, current month")
		assert.Equal(t, 3, resultsMap["Parent B"][month11AgoStr], "Parent B, 11 months ago")

		// Assertions for Parent C
		// The query `assignment_date >= ?` where ? is firstDayOfRange (e.g., 2024-06-01 for a May 2025 run with 12 months)
		// So, monthsAgo(12) which is May 2024, should NOT be included.
		// monthsAgo(11) which is June 2024, SHOULD be included.
		month12AgoStr := monthsAgo(12).Format("2006-01")

		_, twelveMonthsAgoExists := resultsMap["Parent C"][month12AgoStr]
		assert.False(t, twelveMonthsAgoExists, "Parent C, 12 months ago (e.g. May 2024) should NOT be included")
		assert.Equal(t, 1, resultsMap["Parent C"][month11AgoStr], "Parent C, 11 months ago (e.g. June 2024)")

		// Total number of stat rows
		// Parent A: 2 rows (month1ago, month3ago)
		// Parent B: 2 rows (currentMonth, month11ago)
		// Parent C: 1 row (month11ago)
		// Total = 5
		assert.Len(t, stats, 5, "Total number of stat rows")
	})

	t.Run("Lookback for 1 month", func(t *testing.T) {
		stats, err := tracker.GetParentMonthlyStatsForLastNMonths(testReferenceTime, 1)
		assert.NoError(t, err)

		resultsMap := make(map[string]map[string]int)
		for _, s := range stats {
			if _, ok := resultsMap[s.ParentName]; !ok {
				resultsMap[s.ParentName] = make(map[string]int)
			}
			resultsMap[s.ParentName][s.MonthYear] = s.Count
		}
		currentMonthStr := testReferenceTime.Format("2006-01")

		// Parent A: No assignments in current month
		// Parent B: 1 assignment in current month
		// Parent C: No assignments in current month
		assert.Equal(t, 1, resultsMap["Parent B"][currentMonthStr], "Parent B, current month for 1 month lookback")
		_, parentAExists := resultsMap["Parent A"]
		assert.False(t, parentAExists, "Parent A should not have stats for 1 month lookback")
		_, parentCExists := resultsMap["Parent C"]
		assert.False(t, parentCExists, "Parent C should not have stats for 1 month lookback")
		assert.Len(t, stats, 1)
	})

	t.Run("Lookback for 2 months", func(t *testing.T) {
		// This should include current month and (current month - 1)
		stats, err := tracker.GetParentMonthlyStatsForLastNMonths(testReferenceTime, 2)
		assert.NoError(t, err)

		resultsMap := make(map[string]map[string]int)
		for _, s := range stats {
			if _, ok := resultsMap[s.ParentName]; !ok {
				resultsMap[s.ParentName] = make(map[string]int)
			}
			resultsMap[s.ParentName][s.MonthYear] = s.Count
		}
		currentMonthStr := testReferenceTime.Format("2006-01") // May 2025
		month1AgoStr := monthsAgo(1).Format("2006-01")         // April 2025

		// Parent A: 2 assignments 1 month ago
		// Parent B: 1 assignment current month
		// Parent C: No assignments in these 2 months
		assert.Equal(t, 2, resultsMap["Parent A"][month1AgoStr], "Parent A, 1 month ago for 2 months lookback")
		assert.Equal(t, 1, resultsMap["Parent B"][currentMonthStr], "Parent B, current month for 2 months lookback")
		_, parentCExists := resultsMap["Parent C"]
		assert.False(t, parentCExists, "Parent C should not have stats for 2 months lookback")
		assert.Len(t, stats, 2) // Parent A (1 row), Parent B (1 row)
	})
}

// TestSaveAndGetAssignmentDetails tests saving and retrieving assignment details
func TestSaveAndGetAssignmentDetails(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Create an assignment first
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	assignment, err := tracker.RecordAssignment("Alice", date, false, DecisionReasonTotalCount)
	assert.NoError(t, err)
	assert.NotNil(t, assignment)

	// Create stats for both parents
	statsA := Stats{
		TotalAssignments: 5,
		Last30Days:       3,
	}
	statsB := Stats{
		TotalAssignments: 7,
		Last30Days:       4,
	}

	// Save assignment details
	err = tracker.SaveAssignmentDetails(assignment.ID, date, "Alice", statsA, "Bob", statsB)
	assert.NoError(t, err)

	// Retrieve assignment details
	details, err := tracker.GetAssignmentDetails(assignment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, details)

	// Verify the details
	assert.Equal(t, assignment.ID, details.AssignmentID)
	assert.Equal(t, date.Format("2006-01-02"), details.CalculationDate.Format("2006-01-02"))
	assert.Equal(t, "Alice", details.ParentAName)
	assert.Equal(t, 5, details.ParentATotalCount)
	assert.Equal(t, 3, details.ParentALast30Days)
	assert.Equal(t, "Bob", details.ParentBName)
	assert.Equal(t, 7, details.ParentBTotalCount)
	assert.Equal(t, 4, details.ParentBLast30Days)
}

// TestGetAssignmentDetailsNotFound tests retrieving non-existent assignment details
func TestGetAssignmentDetailsNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Try to get details for non-existent assignment
	details, err := tracker.GetAssignmentDetails(99999)
	assert.NoError(t, err)
	assert.Nil(t, details)
}

// TestAssignmentDetailsCascadeDelete tests that assignment details are deleted when assignment is deleted
func TestAssignmentDetailsCascadeDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Create an assignment
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	assignment, err := tracker.RecordAssignment("Alice", date, false, DecisionReasonTotalCount)
	assert.NoError(t, err)

	// Save assignment details
	statsA := Stats{TotalAssignments: 5, Last30Days: 3}
	statsB := Stats{TotalAssignments: 7, Last30Days: 4}
	err = tracker.SaveAssignmentDetails(assignment.ID, date, "Alice", statsA, "Bob", statsB)
	assert.NoError(t, err)

	// Verify details exist
	details, err := tracker.GetAssignmentDetails(assignment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, details)

	// Delete the assignment
	_, err = db.Conn().Exec("DELETE FROM assignments WHERE id = ?", assignment.ID)
	assert.NoError(t, err)

	// Verify details are also deleted (cascade)
	details, err = tracker.GetAssignmentDetails(assignment.ID)
	assert.NoError(t, err)
	assert.Nil(t, details)
}

func TestUpdateAssignmentToBabysitter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	assignment, err := tracker.RecordAssignment("Alice", date, false, DecisionReasonAlternating)
	assert.NoError(t, err)

	err = tracker.UpdateAssignmentToBabysitter(assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	updated, err := tracker.GetAssignmentByID(assignment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, CaregiverTypeBabysitter, updated.CaregiverType)
	assert.Equal(t, "Dawn", updated.Parent)
	assert.True(t, updated.Override)
	assert.Equal(t, DecisionReasonOverride, updated.DecisionReason)

	err = tracker.UpdateAssignmentParent(assignment.ID, "Bob", true)
	assert.NoError(t, err)

	updated, err = tracker.GetAssignmentByID(assignment.ID)
	assert.NoError(t, err)
	assert.Equal(t, CaregiverTypeParent, updated.CaregiverType)
	assert.Equal(t, "Bob", updated.Parent)
}

func TestGetParentStatsUntil_BabysitterShiftCountsForBothParents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	until := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Alice", until.AddDate(0, 0, -10), false, DecisionReasonTotalCount)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", until.AddDate(0, 0, -8), false, DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", until.AddDate(0, 0, -5), true)
	assert.NoError(t, err)

	stats, err := tracker.GetParentStatsUntil(until)
	assert.NoError(t, err)
	// Babysitter shift adds +1 to both parents: Alice=1+1=2, Bob=1+1=2
	assert.Equal(t, 2, stats["Alice"].TotalAssignments)
	assert.Equal(t, 2, stats["Alice"].Last30Days)
	assert.Equal(t, 2, stats["Bob"].TotalAssignments)
	assert.Equal(t, 2, stats["Bob"].Last30Days)
	_, exists := stats["Dawn"]
	assert.False(t, exists, "babysitter should not appear as a separate parent in stats")
}

func TestUnlockAssignment_ClearsBabysitterState(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	assignment, err := tracker.RecordAssignment("Alice", date, true, DecisionReasonOverride)
	assert.NoError(t, err)

	err = tracker.UpdateAssignmentToBabysitter(assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	err = tracker.UnlockAssignment(assignment.ID)
	assert.NoError(t, err)

	updated, err := tracker.GetAssignmentByID(assignment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.False(t, updated.Override)
	assert.Equal(t, CaregiverTypeParent, updated.CaregiverType)
	// parent_name retains the babysitter name after unlock (the scheduler
	// will overwrite it when it regenerates the schedule in the handler).
	assert.Equal(t, "Dawn", updated.Parent)
	assert.Equal(t, DecisionReason(""), updated.DecisionReason)
}

// TestSaveAssignmentDetailsUpsert tests that SaveAssignmentDetails can update existing records
func TestSaveAssignmentDetailsUpsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Create an assignment
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	assignment, err := tracker.RecordAssignment("Alice", date, false, DecisionReasonTotalCount)
	assert.NoError(t, err)

	// Save initial assignment details
	statsA := Stats{TotalAssignments: 5, Last30Days: 3}
	statsB := Stats{TotalAssignments: 7, Last30Days: 4}
	err = tracker.SaveAssignmentDetails(assignment.ID, date, "Alice", statsA, "Bob", statsB)
	assert.NoError(t, err)

	// Retrieve and verify initial details
	details, err := tracker.GetAssignmentDetails(assignment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, details)
	assert.Equal(t, 5, details.ParentATotalCount)
	assert.Equal(t, 7, details.ParentBTotalCount)

	// Update the details with new stats (simulating schedule recalculation)
	statsA2 := Stats{TotalAssignments: 10, Last30Days: 6}
	statsB2 := Stats{TotalAssignments: 12, Last30Days: 8}
	err = tracker.SaveAssignmentDetails(assignment.ID, date, "Alice", statsA2, "Bob", statsB2)
	assert.NoError(t, err)

	// Retrieve and verify updated details
	updatedDetails, err := tracker.GetAssignmentDetails(assignment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedDetails)
	assert.Equal(t, assignment.ID, updatedDetails.AssignmentID)
	assert.Equal(t, 10, updatedDetails.ParentATotalCount)
	assert.Equal(t, 12, updatedDetails.ParentBTotalCount)
	assert.Equal(t, 6, updatedDetails.ParentALast30Days)
	assert.Equal(t, 8, updatedDetails.ParentBLast30Days)

	// Verify there's still only one record (not duplicate)
	var count int
	err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignment_details WHERE assignment_id = ?", assignment.ID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Should only have one record after upsert")
}

func TestRecordBabysitterAssignment(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	t.Run("Insert new babysitter assignment", func(t *testing.T) {
		date := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
		assignment, err := tracker.RecordBabysitterAssignment("Dawn", date, true)
		assert.NoError(t, err)
		assert.NotNil(t, assignment)
		assert.Equal(t, "Dawn", assignment.Parent)
		assert.Equal(t, CaregiverTypeBabysitter, assignment.CaregiverType)
		assert.True(t, assignment.Override)
		assert.Equal(t, DecisionReasonOverride, assignment.DecisionReason)
		assert.Equal(t, date.Format("2006-01-02"), assignment.Date.Format("2006-01-02"))
	})

	t.Run("Upsert overwrites existing parent assignment", func(t *testing.T) {
		date := time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)
		// First record a parent
		original, err := tracker.RecordAssignment("Alice", date, false, DecisionReasonTotalCount)
		assert.NoError(t, err)
		assert.Equal(t, CaregiverTypeParent, original.CaregiverType)

		// Now record babysitter on same date
		updated, err := tracker.RecordBabysitterAssignment("Dawn", date, true)
		assert.NoError(t, err)
		assert.Equal(t, original.ID, updated.ID, "should be the same row via upsert")
		assert.Equal(t, CaregiverTypeBabysitter, updated.CaregiverType)
		assert.Equal(t, "Dawn", updated.Parent)
	})
}

func TestGetLastAssignmentDate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	t.Run("No assignments returns zero time", func(t *testing.T) {
		date, err := tracker.GetLastAssignmentDate()
		assert.NoError(t, err)
		assert.True(t, date.IsZero())
	})

	t.Run("Returns latest assignment date", func(t *testing.T) {
		_, err := tracker.RecordAssignment("Alice", time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), false, DecisionReasonTotalCount)
		assert.NoError(t, err)
		_, err = tracker.RecordAssignment("Bob", time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC), false, DecisionReasonAlternating)
		assert.NoError(t, err)

		date, err := tracker.GetLastAssignmentDate()
		assert.NoError(t, err)
		assert.Equal(t, "2025-03-15", date.Format("2006-01-02"))
	})
}

func TestGetBabysitterMonthlyStatsForLastNMonths(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Seed data: some parent + babysitter assignments across months
	refTime := time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC)
	monthsAgo := func(n int) time.Time {
		return time.Date(refTime.Year(), refTime.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -n, 5)
	}

	// Parent assignments (should NOT appear in babysitter stats)
	_, err = tracker.RecordAssignment("Alice", monthsAgo(0), false, DecisionReasonTotalCount)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", monthsAgo(1), false, DecisionReasonAlternating)
	assert.NoError(t, err)

	// Babysitter assignments
	_, err = tracker.RecordBabysitterAssignment("Dawn", monthsAgo(0).AddDate(0, 0, 1), true)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", monthsAgo(0).AddDate(0, 0, 2), true)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Eve", monthsAgo(2), true)
	assert.NoError(t, err)

	t.Run("Returns only babysitter stats", func(t *testing.T) {
		stats, err := tracker.GetBabysitterMonthlyStatsForLastNMonths(refTime, 12)
		assert.NoError(t, err)

		resultsMap := make(map[string]map[string]int)
		for _, s := range stats {
			if _, ok := resultsMap[s.ParentName]; !ok {
				resultsMap[s.ParentName] = make(map[string]int)
			}
			resultsMap[s.ParentName][s.MonthYear] = s.Count
		}

		currentMonth := refTime.Format("2006-01")
		assert.Equal(t, 2, resultsMap["Dawn"][currentMonth], "Dawn should have 2 babysitter assignments in current month")
		_, aliceExists := resultsMap["Alice"]
		assert.False(t, aliceExists, "parent-only assignments should not appear")
		_, bobExists := resultsMap["Bob"]
		assert.False(t, bobExists, "parent-only assignments should not appear")
	})

	t.Run("Lookback for 1 month excludes older entries", func(t *testing.T) {
		stats, err := tracker.GetBabysitterMonthlyStatsForLastNMonths(refTime, 1)
		assert.NoError(t, err)

		for _, s := range stats {
			assert.NotEqual(t, "Eve", s.ParentName, "Eve's assignment is older and should not appear in 1-month lookback")
		}
	})

	t.Run("No babysitter assignments returns empty", func(t *testing.T) {
		// Query a narrow range that only contains parent assignments
		parentOnlyRef := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)
		stats, err := tracker.GetBabysitterMonthlyStatsForLastNMonths(parentOnlyRef, 1)
		assert.NoError(t, err)
		assert.Empty(t, stats)
	})
}

func TestUnlockAssignment_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	err = tracker.UnlockAssignment(99999)
	assert.Error(t, err, "unlocking a nonexistent assignment should fail")
	assert.Contains(t, err.Error(), "assignment not found")
}

// TestGetLastAssignmentsUntil verifies that GetLastAssignmentsUntil returns all
// caregiver types (parents and babysitters) in reverse chronological order.
func TestGetLastAssignmentsUntil(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	day1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Alice", day1, false, DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Sitter", day2, false)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day3, false, DecisionReasonAlternating)
	assert.NoError(t, err)

	// GetLastAssignmentsUntil should return all three (parent + babysitter + parent).
	all, err := tracker.GetLastAssignmentsUntil(5, until)
	assert.NoError(t, err)
	assert.Len(t, all, 3, "should include parent and babysitter assignments")
	assert.Equal(t, "Bob", all[0].Parent)
	assert.Equal(t, CaregiverTypeParent, all[0].CaregiverType)
	assert.Equal(t, CaregiverTypeBabysitter, all[1].CaregiverType)
	assert.Equal(t, "Alice", all[2].Parent)
	assert.Equal(t, CaregiverTypeParent, all[2].CaregiverType)
}
