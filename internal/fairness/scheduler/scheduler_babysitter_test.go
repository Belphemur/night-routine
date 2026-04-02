package scheduler

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/stretchr/testify/assert"
)

// newBabysitterTestConfig creates a config with no unavailability for predictable tests.
func newBabysitterTestConfig() *config.Config {
	return &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "Alice",
			ParentB: "Bob",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{},
			ParentBUnavailable: []string{},
		},
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// 1. Babysitter override is always treated as fixed
// ──────────────────────────────────────────────────────────────────────────────

// TestBabysitterOverrideIsFixed verifies that a babysitter assignment (which
// has override=true) is never recalculated by GenerateSchedule, even when it
// falls on a future day that would normally be recalculated.
func TestBabysitterOverrideIsFixed(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC) // Monday
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	// Generate initial: day1=Alice, day2=Bob, day3=Alice
	initial, err := sched.GenerateSchedule(day1, day3, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 3)

	// Set day2 (future) to babysitter
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Regenerate from day1 — day2 must remain babysitter "Dawn" (fixed override)
	recalc, err := sched.GenerateSchedule(day1, day3, day1)
	assert.NoError(t, err)
	assert.Len(t, recalc, 3)

	assert.Equal(t, "Dawn", recalc[1].Parent, "day2 babysitter override must remain fixed")
	assert.Equal(t, fairness.CaregiverTypeBabysitter, recalc[1].CaregiverType)
}

// ──────────────────────────────────────────────────────────────────────────────
// 2. Babysitter excluded from TotalCount path
// ──────────────────────────────────────────────────────────────────────────────

// TestBabysitterExcludedFromTotalCount verifies that converting a parent
// assignment to babysitter removes it from parent stats, causing the TotalCount
// fairness path to pick the parent who now has fewer assignments.
func TestBabysitterExcludedFromTotalCount(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// day1=Alice, day2=Bob, day3=Alice, day4=Bob
	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)

	initial, err := sched.GenerateSchedule(day1, day4, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 4)
	assert.Equal(t, "Alice", initial[0].Parent)
	assert.Equal(t, "Bob", initial[1].Parent)
	assert.Equal(t, "Alice", initial[2].Parent)
	assert.Equal(t, "Bob", initial[3].Parent)

	// Convert day2 (Bob) to babysitter → parent stats: Alice=1(day1), Bob=0
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Regenerate: day3-day4 recalculate. Stats before day3: Alice=1(day1), Bob=0
	// day3 → Bob (TotalCount, fewer assignments)
	recalc, err := sched.GenerateSchedule(day1, day4, day3)
	assert.NoError(t, err)
	assert.Len(t, recalc, 4)

	assert.Equal(t, "Bob", recalc[2].Parent, "day3 should be Bob (TotalCount: Alice=1, Bob=0)")
	assert.Equal(t, fairness.DecisionReasonTotalCount, recalc[2].DecisionReason)
}

// ──────────────────────────────────────────────────────────────────────────────
// 3. Babysitter excluded from RecentCount path
// ──────────────────────────────────────────────────────────────────────────────

