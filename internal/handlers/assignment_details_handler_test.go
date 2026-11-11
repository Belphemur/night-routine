package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type recordingCalendarService struct {
	noopCalendarService
	syncCalls int
}

func (r *recordingCalendarService) SyncSchedule(_ context.Context, _ []*Scheduler.Assignment) error {
	r.syncCalls++
	return nil
}

func testCurrentDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func setupTestAssignmentDetailsHandler(t *testing.T, authenticated bool) (*AssignmentDetailsHandler, *fairness.Tracker, *database.DB, func()) {
	// Create test database
	dbOpts := database.SQLiteOptions{
		Path:        ":memory:",
		Mode:        "rwc",
		Cache:       database.CacheShared,
		Journal:     database.JournalWAL,
		ForeignKeys: true,
		BusyTimeout: 5000,
	}

	db, err := database.New(dbOpts)
	require.NoError(t, err)

	err = db.MigrateDatabase()
	require.NoError(t, err)

	// Create token store
	tokenStore, err := database.NewTokenStore(db)
	require.NoError(t, err)

	if authenticated {
		// Save a token to simulate authenticated state
		testToken := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			TokenType:    "Bearer",
		}
		err = tokenStore.SaveToken(testToken)
		require.NoError(t, err)
	}

	// Create tracker
	tracker, err := fairness.New(db)
	require.NoError(t, err)

	// Create config
	oauthCfg := &oauth2.Config{}

	// Create token manager
	tokenManager := token.NewTokenManager(tokenStore, oauthCfg)

	// Create config adapter — single source of truth for all config reads
	cfgStore, err := database.NewConfigStore(db)
	require.NoError(t, err)
	err = cfgStore.SaveParents("Alice", "Bob")
	require.NoError(t, err)
	err = cfgStore.SaveAvailability("parent_a", []string{})
	require.NoError(t, err)
	err = cfgStore.SaveAvailability("parent_b", []string{})
	require.NoError(t, err)
	err = cfgStore.SaveSchedule("daily", 7, 5, constants.StatsOrderDesc)
	require.NoError(t, err)
	configAdapter := database.NewConfigAdapter(cfgStore, oauthCfg)

	// Create base handler
	baseHandler, err := NewBaseHandler(configAdapter, tokenStore, tokenManager, tracker, "test-version", "test-logo-version")
	require.NoError(t, err)

	// Create assignment details handler with scheduler and no-op external integrations.
	sched := Scheduler.New(configAdapter, tracker)
	handler := NewAssignmentDetailsHandler(baseHandler, tracker, sched, &noopCalendarService{}, configAdapter)

	cleanup := func() {
		db.Close()
	}

	return handler, tracker, db, cleanup
}

