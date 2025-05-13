package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTracker is a mock implementation of the fairness.TrackerInterface
type MockTracker struct {
	mock.Mock
}

func (m *MockTracker) GetLastAssignmentDate() (time.Time, error) {
	args := m.Called()
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *MockTracker) RecordAssignment(parent string, date time.Time, override bool, decisionReason fairness.DecisionReason) (*fairness.Assignment, error) {
	args := m.Called(parent, date, override, decisionReason)
	return args.Get(0).(*fairness.Assignment), args.Error(1)
}

func (m *MockTracker) GetLastAssignmentsUntil(n int, until time.Time) ([]*fairness.Assignment, error) {
	args := m.Called(n, until)
	return args.Get(0).([]*fairness.Assignment), args.Error(1)
}

func (m *MockTracker) GetParentStatsUntil(until time.Time) (map[string]fairness.Stats, error) {
	args := m.Called(until)
	return args.Get(0).(map[string]fairness.Stats), args.Error(1)
}

func (m *MockTracker) GetAssignmentByID(id int64) (*fairness.Assignment, error) {
	args := m.Called(id)
	return args.Get(0).(*fairness.Assignment), args.Error(1)
}

func (m *MockTracker) GetAssignmentByDate(date time.Time) (*fairness.Assignment, error) {
	args := m.Called(date)
	return args.Get(0).(*fairness.Assignment), args.Error(1)
}

func (m *MockTracker) UpdateAssignmentGoogleCalendarEventID(id int64, googleCalendarEventID string) error {
	args := m.Called(id, googleCalendarEventID)
	return args.Error(0)
}

func (m *MockTracker) GetAssignmentByGoogleCalendarEventID(eventID string) (*fairness.Assignment, error) {
	args := m.Called(eventID)
	return args.Get(0).(*fairness.Assignment), args.Error(1)
}

func (m *MockTracker) GetAssignmentsInRange(start, end time.Time) ([]*fairness.Assignment, error) {
	args := m.Called(start, end)
	return args.Get(0).([]*fairness.Assignment), args.Error(1)
}

func (m *MockTracker) UpdateAssignmentParent(id int64, parent string, override bool) error {
	args := m.Called(id, parent, override)
	return args.Error(0)
}

func (m *MockTracker) GetParentMonthlyStatsForLastNMonths(nMonths int) ([]fairness.MonthlyStatRow, error) {
	args := m.Called(nMonths)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]fairness.MonthlyStatRow), args.Error(1)
}

// MockCalendarService is a mock implementation of the calendar.CalendarService interface
type MockCalendarService struct {
	mock.Mock
}

