package handlers

//go:generate pnpm run build:css

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/logging"
	"github.com/belphemur/night-routine/internal/token"
	"github.com/rs/zerolog"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed assets/css/*.css
var assetsFS embed.FS

// BaseHandler contains common handler functionality
type BaseHandler struct {
	tmpl          *template.Template
	TokenStore    *database.TokenStore
	TokenManager  *token.TokenManager
	RuntimeConfig *config.RuntimeConfig
	Tracker       fairness.TrackerInterface
	logger        zerolog.Logger
	cssETag       string // Cached ETag for CSS file
	cssContent    []byte // Cached CSS file content
}

// NewBaseHandler creates a common base handler with shared components
func NewBaseHandler(runtimeCfg *config.RuntimeConfig, tokenStore *database.TokenStore, tokenManager *token.TokenManager, tracker fairness.TrackerInterface) (*BaseHandler, error) {
	logger := logging.GetLogger("base-handler")
	logger.Debug().Msg("Parsing templates")

	// Define custom template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"js": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
	}

	// Parse only layout.html initially
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/layout.html")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse templates")
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	logger.Debug().Msg("Templates parsed successfully")

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

	return &BaseHandler{
		tmpl:          tmpl, // Updated field name
		TokenStore:    tokenStore,
		TokenManager:  tokenManager,
		RuntimeConfig: runtimeCfg,
		Tracker:       tracker,
		logger:        logger,
		cssETag:       etag,
		cssContent:    css,
	}, nil
}

// RenderTemplate renders a template with the given data
func (h *BaseHandler) RenderTemplate(w http.ResponseWriter, name string, data interface{}) {
	h.logger.Debug().Str("template_name", name).Msg("Executing template")

	// Clone the base template (which contains layout.html)
	tmpl, err := h.tmpl.Clone()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to clone template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse the specific page template into the clone
	_, err = tmpl.ParseFS(templateFS, "templates/"+name)
	if err != nil {
		h.logger.Error().Err(err).Str("template", name).Msg("Failed to parse page template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		h.logger.Error().Err(err).Str("template", name).Msg("Failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// CheckAuthentication checks if the user is authenticated
func (h *BaseHandler) CheckAuthentication(ctx context.Context, logger zerolog.Logger) bool {
	logger.Debug().Msg("Checking authentication status")
	hasToken, err := h.TokenManager.HasToken()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to check token existence")
		return false
	}
	if !hasToken {
		logger.Debug().Msg("No token found")
		return false
	}

	token, err := h.TokenManager.GetValidToken(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to validate token")
		return false
	}
	if token == nil {
		logger.Debug().Msg("Token is nil")
		return false
	}

	logger.Debug().Msg("User is authenticated")
	return true
}

// RegisterStaticRoutes registers static asset routes
func (h *BaseHandler) RegisterStaticRoutes() {
	http.HandleFunc("/static/css/tailwind.css", h.serveTailwindCSS)
}

// serveTailwindCSS serves the embedded Tailwind CSS file with ETag support
func (h *BaseHandler) serveTailwindCSS(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Msg("Serving Tailwind CSS")

	// Check If-None-Match header for ETag validation (RFC 7232)
	// Supports multiple ETags and wildcard '*'
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
		if h.matchesETag(ifNoneMatch) {
			h.logger.Debug().Str("if_none_match", ifNoneMatch).Msg("ETag matches - returning 304 Not Modified")
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Set cache headers and ETag
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", h.cssETag)

	if _, err := w.Write(h.cssContent); err != nil {
		h.logger.Error().Err(err).Msg("Failed to write CSS response")
	}
}

// matchesETag checks if the If-None-Match header matches the current ETag
// Supports multiple ETags separated by commas and wildcard '*' as per RFC 7232
func (h *BaseHandler) matchesETag(ifNoneMatch string) bool {
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
	var etags []string
	parts := strings.Split(header, ",")

	for _, part := range parts {
		etag := strings.TrimSpace(part)
		if etag != "" {
			etags = append(etags, etag)
		}
	}

	return etags
}

// BasePageData contains common data for all pages
type BasePageData struct {
	CurrentYear     int
	CurrentPath     string
	IsAuthenticated bool
}

// NewBasePageData creates a new BasePageData with common fields populated
func (h *BaseHandler) NewBasePageData(r *http.Request, isAuthenticated bool) BasePageData {
	return BasePageData{
		CurrentYear:     time.Now().Year(),
		CurrentPath:     r.URL.Path,
		IsAuthenticated: isAuthenticated,
	}
}
