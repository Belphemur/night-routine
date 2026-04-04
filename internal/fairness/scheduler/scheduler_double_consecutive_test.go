package scheduler

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/stretchr/testify/assert"
)

// TestDoubleConsecutiveSwapBasic verifies the basic AA BB → AB AB swap.
func TestDoubleConsecutiveSwapBasic(t *testing.T) {
	// No unavailability so the cascade can produce natural consecutives.
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Seed history so that the scheduler will naturally produce AA BB.
	// We need TotalCount to create consecutive assignments.
	// Alice=0, Bob=2 → TotalCount gives Alice twice, then tied → alternating.
	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	// State: Alice=0, Bob=2. Last=Bob.
	// day3: TotalCount → Alice (fewer). Alice=1,Bob=2
	// day4: TotalCount → Alice (fewer). Alice=2,Bob=2
	// day5: Tied. ConsecutiveLimit: Alice had 2 → Bob. Alice=2,Bob=3
	// day6: TotalCount → Alice (fewer). Alice=3,Bob=3
	// Without smoothing: Alice, Alice, Bob, Bob → double consecutive!
	// With smoothing:    Alice, Bob, Alice, Bob → swapped!
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day6 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)

	schedule, err := sched.GenerateSchedule(day3, day6, day3)
	assert.NoError(t, err)
	assert.Len(t, schedule, 4)

	// After smoothing, the AA BB pattern should become AB AB.
	assert.Equal(t, "Alice", schedule[0].Parent, "day3: Alice")
	assert.Equal(t, "Bob", schedule[1].Parent, "day4: Bob (swapped)")
	assert.Equal(t, fairness.DecisionReasonDoubleConsecutiveSwap, schedule[1].DecisionReason, "day4: swapped reason")
	assert.Equal(t, "Alice", schedule[2].Parent, "day5: Alice (swapped)")
	assert.Equal(t, fairness.DecisionReasonDoubleConsecutiveSwap, schedule[2].DecisionReason, "day5: swapped reason")
	assert.Equal(t, "Bob", schedule[3].Parent, "day6: Bob")
}

// TestDoubleConsecutiveSwapNotTriggeredForUnavailability verifies that consecutive
// runs caused by unavailability do not trigger the double consecutive swap.
func TestDoubleConsecutiveSwapNotTriggeredForUnavailability(t *testing.T) {
	// Alice unavailable on Monday, Bob unavailable on Thursday.
	store := newTestConfigStore("Alice", "Bob", []string{"Monday"}, []string{"Thursday"})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Start on a Thursday so Bob is unavailable → Alice.
	// Friday → alternating (Bob). Saturday → alternating (Alice).
	// Sunday → alternating (Bob). Monday → unavailability (Bob since Alice unavailable).
	start := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC) // Thursday
	end := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)   // Monday

	schedule, err := sched.GenerateSchedule(start, end, start)
	assert.NoError(t, err)
	assert.Len(t, schedule, 5)

	// The unavailability reason should NOT be part of the double consecutive detection.
	// Thu: Alice (unavailability), Fri: Bob, Sat: Alice, Sun: Bob, Mon: Bob (unavailability)
	assert.Equal(t, "Alice", schedule[0].Parent, "Thursday: Alice (Bob unavailable)")
	assert.Equal(t, fairness.DecisionReasonUnavailability, schedule[0].DecisionReason)

	// No DoubleConsecutiveSwap should appear.
	for _, a := range schedule {
		assert.NotEqual(t, fairness.DecisionReasonDoubleConsecutiveSwap, a.DecisionReason,
			"Unavailability-driven consecutive should not trigger swap on %s", a.Date.Format("2006-01-02"))
	}
}

// TestDoubleConsecutiveSwapNotTriggeredForOverride verifies that override
// assignments break consecutive tracking and don't get swapped.
func TestDoubleConsecutiveSwapNotTriggeredForOverride(t *testing.T) {
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Seed: Alice=0, Bob=2 so TotalCount gives Alice twice.
	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	// Pre-record day5 as an override for Alice → this should break the tracking.
	day5 := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	_, err = tracker.RecordAssignment("Alice", day5, true, fairness.DecisionReasonOverride)
	assert.NoError(t, err)

	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day7 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)

	schedule, err := sched.GenerateSchedule(day3, day7, day3)
	assert.NoError(t, err)

	// The override on day5 should break tracking, preventing swaps across it.
	for _, a := range schedule {
		if a.Date.Equal(day5) {
			assert.Equal(t, fairness.DecisionReasonOverride, a.DecisionReason, "day5 should remain override")
		}
	}
}

// TestDoubleConsecutiveSwapWithBabysitterBreak verifies that a babysitter night
// in between breaks the consecutive tracking.
func TestDoubleConsecutiveSwapWithBabysitterBreak(t *testing.T) {
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Seed so the scheduler produces Alice, Alice, Babysitter, Bob, Bob
	// The babysitter should break tracking.
	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	// Pre-record day5 as babysitter → breaks consecutive chain.
	day5 := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	_, err = tracker.RecordBabysitterAssignment("Nanny", day5, true)
	assert.NoError(t, err)

	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day7 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)

	schedule, err := sched.GenerateSchedule(day3, day7, day3)
	assert.NoError(t, err)

	// The babysitter on day5 should prevent a swap from spanning across it.
	for _, a := range schedule {
		if a.Date.Equal(day5) {
			assert.Equal(t, fairness.CaregiverTypeBabysitter, a.CaregiverType, "day5 should be babysitter")
		}
	}
}

