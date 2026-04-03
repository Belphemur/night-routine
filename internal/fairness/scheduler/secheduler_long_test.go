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
// Balance verification over long periods
// ──────────────────────────────────────────────────────────────────────────────

// TestBalanceOverLongPeriods verifies that over extended periods with no
// unavailability, the scheduler maintains fair assignment distribution.
// TotalCount may create temporary back-to-back assignments to correct
// imbalances, but the overall distribution remains fair.
func TestBalanceOverLongPeriods(t *testing.T) {
	testCases := []struct {
		name                string
		days                int
		expectedAssignments map[string]int
	}{
		{
			name: "30 days - no unavailability, balanced distribution",
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
			for day := 0; day < tc.days; day++ {
				date := startDate.AddDate(0, 0, day)

				assignment, err := scheduler.assignForDate(date, cfg)
				assert.NoError(t, err)
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

// TestUnavailabilityImbalanceCorrectedByTotalCount verifies that when
// unavailability causes a TotalCount imbalance, the algorithm correctly assigns
// the parent with fewer assignments to restore balance.
//
// Scenario: Bob is unavailable on Wednesday. After Wednesday (Alice forced by
// unavailability), Alice has more assignments than Bob. Thursday correctly
// assigns Bob via TotalCount to restore balance.
func TestUnavailabilityImbalanceCorrectedByTotalCount(t *testing.T) {
	store := newTestConfigStore("Alice", "Bob", []string{}, []string{"Wednesday"})

	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := fairness.New(db)
	assert.NoError(t, err)
	scheduler := New(store, tracker)
	cfg := testScheduleConfig(store)

	startDate := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC) // Monday

	type dayResult struct {
		Parent         string
		DecisionReason fairness.DecisionReason
	}

	var results []dayResult
	for day := range 14 {
		date := startDate.AddDate(0, 0, day)
		a, err := scheduler.assignForDate(date, cfg)
		assert.NoError(t, err)
		results = append(results, dayResult{Parent: a.Parent, DecisionReason: a.DecisionReason})
	}

	// Wednesday (day index 2) must be Alice via Unavailability (Bob is unavailable).
	assert.Equal(t, "Alice", results[2].Parent, "Wed should be Alice (Bob unavailable)")
	assert.Equal(t, fairness.DecisionReasonUnavailability, results[2].DecisionReason,
		"Wed decision reason should be Unavailability")

	// Thursday (day index 3) should be Bob. TotalCount sees the imbalance
	// (Alice has more assignments) and corrects it by assigning Bob.
	assert.Equal(t, "Bob", results[3].Parent,
		"Thu should be Bob (TotalCount correction after unavailability)")

	// Verify balance is maintained over the 14 days
	aliceCount, bobCount := 0, 0
	for _, r := range results {
		if r.Parent == "Alice" {
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
