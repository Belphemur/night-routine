package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// noopCalendarService is a minimal CalendarService stub that does nothing.
type noopCalendarService struct{}

func (n *noopCalendarService) Initialize(_ context.Context) error               { return nil }
func (n *noopCalendarService) IsInitialized() bool                              { return true }
func (n *noopCalendarService) SetupNotificationChannel(_ context.Context) error { return nil }
func (n *noopCalendarService) SyncSchedule(_ context.Context, _ []*Scheduler.Assignment) error {
	return nil
}
func (n *noopCalendarService) StopNotificationChannel(_ context.Context, _, _ string) error {
	return nil
}
func (n *noopCalendarService) StopAllNotificationChannels(_ context.Context) error { return nil }
func (n *noopCalendarService) VerifyNotificationChannel(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

// noopConfigStore is a minimal ConfigStoreInterface stub that returns safe defaults.
type noopConfigStore struct{}

func (n *noopConfigStore) GetParents() (string, string, error) { return "ParentA", "ParentB", nil }
func (n *noopConfigStore) GetAvailability(_ string) ([]string, error) {
	return []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}, nil
}
func (n *noopConfigStore) GetSchedule() (string, int, int, constants.StatsOrder, error) {
	return "daily", 30, 7, constants.StatsOrderDesc, nil
}
func (n *noopConfigStore) GetOAuthConfig() *oauth2.Config { return &oauth2.Config{} }

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
	oauthCfg := &oauth2.Config{}

	// Create token manager
	tokenManager := token.NewTokenManager(tokenStore, oauthCfg)

	// Create config adapter — single source of truth for all config reads.
	// The unlock handler doesn't need any live config, but BaseHandler requires it.
	// Use an empty in-memory store; no DB is needed for OAuth/schedule here.
	configAdapter := database.NewConfigAdapter(nil, oauthCfg)

	// Create base handler
	baseHandler, err := NewBaseHandler(configAdapter, tokenStore, tokenManager, tracker, "test-version", "test-logo-version")
	require.NoError(t, err)

	// Create unlock handler with a real lightweight scheduler backed by a minimal config.
	// ParentA/ParentB must match names used in test assignments.
	fileConfig := &config.Config{
		Parents: config.ParentsConfig{ParentA: "ParentA", ParentB: "ParentB"},
	}
	sched := Scheduler.New(fileConfig, tracker)
	handler := NewUnlockHandler(baseHandler, tracker, sched, &noopCalendarService{}, &noopConfigStore{})

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

func TestUnlockHandler_HandleUnlock_BabysitterOverrideSuccess(t *testing.T) {
	handler, tracker, _, cleanup := setupTestUnlockHandler(t, true)
	defer cleanup()

	// Create an overridden assignment and convert it to babysitter to mirror UI flow.
	assignment, err := tracker.RecordAssignment("ParentA", time.Now(), true, fairness.DecisionReasonOverride)
	require.NoError(t, err)

	err = tracker.UpdateAssignmentToBabysitter(assignment.ID, "Dawn", true)
	require.NoError(t, err)

	formData := url.Values{}
	formData.Set("assignment_id", strconv.FormatInt(assignment.ID, 10))

	req := httptest.NewRequest(http.MethodPost, "/unlock", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUnlock(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "success="+SuccessCodeAssignmentUnlocked)

	updatedAssignment, err := tracker.GetAssignmentByID(assignment.ID)
	require.NoError(t, err)
	// UnlockAssignment clears override, babysitter metadata, and restores parent caregiver type.
	// The subsequent schedule recalculation overwrites parent_name with a real parent.
	assert.False(t, updatedAssignment.Override)
	assert.Equal(t, fairness.CaregiverTypeParent, updatedAssignment.CaregiverType)
	assert.Empty(t, updatedAssignment.BabysitterName)
	// After unlock + recalculation, parent_name is reassigned by the scheduler (not the stale babysitter name).
	assert.Equal(t, "ParentA", updatedAssignment.Parent)
}
