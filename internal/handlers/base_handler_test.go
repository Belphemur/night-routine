package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func setupTestBaseHandler(t *testing.T) (*BaseHandler, func()) {
	// Create test database
	dbOpts := database.SQLiteOptions{
		Path:        ":memory:",
		Mode:        "memory",
		Cache:       database.CacheShared,
		Journal:     database.JournalMemory,
		ForeignKeys: true,
		BusyTimeout: 5000,
	}

	db, err := database.New(dbOpts)
	require.NoError(t, err)

	err = db.MigrateDatabase()
	require.NoError(t, err)

	// Initialize required components
	tokenStore, err := database.NewTokenStore(db)
	require.NoError(t, err)

	oauthConfig := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/oauth/callback",
	}
	tokenManager := token.NewTokenManager(tokenStore, oauthConfig)

	tracker, err := fairness.New(db)
	require.NoError(t, err)

	runtimeConfig := &config.RuntimeConfig{
		Config: &config.Config{
			Parents: config.ParentsConfig{
				ParentA: "Parent A",
				ParentB: "Parent B",
			},
		},
	}

	// Create base handler
	handler, err := NewBaseHandler(runtimeConfig, tokenStore, tokenManager, tracker)
	require.NoError(t, err)

	cleanup := func() {
		err := db.Close()
		assert.NoError(t, err)
	}

	return handler, cleanup
}

func TestServeTailwindCSS_ETag(t *testing.T) {
	handler, cleanup := setupTestBaseHandler(t)
	defer cleanup()

	require.NotEmpty(t, handler.cssETag, "ETag should be calculated during initialization")

	t.Run("Initial request returns ETag", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		w := httptest.NewRecorder()

		handler.serveTailwindCSS(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/css; charset=utf-8", w.Header().Get("Content-Type"))
		assert.NotEmpty(t, w.Header().Get("ETag"), "ETag header should be present")
		assert.Equal(t, handler.cssETag, w.Header().Get("ETag"))
		assert.NotEmpty(t, w.Body.Bytes(), "CSS content should be present")
	})

	t.Run("Request with matching ETag returns 304", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		req.Header.Set("If-None-Match", handler.cssETag)
		w := httptest.NewRecorder()

		handler.serveTailwindCSS(w, req)

		assert.Equal(t, http.StatusNotModified, w.Code)
		assert.Empty(t, w.Body.Bytes(), "No content should be returned for 304")
	})

	t.Run("Request with non-matching ETag returns full content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		req.Header.Set("If-None-Match", "invalid-etag")
		w := httptest.NewRecorder()

		handler.serveTailwindCSS(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, handler.cssETag, w.Header().Get("ETag"))
		assert.NotEmpty(t, w.Body.Bytes(), "CSS content should be present")
	})

	t.Run("Cache-Control header is set correctly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		w := httptest.NewRecorder()

		handler.serveTailwindCSS(w, req)

		assert.Equal(t, "public, max-age=31536000, immutable", w.Header().Get("Cache-Control"))
	})
}
