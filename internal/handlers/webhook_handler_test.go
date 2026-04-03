package handlers

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	gcalendar "google.golang.org/api/calendar/v3"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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

func (m *MockTracker) RecordBabysitterAssignment(name string, date time.Time, override bool) (*fairness.Assignment, error) {
	args := m.Called(name, date, override)
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

func (m *MockTracker) UpdateAssignmentToBabysitter(id int64, babysitterName string, override bool) error {
	args := m.Called(id, babysitterName, override)
	return args.Error(0)
}

func (m *MockTracker) GetParentMonthlyStatsForLastNMonths(referenceTime time.Time, nMonths int) ([]fairness.MonthlyStatRow, error) {
	args := m.Called(referenceTime, nMonths)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]fairness.MonthlyStatRow), args.Error(1)
}

func (m *MockTracker) GetBabysitterMonthlyStatsForLastNMonths(referenceTime time.Time, nMonths int) ([]fairness.MonthlyStatRow, error) {
	args := m.Called(referenceTime, nMonths)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]fairness.MonthlyStatRow), args.Error(1)
}

func (m *MockTracker) UnlockAssignment(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockTracker) SaveAssignmentDetails(assignmentID int64, calculationDate time.Time, parentAName string, statsA fairness.Stats, parentBName string, statsB fairness.Stats) error {
	args := m.Called(assignmentID, calculationDate, parentAName, statsA, parentBName, statsB)
	return args.Error(0)
}

func (m *MockTracker) GetAssignmentDetails(assignmentID int64) (*fairness.AssignmentDetails, error) {
	args := m.Called(assignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*fairness.AssignmentDetails), args.Error(1)
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

func (m *MockScheduler) UpdateAssignmentToBabysitter(id int64, babysitterName string, override bool) error {
	args := m.Called(id, babysitterName, override)
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

// MockConfigStore is a mock implementation of config.ConfigStoreInterface
type MockConfigStore struct {
	mock.Mock
}

func (m *MockConfigStore) GetParents() (string, string, error) {
	hasExpectation := false
	for _, call := range m.ExpectedCalls {
		if call.Method == "GetParents" {
			hasExpectation = true
			break
		}
	}
	if !hasExpectation {
		return "ParentA", "ParentB", nil
	}

	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockConfigStore) GetAvailability(parent string) ([]string, error) {
	args := m.Called(parent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigStore) GetSchedule() (string, int, int, constants.StatsOrder, error) {
	args := m.Called()
	return args.String(0), args.Int(1), args.Int(2), args.Get(3).(constants.StatsOrder), args.Error(4)
}

func (m *MockConfigStore) GetOAuthConfig() *oauth2.Config {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*oauth2.Config)
}

// defaultTestThresholdDays is the PastEventThresholdDays value used in tests that exercise
// recalculateSchedule.  The threshold is not the subject of those tests so a named constant
// avoids a confusing bare magic number.
const defaultTestThresholdDays = 5

func TestWebhookHandler_RecalculateSchedule(t *testing.T) {
	now := time.Now()
	fromDate := now.Truncate(24 * time.Hour)
	ctx := context.Background()

	// setupMocks configures tracker/scheduler/calendar mocks for each test case.
	// ConfigStore is set up separately (below) because its setup is uniform across
	// all sub-tests (it always returns configLookAheadDays).  Keeping them apart
	// avoids changing the four-argument closure signature just for one extra mock.
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
			mockConfigStore := new(MockConfigStore)

			tt.setupMocks(mockTracker, mockScheduler, mockCalService)

			// Set up config store mock to return look-ahead days live from "database".
			// Use Maybe() because GetSchedule is only called when the last-assignment
			// date is zero or before fromDate (test cases that return a real last date
			// take a different branch and never call GetSchedule).
			mockConfigStore.On("GetSchedule").Maybe().Return("daily", tt.configLookAheadDays, defaultTestThresholdDays, constants.StatsOrderDesc, nil)

			// Create handler with mocked dependencies
			handler := &WebhookHandler{
				BaseHandler: &BaseHandler{
					TokenStore:  nil,
					Tracker:     mockTracker,
					ConfigStore: mockConfigStore,
				},
				CalendarService: mockCalService,
				Scheduler:       mockScheduler,
				ConfigStore:     mockConfigStore,
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
			mockConfigStore.AssertExpectations(t)
		})
	}
}