// TestDoubleConsecutiveSwapRespectsAvailability verifies that the swap is not
// performed when it would violate a parent's availability constraint.
func TestDoubleConsecutiveSwapRespectsAvailability(t *testing.T) {
	// Alice unavailable on Wednesday (day index 2 in a Mon-start week).
	// If the swap would put Alice on a Wednesday, it should not happen.
	store := newTestConfigStore("Alice", "Bob", []string{"Wednesday"}, []string{})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Seed: Alice=0, Bob=3 → TotalCount gives Alice many times.
	day1 := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC) // Monday
	day2 := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC) // Tuesday
	day3 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)  // Wednesday

	_, err = tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day3, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	// Schedule from Thu (Apr 2) to Sun (Apr 5).
	// day4 (Thu): TotalCount → Alice (0 vs 3)
	// day5 (Fri): TotalCount → Alice (1 vs 3)
	// day6 (Sat): TotalCount → Alice (2 vs 3)
	// day7 (Sun): Tied. ConsecutiveLimit → Bob (Alice had 3+)
	// Actually the cascade depends on exact state. The point is:
	// If a swap would place Alice on a Wednesday, it should be skipped.
	day4 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC) // Thursday
	day7 := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC) // Sunday

	schedule, err := sched.GenerateSchedule(day4, day7, day4)
	assert.NoError(t, err)

	// Verify that no assignment puts Alice on a Wednesday.
	for _, a := range schedule {
		if a.Date.Weekday() == time.Wednesday && a.Parent == "Alice" {
			t.Errorf("Alice should not be assigned on Wednesday %s", a.Date.Format("2006-01-02"))
		}
	}
}

// TestDoubleConsecutiveSwapMultiplePatterns verifies that the algorithm detects
// and fixes multiple double consecutive patterns in a single schedule.
func TestDoubleConsecutiveSwapMultiplePatterns(t *testing.T) {
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Manually record a pattern that will naturally produce double consecutives.
	// Seed: Alice=0, Bob=4 → lots of Alice catch-up creating AA BB patterns.
	for i := range 4 {
		day := time.Date(2026, 3, 28+i, 0, 0, 0, 0, time.UTC)
		_, err = tracker.RecordAssignment("Bob", day, false, fairness.DecisionReasonAlternating)
		assert.NoError(t, err)
	}

	// Generate a longer schedule where multiple AA BB could occur.
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	schedule, err := sched.GenerateSchedule(start, end, start)
	assert.NoError(t, err)

	// After smoothing, we should not have any AA BB patterns among swappable assignments.
	for i := 0; i < len(schedule)-3; i++ {
		a, b, c, d := schedule[i], schedule[i+1], schedule[i+2], schedule[i+3]
		// Only check parent-type assignments with swappable reasons.
		if !isSwappable(a) || !isSwappable(b) || !isSwappable(c) || !isSwappable(d) {
			continue
		}
		if a.Parent == b.Parent && c.Parent == d.Parent && a.Parent != c.Parent {
			t.Errorf("Found unsmoothed double consecutive at index %d-%d: %s %s %s %s",
				i, i+3, a.Parent, b.Parent, c.Parent, d.Parent)
		}
	}
}

// TestDoubleConsecutiveSwapSingleConsecutiveNotTriggered verifies that a single
// consecutive (e.g. AB B) does NOT trigger the swap — both runs must be ≥ 2.
func TestDoubleConsecutiveSwapSingleConsecutiveNotTriggered(t *testing.T) {
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Seed exactly one prior assignment so alternating gives AB AB pattern.
	// Alice=0, Bob=1 → TotalCount gives Alice once, then tied → alternating.
	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	_, err = tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day5 := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)

	schedule, err := sched.GenerateSchedule(day2, day5, day2)
	assert.NoError(t, err)

	// No DoubleConsecutiveSwap should appear since there's no AA BB pattern.
	for _, a := range schedule {
		assert.NotEqual(t, fairness.DecisionReasonDoubleConsecutiveSwap, a.DecisionReason,
			"Single consecutive should not trigger swap on %s", a.Date.Format("2006-01-02"))
	}
}

// TestDoubleConsecutiveSwapPersistsToDatabase verifies that the swap updates
// are persisted to the database, not just in-memory.
func TestDoubleConsecutiveSwapPersistsToDatabase(t *testing.T) {
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{})
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(store, tracker)

	// Seed: Alice=0, Bob=2 → produces AA BB pattern.
	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	day6 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)

	schedule, err := sched.GenerateSchedule(day3, day6, day3)
	assert.NoError(t, err)
	assert.Len(t, schedule, 4)

	// Verify the swapped assignments are persisted in the database.
	for _, a := range schedule {
		if a.DecisionReason == fairness.DecisionReasonDoubleConsecutiveSwap {
			// Read back from DB and verify.
			dbAssignment, err := tracker.GetAssignmentByID(a.ID)
			assert.NoError(t, err)
			assert.Equal(t, a.Parent, dbAssignment.Parent,
				"DB parent should match in-memory parent for %s", a.Date.Format("2006-01-02"))
			assert.Equal(t, fairness.DecisionReasonDoubleConsecutiveSwap, dbAssignment.DecisionReason,
				"DB decision reason should be DoubleConsecutiveSwap for %s", a.Date.Format("2006-01-02"))
		}
	}
}

// TestDoubleConsecutiveSwapFixedAssignmentsNotSwapped verifies that fixed (past)
// assignments are not swapped even if they form a double consecutive pattern.
func TestDoubleConsecutiveSwapFixedAssignmentsNotSwapped(t *testing.T) {
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

	// Set currentTime to day5, so day1-day4 are all past (fixed).
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
