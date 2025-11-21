package handlers

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/belphemur/night-routine/internal/logging"
	"github.com/rs/zerolog"
)

//go:embed assets/css/*.css
var assetsFS embed.FS

// StaticHandler manages static file serving with ETag support
type StaticHandler struct {
	logger     zerolog.Logger
	cssETag    string // Cached ETag for CSS file
	cssContent []byte // Cached CSS file content
}

// NewStaticHandler creates a new static file handler
func NewStaticHandler() (*StaticHandler, error) {
	logger := logging.GetLogger("static-handler")

	// Pre-load and cache CSS file with ETag
	css, err := assetsFS.ReadFile("assets/css/tailwind.css")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read Tailwind CSS for ETag calculation")
		return nil, fmt.Errorf("failed to read CSS file: %w", err)
	}

	// Calculate SHA-256 hash for ETag (quoted as per RFC 7232)
	hash := sha256.Sum256(css)
	etag := fmt.Sprintf("\"%s\"", hex.EncodeToString(hash[:]))
	logger.Debug().Str("etag", etag).Int("content_size", len(css)).Msg("Cached CSS file with ETag")

	return &StaticHandler{
		logger:     logger,
		cssETag:    etag,
		cssContent: css,
	}, nil
}

// RegisterRoutes registers static asset routes
func (h *StaticHandler) RegisterRoutes() {
	http.HandleFunc("/static/css/tailwind.css", h.serveTailwindCSS)
}

// serveTailwindCSS serves the embedded Tailwind CSS file with ETag support
func (h *StaticHandler) serveTailwindCSS(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Msg("Serving Tailwind CSS")

	// Set ETag header first (required for both 200 and 304 responses per RFC 7232)
	w.Header().Set("ETag", h.cssETag)

	// Check If-None-Match header for ETag validation (RFC 7232)
	// Supports multiple ETags and wildcard '*'
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
		if h.matchesETag(ifNoneMatch) {
			h.logger.Debug().Str("if_none_match", ifNoneMatch).Msg("ETag matches - returning 304 Not Modified")
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Set remaining cache headers for 200 response
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=43200, must-revalidate")

	if _, err := w.Write(h.cssContent); err != nil {
		h.logger.Error().Err(err).Msg("Failed to write CSS response")
	}
}

// matchesETag checks if the If-None-Match header matches the current ETag
// Supports multiple ETags separated by commas and wildcard '*' as per RFC 7232
func (h *StaticHandler) matchesETag(ifNoneMatch string) bool {
	// Handle wildcard
	if ifNoneMatch == "*" {
		return true
	}

	// Parse comma-separated ETags
	// Simple implementation that handles quoted and unquoted ETags
	etags := parseETags(ifNoneMatch)
	for _, etag := range etags {
		if etag == h.cssETag {
			return true
		}
	}
	return false
}

// parseETags parses comma-separated ETags from If-None-Match header
// This is a simplified implementation that handles the common case of
// comma-separated quoted ETags. For full RFC 7232 compliance with
// escaped quotes, a more sophisticated parser would be needed.
func parseETags(header string) []string {
	parts := strings.Split(header, ",")
	// Pre-allocate slice with capacity to reduce allocations
	etags := make([]string, 0, len(parts))

	for _, part := range parts {
		etag := strings.TrimSpace(part)
		if etag != "" {
			etags = append(etags, etag)
		}
	}

	return etags
}
