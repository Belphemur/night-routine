package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/database"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/token"
)

// BaseHandler contains common handler functionality
type BaseHandler struct {
	Templates    *template.Template
	TokenStore   *database.TokenStore
	TokenManager *token.TokenManager
	Config       *config.Config
	Tracker      *fairness.Tracker
}

// NewBaseHandler creates a common base handler with shared components
func NewBaseHandler(cfg *config.Config, tokenStore *database.TokenStore, tokenManager *token.TokenManager, tracker *fairness.Tracker) (*BaseHandler, error) {
	tmpl, err := template.New("").ParseGlob("internal/handlers/templates/*.html")
	if err != nil {
		return nil, err
	}

	return &BaseHandler{
		Templates:    tmpl,
		TokenStore:   tokenStore,
		TokenManager: tokenManager,
		Config:       cfg,
		Tracker:      tracker,
	}, nil
}

// RenderTemplate is a helper method to render HTML templates
func (h *BaseHandler) RenderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.Templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}
