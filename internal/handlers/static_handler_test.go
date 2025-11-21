package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeTailwindCSS_ETag(t *testing.T) {
	handler, err := NewStaticHandler()
	require.NoError(t, err)
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

	t.Run("ETag is properly quoted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		w := httptest.NewRecorder()

		handler.serveTailwindCSS(w, req)

		etag := w.Header().Get("ETag")
		assert.True(t, len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"',
			"ETag should be quoted as per RFC 7232")
	})

	t.Run("Request with matching ETag returns 304", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		req.Header.Set("If-None-Match", handler.cssETag)
		w := httptest.NewRecorder()

		handler.serveTailwindCSS(w, req)

		assert.Equal(t, http.StatusNotModified, w.Code)
		assert.Empty(t, w.Body.Bytes(), "No content should be returned for 304")
		assert.Equal(t, handler.cssETag, w.Header().Get("ETag"), "ETag header should be present in 304 response per RFC 7232")
	})

	t.Run("Request with wildcard ETag returns 304", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		req.Header.Set("If-None-Match", "*")
		w := httptest.NewRecorder()

		handler.serveTailwindCSS(w, req)

		assert.Equal(t, http.StatusNotModified, w.Code)
		assert.Empty(t, w.Body.Bytes(), "No content should be returned for 304 with wildcard")
	})

	t.Run("Request with multiple ETags including matching one returns 304", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		req.Header.Set("If-None-Match", `"other-etag", `+handler.cssETag+`, "yet-another"`)
		w := httptest.NewRecorder()

		handler.serveTailwindCSS(w, req)

		assert.Equal(t, http.StatusNotModified, w.Code)
		assert.Empty(t, w.Body.Bytes(), "No content should be returned for 304 when one ETag matches")
	})

	t.Run("Request with non-matching ETag returns full content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/tailwind.css", nil)
		req.Header.Set("If-None-Match", `"invalid-etag"`)
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

		assert.Equal(t, "public, max-age=43200, must-revalidate", w.Header().Get("Cache-Control"))
	})
}