// TestProcessEventsIntegration tests the event-processing logic end-to-end against a real SQLite database
func TestProcessEventsIntegration(t *testing.T) {
	// Setup test database in temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_webhook_transaction.db")

	db, err := database.New(database.NewDefaultOptions(dbPath))
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	err = db.MigrateDatabase()
	require.NoError(t, err)

	// Create config store with default schedule settings
	configStore, err := database.NewConfigStore(db)
	require.NoError(t, err)
	err = configStore.SaveSchedule("daily", 7, 5, constants.StatsOrderDesc)
	require.NoError(t, err)
	err = configStore.SaveParents("ParentA", "ParentB")
	require.NoError(t, err)
	err = configStore.SaveAvailability("parent_a", []string{})
	require.NoError(t, err)
	err = configStore.SaveAvailability("parent_b", []string{})
	require.NoError(t, err)

	// Create real tracker and scheduler
	tracker, err := fairness.New(db)
	require.NoError(t, err)

	// Create config adapter (single source of truth; holds OAuth + DB settings)
	configAdapter := database.NewConfigAdapter(configStore, nil)
	scheduler := Scheduler.New(configAdapter, tracker)

	// Create mock calendar service
	mockCalService := &MockCalendarService{}
	mockCalService.On("SyncSchedule", mock.Anything, mock.Anything).Return(nil)

	// Create webhook handler with real database and live config adapter
	handler := &WebhookHandler{
		BaseHandler: &BaseHandler{
			Tracker:     tracker,
			ConfigStore: configAdapter,
		},
		Scheduler:       scheduler,
		CalendarService: mockCalService,
		ConfigStore:     configAdapter,
		logger:          logging.GetLogger("webhook-test"),
	}

	t.Run("Successful Transaction with Multiple Events", func(t *testing.T) {
		ctx := context.Background()

		// Create test events that won't trigger updates (matching parent names)
		events := []*gcalendar.Event{
			{
				Id:      "event1",
				Status:  "confirmed",
				Summary: "[OriginalParent1] 🌃👶Routine", // Same as original parent
				ExtendedProperties: &gcalendar.EventExtendedProperties{
					Private: map[string]string{
						"app": constants.NightRoutineIdentifier,
					},
				},
			},
		}

		// First, create assignment that this event will reference
		assignment1, err := tracker.RecordAssignment("OriginalParent1", time.Now().AddDate(0, 0, 1), false, fairness.DecisionReasonTotalCount)
		require.NoError(t, err)

		// Update assignment with Google Calendar event ID
		err = tracker.UpdateAssignmentGoogleCalendarEventID(assignment1.ID, "event1")
		require.NoError(t, err)

		// Count assignments before transaction
		var countBefore int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments").Scan(&countBefore)
		require.NoError(t, err)

		// Process events (no transaction needed — processEvents is called directly)
		err = handler.processEvents(ctx, events, handler.logger)

		// Should succeed since parent names match (no update needed)
		assert.NoError(t, err)

		// Verify count is unchanged (no new assignments created)
		var countAfter int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments").Scan(&countAfter)
		assert.NoError(t, err)
		assert.Equal(t, countBefore, countAfter)
	})

	t.Run("Scheduler Error Returns Error", func(t *testing.T) {
		ctx := context.Background()

		// Create a mock scheduler that will fail
		mockScheduler := new(MockScheduler)
		mockScheduler.On("GetAssignmentByGoogleCalendarEventID", "event_fail").Return((*Scheduler.Assignment)(nil), errors.New("scheduler error"))

		// Create handler with mock scheduler that will fail
		handlerWithFailingScheduler := &WebhookHandler{
			BaseHandler: &BaseHandler{
				Tracker:     tracker,
				ConfigStore: configAdapter,
			},
			Scheduler:   mockScheduler,
			ConfigStore: configAdapter,
		}

		// Create test event that will cause scheduler to fail
		events := []*gcalendar.Event{
			{
				Id:      "event_fail",
				Status:  "confirmed",
				Summary: "[FailParent] 🌃👶Routine",
				ExtendedProperties: &gcalendar.EventExtendedProperties{
					Private: map[string]string{
						"app": constants.NightRoutineIdentifier,
					},
				},
			},
		}

		// Count assignments before
		var countBefore int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments").Scan(&countBefore)
		require.NoError(t, err)

		// Process events — should fail due to scheduler error
		err = handlerWithFailingScheduler.processEvents(ctx, events, handler.logger)

		// Should fail due to scheduler error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scheduler error")

		// Verify no changes were made
		var countAfter int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments").Scan(&countAfter)
		assert.NoError(t, err)
		assert.Equal(t, countBefore, countAfter)

		mockScheduler.AssertExpectations(t)
	})

	t.Run("Transaction Handles Cancelled Events", func(t *testing.T) {
		ctx := context.Background()

		// Create test events with cancelled status
		events := []*gcalendar.Event{
			{
				Id:      "cancelled_event",
				Status:  "cancelled",
				Summary: "[CancelledParent] 🌃👶Routine",
				ExtendedProperties: &gcalendar.EventExtendedProperties{
					Private: map[string]string{
						"app": constants.NightRoutineIdentifier,
					},
				},
			},
		}

		// Count assignments before transaction
		var countBefore int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments").Scan(&countBefore)
		require.NoError(t, err)

		// Process events (no transaction wrapper — processEvents is called directly)
		err = handler.processEvents(ctx, events, handler.logger)

		// Should succeed (cancelled events are skipped)
		assert.NoError(t, err)

		// Verify no changes were made (cancelled events are ignored)
		var countAfter int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments").Scan(&countAfter)
		assert.NoError(t, err)
		assert.Equal(t, countBefore, countAfter)
	})

	t.Run("Transaction Handles Non-Night-Routine Events", func(t *testing.T) {
		ctx := context.Background()

		// Create test events without Night Routine identifier
		events := []*gcalendar.Event{
			{
				Id:      "external_event",
				Status:  "confirmed",
				Summary: "[ExternalParent] Some Other Event",
				ExtendedProperties: &gcalendar.EventExtendedProperties{
					Private: map[string]string{
						"app": "other-app",
					},
				},
			},
			{
				Id:      "no_properties_event",
				Status:  "confirmed",
				Summary: "[NoPropsParent] Event Without Properties",
				// No ExtendedProperties
			},
		}

		// Count assignments before transaction
		var countBefore int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments").Scan(&countBefore)
		require.NoError(t, err)

		// Process events (no transaction wrapper — processEvents is called directly)
		err = handler.processEvents(ctx, events, handler.logger)

		// Should succeed (non-Night-Routine events are skipped)
		assert.NoError(t, err)

		// Verify no changes were made (external events are ignored)
		var countAfter int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments").Scan(&countAfter)
		assert.NoError(t, err)
		assert.Equal(t, countBefore, countAfter)
	})
}

