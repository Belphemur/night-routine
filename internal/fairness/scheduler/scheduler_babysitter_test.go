package scheduler

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/stretchr/testify/assert"
)

// TestUnlockBabysitterRecalculatesStartDate tests that unlocking a babysitter
// assignment and regenerating from that date actually recalculates the assignment,
// even when the start date is today or in the past. This is a regression test
// for the bug where the scheduler treated today/past non-override assignments
// as "fixed", preventing proper recalculation after unlock.
func TestUnlockBabysitterRecalculatesStartDate(t *testing.T) {
	cfg := &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "Alice",
			ParentB: "Bob",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{},
			ParentBUnavailable: []string{},
		},
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// Scenario:
	// day1=Alice, day2=Bob (initial schedule)
	// day2 is set to babysitter "Dawn" (override)
	// User unlocks day2 → caregiver_type reverts to parent, parent_name stays "Dawn"
	// Regenerate from day2 with currentTime=day2 (today)
	// day2 should be recalculated to a real parent, NOT remain as "Dawn"

	day1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC) // Tuesday
	day2 := time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC) // Wednesday

	// Step 1: Generate initial schedule
	initial, err := sched.GenerateSchedule(day1, day2, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 2)
	assert.Equal(t, "Alice", initial[0].Parent) // day1
	assert.Equal(t, "Bob", initial[1].Parent)   // day2

	// Step 2: Set day2 to babysitter
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Step 3: Unlock day2 (simulates UnlockHandler)
	err = tracker.UnlockAssignment(day2Assignment.ID)
	assert.NoError(t, err)

	// Verify the DB state: parent_name is still "Dawn" but override=false, caregiver_type=parent
	unlocked, err := tracker.GetAssignmentByID(day2Assignment.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Dawn", unlocked.Parent, "parent_name retains babysitter name after unlock")
	assert.Equal(t, fairness.CaregiverTypeParent, unlocked.CaregiverType)
	assert.False(t, unlocked.Override)

	// Step 4: Regenerate from day2 with currentTime=day2 (simulates recalculation after unlock)
	recalc, err := sched.GenerateSchedule(day2, day2, day2)
	assert.NoError(t, err)
	assert.Len(t, recalc, 1)

	// The key assertion: day2 must be recalculated to a real parent, not kept as "Dawn"
	// Alice=1 (day1), Bob=0 after unlock → Bob gets assigned (TotalCount)
	assert.NotEqual(t, "Dawn", recalc[0].Parent, "day2 should NOT keep the babysitter name after unlock + recalculate")
	assert.Equal(t, "Bob", recalc[0].Parent, "day2 should be Bob (fewer total assignments)")
}

// TestBabysitterOnPastDayRecalculatesFutureAssignments tests that setting a
// babysitter on a past date properly recalculates the following assignments.
// Regression test for: "Setting a babysitter in the past isn't properly
// recalculating future assignments"
func TestBabysitterOnPastDayRecalculatesFutureAssignments(t *testing.T) {
	cfg := &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "Alice",
			ParentB: "Bob",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{},
			ParentBUnavailable: []string{},
		},
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// Scenario matching the reported issue:
	// Initial: day1=Alice, day2=Bob, day3=Alice, day4=Bob (alternating)
	// User sets day2 (yesterday) to babysitter "Dawn"
	// day3 (today) should recalculate: Alice has 1 assignment, Bob has 0 parent
	// assignments (day2 is now babysitter), so Bob should be assigned to day3.

	day1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC) // Tuesday
	day2 := time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC) // Wednesday (yesterday)
	day3 := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC) // Thursday (today)
	day4 := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC) // Friday

	// Step 1: Generate initial schedule
	initial, err := sched.GenerateSchedule(day1, day4, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 4)
	assert.Equal(t, "Alice", initial[0].Parent) // day1
	assert.Equal(t, "Bob", initial[1].Parent)   // day2
	assert.Equal(t, "Alice", initial[2].Parent) // day3
	assert.Equal(t, "Bob", initial[3].Parent)   // day4

	// Step 2: Set day2 to babysitter (simulates setting babysitter on yesterday)
	day2Assignment, err := tracker.GetAssignmentByDate(day2)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day2Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Step 3: Regenerate with currentTime = day3 (today)
	// day1 = Alice (past, fixed), day2 = Dawn/babysitter (override, fixed)
	// day3 should recalculate: Alice=1 (day1), Bob=0 parent assignments
	// (day2 is now babysitter so not counted). Bob has fewer → Bob is assigned to day3.
	recalc, err := sched.GenerateSchedule(day1, day4, day3)
	assert.NoError(t, err)
	assert.Len(t, recalc, 4)

	assert.Equal(t, "Alice", recalc[0].Parent, "day1 should be Alice (past, fixed)")
	assert.Equal(t, "Dawn", recalc[1].Parent, "day2 should be Dawn (babysitter override, fixed)")
	assert.Equal(t, fairness.CaregiverTypeBabysitter, recalc[1].CaregiverType, "day2 should be babysitter type")

	// The key assertion: day3 must account for the babysitter on day2.
	// With babysitter filtered from stats and last-assignments, Alice=1, Bob=0
	// so Bob should be chosen (TotalCount).
	assert.Equal(t, "Bob", recalc[2].Parent, "day3 should be Bob (fewer total assignments after babysitter on day2)")
	assert.Equal(t, fairness.DecisionReasonTotalCount, recalc[2].DecisionReason,
		"day3 should have TotalCount reason (Alice=1, Bob=0)")

	// day4: Alice=1, Bob=1 → alternating from Bob → Alice
	assert.Equal(t, "Alice", recalc[3].Parent, "day4 should be Alice (alternating after Bob on day3)")
}