// TestBabysitterExcludedFromRecentCount verifies that with tied total counts
// but differing last-30-day counts (due to babysitter conversion), the
// RecentCount path selects the correct parent.
func TestBabysitterExcludedFromRecentCount(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// Ancient history (>30 days before rDay3): Alice=2, Bob=1
	ancient1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	ancient2 := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	ancient3 := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	_, err = tracker.RecordAssignment("Alice", ancient1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", ancient2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Alice", ancient3, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	// Recent: rDay1=Bob, rDay2=Bob (within 30 days of rDay3)
	rDay1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	rDay2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	rDay3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Bob", rDay1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", rDay2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	// Stats at rDay3: Alice total=2, Bob total=3 → not tied.
	// Convert rDay2 (Bob) to babysitter → Bob total=2, tied with Alice=2
	// Last30 at rDay3: Alice=0, Bob=1(rDay1) → Bob has more recent → Alice wins RecentCount
	rDay2Assignment, err := tracker.GetAssignmentByDate(rDay2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(rDay2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Generate for rDay3 only
	recalc, err := sched.GenerateSchedule(rDay3, rDay3, rDay3)
	assert.NoError(t, err)
	assert.Len(t, recalc, 1)

	assert.Equal(t, "Alice", recalc[0].Parent, "rDay3 should be Alice (RecentCount: Alice=0, Bob=1)")
	assert.Equal(t, fairness.DecisionReasonRecentCount, recalc[0].DecisionReason)
}

// ──────────────────────────────────────────────────────────────────────────────
// 4. Babysitter excluded from ConsecutiveLimit path
// ──────────────────────────────────────────────────────────────────────────────

// TestConsecutiveLimitWithBabysitterGap verifies that the ConsecutiveLimit
// path fires correctly when two same-parent assignments are separated by a
// babysitter day — from the parent-only view they are consecutive.
func TestConsecutiveLimitWithBabysitterGap(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// We need: stats tied (total + last30), last 2 parent assignments = same parent.
	// All within last 30 days so last30 = total for both.
	// Alice=2, Bob=2; parent-only last 2 = Bob, Bob → consecutive limit → Alice
	old1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC) // Within last 30 days of day4
	old2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	_, err = tracker.RecordAssignment("Alice", old1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Alice", old2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC) // Bob
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC) // babysitter
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC) // Bob (override)
	day4 := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC) // recalculate

	_, err = tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", day2, true)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", day3, true, fairness.DecisionReasonOverride)
	assert.NoError(t, err)

	// Stats at day4: Alice=2(old1+old2), Bob=2(day1+day3) → tied total
	// Last30 at day4: all within 30 days → Alice=2, Bob=2 → tied last30
	// Parent-only last assignments: [Bob(day3), Bob(day1), Alice(old2), Alice(old1)]
	// Consecutive: Bob, Bob → count=2 ≥ 2 → ConsecutiveLimit → Alice

	recalc, err := sched.GenerateSchedule(day4, day4, day4)
	assert.NoError(t, err)
	assert.Len(t, recalc, 1)

	assert.Equal(t, "Alice", recalc[0].Parent,
		"day4 should be Alice (ConsecutiveLimit: Bob had 2 consecutive in parent-only view)")
	assert.Equal(t, fairness.DecisionReasonConsecutiveLimit, recalc[0].DecisionReason)
}

// ──────────────────────────────────────────────────────────────────────────────
// 5. Babysitter excluded from Alternating path
// ──────────────────────────────────────────────────────────────────────────────

// TestBabysitterPoisonsAlternatingLogic tests that when parent stats are tied
// and the most recent DB row is a babysitter, the alternating logic correctly
// uses the most recent *parent* assignment instead of the babysitter name.
// Without filtering: "Dawn" != parentB → always picks parentB (Bob) — WRONG.
func TestBabysitterPoisonsAlternatingLogic(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	initial, err := sched.GenerateSchedule(day1, day4, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 4)

	day3Assignment, err := tracker.GetAssignmentByDate(day3)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day3Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Stats at day4: Alice=1(day1), Bob=1(day2) → tied. Alternating from Bob → Alice.
	recalc, err := sched.GenerateSchedule(day1, day4, day4)
	assert.NoError(t, err)
	assert.Len(t, recalc, 4)

	assert.Equal(t, "Alice", recalc[0].Parent, "day1 fixed")
	assert.Equal(t, "Bob", recalc[1].Parent, "day2 fixed")
	assert.Equal(t, "Dawn", recalc[2].Parent, "day3 babysitter fixed")
	assert.Equal(t, fairness.CaregiverTypeBabysitter, recalc[2].CaregiverType)

	assert.Equal(t, "Alice", recalc[3].Parent,
		"day4 should alternate from Bob (last parent), not be poisoned by Dawn")
	assert.Equal(t, fairness.DecisionReasonAlternating, recalc[3].DecisionReason)
}

// TestBabysitterAlternatingFromParentA verifies the alternating path picks
// parentB when the last parent assignment is parentA and a babysitter sits
// in between.
func TestBabysitterAlternatingFromParentA(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// day1=Bob, day2=Alice, day3=babysitter, day4=recalc
	// Stats at day4: Alice=1, Bob=1 → tied. Last parent = Alice(day2) → alternate → Bob
	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Bob", day1, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordAssignment("Alice", day2, false, fairness.DecisionReasonAlternating)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", day3, true)
	assert.NoError(t, err)

	recalc, err := sched.GenerateSchedule(day4, day4, day4)
	assert.NoError(t, err)
	assert.Len(t, recalc, 1)

	assert.Equal(t, "Bob", recalc[0].Parent,
		"day4 should alternate from Alice (last parent) → Bob")
	assert.Equal(t, fairness.DecisionReasonAlternating, recalc[0].DecisionReason)
}

