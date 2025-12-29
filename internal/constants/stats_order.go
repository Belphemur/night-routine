// Package constants provides shared constants for the night-routine application
package constants

import "fmt"

// StatsOrder represents the sort order for statistics display
type StatsOrder string

const (
	// StatsOrderDesc sorts statistics in descending order (newest first)
	StatsOrderDesc StatsOrder = "desc"
	// StatsOrderAsc sorts statistics in ascending order (oldest first)
	StatsOrderAsc StatsOrder = "asc"
)

// IsValid checks if the stats order value is valid
func (s StatsOrder) IsValid() bool {
	return s == StatsOrderDesc || s == StatsOrderAsc
}

// String returns the string representation of the stats order
func (s StatsOrder) String() string {
	return string(s)
}

// ParseStatsOrder parses a string into a StatsOrder type
// Returns an error if the value is invalid
func ParseStatsOrder(s string) (StatsOrder, error) {
	order := StatsOrder(s)
	if !order.IsValid() {
		return "", fmt.Errorf("invalid stats order: %s (must be 'desc' or 'asc')", s)
	}
	return order, nil
}

// GetAllStatsOrders returns all valid stats order values
// This provides a consistent list for UI components
func GetAllStatsOrders() []StatsOrder {
	return []StatsOrder{StatsOrderDesc, StatsOrderAsc}
}
