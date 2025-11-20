package handlers

//go:generate npm run build:css

import (
	"context"
	"embed"
	"html/template"
	"net/http"

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
	Templates     *template.Template
	TokenStore    *database.TokenStore
	TokenManager  *token.TokenManager
	RuntimeConfig *config.RuntimeConfig
	Tracker       fairness.TrackerInterface
	logger        zerolog.Logger
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
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse templates")
		return nil, err
	}
	logger.Debug().Msg("Templates parsed successfully")

	return &BaseHandler{
		Templates:     tmpl,
		TokenStore:    tokenStore,
		TokenManager:  tokenManager,
		RuntimeConfig: runtimeCfg,
		Tracker:       tracker,
		logger:        logger,
	}, nil
}

// RenderTemplate is a helper method to render HTML templates
func (h *BaseHandler) RenderTemplate(w http.ResponseWriter, name string, data interface{}) {
	// Add template name to logger context for this call
	renderLogger := h.logger.With().Str("template_name", name).Logger()
	renderLogger.Debug().Msg("Executing template")
	if err := h.Templates.ExecuteTemplate(w, name, data); err != nil {
		renderLogger.Error().Err(err).Msg("Template execution error")
		// Avoid writing partial templates if header hasn't been written
		if w.Header().Get("Content-Type") == "" {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
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

// serveTailwindCSS serves the embedded Tailwind CSS file
func (h *BaseHandler) serveTailwindCSS(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().Msg("Serving Tailwind CSS")

	css, err := assetsFS.ReadFile("assets/css/tailwind.css")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to read Tailwind CSS")
		http.Error(w, "CSS file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if _, err := w.Write(css); err != nil {
		h.logger.Error().Err(err).Msg("Failed to write CSS response")
	}
}
