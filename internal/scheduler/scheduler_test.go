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
	tracker := fairness.NewMockTracker()
	scheduler := New(cfg, tracker)

	// Test period: 7 days starting from a Sunday
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // Sunday
	end := time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)   // Saturday

	schedule, err := scheduler.GenerateSchedule(start, end)
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
	tracker := fairness.NewMockTracker()
	scheduler := New(cfg, tracker)

	// Add some prior assignments (Alice did the last 2 days)
	yesterday := time.Now().AddDate(0, 0, -1)
	dayBefore := time.Now().AddDate(0, 0, -2)
	tracker.AddAssignment("Alice", yesterday)
	tracker.AddAssignment("Alice", dayBefore)

	// Test period: 3 days starting from today
	start := time.Now()
	end := time.Now().AddDate(0, 0, 2)

	schedule, err := scheduler.GenerateSchedule(start, end)
	assert.NoError(t, err)
	assert.Len(t, schedule, 3)

	// First day should be Alice because bob is unavailable
	assert.Equal(t, "Alice", schedule[0].Parent)
	assert.Equal(t, "Bob", schedule[1].Parent)
	assert.Equal(t, "Bob", schedule[2].Parent)

}

// TestDetermineAssignmentForDate tests the determineAssignmentForDate function
func TestDetermineAssignmentForDate(t *testing.T) {
	cfg := createTestConfig()
	tracker := fairness.NewMockTracker()
	scheduler := New(cfg, tracker)

	// Test unavailability
	monday := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)   // Monday
	thursday := time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC) // Thursday

	// Get empty stats and assignments for testing
	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	lastAssignments := []fairness.Assignment{}

	// Monday: Alice is unavailable
	assignment, err := scheduler.determineAssignmentForDate(monday, lastAssignments, stats)
	assert.NoError(t, err)
	assert.Equal(t, "Bob", assignment.Parent)

	// Thursday: Bob is unavailable
	assignment, err = scheduler.determineAssignmentForDate(thursday, lastAssignments, stats)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", assignment.Parent)
}

// TestAssignForDate tests the assignForDate function including recording the assignment
func TestAssignForDate(t *testing.T) {
	cfg := createTestConfig()
	tracker := fairness.NewMockTracker()
	scheduler := New(cfg, tracker)

	// Test unavailability
	monday := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)   // Monday
	thursday := time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC) // Thursday

	// Get empty stats and assignments for testing
	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	// Monday: Alice is unavailable, so Bob should be assigned
	assignment, err := scheduler.assignForDate(monday)
	assert.NoError(t, err)
	assert.Equal(t, "Bob", assignment.Parent)

	// Verify the assignment was recorded
	recordedAssignments, err := tracker.GetLastAssignments(1)
	assert.NoError(t, err)
	assert.Len(t, recordedAssignments, 1)
	assert.Equal(t, "Bob", recordedAssignments[0].Parent)
	assert.Equal(t, monday.Format("2006-01-02"), recordedAssignments[0].Date.Format("2006-01-02"))

	// Thursday: Bob is unavailable, so Alice should be assigned
	assignment, err = scheduler.assignForDate(thursday)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", assignment.Parent)

	// Verify the assignment was recorded
	recordedAssignments, err = tracker.GetLastAssignments(2)
	assert.NoError(t, err)
	assert.Len(t, recordedAssignments, 2)
	// The most recent assignment should be first
	assert.Equal(t, "Alice", recordedAssignments[0].Parent)
	assert.Equal(t, thursday.Format("2006-01-02"), recordedAssignments[0].Date.Format("2006-01-02"))
}

// TestDetermineNextParent tests the determineNextParent function
func TestDetermineNextParent(t *testing.T) {
	cfg := createTestConfig()
	tracker := fairness.NewMockTracker()
	scheduler := New(cfg, tracker)

	// Test with no prior assignments
	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 12, Last30Days: 5}

	// Alice should be chosen because she has fewer total assignments
	parent := scheduler.determineNextParent([]fairness.Assignment{}, stats)
	assert.Equal(t, "Alice", parent)

	// Test with consecutive assignments
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	lastAssignments := []fairness.Assignment{
		{Parent: "Alice", Date: today},
		{Parent: "Alice", Date: yesterday},
		{Parent: "Bob", Date: dayBefore},
	}

	// Alice should be chosen because Bob has more total assignments
	parent = scheduler.determineNextParent(lastAssignments, stats)
	assert.Equal(t, "Alice", parent)

	// Test with alternation (should take precedence over small imbalances)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 7}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	singleAssignment := []fairness.Assignment{
		{Parent: "Bob", Date: today},
	}

	// Bob should be chosen because we alternate from Alice, and the imbalance is significant
	parent = scheduler.determineNextParent(singleAssignment, stats)
	assert.Equal(t, "Bob", parent)

	// Test with significant monthly imbalance (should override alternation)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 9}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	// Bob should be chosen despite alternation because Alice has 3+ more assignments
	parent = scheduler.determineNextParent(singleAssignment, stats)
	assert.Equal(t, "Bob", parent)
}

// TestBothParentsUnavailable tests the case when both parents are unavailable
func TestBothParentsUnavailable(t *testing.T) {
	cfg := createTestConfig()
	// Make both parents unavailable on Wednesday
	cfg.Availability.ParentAUnavailable = append(cfg.Availability.ParentAUnavailable, "Wednesday")
	cfg.Availability.ParentBUnavailable = append(cfg.Availability.ParentBUnavailable, "Wednesday")

	tracker := fairness.NewMockTracker()
	scheduler := New(cfg, tracker)

	wednesday := time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC) // Wednesday

	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	// Should return an error when both parents are unavailable
	_, err := scheduler.determineAssignmentForDate(wednesday, []fairness.Assignment{}, stats)
	assert.Error(t, err)
}

// TestAlternatingAssignments tests that assignments alternate when everything is balanced
func TestAlternatingAssignments(t *testing.T) {
	cfg := createTestConfig()
	tracker := fairness.NewMockTracker()
	scheduler := New(cfg, tracker)

	// Create balanced stats
	stats := make(map[string]fairness.Stats)
	stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
	stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

	today := time.Now()

	// Last assignment was Alice
	lastAssignments := []fairness.Assignment{
		{Parent: "Alice", Date: today},
	}

	// Next should be Bob
	parent := scheduler.determineNextParent(lastAssignments, stats)
	assert.Equal(t, "Bob", parent)

	// Last assignment was Bob
	lastAssignments = []fairness.Assignment{
		{Parent: "Bob", Date: today},
	}

	// Next should be Alice
	parent = scheduler.determineNextParent(lastAssignments, stats)
	assert.Equal(t, "Alice", parent)
}