// ──────────────────────────────────────────────────────────────────────────────
// 6. Unlock babysitter and recalculate
// ──────────────────────────────────────────────────────────────────────────────

// TestUnlockBabysitterRecalculatesStartDate tests that unlocking a babysitter
// and regenerating from that date recalculates the assignment even when the
// start date is today or in the past.
func TestUnlockBabysitterRecalculatesStartDate(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC)

	initial, err := sched.GenerateSchedule(day1, day2, day1)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", initial[0].Parent)
	assert.Equal(t, "Bob", initial[1].Parent)

	// Set day2 to babysitter then unlock
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)
	err = tracker.UnlockAssignment(day2Assignment.ID)
	assert.NoError(t, err)

	// Verify DB state after unlock
	unlocked, err := tracker.GetAssignmentByID(day2Assignment.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Dawn", unlocked.Parent, "parent_name retains babysitter name after unlock")
	assert.Equal(t, fairness.CaregiverTypeParent, unlocked.CaregiverType)
	assert.False(t, unlocked.Override)

	// Regenerate from day2 — day2 should recalculate to a real parent
	recalc, err := sched.GenerateSchedule(day2, day2, day2)
	assert.NoError(t, err)
	assert.Len(t, recalc, 1)
	assert.Equal(t, "Bob", recalc[0].Parent, "day2 should be Bob (Alice=1, Bob=0 → TotalCount)")
	assert.NotEqual(t, "Dawn", recalc[0].Parent)
}

// TestUnlockBabysitterRecalculatesMultipleDays tests unlocking a babysitter
// and verifying all subsequent days are recalculated properly.
func TestUnlockBabysitterRecalculatesMultipleDays(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	_ = time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC) // day3
	day4 := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)

	// Initial: Alice, Bob, Alice, Bob
	initial, err := sched.GenerateSchedule(day1, day4, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 4)

	// Set day2 to babysitter, then unlock
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)
	err = tracker.UnlockAssignment(day2Assignment.ID)
	assert.NoError(t, err)

	// Regenerate from day2 onward
	recalc, err := sched.GenerateSchedule(day2, day4, day2)
	assert.NoError(t, err)
	assert.Len(t, recalc, 3)

	// day2 recalculates: Alice=1(day1), Bob=0 → Bob (TotalCount)
	assert.Equal(t, "Bob", recalc[0].Parent, "day2 should be Bob after unlock")
	// day3: Alice=1, Bob=1 → alternate from Bob → Alice
	assert.Equal(t, "Alice", recalc[1].Parent, "day3 should be Alice (alternating)")
	// day4: Alice=2, Bob=1 → Bob (TotalCount)
	assert.Equal(t, "Bob", recalc[2].Parent, "day4 should be Bob (TotalCount)")
}

// ──────────────────────────────────────────────────────────────────────────────
// 7. Babysitter on past day recalculates future
// ──────────────────────────────────────────────────────────────────────────────

// TestBabysitterOnPastDayRecalculatesFutureAssignments is the original
// regression test for: "Setting a babysitter in the past isn't properly
// recalculating future assignments"
func TestBabysitterOnPastDayRecalculatesFutureAssignments(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	initial, err := sched.GenerateSchedule(day1, day4, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 4)

	// Set day2 (yesterday) to babysitter
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Regenerate with currentTime=day3: day1 fixed, day2 babysitter fixed
	recalc, err := sched.GenerateSchedule(day1, day4, day3)
	assert.NoError(t, err)
	assert.Len(t, recalc, 4)

	assert.Equal(t, "Alice", recalc[0].Parent, "day1 fixed")
	assert.Equal(t, "Dawn", recalc[1].Parent, "day2 babysitter fixed")
	assert.Equal(t, fairness.CaregiverTypeBabysitter, recalc[1].CaregiverType)
	assert.Equal(t, "Bob", recalc[2].Parent, "day3: Alice=1, Bob=0 → TotalCount")
	assert.Equal(t, fairness.DecisionReasonTotalCount, recalc[2].DecisionReason)
	assert.Equal(t, "Alice", recalc[3].Parent, "day4: alternating after Bob")
}

