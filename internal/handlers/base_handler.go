package handlers

import (
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

// BaseHandler contains common handler functionality
type BaseHandler struct {
	Templates    *template.Template
	TokenStore   *database.TokenStore
	TokenManager *token.TokenManager
	Config       *config.Config
	Tracker      fairness.TrackerInterface
	logger       zerolog.Logger
}

// NewBaseHandler creates a common base handler with shared components
func NewBaseHandler(cfg *config.Config, tokenStore *database.TokenStore, tokenManager *token.TokenManager, tracker fairness.TrackerInterface) (*BaseHandler, error) {
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
		Templates:    tmpl,
		TokenStore:   tokenStore,
		TokenManager: tokenManager,
		Config:       cfg,
		Tracker:      tracker,
		logger:       logger,
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