// TestProcessEventChangesTransactionIntegration tests the full processEventChanges method with transaction
func TestProcessEventChangesTransactionIntegration(t *testing.T) {
	// This test would require mocking Google Calendar API calls
	// For now, we focus on testing the transaction wrapper behavior

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_webhook_process_events.db")

	db, err := database.New(database.NewDefaultOptions(dbPath))
	require.NoError(t, err)
	defer db.Close()

	err = db.MigrateDatabase()
	require.NoError(t, err)

	t.Run("Transaction Wrapper Functionality", func(t *testing.T) {
		ctx := context.Background()

		// Test that the transaction wrapper works by verifying database state
		var transactionStarted bool
		var transactionCommitted bool

		// Use WithTransaction directly to test the wrapper
		err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
			transactionStarted = true

			// Perform a simple database operation
			_, err := tx.ExecContext(ctx, `
				INSERT INTO assignments (parent_name, assignment_date, override, decision_reason)
				VALUES (?, ?, ?, ?)
			`, "TransactionTestParent", "2024-12-01", false, "test_transaction")

			if err != nil {
				return err
			}

			transactionCommitted = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, transactionStarted)
		assert.True(t, transactionCommitted)

		// Verify the record was committed
		var count int
		err = db.Conn().QueryRow("SELECT COUNT(*) FROM assignments WHERE parent_name = ?", "TransactionTestParent").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

// TestProcessEventsWithinTransaction_PastEventThreshold tests the configurable past event threshold
func TestProcessEventsWithinTransaction_PastEventThreshold(t *testing.T) {
	// Setup test database in temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_webhook_threshold.db")

	db, err := database.New(database.NewDefaultOptions(dbPath))
	require.NoError(t, err)
	defer db.Close()

	err = db.MigrateDatabase()
	require.NoError(t, err)

	tracker, err := fairness.New(db)
	require.NoError(t, err)

	now := time.Now()

	tests := []struct {
		name              string
		thresholdDays     int
		assignmentDaysAgo int
		expectedProcessed bool
	}{
		{
			name:              "Within default 5 day threshold - should accept",
			thresholdDays:     5,
			assignmentDaysAgo: 3,
			expectedProcessed: true,
		},
		{
			name:              "At exact 5 day threshold boundary - should accept",
			thresholdDays:     5,
			assignmentDaysAgo: 5,
			expectedProcessed: true,
		},
		{
			name:              "Beyond 5 day threshold - should reject",
			thresholdDays:     5,
			assignmentDaysAgo: 6,
			expectedProcessed: false,
		},
		{
			name:              "Within custom 10 day threshold - should accept",
			thresholdDays:     10,
			assignmentDaysAgo: 8,
			expectedProcessed: true,
		},
		{
			name:              "Beyond custom 10 day threshold - should reject",
			thresholdDays:     10,
			assignmentDaysAgo: 11,
			expectedProcessed: false,
		},
		{
			name:              "With 1 day threshold - yesterday should accept",
			thresholdDays:     1,
			assignmentDaysAgo: 1,
			expectedProcessed: true,
		},
		{
			name:              "With 1 day threshold - 2 days ago should reject",
			thresholdDays:     1,
			assignmentDaysAgo: 2,
			expectedProcessed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Calculate the assignment date based on days ago
			assignmentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -tt.assignmentDaysAgo)

			// Create assignment in the database
			assignment, err := tracker.RecordAssignment("OriginalParent", assignmentDate, false, fairness.DecisionReasonTotalCount)
			require.NoError(t, err)

			// Update assignment with Google Calendar event ID
			eventID := "test_event_" + tt.name
			err = tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, eventID)
			require.NoError(t, err)

			// Create real scheduler — uses mockConfigStore for parent/availability lookups
			mockConfigStore := new(MockConfigStore)
			mockConfigStore.On("GetSchedule").Return("daily", 7, tt.thresholdDays, constants.StatsOrderDesc, nil)
			mockConfigStore.On("GetParents").Return("OriginalParent", "NewParent", nil)
			mockConfigStore.On("GetAvailability", mock.Anything).Maybe().Return([]string{}, nil)
			scheduler := Scheduler.New(mockConfigStore, tracker)

			// Create mock calendar service
			mockCalService := &MockCalendarService{}
			if tt.expectedProcessed {
				// Only expect SyncSchedule to be called when processing is expected
				mockCalService.On("SyncSchedule", mock.Anything, mock.Anything).Return(nil)
			}

			// Create webhook handler; threshold is now read from ConfigStore, not RuntimeConfig
			handler := &WebhookHandler{
				BaseHandler: &BaseHandler{
					Tracker:     tracker,
					ConfigStore: mockConfigStore,
				},
				Scheduler:       scheduler,
				CalendarService: mockCalService,
				ConfigStore:     mockConfigStore,
				logger:          logging.GetLogger("webhook-test"),
			}

			// Create test event with changed parent name
			events := []*gcalendar.Event{
				{
					Id:      eventID,
					Status:  "confirmed",
					Summary: "[NewParent] 🌃👶Routine", // Changed from OriginalParent to NewParent
					ExtendedProperties: &gcalendar.EventExtendedProperties{
						Private: map[string]string{
							"app": constants.NightRoutineIdentifier,
						},
					},
				},
			}

			// Process events (no transaction wrapper — processEvents is called directly)
			err = handler.processEvents(ctx, events, handler.logger)

			// Should not error regardless of threshold
			assert.NoError(t, err)

			// Verify the assignment was updated or not based on threshold
			updatedAssignment, err := tracker.GetAssignmentByID(assignment.ID)
			require.NoError(t, err)

			if tt.expectedProcessed {
				// Assignment should be updated with new parent and override flag
				assert.Equal(t, "NewParent", updatedAssignment.Parent, "Assignment parent should be updated when within threshold")
				assert.True(t, updatedAssignment.Override, "Override flag should be set to true")
			} else {
				// Assignment should remain unchanged
				assert.Equal(t, "OriginalParent", updatedAssignment.Parent, "Assignment parent should not be updated when outside threshold")
				assert.False(t, updatedAssignment.Override, "Override flag should remain false")
			}

			mockCalService.AssertExpectations(t)
			mockConfigStore.AssertExpectations(t)
		})
	}
}

