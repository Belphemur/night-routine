package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func setupTestSettingsHandler(t *testing.T) (*SettingsHandler, *database.ConfigStore, *database.DB, func()) {
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

	// Seed initial data
	err = configStore.SaveParents("TestParentA", "TestParentB")
	require.NoError(t, err)
	err = configStore.SaveAvailability("parent_a", []string{"Monday"})
	require.NoError(t, err)
	err = configStore.SaveAvailability("parent_b", []string{"Friday"})
	require.NoError(t, err)
	err = configStore.SaveSchedule("weekly", 30, 5)
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

	// Create base handler
	baseHandler, err := NewBaseHandler(cfg, tokenStore, tokenManager, tracker)
	require.NoError(t, err)

	// Create settings handler
	handler := NewSettingsHandler(baseHandler, configStore)

	cleanup := func() {
		db.Close()
	}

	return handler, configStore, db, cleanup
}

func TestSettingsHandler_HandleSettings_Authenticated(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	handler.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	handler.handleSettings(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "TestParentA")
	assert.Contains(t, w.Body.String(), "TestParentB")
	assert.Contains(t, w.Body.String(), "weekly")
}

func TestSettingsHandler_HandleSettings_WithErrors(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/settings?error=test+error", nil)
	w := httptest.NewRecorder()

	handler.handleSettings(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test error")
}

func TestSettingsHandler_HandleSettings_WithSuccess(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/settings?success=test+success", nil)
	w := httptest.NewRecorder()

	handler.handleSettings(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test success")
}

func TestSettingsHandler_HandleUpdateSettings_Success(t *testing.T) {
	handler, configStore, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("parent_a", "NewParentA")
	formData.Set("parent_b", "NewParentB")
	formData.Add("parent_a_unavailable", "Tuesday")
	formData.Add("parent_a_unavailable", "Thursday")
	formData.Add("parent_b_unavailable", "Wednesday")
	formData.Set("update_frequency", "daily")
	formData.Set("look_ahead_days", "14")
	formData.Set("past_event_threshold_days", "3")

	req := httptest.NewRequest(http.MethodPost, "/settings/update", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUpdateSettings(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/settings?success=")

	// Verify changes
	parentA, parentB, err := configStore.GetParents()
	require.NoError(t, err)
	assert.Equal(t, "NewParentA", parentA)
	assert.Equal(t, "NewParentB", parentB)

	freq, lookAhead, threshold, err := configStore.GetSchedule()
	require.NoError(t, err)
	assert.Equal(t, "daily", freq)
	assert.Equal(t, 14, lookAhead)
	assert.Equal(t, 3, threshold)
}

func TestSettingsHandler_HandleUpdateSettings_NotPost(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/settings/update", nil)
	w := httptest.NewRecorder()

	handler.handleUpdateSettings(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/settings", w.Header().Get("Location"))
}

func TestSettingsHandler_HandleUpdateSettings_InvalidFormData(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/settings/update", strings.NewReader("%"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUpdateSettings(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error=Invalid")
}

func TestSettingsHandler_HandleUpdateSettings_InvalidLookAheadDays(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("parent_a", "TestA")
	formData.Set("parent_b", "TestB")
	formData.Set("update_frequency", "daily")
	formData.Set("look_ahead_days", "invalid")
	formData.Set("past_event_threshold_days", "5")

	req := httptest.NewRequest(http.MethodPost, "/settings/update", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUpdateSettings(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error=Invalid+look+ahead")
}

func TestSettingsHandler_HandleUpdateSettings_InvalidThresholdDays(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("parent_a", "TestA")
	formData.Set("parent_b", "TestB")
	formData.Set("update_frequency", "daily")
	formData.Set("look_ahead_days", "30")
	formData.Set("past_event_threshold_days", "invalid")

	req := httptest.NewRequest(http.MethodPost, "/settings/update", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUpdateSettings(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error=Invalid+past+event")
}

func TestSettingsHandler_HandleUpdateSettings_ParentsSaveFails(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("parent_a", "Same")
	formData.Set("parent_b", "Same") // Same name will fail validation
	formData.Set("update_frequency", "daily")
	formData.Set("look_ahead_days", "30")
	formData.Set("past_event_threshold_days", "5")

	req := httptest.NewRequest(http.MethodPost, "/settings/update", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUpdateSettings(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error=Failed+to+save+parent")
}

func TestSettingsHandler_HandleUpdateSettings_ScheduleSaveFails(t *testing.T) {
	handler, _, _, cleanup := setupTestSettingsHandler(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("parent_a", "TestA")
	formData.Set("parent_b", "TestB")
	formData.Set("update_frequency", "invalid") // Invalid frequency
	formData.Set("look_ahead_days", "30")
	formData.Set("past_event_threshold_days", "5")

	req := httptest.NewRequest(http.MethodPost, "/settings/update", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUpdateSettings(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "error=Failed+to+save+schedule")
}

func TestSettingsHandler_GetAllDaysOfWeek(t *testing.T) {
	days := getAllDaysOfWeek()
	expected := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	assert.Equal(t, expected, days)
}

func TestSettingsHandler_CheckAuthentication_NoToken(t *testing.T) {
	// Create handler without token
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
	defer db.Close()

	err = db.MigrateDatabase()
	require.NoError(t, err)

	configStore, err := database.NewConfigStore(db)
	require.NoError(t, err)

	tokenStore, err := database.NewTokenStore(db)
	require.NoError(t, err)

	tracker, err := fairness.New(db)
	require.NoError(t, err)

	cfg := &config.Config{OAuth: &oauth2.Config{}}
	tokenManager := token.NewTokenManager(tokenStore, cfg.OAuth)

	baseHandler, err := NewBaseHandler(cfg, tokenStore, tokenManager, tracker)
	require.NoError(t, err)

	handler := NewSettingsHandler(baseHandler, configStore)

	// Test unauthenticated access to settings
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	handler.handleSettings(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Should render page but show not authenticated
}

func TestSettingsHandler_HandleUpdateSettings_Unauthenticated(t *testing.T) {
	// Create handler without token
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
	defer db.Close()

	err = db.MigrateDatabase()
	require.NoError(t, err)

	configStore, err := database.NewConfigStore(db)
	require.NoError(t, err)

	tokenStore, err := database.NewTokenStore(db)
	require.NoError(t, err)

	tracker, err := fairness.New(db)
	require.NoError(t, err)

	cfg := &config.Config{OAuth: &oauth2.Config{}}
	tokenManager := token.NewTokenManager(tokenStore, cfg.OAuth)

	baseHandler, err := NewBaseHandler(cfg, tokenStore, tokenManager, tracker)
	require.NoError(t, err)

	handler := NewSettingsHandler(baseHandler, configStore)

	formData := url.Values{}
	formData.Set("parent_a", "TestA")
	formData.Set("parent_b", "TestB")

	req := httptest.NewRequest(http.MethodPost, "/settings/update", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.handleUpdateSettings(w, req)

	// Should redirect to auth
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/auth", w.Header().Get("Location"))
}
