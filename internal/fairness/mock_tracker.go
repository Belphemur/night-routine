package fairness

import (
	"fmt"
	"sort"
	"time"
)

// MockTracker is a mock implementation of TrackerInterface for testing
type MockTracker struct {
	assignments []*Assignment
	nextID      int64
}

// NewMockTracker creates a new MockTracker
func NewMockTracker() *MockTracker {
	return &MockTracker{
		assignments: []*Assignment{},
		nextID:      1,
	}
}

// RecordAssignment records a new assignment
func (m *MockTracker) RecordAssignment(parent string, date time.Time) (*Assignment, error) {
	// Check if there's already an assignment for this date
	existingAssignment, err := m.GetAssignmentByDate(date)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}

	// If there's already an assignment, update it
	if existingAssignment != nil {
		// Only update if the parent has changed
		if existingAssignment.Parent != parent {
			existingAssignment.Parent = parent
			existingAssignment.UpdatedAt = time.Now()
			return existingAssignment, nil
		}

		// Parent hasn't changed, return the existing assignment
		return existingAssignment, nil
	}

	// No existing assignment, create a new one
	return m.RecordAssignmentWithDetails(parent, date, false, "")
}

// RecordAssignmentWithOverride records a new assignment with override flag
func (m *MockTracker) RecordAssignmentWithOverride(parent string, date time.Time, override bool) (*Assignment, error) {
	return m.RecordAssignmentWithDetails(parent, date, override, "")
}

// RecordAssignmentWithDetails records an assignment with all available details
func (m *MockTracker) RecordAssignmentWithDetails(parent string, date time.Time, override bool, googleCalendarEventID string) (*Assignment, error) {
	// Check if there's already an assignment for this date
	existingAssignment, err := m.GetAssignmentByDate(date)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}

	// If there's already an assignment, update it
	if existingAssignment != nil {
		existingAssignment.Parent = parent
		existingAssignment.Override = override
		existingAssignment.GoogleCalendarEventID = googleCalendarEventID
		existingAssignment.UpdatedAt = time.Now()
		return existingAssignment, nil
	}

	// No existing assignment, create a new one
	now := time.Now()

	assignment := &Assignment{
		ID:                    m.nextID,
		Parent:                parent,
		Date:                  date,
		Override:              override,
		GoogleCalendarEventID: googleCalendarEventID,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	m.assignments = append(m.assignments, assignment)
	m.nextID++

	return assignment, nil
}

// GetLastAssignments returns the last n assignments
func (m *MockTracker) GetLastAssignments(n int) ([]*Assignment, error) {
	// Sort assignments by date descending
	sort.Slice(m.assignments, func(i, j int) bool {
		return m.assignments[i].Date.After(m.assignments[j].Date)
	})

	result := make([]*Assignment, 0, n)
	for i := 0; i < n && i < len(m.assignments); i++ {
		result = append(result, m.assignments[i])
	}
	return result, nil
}

// GetParentStats returns statistics for each parent
func (m *MockTracker) GetParentStats() (map[string]Stats, error) {
	stats := make(map[string]Stats)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	for _, a := range m.assignments {
		s := stats[a.Parent]
		s.TotalAssignments++
		if a.Date.After(thirtyDaysAgo) {
			s.Last30Days++
		}
		stats[a.Parent] = s
	}
	return stats, nil
}

// GetAssignmentByID retrieves an assignment by its ID
func (m *MockTracker) GetAssignmentByID(id int64) (*Assignment, error) {
	for _, a := range m.assignments {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, fmt.Errorf("assignment not found: %d", id)
}

// GetAssignmentByDate retrieves an assignment for a specific date
func (m *MockTracker) GetAssignmentByDate(date time.Time) (*Assignment, error) {
	dateStr := date.Format("2006-01-02")

	var result *Assignment
	for _, a := range m.assignments {
		if a.Date.Format("2006-01-02") == dateStr {
			// If multiple assignments exist for the date, take the latest one (highest ID)
			if result == nil || a.ID > result.ID {
				result = a
			}
		}
	}

	return result, nil
}

// GetAssignmentByGoogleCalendarEventID retrieves an assignment by its Google Calendar event ID
func (m *MockTracker) GetAssignmentByGoogleCalendarEventID(eventID string) (*Assignment, error) {
	if eventID == "" {
		return nil, nil
	}

	for _, a := range m.assignments {
		if a.GoogleCalendarEventID == eventID {
			return a, nil
		}
	}

	return nil, nil
}

// GetAssignmentsInRange retrieves all assignments in a date range
func (m *MockTracker) GetAssignmentsInRange(start, end time.Time) ([]*Assignment, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	var result []*Assignment
	for _, a := range m.assignments {
		dateStr := a.Date.Format("2006-01-02")
		if dateStr >= startStr && dateStr <= endStr {
			result = append(result, a)
		}
	}

	// Sort by date
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})

	return result, nil
}

// UpdateAssignmentGoogleCalendarEventID updates an assignment with Google Calendar event ID
func (m *MockTracker) UpdateAssignmentGoogleCalendarEventID(id int64, googleCalendarEventID string) error {
	assignment, err := m.GetAssignmentByID(id)
	if err != nil {
		return err
	}

	assignment.GoogleCalendarEventID = googleCalendarEventID
	assignment.UpdatedAt = time.Now()

	return nil
}

// UpdateAssignmentParent updates the parent for an assignment and sets the override flag
func (m *MockTracker) UpdateAssignmentParent(id int64, parent string, override bool) error {
	assignment, err := m.GetAssignmentByID(id)
	if err != nil {
		return err
	}

	assignment.Parent = parent
	assignment.Override = override
	assignment.UpdatedAt = time.Now()

	return nil
}
