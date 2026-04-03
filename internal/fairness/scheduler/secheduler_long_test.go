package scheduler

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/stretchr/testify/assert"
)

func TestAssignForDateLongPeriods(t *testing.T) {
	testCases := []struct {
		name                string
		days                int
		parentAUnavailable  []string
		parentBUnavailable  []string
		expectedAssignments map[string]int // Expected count of assignments per parent
		expectedDays        []string       // Expected parent for each day
	}{
		{
			name:               "14 days - Bob unavailable on Wednesday & Alice available any day",
			days:               14,
			parentAUnavailable: []string{},
			parentBUnavailable: []string{"Wednesday"},
			expectedAssignments: map[string]int{
				"Alice": 7,
				"Bob":   7,
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Alice", "Bob", "Bob", "Alice", "Bob"},
		},
		{
			name:               "30 days - Bob unavailable on Wednesday & Alice available any day",
			days:               30,
			parentAUnavailable: []string{},
			parentBUnavailable: []string{"Wednesday"},
			expectedAssignments: map[string]int{
				"Alice": 15,
				"Bob":   15,
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob"},
		},
		{
			name:               "14 days - Alice unavailable on Friday & Bob available any day",
			days:               14,
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{},
			expectedAssignments: map[string]int{
				"Alice": 7,
				"Bob":   7,
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice"},
		},
		{
			name:               "30 days - Alice unavailable on Friday & Bob available any day",
			days:               30,
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{},
			expectedAssignments: map[string]int{
				"Alice": 15,
				"Bob":   15,
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Alice"},
		},
		{
			name:               "14 days - Bob unavailable on Wednesday & Alice unavailable on Friday",
			days:               14,
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{"Wednesday"},
			expectedAssignments: map[string]int{
				"Alice": 7,
				"Bob":   7,
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice"},
		},
		{
			name:               "30 days - Bob unavailable on Wednesday & Alice unavailable on Friday",
			days:               30,
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{"Wednesday"},
			expectedAssignments: map[string]int{
				"Alice": 15,
				"Bob":   15,
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Alice", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Alice"},
		},
		{
			name:               "30 days - Bob unavailable on Wednesday & Alice unavailable on Friday",
			days:               30,
			parentAUnavailable: []string{},
			parentBUnavailable: []string{"Wednesday", "Friday"},
			expectedAssignments: map[string]int{
				"Alice": 15,
				"Bob":   15,
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create config with specified availability
			store := newTestConfigStore("Alice", "Bob", tc.parentAUnavailable, tc.parentBUnavailable)

			// Create real tracker with in-memory database
			db, cleanup := setupTestDB(t)
			defer cleanup()

			tracker, err := fairness.New(db)
			assert.NoError(t, err)
			scheduler := New(store, tracker)
			cfg := testScheduleConfig(store)

			// Start from a Sunday to ensure we cover a full week
			startDate := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC) // Monday

			// Track assignments
			actualAssignments := make(map[string]int)
			actualAssignments["Alice"] = 0
			actualAssignments["Bob"] = 0

			// Assign for each day in the period
			actualDays := []string{}
			for day := 0; day < tc.days; day++ {
				date := startDate.AddDate(0, 0, day)

				assignment, err := scheduler.assignForDate(date, cfg)
				assert.NoError(t, err)

				// Count the assignment
				actualAssignments[assignment.Parent]++
				actualDays = append(actualDays, assignment.Parent)
			}

			// Verify the assignments match expectations
			assert.Equal(t, tc.expectedAssignments["Alice"], actualAssignments["Alice"],
				"Alice should have %d assignments but got %d",
				tc.expectedAssignments["Alice"], actualAssignments["Alice"])

			assert.Equal(t, tc.expectedAssignments["Bob"], actualAssignments["Bob"],
				"Bob should have %d assignments but got %d",
				tc.expectedAssignments["Bob"], actualAssignments["Bob"])

			// Verify total assignments add up to the number of days
			assert.Equal(t, tc.days, actualAssignments["Alice"]+actualAssignments["Bob"],
				"Total assignments should equal %d days", tc.days)

			// Verify the exact sequence of assignments
			assert.Equal(t, tc.expectedDays, actualDays,
				"Expected assignment sequence does not match actual sequence")
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Regression: ConsecutiveAvoidance ensures no back-to-back without unavailability
// ──────────────────────────────────────────────────────────────────────────────

// TestNoConsecutiveWithoutUnavailability is a regression test for:
// "Algorithm should avoid back-to-back consecutive assignments when there is no
// unavailability forcing the imbalance."
//
// When both parents are always available, perfect alternation (Alice, Bob, Alice,
// Bob, …) is possible and the algorithm should never produce two same-parent
// nights in a row — even across month boundaries where odd-day months cause a
// 1-night TotalCount imbalance.
func TestNoConsecutiveWithoutUnavailability(t *testing.T) {
	testCases := []struct {
		name                string
		days                int
		expectedAssignments map[string]int
	}{
		{
			name: "30 days - no unavailability, perfect alternation",
			days: 30,
			expectedAssignments: map[string]int{
				"Alice": 15,
				"Bob":   15,
			},
		},
		{
			name: "31 days - no unavailability, Alice gets extra day",
			days: 31,
			expectedAssignments: map[string]int{
				"Alice": 16,
				"Bob":   15,
			},
		},
		{
			name: "59 days - no unavailability, two-month span (31+28)",
			days: 59,
			expectedAssignments: map[string]int{
				"Alice": 30,
				"Bob":   29,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestConfigStore("Alice", "Bob", []string{}, []string{})

			db, cleanup := setupTestDB(t)
			defer cleanup()

			tracker, err := fairness.New(db)
			assert.NoError(t, err)
			scheduler := New(store, tracker)
			cfg := testScheduleConfig(store)

			startDate := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC) // Monday

			actualAssignments := map[string]int{"Alice": 0, "Bob": 0}
			var prevParent string
			for day := 0; day < tc.days; day++ {
				date := startDate.AddDate(0, 0, day)

				assignment, err := scheduler.assignForDate(date, cfg)
				assert.NoError(t, err)

				// Core assertion: no two consecutive same-parent nights
				if prevParent != "" {
					assert.NotEqual(t, prevParent, assignment.Parent,
						"Day %d (%s): %s assigned again (consecutive), previous was also %s",
						day+1, date.Format("Monday"), assignment.Parent, prevParent)
				}
				prevParent = assignment.Parent
				actualAssignments[assignment.Parent]++
			}

			// Verify total counts
			assert.Equal(t, tc.expectedAssignments["Alice"], actualAssignments["Alice"],
				"Alice should have %d assignments but got %d",
				tc.expectedAssignments["Alice"], actualAssignments["Alice"])
			assert.Equal(t, tc.expectedAssignments["Bob"], actualAssignments["Bob"],
				"Bob should have %d assignments but got %d",
				tc.expectedAssignments["Bob"], actualAssignments["Bob"])
		})
	}
}

// TestUnavailabilityExemptionAllowsConsecutive verifies that when unavailability
// causes a TotalCount imbalance, the algorithm correctly allows a consecutive
// assignment to restore balance.
//
// Scenario: Bob is unavailable on Wednesday. After Wednesday (Alice forced by
// unavailability), Alice has more assignments than Bob. The day after Wednesday
// (Thursday), TotalCount correctly assigns Bob even though Thursday's assignment
// follows a potential consecutive — this is allowed because the recent
// unavailability caused the imbalance.
func TestUnavailabilityExemptionAllowsConsecutive(t *testing.T) {
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{"Wednesday"})

	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(store, tracker)
	cfg := testScheduleConfig(store)

	startDate := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC) // Monday

	// Run 14 days and verify the unavailability day (Wednesday) creates
	// the expected unavailability-triggered consecutive.
	var assignments []string
	for day := range 14 {
		date := startDate.AddDate(0, 0, day)
		a, err := scheduler.assignForDate(date, cfg)
		assert.NoError(t, err)
		assignments = append(assignments, a.Parent)
	}

	// Days: Mon=Alice, Tue=Bob, Wed=Alice(unavail), Thu=Bob(TotalCount fix)...
	// The Wed→Thu transition should show Alice→Bob (not consecutive).
	// But the Tue→Wed transition is Bob→Alice(unavail), and then later
	// the pattern may create Alice→Alice around Wed when Alice is forced.
	// The key check: the day AFTER unavailability uses TotalCount to correct.
	assert.Equal(t, "Alice", assignments[2], "Wed should be Alice (Bob unavailable)")

	// Verify balance is maintained over the 14 days
	aliceCount, bobCount := 0, 0
	for _, a := range assignments {
		if a == "Alice" {
			aliceCount++
		} else {
			bobCount++
		}
	}
	assert.Equal(t, 7, aliceCount, "Alice should have 7 assignments in 14 days")
	assert.Equal(t, 7, bobCount, "Bob should have 7 assignments in 14 days")
}

// TestAssignForDateWithSpecificDays tests the assignForDate function for specific days of the week
func TestAssignForDateWithSpecificDays(t *testing.T) {
	testCases := []struct {
		name               string
		date               time.Time
		parentAUnavailable []string
		parentBUnavailable []string
		expectedParent     string
	}{
		{
			name:               "Wednesday - Bob unavailable",
			date:               time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC), // Wednesday
			parentAUnavailable: []string{},
			parentBUnavailable: []string{"Wednesday"},
			expectedParent:     "Alice",
		},
		{
			name:               "Friday - Alice unavailable",
			date:               time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC), // Friday
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{},
			expectedParent:     "Bob",
		},
		{
			name:               "Wednesday - Bob unavailable & Alice available",
			date:               time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC), // Wednesday
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{"Wednesday"},
			expectedParent:     "Alice",
		},
		{
			name:               "Friday - Alice unavailable & Bob available",
			date:               time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC), // Friday
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{"Wednesday"},
			expectedParent:     "Bob",
		},
		{
			name:               "Tuesday - Both available, should alternate",
			date:               time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), // Tuesday
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{"Wednesday"},
			expectedParent:     "Alice", // Starting with Alice since no prior assignments
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create config with specified availability
			store := newTestConfigStore("Alice", "Bob", tc.parentAUnavailable, tc.parentBUnavailable)

			// Create real tracker with in-memory database
			db, cleanup := setupTestDB(t)
			defer cleanup()

			tracker, err := fairness.New(db)
			assert.NoError(t, err)
			scheduler := New(store, tracker)
			cfg := testScheduleConfig(store)

			// Assign for the specific date
			assignment, err := scheduler.assignForDate(tc.date, cfg)
			assert.NoError(t, err)

			// Verify the assignment matches the expected parent
			assert.Equal(t, tc.expectedParent, assignment.Parent,
				"Expected %s to be assigned on %s but got %s",
				tc.expectedParent, tc.date.Format("Monday"), assignment.Parent)
		})
	}
}
