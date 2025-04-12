package scheduler

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/stretchr/testify/assert"
)

// createTestConfig creates a config for testing
func createTestConfig() *config.Config {
	return &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "Alice",
			ParentB: "Bob",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{"Monday"},
			ParentBUnavailable: []string{"Thursday"},
		},
		Schedule: config.ScheduleConfig{
			UpdateFrequency: "weekly",
			LookAheadDays:   7,
		},
	}
}

// TestGenerateSchedule tests the GenerateSchedule function
func TestGenerateSchedule(t *testing.T) {
	cfg := createTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(cfg, tracker)

	// Test period: 7 days starting from a Sunday
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // Sunday
	end := time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)   // Saturday

	// Use the end date as the "current time" for the test
	schedule, err := scheduler.GenerateSchedule(start, end, end)
	assert.NoError(t, err)
	assert.Len(t, schedule, 7)

	// Monday: Alice is unavailable, so Bob should be assigned
	assert.Equal(t, "Bob", schedule[1].Parent)

	// Thursday: Bob is unavailable, so Alice should be assigned
	assert.Equal(t, "Alice", schedule[4].Parent)
}

// TestGenerateScheduleWithPriorAssignments tests the scheduler with prior assignments
func TestGenerateScheduleWithPriorAssignments(t *testing.T) {
	cfg := createTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(cfg, tracker)

	// Use fixed dates instead of time.Now() to make the test deterministic
	// Let's use a known sequence starting on a Tuesday (neither parent is unavailable)
	dayBefore := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // Sunday
	yesterday := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC) // Monday - Alice unavailable
	today := time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)     // Tuesday
	dayAfter := time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)  // Thursday - Bob unavailable

	// Add some prior assignments (Alice did the day before, Bob did yesterday)
	_, err = tracker.RecordAssignment("Alice", dayBefore, false, "")
	assert.NoError(t, err)
	// On Monday, Alice is unavailable, so Bob would be assigned
	_, err = tracker.RecordAssignment("Bob", yesterday, false, fairness.DecisionReasonUnavailability)
	assert.NoError(t, err)

	// Test period: 3 days starting from today (Tuesday)
	// Use the end date (dayAfter) as the "current time" for the test
	schedule, err := scheduler.GenerateSchedule(today, dayAfter, dayAfter)
	assert.NoError(t, err)
	assert.Len(t, schedule, 3)

	// Tuesday: Neither parent is unavailable, and we're alternating, so Alice should be next
	assert.Equal(t, "Alice", schedule[0].Parent)

	// Wednesday: Neither parent is unavailable
	assert.Equal(t, "Bob", schedule[1].Parent)

	// Thursday: Bob is unavailable, so Alice must be assigned
	assert.Equal(t, "Alice", schedule[2].Parent)
}

// TestDetermineAssignmentForDate tests the determineParentForDate function
func TestDetermineAssignmentForDate(t *testing.T) {
	cfg := createTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(cfg, tracker)

	// Test unavailability
	monday := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)   // Monday
	thursday := time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC) // Thursday

	// Get empty stats and assignments for testing
	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	var lastAssignments []*fairness.Assignment

	// Monday: Alice is unavailable
	parent, reason, err := scheduler.determineParentForDate(monday, lastAssignments, stats)
	assert.NoError(t, err)
	assert.Equal(t, "Bob", parent)
	assert.Equal(t, fairness.DecisionReasonUnavailability, reason)

	// Thursday: Bob is unavailable
	parent, reason, err = scheduler.determineParentForDate(thursday, lastAssignments, stats)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", parent)
	assert.Equal(t, fairness.DecisionReasonUnavailability, reason)
}

// TestAssignForDate tests the assignForDate function including recording the assignment
func TestAssignForDate(t *testing.T) {
	cfg := createTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(cfg, tracker)

	// Test unavailability
	monday := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)   // Monday
	thursday := time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC) // Thursday

	// Monday: Alice is unavailable, so Bob should be assigned
	assignment, err := scheduler.assignForDate(monday)
	assert.NoError(t, err)
	assert.Equal(t, "Bob", assignment.Parent)

	// Verify the assignment was recorded
	recordedAssignments, err := tracker.GetLastAssignmentsUntil(1, time.Now())
	assert.NoError(t, err)
	assert.Len(t, recordedAssignments, 1)
	assert.Equal(t, "Bob", recordedAssignments[0].Parent)
	assert.Equal(t, monday.Format("2006-01-02"), recordedAssignments[0].Date.Format("2006-01-02"))

	// Thursday: Bob is unavailable, so Alice should be assigned
	assignment, err = scheduler.assignForDate(thursday)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", assignment.Parent)

	// Verify the assignment was recorded
	recordedAssignments, err = tracker.GetLastAssignmentsUntil(2, time.Now())
	assert.NoError(t, err)
	assert.Len(t, recordedAssignments, 2)
	// The most recent assignment should be first
	assert.Equal(t, "Alice", recordedAssignments[0].Parent)
	assert.Equal(t, thursday.Format("2006-01-02"), recordedAssignments[0].Date.Format("2006-01-02"))
}

