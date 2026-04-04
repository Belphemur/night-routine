package scheduler

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/stretchr/testify/assert"
)

// --- Unit tests for the doubleConsecutiveTracker.observe mechanism ---

// makeAssignment is a test helper that builds a scheduler.Assignment
// with the minimum fields needed for double-consecutive tracking.
func makeAssignment(id int64, parent string, date time.Time, reason fairness.DecisionReason, caregiverType fairness.CaregiverType) *Assignment {
	return &Assignment{
		ID:             id,
		Parent:         parent,
		Date:           date,
		DecisionReason: reason,
		CaregiverType:  caregiverType,
	}
}

func noUnavailabilityCfg() *scheduleConfig {
	return &scheduleConfig{
		parentA:            "Alice",
		parentB:            "Bob",
		parentAUnavailable: []string{},
		parentBUnavailable: []string{},
	}
}

func testTracker(t *testing.T) fairness.TrackerInterface {
	t.Helper()
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	return tracker
}

// TestObserveDetectsDoubleConsecutive directly tests the tracker with an
// AA BB schedule slice and verifies it swaps the boundary assignments.
func TestObserveDetectsDoubleConsecutive(t *testing.T) {
	tracker := testTracker(t)
	cfg := noUnavailabilityCfg()

	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)

	// Pre-record all 4 assignments in the DB so upsert works.
	a1, _ := tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	a2, _ := tracker.RecordAssignment("Alice", day2, false, fairness.DecisionReasonTotalCount)
	a3, _ := tracker.RecordAssignment("Bob", day3, false, fairness.DecisionReasonConsecutiveLimit)
	a4, _ := tracker.RecordAssignment("Bob", day4, false, fairness.DecisionReasonAlternating)

	schedule := []*Assignment{
		makeAssignment(a1.ID, "Alice", day1, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a2.ID, "Alice", day2, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a3.ID, "Bob", day3, fairness.DecisionReasonConsecutiveLimit, fairness.CaregiverTypeParent),
		makeAssignment(a4.ID, "Bob", day4, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))

	// Feed assignments one by one.
	swapped := false
	for i := range schedule {
		if dc.observe(schedule, i, cfg, tracker) {
			swapped = true
		}
	}

	assert.True(t, swapped, "Should detect and swap the AA BB pattern")

	// After swap: Alice, Bob, Alice, Bob  (boundary positions 1 and 2 swapped).
	assert.Equal(t, "Alice", schedule[0].Parent, "day1 unchanged")
	assert.Equal(t, "Bob", schedule[1].Parent, "day2 swapped to Bob")
	assert.Equal(t, fairness.DecisionReasonDoubleConsecutiveSwap, schedule[1].DecisionReason)
	assert.Equal(t, "Alice", schedule[2].Parent, "day3 swapped to Alice")
	assert.Equal(t, fairness.DecisionReasonDoubleConsecutiveSwap, schedule[2].DecisionReason)
	assert.Equal(t, "Bob", schedule[3].Parent, "day4 unchanged")

	// Verify the DB was updated via the upsert.
	dbA2, _ := tracker.GetAssignmentByDate(day2)
	assert.Equal(t, "Bob", dbA2.Parent)
	assert.Equal(t, fairness.DecisionReasonDoubleConsecutiveSwap, dbA2.DecisionReason)

	dbA3, _ := tracker.GetAssignmentByDate(day3)
	assert.Equal(t, "Alice", dbA3.Parent)
	assert.Equal(t, fairness.DecisionReasonDoubleConsecutiveSwap, dbA3.DecisionReason)
}

// TestObserveReversedPattern verifies BB AA is also detected and swapped.
func TestObserveReversedPattern(t *testing.T) {
	tracker := testTracker(t)
	cfg := noUnavailabilityCfg()

	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)

	a1, _ := tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonTotalCount)
	a2, _ := tracker.RecordAssignment("Bob", day2, false, fairness.DecisionReasonTotalCount)
	a3, _ := tracker.RecordAssignment("Alice", day3, false, fairness.DecisionReasonConsecutiveLimit)
	a4, _ := tracker.RecordAssignment("Alice", day4, false, fairness.DecisionReasonAlternating)

	schedule := []*Assignment{
		makeAssignment(a1.ID, "Bob", day1, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a2.ID, "Bob", day2, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a3.ID, "Alice", day3, fairness.DecisionReasonConsecutiveLimit, fairness.CaregiverTypeParent),
		makeAssignment(a4.ID, "Alice", day4, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))
	swapped := false
	for i := range schedule {
		if dc.observe(schedule, i, cfg, tracker) {
			swapped = true
		}
	}

	assert.True(t, swapped, "Should detect BB AA pattern")
	assert.Equal(t, "Bob", schedule[0].Parent)
	assert.Equal(t, "Alice", schedule[1].Parent, "swapped to Alice")
	assert.Equal(t, "Bob", schedule[2].Parent, "swapped to Bob")
	assert.Equal(t, "Alice", schedule[3].Parent)
}

