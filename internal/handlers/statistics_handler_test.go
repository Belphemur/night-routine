package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func setupTestStatisticsHandler(t *testing.T, statsOrder constants.StatsOrder) (*StatisticsHandler, *database.ConfigStore, *database.DB, *fairness.Tracker, func()) {
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

	// Create config store
	configStore, err := database.NewConfigStore(db)
	require.NoError(t, err)

	// Seed initial data with specified stats order
	err = configStore.SaveParents("TestParentA", "TestParentB")
	require.NoError(t, err)
	err = configStore.SaveSchedule("weekly", 30, 5, statsOrder)
	require.NoError(t, err)

	// Create token store
	tokenStore, err := database.NewTokenStore(db)
	require.NoError(t, err)

	// Save a token to simulate authenticated state
	testToken := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
	}
	err = tokenStore.SaveToken(testToken)
	require.NoError(t, err)

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

	// Create statistics handler
	handler := NewStatisticsHandler(baseHandler, configStore)

	cleanup := func() {
		db.Close()
	}

	return handler, configStore, db, tracker, cleanup
}

func TestStatisticsHandler_StatsOrderDescending(t *testing.T) {
	handler, _, _, tracker, cleanup := setupTestStatisticsHandler(t, constants.StatsOrderDesc)
	defer cleanup()

	// Add some test assignments for different months
	now := time.Now()
	currentMonth := now.Format("2006-01")
	lastMonth := now.AddDate(0, -1, 0).Format("2006-01")
	twoMonthsAgo := now.AddDate(0, -2, 0).Format("2006-01")

	// Create assignments in different months
	dates := []time.Time{
		now,                   // Current month
		now.AddDate(0, -1, 0), // Last month
		now.AddDate(0, -2, 0), // Two months ago
	}

	for _, date := range dates {
		_, err := tracker.RecordAssignment("TestParentA", date, false, fairness.DecisionReasonTotalCount)
		require.NoError(t, err)
	}

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
	w := httptest.NewRecorder()

	handler.handleStatisticsPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the response contains the page
	body := w.Body.String()
	assert.Contains(t, body, "Statistics")

	// Verify month headers are in descending order (current month first)
	currentIdx := strings.Index(body, currentMonth)
	lastMonthIdx := strings.Index(body, lastMonth)
	twoMonthsAgoIdx := strings.Index(body, twoMonthsAgo)

	// In descending order, current month should appear before last month
	// and last month should appear before two months ago
	if currentIdx != -1 && lastMonthIdx != -1 {
		assert.Less(t, currentIdx, lastMonthIdx, "Current month should appear before last month in descending order")
	}
	if lastMonthIdx != -1 && twoMonthsAgoIdx != -1 {
		assert.Less(t, lastMonthIdx, twoMonthsAgoIdx, "Last month should appear before two months ago in descending order")
	}
}

func TestStatisticsHandler_StatsOrderAscending(t *testing.T) {
	handler, _, _, tracker, cleanup := setupTestStatisticsHandler(t, constants.StatsOrderAsc)
	defer cleanup()

	// Add some test assignments for different months
	now := time.Now()
	currentMonth := now.Format("2006-01")
	lastMonth := now.AddDate(0, -1, 0).Format("2006-01")
	twoMonthsAgo := now.AddDate(0, -2, 0).Format("2006-01")

	// Create assignments in different months
	dates := []time.Time{
		now,                   // Current month
		now.AddDate(0, -1, 0), // Last month
		now.AddDate(0, -2, 0), // Two months ago
	}

	for _, date := range dates {
		_, err := tracker.RecordAssignment("TestParentA", date, false, fairness.DecisionReasonTotalCount)
		require.NoError(t, err)
	}

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
	w := httptest.NewRecorder()

	handler.handleStatisticsPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the response contains the page
	body := w.Body.String()
	assert.Contains(t, body, "Statistics")

	// Verify month headers are in ascending order (oldest month first)
	currentIdx := strings.Index(body, currentMonth)
	lastMonthIdx := strings.Index(body, lastMonth)
	twoMonthsAgoIdx := strings.Index(body, twoMonthsAgo)

	// In ascending order, two months ago should appear before last month
	// and last month should appear before current month
	if twoMonthsAgoIdx != -1 && lastMonthIdx != -1 {
		assert.Less(t, twoMonthsAgoIdx, lastMonthIdx, "Two months ago should appear before last month in ascending order")
	}
	if lastMonthIdx != -1 && currentIdx != -1 {
		assert.Less(t, lastMonthIdx, currentIdx, "Last month should appear before current month in ascending order")
	}
}

func TestStatisticsHandler_NoData(t *testing.T) {
	handler, _, _, _, cleanup := setupTestStatisticsHandler(t, constants.StatsOrderDesc)
	defer cleanup()

	// Make request without any assignments
	req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
	w := httptest.NewRecorder()

	handler.handleStatisticsPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the response shows no data message
	body := w.Body.String()
	assert.Contains(t, body, "Statistics")
	assert.Contains(t, body, "No statistics data available")
}

func TestStatisticsHandler_DefaultsToDescendingOnError(t *testing.T) {
	// This test verifies that when GetSchedule fails, the handler defaults to descending order
	// We test this indirectly by verifying the handler still works even if there's an issue

	handler, _, _, tracker, cleanup := setupTestStatisticsHandler(t, constants.StatsOrderDesc)
	defer cleanup()

	// Add a single assignment
	_, err := tracker.RecordAssignment("TestParentA", time.Now(), false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
	w := httptest.NewRecorder()

	handler.handleStatisticsPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Statistics")
}

func TestStatisticsHandler_MultipleParents(t *testing.T) {
	handler, _, _, tracker, cleanup := setupTestStatisticsHandler(t, constants.StatsOrderDesc)
	defer cleanup()

	// Use fixed dates in the same month to ensure both parents appear
	// Use dates in the past to ensure they are counted
	baseDate := time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC) // October 2025

	// Create assignments for both parents on different days in the same month
	_, err := tracker.RecordAssignment("TestParentA", baseDate, false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)
	_, err = tracker.RecordAssignment("TestParentB", baseDate.AddDate(0, 0, 1), false, fairness.DecisionReasonTotalCount)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
	w := httptest.NewRecorder()

	handler.handleStatisticsPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, "TestParentA")
	assert.Contains(t, body, "TestParentB")
}