// ──────────────────────────────────────────────────────────────────────────────
// 8. Multiple babysitter days
// ──────────────────────────────────────────────────────────────────────────────

// TestMultipleBabysitterDays verifies correct recalculation when multiple days
// in a schedule are converted to babysitter assignments.
func TestMultipleBabysitterDays(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)  // Alice
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)  // Bob → babysitter
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)  // Alice → babysitter
	day4 := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)  // Bob
	day5 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC) // recalculate

	initial, err := sched.GenerateSchedule(day1, day5, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 5)

	// Convert day2 and day3 to babysitters
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	day3Assignment, err := tracker.GetAssignmentByDate(day3)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day3Assignment.ID, "Eve", true)
	assert.NoError(t, err)

	// Regenerate from day4 onward
	// Stats at day4: Alice=1(day1), Bob=0 (day2 now babysitter) → Bob TotalCount
	recalc, err := sched.GenerateSchedule(day1, day5, day4)
	assert.NoError(t, err)
	assert.Len(t, recalc, 5)

	assert.Equal(t, "Alice", recalc[0].Parent, "day1 fixed")
	assert.Equal(t, "Dawn", recalc[1].Parent, "day2 babysitter fixed")
	assert.Equal(t, "Eve", recalc[2].Parent, "day3 babysitter fixed")
	assert.Equal(t, "Bob", recalc[3].Parent, "day4: Alice=1, Bob=0 → TotalCount")
	assert.Equal(t, fairness.DecisionReasonTotalCount, recalc[3].DecisionReason)
	assert.Equal(t, "Alice", recalc[4].Parent, "day5: Alice=1, Bob=1 → alternate from Bob → Alice")
}

// TestConsecutiveBabysitterDays verifies that multiple consecutive babysitter
// days are all treated as fixed and properly excluded from fairness calculations.
func TestConsecutiveBabysitterDays(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)  // Alice
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)  // babysitter
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)  // babysitter
	day4 := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)  // babysitter
	day5 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC) // recalculate

	// Record day1 and set day2-day4 as consecutive babysitter days
	_, err = tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", day2, true)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", day3, true)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", day4, true)
	assert.NoError(t, err)

	// Generate day5: only parent assignment is Alice(day1)
	// Stats: Alice=1, Bob=0 → Bob TotalCount
	recalc, err := sched.GenerateSchedule(day5, day5, day5)
	assert.NoError(t, err)
	assert.Len(t, recalc, 1)

	assert.Equal(t, "Bob", recalc[0].Parent, "day5: Alice=1, Bob=0 → Bob (TotalCount)")
	assert.Equal(t, fairness.DecisionReasonTotalCount, recalc[0].DecisionReason)
}

// ──────────────────────────────────────────────────────────────────────────────
// 9. Babysitter on boundary days (first/last)
// ──────────────────────────────────────────────────────────────────────────────

// TestBabysitterOnFirstDayOfSchedule verifies that a babysitter on the first
// day of the schedule is treated as fixed and doesn't disrupt subsequent
// recalculations.
func TestBabysitterOnFirstDayOfSchedule(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC) // babysitter
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC) // recalculate
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC) // recalculate

	_, err = tracker.RecordBabysitterAssignment("Dawn", day1, true)
	assert.NoError(t, err)

	// Generate day1-day3 with currentTime=day2 (day1 is past + override, fixed)
	recalc, err := sched.GenerateSchedule(day1, day3, day2)
	assert.NoError(t, err)
	assert.Len(t, recalc, 3)

	assert.Equal(t, "Dawn", recalc[0].Parent, "day1 babysitter fixed")
	assert.Equal(t, fairness.CaregiverTypeBabysitter, recalc[0].CaregiverType)
	// day2: no parent history → Alice (TotalCount, parentA ≤ parentB)
	assert.Equal(t, "Alice", recalc[1].Parent, "day2: first parent assignment → Alice")
	// day3: Alice=1, Bob=0 → Bob
	assert.Equal(t, "Bob", recalc[2].Parent, "day3: Alice=1, Bob=0 → Bob")
}

