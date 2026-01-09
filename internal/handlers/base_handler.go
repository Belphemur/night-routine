package handlers

//go:generate pnpm run build

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
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

// BaseHandler contains common handler functionality
type BaseHandler struct {
	tmpl          *template.Template
	TokenStore    *database.TokenStore
	TokenManager  *token.TokenManager
	RuntimeConfig *config.RuntimeConfig
	Tracker       fairness.TrackerInterface
	cssVersion    string
	logoVersion   string
	logger        zerolog.Logger
}

// NewBaseHandler creates a common base handler with shared components
func NewBaseHandler(runtimeCfg *config.RuntimeConfig, tokenStore *database.TokenStore, tokenManager *token.TokenManager, tracker fairness.TrackerInterface, cssVersion, logoVersion string) (*BaseHandler, error) {
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

	return &BaseHandler{
		tmpl:          tmpl, // Updated field name
		TokenStore:    tokenStore,
		TokenManager:  tokenManager,
		RuntimeConfig: runtimeCfg,
		Tracker:       tracker,
		cssVersion:    cssVersion,
		logoVersion:   logoVersion,
		logger:        logger,
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

// BasePageData contains common data for all pages
type BasePageData struct {
	CurrentYear     int
	CurrentPath     string
	IsAuthenticated bool
	CSSETag         string
	LogoETag        string
}

// NewBasePageData creates a new BasePageData with common fields populated
func (h *BaseHandler) NewBasePageData(r *http.Request, isAuthenticated bool) BasePageData {
	return BasePageData{
		CurrentYear:     time.Now().Year(),
		CurrentPath:     r.URL.Path,
		IsAuthenticated: isAuthenticated,
		CSSETag:         h.cssVersion,
		LogoETag:        h.logoVersion,
	}
}