func TestHandleGetAssignmentDetails_Success(t *testing.T) {
	// Setup
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	// Create an assignment and save details
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	assignment, err := tracker.RecordAssignment("Alice", date, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	statsA := fairness.Stats{TotalAssignments: 5, Last30Days: 3}
	statsB := fairness.Stats{TotalAssignments: 7, Last30Days: 4}
	err = tracker.SaveAssignmentDetails(assignment.ID, date, "Alice", statsA, "Bob", statsB)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/assignment-details?assignment_id="+strconv.FormatInt(assignment.ID, 10), nil)
	w := httptest.NewRecorder()

	// Execute
	handler.handleGetAssignmentDetails(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response AssignmentDetailsResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, assignment.ID, response.AssignmentID)
	assert.Equal(t, "2025-01-15", response.CalculationDate)
	assert.Equal(t, "Alice", response.ParentAName)
	assert.Equal(t, 5, response.ParentATotalCount)
	assert.Equal(t, 3, response.ParentALast30Days)
	assert.Equal(t, "Bob", response.ParentBName)
	assert.Equal(t, 7, response.ParentBTotalCount)
	assert.Equal(t, 4, response.ParentBLast30Days)
	assert.Equal(t, fairness.CaregiverTypeParent.String(), response.CaregiverType)
}

func TestHandleGetAssignmentDetails_BabysitterAssignment(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	date := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
	assignment, err := tracker.RecordAssignment("Alice", date, false, fairness.DecisionReasonAlternating)
	require.NoError(t, err)
	require.NoError(t, tracker.UpdateAssignmentToBabysitter(assignment.ID, "Dawn", true))

	req := httptest.NewRequest(http.MethodGet, "/api/assignment-details?assignment_id="+strconv.FormatInt(assignment.ID, 10), nil)
	w := httptest.NewRecorder()

	handler.handleGetAssignmentDetails(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response AssignmentDetailsResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, fairness.CaregiverTypeBabysitter.String(), response.CaregiverType)
	assert.Equal(t, "Dawn", response.ParentName)
}

func TestHandleGetAssignmentDetails_NotFound(t *testing.T) {
	// Setup
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	// Create request for non-existent assignment
	req := httptest.NewRequest(http.MethodGet, "/api/assignment-details?assignment_id=999", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.handleGetAssignmentDetails(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Assignment details not found", response["error"])
}

func TestHandleGetAssignmentDetails_MissingParameter(t *testing.T) {
	// Setup
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	// Create request without assignment_id parameter
	req := httptest.NewRequest(http.MethodGet, "/api/assignment-details", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.handleGetAssignmentDetails(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Missing assignment_id parameter", response["error"])
}

func TestHandleGetAssignmentDetails_InvalidID(t *testing.T) {
	// Setup
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	// Create request with invalid assignment_id
	req := httptest.NewRequest(http.MethodGet, "/api/assignment-details?assignment_id=invalid", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.handleGetAssignmentDetails(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid assignment_id format", response["error"])
}

func TestHandleGetAssignmentDetails_Unauthenticated(t *testing.T) {
	// Setup
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, false) // Not authenticated
	defer cleanup()

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/assignment-details?assignment_id=123", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.handleGetAssignmentDetails(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["error"])
}

func TestHandleGetAssignmentDetails_WrongMethod(t *testing.T) {
	// Setup
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	// Create POST request (should only accept GET)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-details?assignment_id=123", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.handleGetAssignmentDetails(w, req)

	// Assert
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleSetAssignmentBabysitter_Success(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	date := testCurrentDate()
	assignment, err := tracker.RecordAssignment("Alice", date, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignment.ID, 10) + `,"babysitter_name":"Dawn"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-babysitter", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSetAssignmentBabysitter(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	updated, err := tracker.GetAssignmentByID(assignment.ID)
	require.NoError(t, err)
	assert.Equal(t, fairness.CaregiverTypeBabysitter, updated.CaregiverType)
	assert.Equal(t, "Dawn", updated.Parent)
}

func TestHandleSetAssignmentBabysitter_InvalidPayload(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/assignment-babysitter", bytes.NewBufferString("bad json"))
	w := httptest.NewRecorder()

	handler.handleSetAssignmentBabysitter(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetAssignmentBabysitter_TriggersScheduleRecalculation(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	recordingSvc := &recordingCalendarService{}
	handler.CalendarService = recordingSvc

	date := testCurrentDate()
	assignment, err := tracker.RecordAssignment("Alice", date, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignment.ID, 10) + `,"babysitter_name":"Dawn"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-babysitter", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSetAssignmentBabysitter(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, recordingSvc.syncCalls, "setting babysitter should trigger schedule recalculation/sync")
}

func TestHandleSetAssignmentBabysitter_Unauthenticated(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, false)
	defer cleanup()

	payload := []byte(`{"assignment_id":1,"babysitter_name":"Dawn"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-babysitter", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSetAssignmentBabysitter(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleSetAssignmentBabysitter_WrongMethod(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/assignment-babysitter", nil)
	w := httptest.NewRecorder()

	handler.handleSetAssignmentBabysitter(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleSetAssignmentBabysitter_MissingFields(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	tests := []struct {
		name    string
		payload string
	}{
		{"zero assignment_id", `{"assignment_id":0,"babysitter_name":"Dawn"}`},
		{"empty babysitter_name", `{"assignment_id":1,"babysitter_name":""}`},
		{"whitespace babysitter_name", `{"assignment_id":1,"babysitter_name":"   "}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/assignment-babysitter", bytes.NewBufferString(tt.payload))
			w := httptest.NewRecorder()
			handler.handleSetAssignmentBabysitter(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandleSetAssignmentBabysitter_NotFound(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	payload := []byte(`{"assignment_id":99999,"babysitter_name":"Dawn"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-babysitter", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSetAssignmentBabysitter(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSetAssignmentBabysitter_NameTooLong(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	longName := strings.Repeat("a", 81)
	payload := []byte(`{"assignment_id":1,"babysitter_name":"` + longName + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-babysitter", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSetAssignmentBabysitter(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "maximum length")
}

func TestHandleSetAssignmentBabysitter_PastThreshold(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	// Create an assignment far in the past (beyond the configured 5-day threshold)
	oldDate := testCurrentDate().AddDate(0, 0, -30)
	assignment, err := tracker.RecordAssignment("Alice", oldDate, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignment.ID, 10) + `,"babysitter_name":"Dawn"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-babysitter", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSetAssignmentBabysitter(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "too far in the past")
}

func TestHandleSwapAssignments_Success(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	handler.CalendarService = &recordingCalendarService{}

	now := testCurrentDate()
	dateA := now.AddDate(0, 0, 1)
	dateB := now.AddDate(0, 0, 2)

	assignmentA, err := tracker.RecordAssignment("Alice", dateA, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)
	assignmentB, err := tracker.RecordAssignment("Bob", dateB, false, fairness.DecisionReasonAlternating)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignmentA.ID, 10) + `,"swap_with_date":"` + dateB.Format("2006-01-02") + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Both rows swapped: dateA now carries Bob, dateB carries Alice. The
	// swap is recorded as an override so schedule recalculation keeps it.
	updatedA, err := tracker.GetAssignmentByDate(dateA)
	require.NoError(t, err)
	require.NotNil(t, updatedA)
	assert.Equal(t, "Bob", updatedA.Parent)
	assert.Equal(t, fairness.DecisionReasonManualSwap, updatedA.DecisionReason)
	assert.Equal(t, fairness.CaregiverTypeParent, updatedA.CaregiverType)
	assert.True(t, updatedA.Override)

	updatedB, err := tracker.GetAssignmentByDate(dateB)
	require.NoError(t, err)
	require.NotNil(t, updatedB)
	assert.Equal(t, "Alice", updatedB.Parent)
	assert.Equal(t, fairness.DecisionReasonManualSwap, updatedB.DecisionReason)
	assert.Equal(t, fairness.CaregiverTypeParent, updatedB.CaregiverType)
	assert.True(t, updatedB.Override)

	_ = assignmentB
}

func TestHandleSwapAssignments_Unauthenticated(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, false)
	defer cleanup()

	payload := []byte(`{"assignment_id":1,"swap_with_date":"2025-01-16"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleSwapAssignments_WrongMethod(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/assignment-swap", nil)
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleSwapAssignments_InvalidPayload(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewBufferString("bad json"))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSwapAssignments_MissingFields(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	tests := []struct {
		name    string
		payload string
	}{
		{"zero assignment_id", `{"assignment_id":0,"swap_with_date":"2025-01-16"}`},
		{"empty swap_with_date", `{"assignment_id":1,"swap_with_date":""}`},
		{"whitespace swap_with_date", `{"assignment_id":1,"swap_with_date":"   "}`},
		{"bad date format", `{"assignment_id":1,"swap_with_date":"16-01-2025"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewBufferString(tt.payload))
			w := httptest.NewRecorder()
			handler.handleSwapAssignments(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandleSwapAssignments_SourceNotFound(t *testing.T) {
	handler, _, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	payload := []byte(`{"assignment_id":99999,"swap_with_date":"2025-01-16"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSwapAssignments_PastThreshold(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	// Source assignment far in the past (beyond the configured 5-day threshold)
	oldDate := testCurrentDate().AddDate(0, 0, -30)
	assignment, err := tracker.RecordAssignment("Alice", oldDate, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignment.ID, 10) + `,"swap_with_date":"` + testCurrentDate().Format("2006-01-02") + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "too far in the past")
}

func TestHandleSwapAssignments_TargetDatePastThreshold(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	now := testCurrentDate()
	targetDate := now.AddDate(0, 0, -30)

	assignment, err := tracker.RecordAssignment("Alice", now, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", targetDate, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignment.ID, 10) + `,"swap_with_date":"` + targetDate.Format("2006-01-02") + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "too far in the past")
}

func TestHandleSwapAssignments_TargetNotFound(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	now := testCurrentDate()
	assignment, err := tracker.RecordAssignment("Alice", now.AddDate(0, 0, 1), false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	// Target date has no assignment at all.
	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignment.ID, 10) + `,"swap_with_date":"` + now.AddDate(0, 0, 3).Format("2006-01-02") + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSwapAssignments_SameDate(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	now := testCurrentDate()
	assignment, err := tracker.RecordAssignment("Alice", now.AddDate(0, 0, 1), false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignment.ID, 10) + `,"swap_with_date":"` + assignment.Date.Format("2006-01-02") + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSwapAssignments_RejectsBabysitterAssignment(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	now := testCurrentDate()

	babysitterAssignment, err := tracker.RecordAssignment("Alice", now.AddDate(0, 0, 1), false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)
	require.NoError(t, tracker.UpdateAssignmentToBabysitter(babysitterAssignment.ID, "Dawn", true))

	// Create the target assignment as a normal parent assignment.
	_, err = tracker.RecordAssignment("Bob", now.AddDate(0, 0, 2), false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(babysitterAssignment.ID, 10) + `,"swap_with_date":"` + now.AddDate(0, 0, 2).Format("2006-01-02") + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "parent assignments")
}

func TestHandleSwapAssignments_RejectsUnavailableParent(t *testing.T) {
	handler, tracker, db, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	now := testCurrentDate()
	dateA := now.AddDate(0, 0, 1)
	dateB := now.AddDate(0, 0, 2)

	// Alice (parent_a) takes dateA, Bob (parent_b) takes dateB.
	// After the swap Alice would take dateB — make parent_a unavailable on
	// dateB's weekday so the swap must be rejected.
	_, err := tracker.RecordAssignment("Alice", dateA, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", dateB, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	cfgStore, err := database.NewConfigStore(db)
	require.NoError(t, err)
	err = cfgStore.SaveAvailability("parent_a", []string{dateB.Format("Monday")})
	require.NoError(t, err)

	// Fetch the real assignment ID for dateA.
	assignmentA, err := tracker.GetAssignmentByDate(dateA)
	require.NoError(t, err)
	require.NotNil(t, assignmentA)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignmentA.ID, 10) + `,"swap_with_date":"` + dateB.Format("2006-01-02") + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "unavailable")

	// Nothing should have been swapped.
	keptA, err := tracker.GetAssignmentByDate(dateA)
	require.NoError(t, err)
	assert.Equal(t, "Alice", keptA.Parent)
	keptB, err := tracker.GetAssignmentByDate(dateB)
	require.NoError(t, err)
	assert.Equal(t, "Bob", keptB.Parent)
}

func TestHandleSwapAssignments_TriggersScheduleRecalculation(t *testing.T) {
	handler, tracker, _, cleanup := setupTestAssignmentDetailsHandler(t, true)
	defer cleanup()

	recordingSvc := &recordingCalendarService{}
	handler.CalendarService = recordingSvc

	now := testCurrentDate()
	dateA := now.AddDate(0, 0, 1)
	dateB := now.AddDate(0, 0, 2)

	assignmentA, err := tracker.RecordAssignment("Alice", dateA, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", dateB, false, fairness.DecisionReasonAlternating)
	require.NoError(t, err)

	payload := []byte(`{"assignment_id":` + strconv.FormatInt(assignmentA.ID, 10) + `,"swap_with_date":"` + dateB.Format("2006-01-02") + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assignment-swap", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.handleSwapAssignments(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, recordingSvc.syncCalls, "swapping assignments should trigger schedule recalculation/sync")
}