// TestWebhookHandler_DynamicConfigReading verifies that updating settings (via ConfigStore)
// takes effect in the webhook handler immediately, without an application restart.
// This is the core regression test for the issue: "updating the settings doesn't impact
// the webhook worker".
func TestWebhookHandler_DynamicConfigReading(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_webhook_dynamic.db")

	db, err := database.New(database.NewDefaultOptions(dbPath))
	require.NoError(t, err)
	defer db.Close()

	err = db.MigrateDatabase()
	require.NoError(t, err)

	// Set up the live config store with an initial threshold of 3 days
	configStore, err := database.NewConfigStore(db)
	require.NoError(t, err)
	err = configStore.SaveSchedule("daily", 7, 3, constants.StatsOrderDesc)
	require.NoError(t, err)
	err = configStore.SaveParents("ParentA", "ParentB")
	require.NoError(t, err)
	err = configStore.SaveAvailability("parent_a", []string{})
	require.NoError(t, err)
	err = configStore.SaveAvailability("parent_b", []string{})
	require.NoError(t, err)

	tracker, err := fairness.New(db)
	require.NoError(t, err)

	// Build the config adapter — single source of truth; configStore holds the live DB values
	configAdapter := database.NewConfigAdapter(configStore, nil)
	sched := Scheduler.New(configAdapter, tracker)

	mockCalService := &MockCalendarService{}
	mockCalService.On("SyncSchedule", mock.Anything, mock.Anything).Return(nil)

	// Build the webhook handler once; it reads config dynamically from configAdapter
	handler := &WebhookHandler{
		BaseHandler: &BaseHandler{
			Tracker:     tracker,
			ConfigStore: configAdapter,
		},
		Scheduler:       sched,
		CalendarService: mockCalService,
		ConfigStore:     configAdapter,
		logger:          logging.GetLogger("webhook-test"),
	}

	ctx := context.Background()
	now := time.Now()

	// Create an assignment 5 days ago — outside the initial 3-day threshold
	assignmentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -5)
	assignment, err := tracker.RecordAssignment("OriginalParent", assignmentDate, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)
	eventID := "dynamic_test_event"
	err = tracker.UpdateAssignmentGoogleCalendarEventID(assignment.ID, eventID)
	require.NoError(t, err)

	events := []*gcalendar.Event{
		{
			Id:      eventID,
			Status:  "confirmed",
			Summary: "[NewParent] 🌃👶Routine",
			ExtendedProperties: &gcalendar.EventExtendedProperties{
				Private: map[string]string{
					"app": constants.NightRoutineIdentifier,
				},
			},
		},
	}

	t.Run("Rejects event with initial 3-day threshold (assignment is 5 days old)", func(t *testing.T) {
		err = handler.processEvents(ctx, events, handler.logger)
		require.NoError(t, err)

		updatedAssignment, err := tracker.GetAssignmentByID(assignment.ID)
		require.NoError(t, err)
		assert.Equal(t, "OriginalParent", updatedAssignment.Parent, "should not update: assignment is outside the 3-day threshold")
		assert.False(t, updatedAssignment.Override)
	})

	// Simulate the user updating "Past Event Threshold (Days)" to 7 via the settings UI.
	// The handler is NOT restarted — it must pick up the new value dynamically.
	err = configStore.SaveSchedule("daily", 7, 7, constants.StatsOrderDesc)
	require.NoError(t, err)

	t.Run("Accepts same event after threshold is updated to 7 days (no restart needed)", func(t *testing.T) {
		err = handler.processEvents(ctx, events, handler.logger)
		require.NoError(t, err)

		updatedAssignment, err := tracker.GetAssignmentByID(assignment.ID)
		require.NoError(t, err)
		assert.Equal(t, "NewParent", updatedAssignment.Parent, "should update: assignment is now within the 7-day threshold")
		assert.True(t, updatedAssignment.Override, "override flag should be set after parent change")
	})
}