// TestBabysitterPoisonsAlternatingLogic tests that when parent stats are tied
// and the most recent DB row is a babysitter, the alternating/consecutive logic
// correctly uses the most recent *parent* assignment instead of the babysitter.
// Without filtering babysitter rows from GetLastParentAssignmentsUntil, the
// babysitter name doesn't match either parent, causing the alternating fallback
// to always pick parentB regardless of which parent should actually be next.
func TestBabysitterPoisonsAlternatingLogic(t *testing.T) {
	cfg := &config.Config{
		Parents: config.ParentsConfig{
			ParentA: "Alice",
			ParentB: "Bob",
		},
		Availability: config.AvailabilityConfig{
			ParentAUnavailable: []string{},
			ParentBUnavailable: []string{},
		},
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	sched := New(cfg, tracker)

	// Scenario where stats are tied and the alternating/consecutive path decides:
	// day1=Alice, day2=Bob, day3=babysitter "Dawn", day4=recalculate
	// Parent stats at day4: Alice=1, Bob=1 (tied total and last-30)
	// Last parent assignment before day4 (with filter): Bob (day2)
	// Expected via alternating: Bob→Alice
	// Bug (without filter): lastAssignments[0] = "Dawn", "Dawn" != parentB → returns parentB (Bob) WRONG

	day1 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC) // Tuesday
	day3 := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC) // Thursday
	day4 := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC) // Friday

	// Step 1: Generate initial schedule (day1-day4)
	initial, err := sched.GenerateSchedule(day1, day4, day1)
	assert.NoError(t, err)
	assert.Len(t, initial, 4)
	assert.Equal(t, "Alice", initial[0].Parent) // day1
	assert.Equal(t, "Bob", initial[1].Parent)   // day2
	assert.Equal(t, "Alice", initial[2].Parent) // day3
	assert.Equal(t, "Bob", initial[3].Parent)   // day4

	// Step 2: Set day3 to babysitter "Dawn"
	// This makes parent stats: Alice=1 (day1), Bob=1 (day2) — tied
	day3Assignment, err := tracker.GetAssignmentByDate(day3)
	assert.NoError(t, err)
	err = tracker.UpdateAssignmentToBabysitter(day3Assignment.ID, "Dawn", true)
	assert.NoError(t, err)

	// Step 3: Regenerate from day4 (only day4 recalculates)
	// Stats: Alice=1, Bob=1 (tied total + last30)
	// Falls through to alternating logic.
	// With filter: last parent assignment = Bob (day2) → alternate → Alice ✓
	// Without filter: last assignment = Dawn (day3) → "Dawn" != parentB → parentB (Bob) ✗
	recalc, err := sched.GenerateSchedule(day1, day4, day4)
	assert.NoError(t, err)
	assert.Len(t, recalc, 4)

	assert.Equal(t, "Alice", recalc[0].Parent, "day1 should remain Alice (past, fixed)")
	assert.Equal(t, "Bob", recalc[1].Parent, "day2 should remain Bob (past, fixed)")
	assert.Equal(t, "Dawn", recalc[2].Parent, "day3 should remain Dawn (babysitter override, fixed)")
	assert.Equal(t, fairness.CaregiverTypeBabysitter, recalc[2].CaregiverType, "day3 should be babysitter type")

	// The key assertion: with tied stats, the alternating logic must see Bob (day2)
	// as the last parent — not Dawn (day3) — and alternate to Alice.
	assert.Equal(t, "Alice", recalc[3].Parent,
		"day4 should be Alice (alternating from last parent Bob, not poisoned by babysitter Dawn)")
	assert.Equal(t, fairness.DecisionReasonAlternating, recalc[3].DecisionReason,
		"day4 should have Alternating reason (stats tied, alternate from Bob)")
}
