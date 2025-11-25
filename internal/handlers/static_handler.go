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

//go:embed assets/css/*.css assets/images/*.png
var assetsFS embed.FS

// StaticHandler manages static file serving with ETag support
type StaticHandler struct {
	logger         zerolog.Logger
	cssETag        string // Cached ETag for CSS file
	cssContent     []byte // Cached CSS file content
	faviconETag    string // Cached ETag for favicon
	faviconContent []byte // Cached favicon content
	logoETag       string // Cached ETag for logo
	logoContent    []byte // Cached logo content
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

	// Calculate SHA-256 hash for CSS ETag
	cssHash := sha256.Sum256(css)
	cssETag := fmt.Sprintf("\"%s\"", hex.EncodeToString(cssHash[:]))
	logger.Debug().Str("etag", cssETag).Int("content_size", len(css)).Msg("Cached CSS file with ETag")

	// Pre-load and cache Favicon with ETag
	favicon, err := assetsFS.ReadFile("assets/images/favicon.png")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read Favicon for ETag calculation")
		return nil, fmt.Errorf("failed to read Favicon file: %w", err)
	}

	// Calculate SHA-256 hash for Favicon ETag
	faviconHash := sha256.Sum256(favicon)
	faviconETag := fmt.Sprintf("\"%s\"", hex.EncodeToString(faviconHash[:]))
	logger.Debug().Str("etag", faviconETag).Int("content_size", len(favicon)).Msg("Cached Favicon file with ETag")

	// Pre-load and cache Logo with ETag
	logo, err := assetsFS.ReadFile("assets/images/logo.png")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read Logo for ETag calculation")
		return nil, fmt.Errorf("failed to read Logo file: %w", err)
	}

	// Calculate SHA-256 hash for Logo ETag
	logoHash := sha256.Sum256(logo)
	logoETag := fmt.Sprintf("\"%s\"", hex.EncodeToString(logoHash[:]))
	logger.Debug().Str("etag", logoETag).Int("content_size", len(logo)).Msg("Cached Logo file with ETag")

	return &StaticHandler{
		logger:         logger,
		cssETag:        cssETag,
		cssContent:     css,
		faviconETag:    faviconETag,
		faviconContent: favicon,
		logoETag:       logoETag,
		logoContent:    logo,
	}, nil
}

// RegisterRoutes registers static asset routes
func (h *StaticHandler) RegisterRoutes() {
	http.HandleFunc("/static/css/tailwind.css", h.serveTailwindCSS)
	http.HandleFunc("/favicon.ico", h.serveFavicon)               // Standard browser location
	http.HandleFunc("/static/images/favicon.png", h.serveFavicon) // Explicit path
	http.HandleFunc("/static/images/logo.png", h.serveLogo)       // Logo path
}

// serveTailwindCSS serves the embedded Tailwind CSS file with ETag support
func (h *StaticHandler) serveTailwindCSS(w http.ResponseWriter, r *http.Request) {
	h.serveAsset(w, r, h.cssContent, h.cssETag, "text/css; charset=utf-8")
}

// serveFavicon serves the embedded favicon file with ETag support
func (h *StaticHandler) serveFavicon(w http.ResponseWriter, r *http.Request) {
	h.serveAsset(w, r, h.faviconContent, h.faviconETag, "image/png")
}

// serveLogo serves the embedded logo file with ETag support
func (h *StaticHandler) serveLogo(w http.ResponseWriter, r *http.Request) {
	h.serveAsset(w, r, h.logoContent, h.logoETag, "image/png")
}

// serveAsset is a helper to serve static assets with ETag support
func (h *StaticHandler) serveAsset(w http.ResponseWriter, r *http.Request, content []byte, etag string, contentType string) {
	// Set ETag header first
	w.Header().Set("ETag", etag)

	// Check If-None-Match header
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
		if h.matchesETag(ifNoneMatch, etag) {
			h.logger.Debug().Str("if_none_match", ifNoneMatch).Msg("ETag matches - returning 304 Not Modified")
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Set remaining cache headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=43200, must-revalidate")

	if _, err := w.Write(content); err != nil {
		h.logger.Error().Err(err).Msg("Failed to write response")
	}
}

// matchesETag checks if the If-None-Match header matches the current ETag
func (h *StaticHandler) matchesETag(ifNoneMatch, currentETag string) bool {
	if ifNoneMatch == "*" {
		return true
	}
	etags := parseETags(ifNoneMatch)
	for _, etag := range etags {
		if etag == currentETag {
			return true
		}
	}
	return false
}

// parseETags parses comma-separated ETags from If-None-Match header
func parseETags(header string) []string {
	parts := strings.Split(header, ",")
	etags := make([]string, 0, len(parts))
	for _, part := range parts {
		etag := strings.TrimSpace(part)
		if etag != "" {
			etags = append(etags, etag)
		}
	}
	return etags
}

// GetCSSETag returns the ETag for the CSS file, stripping quotes
func (h *StaticHandler) GetCSSETag() string {
	return strings.Trim(h.cssETag, "\"")
}

// GetFaviconETag returns the ETag for the favicon, stripping quotes
func (h *StaticHandler) GetFaviconETag() string {
	return strings.Trim(h.faviconETag, "\"")
}

// GetLogoETag returns the ETag for the logo, stripping quotes
func (h *StaticHandler) GetLogoETag() string {
	return strings.Trim(h.logoETag, "\"")
}