// TestObserveNoSwapForSingleConsecutive verifies that A B B does NOT trigger
// a swap because the first run has only 1 assignment.
func TestObserveNoSwapForSingleConsecutive(t *testing.T) {
	tracker := testTracker(t)
	cfg := noUnavailabilityCfg()

	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)

	a1, _ := tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	a2, _ := tracker.RecordAssignment("Bob", day2, false, fairness.DecisionReasonAlternating)
	a3, _ := tracker.RecordAssignment("Bob", day3, false, fairness.DecisionReasonAlternating)

	schedule := []*Assignment{
		makeAssignment(a1.ID, "Alice", day1, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a2.ID, "Bob", day2, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
		makeAssignment(a3.ID, "Bob", day3, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))
	for i := range schedule {
		assert.False(t, dc.observe(schedule, i, cfg, tracker), "No swap for single consecutive at index %d", i)
	}
}

// TestObserveBabysitterBreaksTracking verifies that a babysitter assignment
// in the middle of a potential double consecutive breaks tracking.
func TestObserveBabysitterBreaksTracking(t *testing.T) {
	tracker := testTracker(t)
	cfg := noUnavailabilityCfg()

	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)
	day5 := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)

	a1, _ := tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	a2, _ := tracker.RecordAssignment("Alice", day2, false, fairness.DecisionReasonTotalCount)
	a3, _ := tracker.RecordBabysitterAssignment("Nanny", day3, true)
	a4, _ := tracker.RecordAssignment("Bob", day4, false, fairness.DecisionReasonAlternating)
	a5, _ := tracker.RecordAssignment("Bob", day5, false, fairness.DecisionReasonAlternating)

	schedule := []*Assignment{
		makeAssignment(a1.ID, "Alice", day1, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a2.ID, "Alice", day2, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a3.ID, "Nanny", day3, fairness.DecisionReasonOverride, fairness.CaregiverTypeBabysitter),
		makeAssignment(a4.ID, "Bob", day4, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
		makeAssignment(a5.ID, "Bob", day5, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))
	for i := range schedule {
		assert.False(t, dc.observe(schedule, i, cfg, tracker), "Babysitter breaks tracking at index %d", i)
	}
}

// TestObserveOverrideBreaksTracking verifies that an override assignment
// in the middle breaks consecutive tracking.
func TestObserveOverrideBreaksTracking(t *testing.T) {
	tracker := testTracker(t)
	cfg := noUnavailabilityCfg()

	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)
	day5 := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)

	a1, _ := tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	a2, _ := tracker.RecordAssignment("Alice", day2, false, fairness.DecisionReasonTotalCount)
	a3, _ := tracker.RecordAssignment("Alice", day3, true, fairness.DecisionReasonOverride)
	a4, _ := tracker.RecordAssignment("Bob", day4, false, fairness.DecisionReasonAlternating)
	a5, _ := tracker.RecordAssignment("Bob", day5, false, fairness.DecisionReasonAlternating)

	schedule := []*Assignment{
		makeAssignment(a1.ID, "Alice", day1, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a2.ID, "Alice", day2, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a3.ID, "Alice", day3, fairness.DecisionReasonOverride, fairness.CaregiverTypeParent),
		makeAssignment(a4.ID, "Bob", day4, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
		makeAssignment(a5.ID, "Bob", day5, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))
	for i := range schedule {
		assert.False(t, dc.observe(schedule, i, cfg, tracker), "Override breaks tracking at index %d", i)
	}
}

// TestObserveUnavailabilityBreaksTracking verifies that an unavailability
// assignment breaks consecutive tracking.
func TestObserveUnavailabilityBreaksTracking(t *testing.T) {
	tracker := testTracker(t)
	cfg := noUnavailabilityCfg()

	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)
	day5 := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)

	a1, _ := tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	a2, _ := tracker.RecordAssignment("Alice", day2, false, fairness.DecisionReasonTotalCount)
	a3, _ := tracker.RecordAssignment("Bob", day3, false, fairness.DecisionReasonUnavailability)
	a4, _ := tracker.RecordAssignment("Bob", day4, false, fairness.DecisionReasonAlternating)
	a5, _ := tracker.RecordAssignment("Bob", day5, false, fairness.DecisionReasonAlternating)

	schedule := []*Assignment{
		makeAssignment(a1.ID, "Alice", day1, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a2.ID, "Alice", day2, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a3.ID, "Bob", day3, fairness.DecisionReasonUnavailability, fairness.CaregiverTypeParent),
		makeAssignment(a4.ID, "Bob", day4, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
		makeAssignment(a5.ID, "Bob", day5, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))
	for i := range schedule {
		assert.False(t, dc.observe(schedule, i, cfg, tracker), "Unavailability breaks tracking at index %d", i)
	}
}

