package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
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

func setupTestUnlockHandler(t *testing.T, authenticated bool) (*UnlockHandler, *fairness.Tracker, *database.DB, func()) {
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
	baseHandler, err := NewBaseHandler(runtimeCfg, tokenStore, tokenManager, tracker, "test-version")
	require.NoError(t, err)

	// Create unlock handler
	handler := NewUnlockHandler(baseHandler, tracker)

	cleanup := func() {
		db.Close()
	}

	return handler, tracker, db, cleanup
}

func TestUnlockHandler_HandleUnlock_Unauthenticated(t *testing.T) {
	handler, _, _, cleanup := setupTestUnlockHandler(t, false)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/unlock", nil)
	w := httptest.NewRecorder()

	handler.handleUnlock(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error="+ErrCodeUnauthorized)
}

func TestUnlockHandler_HandleUnlock_InvalidMethod(t *testing.T) {
	handler, _, _, cleanup := setupTestUnlockHandler(t, true)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/unlock", nil)
	w := httptest.NewRecorder()

	handler.handleUnlock(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestUnlockHandler_HandleUnlock_MissingAssignmentID(t *testing.T) {
	handler, _, _, cleanup := setupTestUnlockHandler(t, true)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/unlock", nil)
	w := httptest.NewRecorder()

	handler.handleUnlock(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error="+ErrCodeMissingAssignmentID)
}

func TestUnlockHandler_HandleUnlock_InvalidAssignmentID(t *testing.T) {
	handler, _, _, cleanup := setupTestUnlockHandler(t, true)
	defer cleanup()

	formData := url.Values{}
	formData.Set("assignment_id", "invalid")

	req := httptest.NewRequest(http.MethodPost, "/unlock", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUnlock(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error="+ErrCodeInvalidAssignmentID)
}

func TestUnlockHandler_HandleUnlock_AssignmentNotFound(t *testing.T) {
	handler, _, _, cleanup := setupTestUnlockHandler(t, true)
	defer cleanup()

	formData := url.Values{}
	formData.Set("assignment_id", "999")

	req := httptest.NewRequest(http.MethodPost, "/unlock", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUnlock(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error="+ErrCodeInvalidAssignmentID)
}

func TestUnlockHandler_HandleUnlock_NotOverridden(t *testing.T) {
	handler, tracker, _, cleanup := setupTestUnlockHandler(t, true)
	defer cleanup()

	// Create a non-overridden assignment
	assignment, err := tracker.RecordAssignment("ParentA", time.Now(), false, fairness.DecisionReasonAlternating)
	require.NoError(t, err)

	formData := url.Values{}
	formData.Set("assignment_id", strconv.FormatInt(assignment.ID, 10))

	req := httptest.NewRequest(http.MethodPost, "/unlock", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUnlock(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error="+ErrCodeNotOverridden)
}

func TestUnlockHandler_HandleUnlock_Success(t *testing.T) {
	handler, tracker, _, cleanup := setupTestUnlockHandler(t, true)
	defer cleanup()

	// Create an overridden assignment
	assignment, err := tracker.RecordAssignment("ParentA", time.Now(), true, fairness.DecisionReasonOverride)
	require.NoError(t, err)

	formData := url.Values{}
	formData.Set("assignment_id", strconv.FormatInt(assignment.ID, 10))

	req := httptest.NewRequest(http.MethodPost, "/unlock", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUnlock(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "success="+SuccessCodeAssignmentUnlocked)

	// Verify it is no longer overridden
	updatedAssignment, err := tracker.GetAssignmentByID(assignment.ID)
	require.NoError(t, err)
	assert.False(t, updatedAssignment.Override)
}