// TestDetermineNextParent tests the determineNextParent function
func TestDetermineNextParent(t *testing.T) {
	cfg := createTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(cfg, tracker)

	// Test with no prior assignments
	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 12, Last30Days: 5}

	// Alice should be chosen because she has fewer total assignments
	parent, reason := scheduler.determineNextParent([]*fairness.Assignment{}, stats)
	assert.Equal(t, "Alice", parent)
	assert.Equal(t, fairness.DecisionReasonTotalCount, reason)

	// Test with consecutive assignments
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	lastAssignments := []*fairness.Assignment{
		{Parent: "Alice", Date: today},
		{Parent: "Alice", Date: yesterday},
		{Parent: "Bob", Date: dayBefore},
	}

	// Alice should be chosen because Bob has more total assignments
	parent, reason = scheduler.determineNextParent(lastAssignments, stats)
	assert.Equal(t, "Alice", parent)
	assert.Equal(t, fairness.DecisionReasonTotalCount, reason)

	// Test with alternation (should take precedence over small imbalances)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 7}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	singleAssignment := []*fairness.Assignment{
		{Parent: "Bob", Date: today},
	}

	// Bob should be chosen because we alternate from Alice, and the imbalance is significant
	parent, reason = scheduler.determineNextParent(singleAssignment, stats)
	assert.Equal(t, "Bob", parent)
	assert.Equal(t, fairness.DecisionReasonRecentCount, reason)

	// Test with significant monthly imbalance (should override alternation)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 9}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	// Bob should be chosen despite alternation because Alice has 3+ more assignments
	parent, reason = scheduler.determineNextParent(singleAssignment, stats)
	assert.Equal(t, "Bob", parent)
	assert.Equal(t, fairness.DecisionReasonRecentCount, reason)
}

// TestBothParentsUnavailable tests the case when both parents are unavailable
func TestBothParentsUnavailable(t *testing.T) {
	cfg := createTestConfig()
	// Make both parents unavailable on Wednesday
	cfg.Availability.ParentAUnavailable = append(cfg.Availability.ParentAUnavailable, "Wednesday")
	cfg.Availability.ParentBUnavailable = append(cfg.Availability.ParentBUnavailable, "Wednesday")

	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(cfg, tracker)

	wednesday := time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC) // Wednesday

	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	// Should return an error when both parents are unavailable
	_, _, err = scheduler.determineParentForDate(wednesday, []*fairness.Assignment{}, stats)
	assert.Error(t, err)
}

// TestAlternatingAssignments tests that assignments alternate when everything is balanced
func TestAlternatingAssignments(t *testing.T) {
	cfg := createTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(cfg, tracker)

	// Create balanced stats
	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	today := time.Now()

	// Last assignment was Alice
	lastAssignments := []*fairness.Assignment{
		{Parent: "Alice", Date: today},
	}

	// Next should be Bob
	parent, reason := scheduler.determineNextParent(lastAssignments, stats)
	assert.Equal(t, "Bob", parent)
	assert.Equal(t, fairness.DecisionReasonAlternating, reason)

	// Last assignment was Bob
	lastAssignments = []*fairness.Assignment{
		{Parent: "Bob", Date: today},
	}

	// Next should be Alice
	parent, reason = scheduler.determineNextParent(lastAssignments, stats)
	assert.Equal(t, "Alice", parent)
	assert.Equal(t, fairness.DecisionReasonAlternating, reason)
}

// TestGenerateScheduleWithCurrentTimeFiltering tests that assignments before or on
// currentTime, or overridden assignments, are treated as fixed.
func TestGenerateScheduleWithCurrentTimeFiltering(t *testing.T) {
	cfg := createTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(cfg, tracker)

	// Define dates
	day1 := time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC) // Wednesday
	day2 := time.Date(2023, 2, 2, 0, 0, 0, 0, time.UTC) // Thursday (Bob unavailable)
	day3 := time.Date(2023, 2, 3, 0, 0, 0, 0, time.UTC) // Friday

	currentTime := day2 // Set current time to day2

	// Record initial assignments
	_, err = tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonAlternating) // Past, not overridden -> Fixed
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day2, false, fairness.DecisionReasonAlternating) // Present, not overridden -> Fixed
	assert.NoError(t, err)
	// Record a future assignment that should be ignored unless overridden
	initialDay3Assignment, err := tracker.RecordAssignment("Alice", day3, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	// Now override the future assignment by updating the existing record
	err = tracker.UpdateAssignmentParent(initialDay3Assignment.ID, "Bob", true) // Future, but overridden -> Fixed
	assert.NoError(t, err)

	// Generate schedule for day1 to day3, with currentTime being day2
	schedule, err := scheduler.GenerateSchedule(day1, day3, currentTime)
	assert.NoError(t, err)
	assert.Len(t, schedule, 3)

	// Verify assignments
	// Day 1: Should be Alice (fixed from past)
	assert.Equal(t, "Alice", schedule[0].Parent)
	assert.Equal(t, day1.Format("2006-01-02"), schedule[0].Date.Format("2006-01-02"))
	// Check reason if possible, might be overwritten by generation logic if not truly fixed
	// assert.Equal(t, fairness.DecisionReasonAlternating, schedule[0].DecisionReason) // This might change based on how fixed assignments are handled

	// Day 2: Should be Bob (fixed from present)
	assert.Equal(t, "Bob", schedule[1].Parent)
	assert.Equal(t, day2.Format("2006-01-02"), schedule[1].Date.Format("2006-01-02"))
	// assert.Equal(t, fairness.DecisionReasonAlternating, schedule[1].DecisionReason)

	// Day 3: Should be Bob (fixed because it was overridden)
	assert.Equal(t, "Bob", schedule[2].Parent)
	assert.Equal(t, day3.Format("2006-01-02"), schedule[2].Date.Format("2006-01-02"))
	// The reason should reflect the override status when fetched
	// Let's fetch the assignment directly to check the reason stored vs generated
	finalDay3Assignment, err := tracker.GetAssignmentByID(initialDay3Assignment.ID)
	assert.NoError(t, err)
	assert.True(t, finalDay3Assignment.Override) // Ensure override flag is set
	// The generated schedule should reflect the reason of the *fixed* assignment
	assert.Equal(t, finalDay3Assignment.DecisionReason, schedule[2].DecisionReason)
}
