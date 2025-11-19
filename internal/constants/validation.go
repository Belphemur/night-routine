// Package constants provides shared constants for the night-routine application
package constants

// ValidDaysOfWeek is a map of valid day-of-week names
// Used for validating user input in settings and configuration
var ValidDaysOfWeek = map[string]bool{
	"Monday":    true,
	"Tuesday":   true,
	"Wednesday": true,
	"Thursday":  true,
	"Friday":    true,
	"Saturday":  true,
	"Sunday":    true,
}

// IsValidDayOfWeek checks if a given day string is a valid day of the week
func IsValidDayOfWeek(day string) bool {
	return ValidDaysOfWeek[day]
}
