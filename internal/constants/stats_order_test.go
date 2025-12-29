package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsOrder_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		order    StatsOrder
		expected bool
	}{
		{
			name:     "desc is valid",
			order:    StatsOrderDesc,
			expected: true,
		},
		{
			name:     "asc is valid",
			order:    StatsOrderAsc,
			expected: true,
		},
		{
			name:     "empty string is invalid",
			order:    StatsOrder(""),
			expected: false,
		},
		{
			name:     "random string is invalid",
			order:    StatsOrder("random"),
			expected: false,
		},
		{
			name:     "DESC uppercase is invalid",
			order:    StatsOrder("DESC"),
			expected: false,
		},
		{
			name:     "ASC uppercase is invalid",
			order:    StatsOrder("ASC"),
			expected: false,
		},
		{
			name:     "descending full word is invalid",
			order:    StatsOrder("descending"),
			expected: false,
		},
		{
			name:     "ascending full word is invalid",
			order:    StatsOrder("ascending"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.order.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatsOrder_String(t *testing.T) {
	tests := []struct {
		name     string
		order    StatsOrder
		expected string
	}{
		{
			name:     "desc returns desc",
			order:    StatsOrderDesc,
			expected: "desc",
		},
		{
			name:     "asc returns asc",
			order:    StatsOrderAsc,
			expected: "asc",
		},
		{
			name:     "custom value returns itself",
			order:    StatsOrder("custom"),
			expected: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.order.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseStatsOrder(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    StatsOrder
		expectError bool
	}{
		{
			name:        "parse desc",
			input:       "desc",
			expected:    StatsOrderDesc,
			expectError: false,
		},
		{
			name:        "parse asc",
			input:       "asc",
			expected:    StatsOrderAsc,
			expectError: false,
		},
		{
			name:        "parse empty string fails",
			input:       "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "parse invalid string fails",
			input:       "invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "parse DESC uppercase fails",
			input:       "DESC",
			expected:    "",
			expectError: true,
		},
		{
			name:        "parse ASC uppercase fails",
			input:       "ASC",
			expected:    "",
			expectError: true,
		},
		{
			name:        "parse descending full word fails",
			input:       "descending",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseStatsOrder(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid stats order")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetAllStatsOrders(t *testing.T) {
	orders := GetAllStatsOrders()

	assert.Len(t, orders, 2)
	assert.Contains(t, orders, StatsOrderDesc)
	assert.Contains(t, orders, StatsOrderAsc)

	// Verify order (desc should come first as it's the default)
	assert.Equal(t, StatsOrderDesc, orders[0])
	assert.Equal(t, StatsOrderAsc, orders[1])
}

func TestStatsOrderConstants(t *testing.T) {
	// Verify the constant values are as expected
	assert.Equal(t, "desc", string(StatsOrderDesc))
	assert.Equal(t, "asc", string(StatsOrderAsc))
}
