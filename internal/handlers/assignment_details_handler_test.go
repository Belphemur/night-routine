package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

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
	cfg := &config.Config{
		OAuth: &oauth2.Config{},
	}

	// Create token manager
	tokenManager := token.NewTokenManager(tokenStore, cfg.OAuth)

	// Create runtime config
	runtimeCfg := &config.RuntimeConfig{
		Config: cfg,
	}

	// Create base handler
	baseHandler, err := NewBaseHandler(runtimeCfg, tokenStore, tokenManager, tracker, "test-version", "test-logo-version")
	require.NoError(t, err)

	// Create assignment details handler
	handler := NewAssignmentDetailsHandler(baseHandler, tracker)

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