func (m *MockCalendarService) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCalendarService) IsInitialized() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockCalendarService) SetupNotificationChannel(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// SyncSchedule mocks the SyncSchedule method of the CalendarService interface
func (m *MockCalendarService) SyncSchedule(ctx context.Context, assignments []*Scheduler.Assignment) error {
	args := m.Called(ctx, mock.Anything)
	return args.Error(0)
}

func (m *MockCalendarService) StopNotificationChannel(ctx context.Context, channelID, resourceID string) error {
	args := m.Called(ctx, channelID, resourceID)
	return args.Error(0)
}

func (m *MockCalendarService) StopAllNotificationChannels(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCalendarService) VerifyNotificationChannel(ctx context.Context, channelID, resourceID string) (bool, error) {
	args := m.Called(ctx, channelID, resourceID)
	return args.Bool(0), args.Error(1)
}

// MockScheduler is a mock implementation of the Scheduler.SchedulerInterface
type MockScheduler struct {
	mock.Mock
}

// GenerateSchedule mocks the GenerateSchedule method of the SchedulerInterface
func (m *MockScheduler) GenerateSchedule(fromDate, endDate time.Time, currentTime time.Time) ([]*Scheduler.Assignment, error) {
	// Note: We use mock.Anything for currentTime in expectations as it's often time.Now()
	args := m.Called(fromDate, endDate, currentTime)
	// Ensure the returned slice is correctly typed
	if assignments, ok := args.Get(0).([]*Scheduler.Assignment); ok {
		return assignments, args.Error(1)
	}
	// Return nil slice if the type assertion fails or if nil was returned
	return nil, args.Error(1)
}

func (m *MockScheduler) UpdateAssignmentParent(id int64, parent string, override bool) error {
	args := m.Called(id, parent, override)
	return args.Error(0)
}

func (m *MockScheduler) GetAssignmentByGoogleCalendarEventID(eventID string) (*Scheduler.Assignment, error) {
	args := m.Called(eventID)
	return args.Get(0).(*Scheduler.Assignment), args.Error(1)
}

func (m *MockScheduler) UpdateGoogleCalendarEventID(assignment *Scheduler.Assignment, eventID string) error {
	args := m.Called(assignment, eventID)
	return args.Error(0)
}

func TestWebhookHandler_RecalculateSchedule(t *testing.T) {
	now := time.Now()
	fromDate := now.Truncate(24 * time.Hour)
	ctx := context.Background()

	tests := []struct {
		name                string
		setupMocks          func(*MockTracker, *MockScheduler, *MockCalendarService)
		configLookAheadDays int
		expectedError       string
	}{
		{
			name: "Success with existing last assignment date",
			setupMocks: func(tracker *MockTracker, scheduler *MockScheduler, calService *MockCalendarService) {
				lastDate := fromDate.AddDate(0, 0, 5)
				schedulerAssignments := []*Scheduler.Assignment{
					{GoogleCalendarEventID: "event1"},
					{GoogleCalendarEventID: "event2"},
				}

				// Set up expectations for tracker
				tracker.On("GetLastAssignmentDate").Return(lastDate, nil)

				// Set up expectations for scheduler
				// Use mock.Anything for the currentTime argument
				scheduler.On("GenerateSchedule", fromDate, lastDate, mock.AnythingOfType("time.Time")).Return(schedulerAssignments, nil)

				// Set up expectations for calendar service
				calService.On("SyncSchedule", ctx, mock.Anything).Return(nil)
			},
			configLookAheadDays: 7,
			expectedError:       "",
		},
		{
			name: "Success with no existing assignments",
			setupMocks: func(tracker *MockTracker, scheduler *MockScheduler, calService *MockCalendarService) {
				zeroTime := time.Time{}
				lookAheadEndDate := fromDate.AddDate(0, 0, 7)
				schedulerAssignments := []*Scheduler.Assignment{
					{GoogleCalendarEventID: "event1"},
				}

				// Set up expectations for tracker
				tracker.On("GetLastAssignmentDate").Return(zeroTime, nil)

				// Set up expectations for scheduler
				// Use mock.Anything for the currentTime argument
				scheduler.On("GenerateSchedule", fromDate, lookAheadEndDate, mock.AnythingOfType("time.Time")).Return(schedulerAssignments, nil)

				// Set up expectations for calendar service
				calService.On("SyncSchedule", ctx, mock.Anything).Return(nil)
			},
			configLookAheadDays: 7,
			expectedError:       "",
		},
		{
			name: "Error getting last assignment date",
			setupMocks: func(tracker *MockTracker, scheduler *MockScheduler, calService *MockCalendarService) {
				tracker.On("GetLastAssignmentDate").Return(time.Time{}, errors.New("database error"))
			},
			configLookAheadDays: 7,
			expectedError:       "failed to get last assignment date: database error",
		},
		{
			name: "Error generating schedule",
			setupMocks: func(tracker *MockTracker, scheduler *MockScheduler, calService *MockCalendarService) {
				lastDate := fromDate.AddDate(0, 0, 5)

				// Set up expectations for tracker
				tracker.On("GetLastAssignmentDate").Return(lastDate, nil)

				// Set up expectations for scheduler with error
				// Use mock.Anything for the currentTime argument
				scheduler.On("GenerateSchedule", fromDate, lastDate, mock.AnythingOfType("time.Time")).Return([]*Scheduler.Assignment{}, errors.New("generation error"))
			},
			configLookAheadDays: 7,
			expectedError:       "failed to generate schedule: generation error",
		},
		{
			name: "Error syncing schedule",
			setupMocks: func(tracker *MockTracker, scheduler *MockScheduler, calService *MockCalendarService) {
				lastDate := fromDate.AddDate(0, 0, 5)
				schedulerAssignments := []*Scheduler.Assignment{
					{GoogleCalendarEventID: "event1"},
				}

				// Set up expectations for tracker
				tracker.On("GetLastAssignmentDate").Return(lastDate, nil)

				// Set up expectations for scheduler
				// Use mock.Anything for the currentTime argument
				scheduler.On("GenerateSchedule", fromDate, lastDate, mock.AnythingOfType("time.Time")).Return(schedulerAssignments, nil)

				// Set up expectations for calendar service with error
				calService.On("SyncSchedule", ctx, mock.Anything).Return(errors.New("sync error"))
			},
			configLookAheadDays: 7,
			expectedError:       "failed to sync schedule: sync error",
		},
		{
			name: "Success with filtered assignments",
			setupMocks: func(tracker *MockTracker, scheduler *MockScheduler, calService *MockCalendarService) {
				lastDate := fromDate.AddDate(0, 0, 5)

				// Assignments with and without event IDs
				schedulerAssignments := []*Scheduler.Assignment{
					{GoogleCalendarEventID: "event1"},
					{GoogleCalendarEventID: ""}, // Should be filtered out
					{GoogleCalendarEventID: "event3"},
				}

				// Set up expectations for tracker
				tracker.On("GetLastAssignmentDate").Return(lastDate, nil)

				// Set up expectations for scheduler
				// Use mock.Anything for the currentTime argument
				scheduler.On("GenerateSchedule", fromDate, lastDate, mock.AnythingOfType("time.Time")).Return(schedulerAssignments, nil)

				// Set up expectations for calendar service
				calService.On("SyncSchedule", ctx, mock.Anything).Return(nil)
			},
			configLookAheadDays: 7,
			expectedError:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockTracker := new(MockTracker)
			mockScheduler := new(MockScheduler)
			mockCalService := new(MockCalendarService)

			tt.setupMocks(mockTracker, mockScheduler, mockCalService)

			// Create handler with mocked dependencies
			handler := &WebhookHandler{
				BaseHandler: &BaseHandler{
					TokenStore: nil,
					Tracker:    mockTracker,
				},
				CalendarService: mockCalService,
				Scheduler:       mockScheduler,
				Config: &config.Config{
					Schedule: config.ScheduleConfig{
						LookAheadDays: tt.configLookAheadDays,
					},
				},
			}

			// Execute test
			err := handler.recalculateSchedule(ctx, fromDate)

			// Verify results
			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify all mocks
			mockTracker.AssertExpectations(t)
			mockScheduler.AssertExpectations(t)
			mockCalService.AssertExpectations(t)
		})
	}
}
