package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidDayOfWeek(t *testing.T) {
	tests := []struct {
		name     string
		day      string
		expected bool
	}{
		{"Valid Monday", "Monday", true},
		{"Valid Tuesday", "Tuesday", true},
		{"Valid Wednesday", "Wednesday", true},
		{"Valid Thursday", "Thursday", true},
		{"Valid Friday", "Friday", true},
		{"Valid Saturday", "Saturday", true},
		{"Valid Sunday", "Sunday", true},
		{"Invalid lowercase", "monday", false},
		{"Invalid empty", "", false},
		{"Invalid random", "NotADay", false},
		{"Invalid number", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidDayOfWeek(tt.day)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidDaysOfWeek(t *testing.T) {
	// Ensure all 7 days are present
	assert.Len(t, ValidDaysOfWeek, 7)

	// Verify all expected days exist
	expectedDays := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for _, day := range expectedDays {
		assert.True(t, ValidDaysOfWeek[day], "Day %s should be in ValidDaysOfWeek", day)
	}
}

func TestGetAllDaysOfWeek(t *testing.T) {
	days := GetAllDaysOfWeek()

	// Check we have exactly 7 days
	assert.Len(t, days, 7, "Should return exactly 7 days")

	// Check the order is correct
	expectedOrder := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	assert.Equal(t, expectedOrder, days, "Days should be in correct order")

	// Check all days are valid
	for _, day := range days {
		assert.True(t, IsValidDayOfWeek(day), "Day %s should be valid", day)
	}

	// Check all valid days are included
	for day := range ValidDaysOfWeek {
		found := false
		for _, d := range days {
			if d == day {
				found = true
				break
			}
		}
		assert.True(t, found, "Valid day %s should be in returned list", day)
	}
}
