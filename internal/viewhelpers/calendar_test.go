package viewhelpers

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/stretchr/testify/assert"
)

// Helper to create time.Time from YYYY-MM-DD string
func date(t *testing.T, dateStr string) time.Time {
	t.Helper()
	// Use UTC for consistency in tests, assuming dates from scheduler/db might not have location
	tm, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t.Fatalf("Failed to parse date '%s': %v", dateStr, err)
	}
	return tm // Returns time with zero time zone offset (UTC)
}

func TestCalculateCalendarRange(t *testing.T) {
	testCases := []struct {
		name          string
		refDate       time.Time
		expectedStart time.Time
		expectedEnd   time.Time // Check day, month, year, and weekday
	}{
		{
			name:          "Regular Month (April 2025)", // Starts Wednesday, Ends Wednesday
			refDate:       date(t, "2025-04-15"),
			expectedStart: date(t, "2025-03-31"), // Monday
			expectedEnd:   date(t, "2025-05-04"), // Sunday
		},
		{
			name:          "Month starting on Monday (Oct 2023)", // Oct 1st 2023 was Sunday
			refDate:       date(t, "2023-10-10"),
			expectedStart: date(t, "2023-09-25"), // Monday
			expectedEnd:   date(t, "2023-11-05"), // Sunday (Oct 31st is Tue, week ends Nov 5th)
		},
		{
			name:          "Month ending on Sunday (Dec 2023)", // Dec 31st 2023 was Sunday
			refDate:       date(t, "2023-12-15"),
			expectedStart: date(t, "2023-11-27"), // Monday
			expectedEnd:   date(t, "2023-12-31"), // Sunday
		},
		{
			name:          "February Non-Leap Year (Feb 2025)", // Starts Saturday, Ends Friday
			refDate:       date(t, "2025-02-10"),
			expectedStart: date(t, "2025-01-27"), // Monday
			expectedEnd:   date(t, "2025-03-02"), // Sunday
		},
		{
			name:          "February Leap Year (Feb 2024)", // Starts Thursday, Ends Friday
			refDate:       date(t, "2024-02-15"),
			expectedStart: date(t, "2024-01-29"), // Monday
			expectedEnd:   date(t, "2024-03-03"), // Sunday
		},
		{
			name:          "Month starting on Sunday (Sep 2024)", // Sep 1st 2024 is Sunday
			refDate:       date(t, "2024-09-10"),
			expectedStart: date(t, "2024-08-26"), // Monday
			expectedEnd:   date(t, "2024-10-06"), // Sunday (Sep 30th is Mon)
		},
		{
			name:          "Edge case month (Jan 2024)", // Starts Mon, Ends Wed
			refDate:       date(t, "2024-01-10"),
			expectedStart: date(t, "2024-01-01"), // Monday
			expectedEnd:   date(t, "2024-02-04"), // Sunday
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startDate, endDate := CalculateCalendarRange(tc.refDate)

			// Compare only Year, Month, Day
			assert.Equal(t, tc.expectedStart.Year(), startDate.Year(), "Start Year mismatch")
			assert.Equal(t, tc.expectedStart.Month(), startDate.Month(), "Start Month mismatch")
			assert.Equal(t, tc.expectedStart.Day(), startDate.Day(), "Start Day mismatch")
			assert.Equal(t, time.Monday, startDate.Weekday(), "Start date should be Monday")

			assert.Equal(t, tc.expectedEnd.Year(), endDate.Year(), "End Year mismatch")
			assert.Equal(t, tc.expectedEnd.Month(), endDate.Month(), "End Month mismatch")
			assert.Equal(t, tc.expectedEnd.Day(), endDate.Day(), "End Day mismatch")
			assert.Equal(t, time.Sunday, endDate.Weekday(), "End date should be Sunday")
		})
	}
}

