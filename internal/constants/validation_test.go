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
