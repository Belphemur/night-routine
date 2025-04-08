package viewhelpers

import (
	"fmt"
	"time"

	"github.com/belphemur/night-routine/internal/fairness/scheduler"
)

// CalendarDay represents a single day cell in the calendar view.
type CalendarDay struct {
	Date           time.Time
	DayOfMonth     int
	IsCurrentMonth bool                  // Is this day within the primary month being displayed?
	Assignment     *scheduler.Assignment // Assignment for this day (nil if none)
}

// CalculateCalendarRange determines the start and end dates for a calendar view
// that displays full weeks (Monday to Sunday) containing the month of the refDate.
func CalculateCalendarRange(refDate time.Time) (startDate time.Time, endDate time.Time) {
	year, month, _ := refDate.Date()
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, refDate.Location())
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	// Find the Monday of the week containing the first day of the month.
	// Go's Weekday starts with Sunday = 0, Monday = 1, ..., Saturday = 6.
	// We want Monday = 0, ..., Sunday = 6 for calculation.
	startDayOfWeek := int(firstOfMonth.Weekday()) // 0 (Sun) to 6 (Sat)
	daysToSubtract := startDayOfWeek - 1          // Days to go back to reach Monday
	if startDayOfWeek == 0 {                      // If Sunday (0), go back 6 days to get Monday
		daysToSubtract = 6
	}
	startDate = firstOfMonth.AddDate(0, 0, -daysToSubtract)

	// Find the Sunday of the week containing the last day of the month.
	endDayOfWeek := int(lastOfMonth.Weekday()) // 0 (Sun) to 6 (Sat)
	daysToAdd := 0
	if endDayOfWeek != 0 { // If not Sunday, calculate days to add to reach Sunday
		daysToAdd = 7 - endDayOfWeek
	}
	endDate = lastOfMonth.AddDate(0, 0, daysToAdd)

	// Ensure start and end times are at the beginning/end of the day for range queries
	// Although GenerateSchedule might not strictly need this, it's good practice for range boundaries.
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	// For the end date, we want to include the whole day, so we go to the start of the *next* day
	// and GenerateSchedule should use an inclusive start and exclusive end, or handle <= end date.
	// Let's stick to the end of the target day for clarity here, assuming GenerateSchedule is inclusive.
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	return startDate, endDate
}

// StructureAssignmentsForTemplate organizes assignments into a weekly structure for the template.
func StructureAssignmentsForTemplate(startDate, endDate time.Time, assignments []*scheduler.Assignment) (monthName string, weeks [][]CalendarDay) {
	// Determine the primary month being displayed (month of the first day of the range that isn't padding)
	// A simpler way: the primary month is the month of the 15th day within the range.
	midPointDate := startDate.AddDate(0, 0, 14) // Approx middle of the displayed range
	primaryMonth := midPointDate.Month()
	primaryYear := midPointDate.Year()
	monthName = fmt.Sprintf("%s %d", primaryMonth.String(), primaryYear)

	assignmentMap := make(map[string]*scheduler.Assignment)
	for _, a := range assignments {
		if a != nil {
			// Use UTC date string for map key to avoid timezone issues if dates aren't consistent
			dateStr := a.Date.UTC().Format("2006-01-02")
			assignmentMap[dateStr] = a
		}
	}

	var currentWeek []CalendarDay

	currentDate := startDate // Reset currentDate to the actual start
	for !currentDate.After(endDate) {
		dateStr := currentDate.UTC().Format("2006-01-02") // Use UTC for map lookup and today comparison
		day := CalendarDay{
			Date:           currentDate,
			DayOfMonth:     currentDate.Day(),
			IsCurrentMonth: currentDate.Month() == primaryMonth && currentDate.Year() == primaryYear,
			Assignment:     assignmentMap[dateStr], // Will be nil if no assignment
		}
		currentWeek = append(currentWeek, day)

		// If Sunday, or if it's the very last day, end the week
		if currentDate.Weekday() == time.Sunday || !currentDate.Before(endDate) {
			weeks = append(weeks, currentWeek)
			currentWeek = []CalendarDay{} // Start a new week
		}

		// Move to the next day
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return monthName, weeks
}