func TestStructureAssignmentsForTemplate(t *testing.T) {
	// Use April 2025 range from previous test
	refDate := date(t, "2025-04-15")
	startDate, endDate := CalculateCalendarRange(refDate) // 2025-03-31 to 2025-05-04

	assignments := []*scheduler.Assignment{
		{Date: date(t, "2025-04-01"), Parent: "ParentA"},
		{Date: date(t, "2025-04-03"), Parent: "ParentB"},
		{Date: date(t, "2025-04-15"), Parent: "ParentA"},
		{Date: date(t, "2025-04-30"), Parent: "ParentB"},
		{Date: date(t, "2025-05-02"), Parent: "ParentA"}, // Padding day assignment
		{Date: date(t, "2025-03-31"), Parent: "ParentB"}, // Padding day assignment
	}

	monthName, weeks := StructureAssignmentsForTemplate(startDate, endDate, assignments)

	assert.Equal(t, "April 2025", monthName, "Month name mismatch")
	assert.Len(t, weeks, 5, "Should be 5 full weeks") // 31/3-6/4, 7/4-13/4, 14/4-20/4, 21/4-27/4, 28/4-4/5

	// --- Detailed Checks ---
	// Week 1 (Mar 31 - Apr 6)
	assert.Len(t, weeks[0], 7, "Week 1 should have 7 days")
	assert.Equal(t, 31, weeks[0][0].DayOfMonth) // Mar 31
	assert.False(t, weeks[0][0].IsCurrentMonth)
	assert.NotNil(t, weeks[0][0].Assignment)
	assert.Equal(t, "ParentB", weeks[0][0].Assignment.Parent)
	assert.Equal(t, 1, weeks[0][1].DayOfMonth) // Apr 1
	assert.True(t, weeks[0][1].IsCurrentMonth)
	assert.NotNil(t, weeks[0][1].Assignment)
	assert.Equal(t, "ParentA", weeks[0][1].Assignment.Parent)
	assert.Equal(t, 3, weeks[0][3].DayOfMonth) // Apr 3
	assert.True(t, weeks[0][3].IsCurrentMonth)
	assert.NotNil(t, weeks[0][3].Assignment)
	assert.Equal(t, "ParentB", weeks[0][3].Assignment.Parent)
	assert.Equal(t, 6, weeks[0][6].DayOfMonth) // Apr 6
	assert.True(t, weeks[0][6].IsCurrentMonth)
	assert.Nil(t, weeks[0][6].Assignment) // No assignment for Apr 6

	// Week 3 (Apr 14 - Apr 20)
	assert.Len(t, weeks[2], 7, "Week 3 should have 7 days")
	assert.Equal(t, 15, weeks[2][1].DayOfMonth) // Apr 15 (Tuesday)
	assert.True(t, weeks[2][1].IsCurrentMonth)
	assert.NotNil(t, weeks[2][1].Assignment)
	assert.Equal(t, "ParentA", weeks[2][1].Assignment.Parent)

	// Week 5 (Apr 28 - May 4)
	assert.Len(t, weeks[4], 7, "Week 5 should have 7 days")
	assert.Equal(t, 30, weeks[4][2].DayOfMonth) // Apr 30 (Wednesday)
	assert.True(t, weeks[4][2].IsCurrentMonth)
	assert.NotNil(t, weeks[4][2].Assignment)
	assert.Equal(t, "ParentB", weeks[4][2].Assignment.Parent)
	assert.Equal(t, 1, weeks[4][3].DayOfMonth) // May 1
	assert.False(t, weeks[4][3].IsCurrentMonth)
	assert.Nil(t, weeks[4][3].Assignment)
	assert.Equal(t, 2, weeks[4][4].DayOfMonth) // May 2
	assert.False(t, weeks[4][4].IsCurrentMonth)
	assert.NotNil(t, weeks[4][4].Assignment)
	assert.Equal(t, "ParentA", weeks[4][4].Assignment.Parent)
	assert.Equal(t, 4, weeks[4][6].DayOfMonth) // May 4
	assert.False(t, weeks[4][6].IsCurrentMonth)
	assert.Nil(t, weeks[4][6].Assignment)

	// Test IsToday (mocking 'today' as Apr 15, 2025)
	// Need to re-run structure with mocked 'now' - this is harder without DI for time.Now()
	// For now, we assume the IsToday logic inside the function works if date comparison is correct.
	// A more robust test would involve injecting a clock.

	// Test Empty Assignments
	monthNameEmpty, weeksEmpty := StructureAssignmentsForTemplate(startDate, endDate, []*scheduler.Assignment{})
	assert.Equal(t, "April 2025", monthNameEmpty)
	assert.Len(t, weeksEmpty, 5)
	assert.Nil(t, weeksEmpty[0][1].Assignment) // Check a day that had assignment before
	assert.Nil(t, weeksEmpty[4][4].Assignment) // Check another day

}
