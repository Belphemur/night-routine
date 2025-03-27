package scheduler

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/stretchr/testify/assert"
)

// Updated test to include expected days for each parent
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
				"Alice": 15, // Alice should get 15 days
				"Bob":   15, // Bob should get 15 days
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Alice"},
		},
		{
			name:               "14 days - Bob unavailable on Wednesday & Alice unavailable on Friday",
			days:               14,
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{"Wednesday"},
			expectedAssignments: map[string]int{
				"Alice": 7, // Alice should get 7 days
				"Bob":   7, // Bob should get 7 days
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice"},
		},
		{
			name:               "30 days - Bob unavailable on Wednesday & Alice unavailable on Friday",
			days:               30,
			parentAUnavailable: []string{"Friday"},
			parentBUnavailable: []string{"Wednesday"},
			expectedAssignments: map[string]int{
				"Alice": 15, // Alice should get 15 days
				"Bob":   15, // Bob should get 15 days
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Alice", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Alice", "Bob", "Alice"},
		},
		{
			name:               "30 days - Bob unavailable on Wednesday & Alice unavailable on Friday",
			days:               30,
			parentAUnavailable: []string{},
			parentBUnavailable: []string{"Wednesday", "Friday"},
			expectedAssignments: map[string]int{
				"Alice": 15, // Alice should get 15 days
				"Bob":   15, // Bob should get 15 days
			},
			expectedDays: []string{"Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Bob", "Alice", "Alice", "Bob", "Alice", "Bob", "Bob", "Alice", "Bob"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create config with specified availability
			cfg := &config.Config{
				Parents: config.ParentsConfig{
					ParentA: "Alice",
					ParentB: "Bob",
				},
				Availability: config.AvailabilityConfig{
					ParentAUnavailable: tc.parentAUnavailable,
					ParentBUnavailable: tc.parentBUnavailable,
				},
			}

			// Create mock tracker
			tracker := fairness.NewMockTracker()
			scheduler := New(cfg, tracker)

			// Create balanced stats
			stats := make(map[string]fairness.Stats)
			stats["Alice"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}
			stats["Bob"] = fairness.Stats{TotalAssignments: 10, Last30Days: 5}

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

				assignment, err := scheduler.assignForDate(date)
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

// TestAssignForDateWithSpecificDays tests the assignForDate function for specific days of the week
// to verify the correct parent is assigned based on availability
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
			cfg := &config.Config{
				Parents: config.ParentsConfig{
					ParentA: "Alice",
					ParentB: "Bob",
				},
				Availability: config.AvailabilityConfig{
					ParentAUnavailable: tc.parentAUnavailable,
					ParentBUnavailable: tc.parentBUnavailable,
				},
			}

			// Create mock tracker
			tracker := fairness.NewMockTracker()
			scheduler := New(cfg, tracker)

			// Assign for the specific date
			assignment, err := scheduler.assignForDate(tc.date)
			assert.NoError(t, err)

			// Verify the assignment matches the expected parent
			assert.Equal(t, tc.expectedParent, assignment.Parent,
				"Expected %s to be assigned on %s but got %s",
				tc.expectedParent, tc.date.Format("Monday"), assignment.Parent)
		})
	}
}