// TestObserveRespectsAvailabilityConstraints verifies that a swap is skipped
// when it would violate a parent's day-of-week unavailability.
func TestObserveRespectsAvailabilityConstraints(t *testing.T) {
	tracker := testTracker(t)
	// Alice is unavailable on Thursdays.
	cfg := &scheduleConfig{
		parentA:            "Alice",
		parentB:            "Bob",
		parentAUnavailable: []string{"Thursday"},
		parentBUnavailable: []string{},
	}

	// Build AA BB where position 2 (day3) is a Thursday.
	// Swapping would put Alice on Thursday — must be skipped.
	day1 := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC) // Tuesday
	day2 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)  // Wednesday
	day3 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)  // Thursday
	day4 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)  // Friday

	a1, _ := tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	a2, _ := tracker.RecordAssignment("Alice", day2, false, fairness.DecisionReasonTotalCount)
	a3, _ := tracker.RecordAssignment("Bob", day3, false, fairness.DecisionReasonConsecutiveLimit)
	a4, _ := tracker.RecordAssignment("Bob", day4, false, fairness.DecisionReasonAlternating)

	schedule := []*Assignment{
		makeAssignment(a1.ID, "Alice", day1, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a2.ID, "Alice", day2, fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent),
		makeAssignment(a3.ID, "Bob", day3, fairness.DecisionReasonConsecutiveLimit, fairness.CaregiverTypeParent),
		makeAssignment(a4.ID, "Bob", day4, fairness.DecisionReasonAlternating, fairness.CaregiverTypeParent),
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))
	for i := range schedule {
		assert.False(t, dc.observe(schedule, i, cfg, tracker),
			"Swap should be skipped due to availability at index %d", i)
	}

	// Assignments should be unchanged.
	assert.Equal(t, "Alice", schedule[1].Parent, "day2 should remain Alice")
	assert.Equal(t, "Bob", schedule[2].Parent, "day3 should remain Bob")
}

// TestObserveMultiplePatterns verifies that multiple AA BB patterns in a
// single schedule are each detected and swapped independently.
func TestObserveMultiplePatterns(t *testing.T) {
	tracker := testTracker(t)
	cfg := noUnavailabilityCfg()

	days := make([]time.Time, 8)
	for i := range days {
		days[i] = time.Date(2026, 4, 1+i, 0, 0, 0, 0, time.UTC)
	}

	// Record: AA BB AA BB
	assignments := make([]*fairness.Assignment, 8)
	parents := []string{"Alice", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Bob"}
	for i, p := range parents {
		assignments[i], _ = tracker.RecordAssignment(p, days[i], false, fairness.DecisionReasonTotalCount)
	}

	schedule := make([]*Assignment, 8)
	for i := range schedule {
		schedule[i] = makeAssignment(assignments[i].ID, parents[i], days[i],
			fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent)
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))
	swapCount := 0
	for i := range schedule {
		if dc.observe(schedule, i, cfg, tracker) {
			swapCount++
		}
	}

	assert.Equal(t, 2, swapCount, "Should detect two AA BB patterns")
	// After two swaps: AB AB AB AB.
	for i, a := range schedule {
		if i%2 == 0 {
			assert.Equal(t, "Alice", a.Parent, "Even index %d should be Alice", i)
		} else {
			assert.Equal(t, "Bob", a.Parent, "Odd index %d should be Bob", i)
		}
	}
}

// TestObserveLongerRuns verifies that runs > 2 are also detected.
// AAA BBB → boundary swap at endIdx=2 and startIdx=3 → AA B A BB.
func TestObserveLongerRuns(t *testing.T) {
	tracker := testTracker(t)
	cfg := noUnavailabilityCfg()

	days := make([]time.Time, 6)
	for i := range days {
		days[i] = time.Date(2026, 4, 1+i, 0, 0, 0, 0, time.UTC)
	}

	parents := []string{"Alice", "Alice", "Alice", "Bob", "Bob", "Bob"}
	assignments := make([]*fairness.Assignment, 6)
	for i, p := range parents {
		assignments[i], _ = tracker.RecordAssignment(p, days[i], false, fairness.DecisionReasonTotalCount)
	}

	schedule := make([]*Assignment, 6)
	for i := range schedule {
		schedule[i] = makeAssignment(assignments[i].ID, parents[i], days[i],
			fairness.DecisionReasonTotalCount, fairness.CaregiverTypeParent)
	}

	dc := newDoubleConsecutiveTracker(logging.GetLogger("test"))
	swapped := false
	for i := range schedule {
		if dc.observe(schedule, i, cfg, tracker) {
			swapped = true
		}
	}

	assert.True(t, swapped, "Should detect AAA BBB pattern")
	// Boundary swap: [2]=Alice→Bob, [3]=Bob→Alice
	assert.Equal(t, "Alice", schedule[0].Parent)
	assert.Equal(t, "Alice", schedule[1].Parent)
	assert.Equal(t, "Bob", schedule[2].Parent, "boundary swapped")
	assert.Equal(t, "Alice", schedule[3].Parent, "boundary swapped")
	assert.Equal(t, "Bob", schedule[4].Parent)
	assert.Equal(t, "Bob", schedule[5].Parent)
}

