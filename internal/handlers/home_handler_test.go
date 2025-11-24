package handlers

import (
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/viewhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeHandler_flattenCalendarData(t *testing.T) {
	// Create a test HomeHandler (minimal setup needed for this test)
	handler := &HomeHandler{}

	t.Run("empty calendar weeks", func(t *testing.T) {
		result := handler.flattenCalendarData(nil)
		assert.Empty(t, result)

		result = handler.flattenCalendarData([][]viewhelpers.CalendarDay{})
		assert.Empty(t, result)
	})

	t.Run("single day without assignment", func(t *testing.T) {
		date := time.Date(2025, 11, 24, 0, 0, 0, 0, time.UTC)
		weeks := [][]viewhelpers.CalendarDay{
			{
				{
					Date:           date,
					DayOfMonth:     24,
					IsCurrentMonth: true,
					Assignment:     nil,
				},
			},
		}

		result := handler.flattenCalendarData(weeks)
		require.Len(t, result, 1)

		day := result[0]
		assert.Equal(t, "2025-11-24", day.DateStr)
		assert.Equal(t, 24, day.DayOfMonth)
		assert.True(t, day.IsCurrentMonth)
		assert.Equal(t, int64(0), day.AssignmentID)
		assert.Empty(t, day.AssignmentParent)
		assert.Empty(t, day.AssignmentReason)
		assert.False(t, day.IsOverridden)
		assert.Contains(t, day.CSSClasses, "bg-white")
		assert.Contains(t, day.CSSClasses, "border")
	})

	t.Run("single day with ParentA assignment", func(t *testing.T) {
		date := time.Date(2025, 11, 24, 0, 0, 0, 0, time.UTC)
		weeks := [][]viewhelpers.CalendarDay{
			{
				{
					Date:           date,
					DayOfMonth:     24,
					IsCurrentMonth: true,
					Assignment: &scheduler.Assignment{
						ID:             1,
						Date:           date,
						Parent:         "Alice",
						ParentType:     scheduler.ParentTypeA,
						DecisionReason: fairness.DecisionReason("TotalCount"),
					},
				},
			},
		}

		result := handler.flattenCalendarData(weeks)
		require.Len(t, result, 1)

		day := result[0]
		assert.Equal(t, "2025-11-24", day.DateStr)
		assert.Equal(t, 24, day.DayOfMonth)
		assert.True(t, day.IsCurrentMonth)
		assert.Equal(t, int64(1), day.AssignmentID)
		assert.Equal(t, "Alice", day.AssignmentParent)
		assert.Equal(t, "TotalCount", day.AssignmentReason)
		assert.False(t, day.IsOverridden)
		assert.Contains(t, day.CSSClasses, "from-blue-50")
		assert.Contains(t, day.CSSClasses, "to-indigo-100")
		assert.Contains(t, day.CSSClasses, "text-indigo-900")
	})

	t.Run("single day with ParentB assignment", func(t *testing.T) {
		date := time.Date(2025, 11, 25, 0, 0, 0, 0, time.UTC)
		weeks := [][]viewhelpers.CalendarDay{
			{
				{
					Date:           date,
					DayOfMonth:     25,
					IsCurrentMonth: true,
					Assignment: &scheduler.Assignment{
						ID:             2,
						Date:           date,
						Parent:         "Bob",
						ParentType:     scheduler.ParentTypeB,
						DecisionReason: fairness.DecisionReason("Fairness"),
					},
				},
			},
		}

		result := handler.flattenCalendarData(weeks)
		require.Len(t, result, 1)

		day := result[0]
		assert.Equal(t, "2025-11-25", day.DateStr)
		assert.Equal(t, 25, day.DayOfMonth)
		assert.True(t, day.IsCurrentMonth)
		assert.Equal(t, int64(2), day.AssignmentID)
		assert.Equal(t, "Bob", day.AssignmentParent)
		assert.Equal(t, "Fairness", day.AssignmentReason)
		assert.False(t, day.IsOverridden)
		assert.Contains(t, day.CSSClasses, "from-amber-50")
		assert.Contains(t, day.CSSClasses, "to-orange-100")
		assert.Contains(t, day.CSSClasses, "text-orange-900")
	})

	t.Run("day with overridden assignment", func(t *testing.T) {
		date := time.Date(2025, 11, 26, 0, 0, 0, 0, time.UTC)
		weeks := [][]viewhelpers.CalendarDay{
			{
				{
					Date:           date,
					DayOfMonth:     26,
					IsCurrentMonth: true,
					Assignment: &scheduler.Assignment{
						ID:             3,
						Date:           date,
						Parent:         "Alice",
						ParentType:     scheduler.ParentTypeA,
						DecisionReason: fairness.DecisionReason("Override"),
					},
				},
			},
		}

		result := handler.flattenCalendarData(weeks)
		require.Len(t, result, 1)

		day := result[0]
		assert.Equal(t, "2025-11-26", day.DateStr)
		assert.True(t, day.IsOverridden)
		assert.Equal(t, "Override", day.AssignmentReason)
		assert.Contains(t, day.CSSClasses, "overridden")
	})

	t.Run("day not in current month", func(t *testing.T) {
		date := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)
		weeks := [][]viewhelpers.CalendarDay{
			{
				{
					Date:           date,
					DayOfMonth:     31,
					IsCurrentMonth: false,
					Assignment:     nil,
				},
			},
		}

		result := handler.flattenCalendarData(weeks)
		require.Len(t, result, 1)

		day := result[0]
		assert.Equal(t, "2025-10-31", day.DateStr)
		assert.False(t, day.IsCurrentMonth)
		assert.Contains(t, day.CSSClasses, "bg-slate-50")
		assert.Contains(t, day.CSSClasses, "text-slate-400")
	})

	t.Run("multiple weeks with mixed data", func(t *testing.T) {
		date1 := time.Date(2025, 11, 24, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2025, 11, 25, 0, 0, 0, 0, time.UTC)
		date3 := time.Date(2025, 11, 26, 0, 0, 0, 0, time.UTC)

		weeks := [][]viewhelpers.CalendarDay{
			{
				{
					Date:           date1,
					DayOfMonth:     24,
					IsCurrentMonth: true,
					Assignment: &scheduler.Assignment{
						ID:             1,
						Parent:         "Alice",
						ParentType:     scheduler.ParentTypeA,
						DecisionReason: fairness.DecisionReason("TotalCount"),
					},
				},
				{
					Date:           date2,
					DayOfMonth:     25,
					IsCurrentMonth: true,
					Assignment:     nil,
				},
			},
			{
				{
					Date:           date3,
					DayOfMonth:     26,
					IsCurrentMonth: true,
					Assignment: &scheduler.Assignment{
						ID:             2,
						Parent:         "Bob",
						ParentType:     scheduler.ParentTypeB,
						DecisionReason: fairness.DecisionReason("Override"),
					},
				},
			},
		}

		result := handler.flattenCalendarData(weeks)
		require.Len(t, result, 3)

		// Check first day
		assert.Equal(t, "2025-11-24", result[0].DateStr)
		assert.Equal(t, "Alice", result[0].AssignmentParent)
		assert.Contains(t, result[0].CSSClasses, "from-blue-50")

		// Check second day (no assignment)
		assert.Equal(t, "2025-11-25", result[1].DateStr)
		assert.Empty(t, result[1].AssignmentParent)
		assert.Contains(t, result[1].CSSClasses, "bg-white")

		// Check third day (overridden)
		assert.Equal(t, "2025-11-26", result[2].DateStr)
		assert.Equal(t, "Bob", result[2].AssignmentParent)
		assert.True(t, result[2].IsOverridden)
		assert.Contains(t, result[2].CSSClasses, "overridden")
	})

	t.Run("CSS classes include all necessary styles", func(t *testing.T) {
		date := time.Date(2025, 11, 24, 0, 0, 0, 0, time.UTC)
		weeks := [][]viewhelpers.CalendarDay{
			{
				{
					Date:           date,
					DayOfMonth:     24,
					IsCurrentMonth: true,
					Assignment: &scheduler.Assignment{
						ID:             1,
						Parent:         "Alice",
						ParentType:     scheduler.ParentTypeA,
						DecisionReason: fairness.DecisionReason("TotalCount"),
					},
				},
			},
		}

		result := handler.flattenCalendarData(weeks)
		require.Len(t, result, 1)

		classes := result[0].CSSClasses
		// Check base classes
		assert.Contains(t, classes, "border")
		assert.Contains(t, classes, "border-slate-200")
		assert.Contains(t, classes, "text-center")
		assert.Contains(t, classes, "align-top")
		assert.Contains(t, classes, "relative")
		assert.Contains(t, classes, "cursor-pointer")
		assert.Contains(t, classes, "transition-all")
		assert.Contains(t, classes, "duration-200")

		// Check hover classes
		assert.Contains(t, classes, "hover:shadow-lg")
	})
}
