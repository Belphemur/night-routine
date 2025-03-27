package fairness

import (
	"sort"
	"time"
)

// MockTracker is an in-memory implementation of Tracker for testing
type MockTracker struct {
	assignments []Assignment
}

// NewMockTracker creates a new MockTracker instance
func NewMockTracker() *MockTracker {
	return &MockTracker{
		assignments: []Assignment{},
	}
}

// RecordAssignment records a new assignment in memory
func (t *MockTracker) RecordAssignment(parent string, date time.Time) error {
	t.assignments = append(t.assignments, Assignment{
		Parent: parent,
		Date:   date,
	})

	// Sort assignments by date in descending order
	sort.Slice(t.assignments, func(i, j int) bool {
		return t.assignments[i].Date.After(t.assignments[j].Date)
	})

	return nil
}

// GetLastAssignments returns the last n assignments
func (t *MockTracker) GetLastAssignments(n int) ([]Assignment, error) {
	if len(t.assignments) == 0 {
		return []Assignment{}, nil
	}

	// Sort assignments by date in descending order
	sort.Slice(t.assignments, func(i, j int) bool {
		return t.assignments[i].Date.After(t.assignments[j].Date)
	})

	if n > len(t.assignments) {
		n = len(t.assignments)
	}

	return t.assignments[:n], nil
}

// GetParentStats returns statistics for each parent
func (t *MockTracker) GetParentStats() (map[string]Stats, error) {
	stats := make(map[string]Stats)

	// Calculate the date 30 days ago
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	for _, a := range t.assignments {
		// Initialize stats for this parent if not already present
		if _, exists := stats[a.Parent]; !exists {
			stats[a.Parent] = Stats{
				TotalAssignments: 0,
				Last30Days:       0,
			}
		}

		// Update stats
		s := stats[a.Parent]
		s.TotalAssignments++

		if a.Date.After(thirtyDaysAgo) || a.Date.Equal(thirtyDaysAgo) {
			s.Last30Days++
		}

		stats[a.Parent] = s
	}

	return stats, nil
}

// AddAssignment adds a pre-existing assignment to the mock tracker
// This is useful for setting up test scenarios
func (t *MockTracker) AddAssignment(parent string, date time.Time) {
	t.assignments = append(t.assignments, Assignment{
		Parent: parent,
		Date:   date,
	})

	// Sort assignments by date in descending order
	sort.Slice(t.assignments, func(i, j int) bool {
		return t.assignments[i].Date.After(t.assignments[j].Date)
	})
}