// --- Integration tests through GenerateSchedule ---

// TestGenerateScheduleNoDoubleConsecutiveInOutput verifies that GenerateSchedule
// never produces AA BB patterns among swappable assignments in various scenarios.
func TestGenerateScheduleNoDoubleConsecutiveInOutput(t *testing.T) {
	tests := []struct {
		name               string
		parentAUnavailable []string
		parentBUnavailable []string
		seedAlice          int
		seedBob            int
		days               int
	}{
		{"balanced no unavailability", []string{}, []string{}, 0, 0, 14},
		{"alice behind", []string{}, []string{}, 0, 5, 14},
		{"bob behind", []string{}, []string{}, 5, 0, 14},
		{"large imbalance", []string{}, []string{}, 0, 10, 30},
		{"with unavailability", []string{"Monday"}, []string{"Thursday"}, 0, 3, 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestConfigStore("Alice", "Bob", tt.parentAUnavailable, tt.parentBUnavailable)
			db, cleanup := setupTestDB(t)
			defer cleanup()

			tracker, err := fairness.New(db)
			assert.NoError(t, err)
			sched := New(store, tracker)

			// Seed prior assignments to create the desired imbalance.
			seedDay := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
			for i := range tt.seedAlice {
				_, err = tracker.RecordAssignment("Alice", seedDay.AddDate(0, 0, i), false, fairness.DecisionReasonAlternating)
				assert.NoError(t, err)
			}
			for i := range tt.seedBob {
				_, err = tracker.RecordAssignment("Bob", seedDay.AddDate(0, 0, tt.seedAlice+i), false, fairness.DecisionReasonAlternating)
				assert.NoError(t, err)
			}

			start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
			end := start.AddDate(0, 0, tt.days-1)

			schedule, err := sched.GenerateSchedule(start, end, start)
			assert.NoError(t, err)
			assert.Len(t, schedule, tt.days)

			// Verify no AA BB pattern among swappable assignments.
			for i := 0; i < len(schedule)-3; i++ {
				a, b, c, d := schedule[i], schedule[i+1], schedule[i+2], schedule[i+3]
				if !isSwappable(a) || !isSwappable(b) || !isSwappable(c) || !isSwappable(d) {
					continue
				}
				if a.Parent == b.Parent && c.Parent == d.Parent && a.Parent != c.Parent {
					t.Errorf("Found unsmoothed double consecutive at index %d-%d: %s(%s) %s(%s) %s(%s) %s(%s)",
						i, i+3,
						a.Parent, a.DecisionReason,
						b.Parent, b.DecisionReason,
						c.Parent, c.DecisionReason,
						d.Parent, d.DecisionReason)
				}
			}
		})
	}
}

// TestGenerateScheduleFixedAssignmentsNotSwapped verifies that past (fixed)
// assignments are never modified by the double consecutive tracker.
func TestGenerateScheduleFixedAssignmentsNotSwapped(t *testing.T) {
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Record past assignments forming AA BB pattern.
	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Alice", day2, false, fairness.DecisionReasonTotalCount)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day3, false, fairness.DecisionReasonConsecutiveLimit)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day4, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	// Set currentTime to day5 so day1-day4 are all past (fixed).
	day5 := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	day7 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)

	schedule, err := sched.GenerateSchedule(day1, day7, day5)
	assert.NoError(t, err)

	// Past assignments (day1-day4) should retain their original reasons.
	for _, a := range schedule {
		if a.Date.Before(day5) {
			assert.NotEqual(t, fairness.DecisionReasonDoubleConsecutiveSwap, a.DecisionReason,
				"Past assignment on %s should not be swapped", a.Date.Format("2006-01-02"))
		}
	}
}

// TestDoubleConsecutiveSwapDecisionReasonString verifies the string representation.
func TestDoubleConsecutiveSwapDecisionReasonString(t *testing.T) {
	assert.Equal(t, "Double Consecutive Swap", fairness.DecisionReasonDoubleConsecutiveSwap.String())
}