// TestBabysitterOnLastDayOfSchedule verifies that a babysitter on the last
// day is treated as fixed.
func TestBabysitterOnLastDayOfSchedule(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	_ = time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)     // day2
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC) // babysitter

	initial, err := sched.GenerateSchedule(day1, day3, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 3)

	// Set last day to babysitter
	day3Assignment, err := tracker.GetAssignmentByDate(day3)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day3Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Regenerate — day3 stays as babysitter
	recalc, err := sched.GenerateSchedule(day1, day3, day1)
	assert.NoError(t, err)
	assert.Len(t, recalc, 3)

	assert.Equal(t, "Dawn", recalc[2].Parent, "day3 babysitter stays fixed on last day")
	assert.Equal(t, fairness.CaregiverTypeBabysitter, recalc[2].CaregiverType)
}

// ──────────────────────────────────────────────────────────────────────────────
// 10. Babysitter replaced by parent override
// ──────────────────────────────────────────────────────────────────────────────

// TestBabysitterReplacedByParentOverride verifies that converting a babysitter
// assignment back to a parent override correctly updates stats and recalculates.
func TestBabysitterReplacedByParentOverride(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	// day1=Alice, day2=babysitter, day3=recalculate
	_, err = tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", day2, true)
	assert.NoError(t, err)

	// Replace babysitter with parent override: day2=Bob(override)
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentParent(day2Assignment.ID, "Bob", true)
	assert.NoError(t, err)

	// Verify day2 is now a parent assignment
	updated, err := tracker.GetAssignmentByID(day2Assignment.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Bob", updated.Parent)
	assert.Equal(t, fairness.CaregiverTypeParent, updated.CaregiverType)
	assert.True(t, updated.Override)

	// Generate day3: Alice=1, Bob=1 → tied → alternate from Bob → Alice
	recalc, err := sched.GenerateSchedule(day3, day3, day3)
	assert.NoError(t, err)
	assert.Len(t, recalc, 1)

	assert.Equal(t, "Alice", recalc[0].Parent, "day3: stats tied, alternate from Bob → Alice")
}

// ──────────────────────────────────────────────────────────────────────────────
// 11. Babysitter with parent unavailability
// ──────────────────────────────────────────────────────────────────────────────

// TestBabysitterWithUnavailableParent verifies that after a babysitter day,
// unavailability rules still apply correctly on the recalculated days.
func TestBabysitterWithUnavailableParent(t *testing.T) {
	cfg := &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "Alice",
			ParentB: "Bob",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{},
			ParentBUnavailable: []string{"Thursday"},
		},
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// Wednesday=babysitter, Thursday=recalculate (Bob unavailable on Thursday)
	wed := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)  // Wednesday
	thu := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)  // Thursday
	fri := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC) // Friday

	_, err = tracker.RecordBabysitterAssignment("Dawn", wed, true)
	assert.NoError(t, err)

	recalc, err := sched.GenerateSchedule(wed, fri, thu)
	assert.NoError(t, err)
	assert.Len(t, recalc, 3)

	assert.Equal(t, "Dawn", recalc[0].Parent, "Wednesday babysitter fixed")
	// Thursday: Bob unavailable → Alice (Unavailability)
	assert.Equal(t, "Alice", recalc[1].Parent, "Thursday: Bob unavailable → Alice")
	assert.Equal(t, fairness.DecisionReasonUnavailability, recalc[1].DecisionReason)
	// Friday: Alice=1, Bob=0 → Bob (TotalCount)
	assert.Equal(t, "Bob", recalc[2].Parent, "Friday: Alice=1, Bob=0 → Bob")
}

// ──────────────────────────────────────────────────────────────────────────────
// 12. Multiple babysitters with different names
// ──────────────────────────────────────────────────────────────────────────────

// TestMultipleDifferentBabysitters verifies that assignments to different
// babysitter names are all properly excluded from parent fairness calculations.
func TestMultipleDifferentBabysitters(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	day1 := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)  // Alice
	day2 := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)  // babysitter Dawn
	day3 := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)  // babysitter Eve
	day4 := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)  // babysitter Frank
	day5 := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC) // recalculate

	_, err = tracker.RecordAssignment("Alice", day1, false, fairness.DecisionReasonTotalCount)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Dawn", day2, true)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Eve", day3, true)
	assert.NoError(t, err)
	_, err = tracker.RecordBabysitterAssignment("Frank", day4, true)
	assert.NoError(t, err)

	// day5: only Alice(day1) in parent stats → Bob (TotalCount)
	recalc, err := sched.GenerateSchedule(day5, day5, day5)
	assert.NoError(t, err)
	assert.Len(t, recalc, 1)

	assert.Equal(t, "Bob", recalc[0].Parent, "day5: only Alice=1 in stats → Bob (TotalCount)")
	assert.Equal(t, fairness.DecisionReasonTotalCount, recalc[0].DecisionReason)
}

// ──────────────────────────────────────────────────────────────────────────────
// 13. Full week scenario with mid-week babysitter
// ──────────────────────────────────────────────────────────────────────────────

// TestFullWeekWithMidWeekBabysitter simulates a realistic week-long schedule
// where a babysitter is set mid-week, verifying the entire schedule is
// correctly recalculated end-to-end.
func TestFullWeekWithMidWeekBabysitter(t *testing.T) {
	cfg := newBabysitterTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// Monday through Sunday
	mon := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	_ = time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC) // tue
	wed := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	thu := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)
	_ = time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC) // fri
	_ = time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC) // sat
	sun := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)

	// Generate full week: Mon=Alice, Tue=Bob, Wed=Alice, Thu=Bob, Fri=Alice, Sat=Bob, Sun=Alice
	initial, err := sched.GenerateSchedule(mon, sun, mon)
	assert.NoError(t, err)
	assert.Len(t, initial, 7)
	assert.Equal(t, "Alice", initial[0].Parent) // Mon
	assert.Equal(t, "Bob", initial[1].Parent)   // Tue
	assert.Equal(t, "Alice", initial[2].Parent) // Wed
	assert.Equal(t, "Bob", initial[3].Parent)   // Thu
	assert.Equal(t, "Alice", initial[4].Parent) // Fri
	assert.Equal(t, "Bob", initial[5].Parent)   // Sat
	assert.Equal(t, "Alice", initial[6].Parent) // Sun

	// Set Wednesday to babysitter (mid-week)
	wedAssignment, err := tracker.GetAssignmentByDate(wed)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(wedAssignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Regenerate with currentTime=Thursday
	// Fixed: Mon=Alice(past), Tue=Bob(past), Wed=Dawn(override)
	// Thu onward: recalculate (after override)
	recalc, err := sched.GenerateSchedule(mon, sun, thu)
	assert.NoError(t, err)
	assert.Len(t, recalc, 7)

	assert.Equal(t, "Alice", recalc[0].Parent, "Mon fixed")
	assert.Equal(t, "Bob", recalc[1].Parent, "Tue fixed")
	assert.Equal(t, "Dawn", recalc[2].Parent, "Wed babysitter fixed")
	assert.Equal(t, fairness.CaregiverTypeBabysitter, recalc[2].CaregiverType)

	// Thu: Stats: Alice=1(Mon), Bob=1(Tue) → tied.
	// Last parent = Bob(Tue) → alternate → Alice
	assert.Equal(t, "Alice", recalc[3].Parent, "Thu: tied stats, alternate from Bob → Alice")

	// Fri: Stats: Alice=2(Mon+Thu), Bob=1(Tue) → Alice has more → Bob
	assert.Equal(t, "Bob", recalc[4].Parent, "Fri: Alice=2, Bob=1 → Bob (TotalCount)")

	// Sat: Stats: Alice=2, Bob=2 → tied, last parent = Bob(Fri) → Alice
	assert.Equal(t, "Alice", recalc[5].Parent, "Sat: tied, alternate from Bob → Alice")

	// Sun: Stats: Alice=3, Bob=2 → Bob has fewer → Bob
	assert.Equal(t, "Bob", recalc[6].Parent, "Sun: Alice=3, Bob=2 → Bob (TotalCount)")
}
