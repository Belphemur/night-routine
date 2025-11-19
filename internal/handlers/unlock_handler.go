package handlers

import (
	"net/http"
	"strconv"

	"github.com/belphemur/night-routine/internal/fairness/scheduler"
)

// UnlockHandler manages unlocking of overridden assignments by removing the override flag,
// allowing them to be re-evaluated by the scheduler.
type UnlockHandler struct {
	*BaseHandler
	Scheduler *scheduler.Scheduler
}

// NewUnlockHandler creates a new unlock handler
func NewUnlockHandler(baseHandler *BaseHandler, scheduler *scheduler.Scheduler) *UnlockHandler {
	return &UnlockHandler{
		BaseHandler: baseHandler,
		Scheduler:   scheduler,
	}
}

// RegisterRoutes registers unlock related routes
func (h *UnlockHandler) RegisterRoutes() {
	http.HandleFunc("/unlock", h.handleUnlock)
}

// handleUnlock handles the request to unlock an overridden assignment
func (h *UnlockHandler) handleUnlock(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleUnlock").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling unlock request")

	if r.Method != http.MethodPost {
		handlerLogger.Warn().Msg("Invalid method for unlock request")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.BaseHandler.CheckAuthentication(r.Context(), handlerLogger) {
		handlerLogger.Warn().Msg("Unauthenticated access attempt to unlock")
		http.Redirect(w, r, "/?error="+ErrCodeUnauthorized, http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to parse form")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	assignmentIDStr := r.FormValue("assignment_id")
	if assignmentIDStr == "" {
		handlerLogger.Warn().Msg("No assignment_id provided")
		http.Redirect(w, r, "/?error="+ErrCodeMissingAssignmentID, http.StatusSeeOther)
		return
	}

	assignmentID, err := strconv.ParseInt(assignmentIDStr, 10, 64)
	if err != nil {
		handlerLogger.Error().Err(err).Str("assignment_id_str", assignmentIDStr).Msg("Invalid assignment ID format")
		http.Redirect(w, r, "/?error="+ErrCodeInvalidAssignmentID, http.StatusSeeOther)
		return
	}

	handlerLogger = handlerLogger.With().Int64("assignment_id", assignmentID).Logger()
	handlerLogger.Debug().Msg("Attempting to unlock assignment")

	if err := h.Scheduler.UnlockAssignment(assignmentID); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to unlock assignment")
		http.Redirect(w, r, "/?error="+ErrCodeUnlockFailed, http.StatusSeeOther)
		return
	}

	handlerLogger.Info().Msg("Assignment unlocked successfully")
	http.Redirect(w, r, "/?success="+SuccessCodeAssignmentUnlocked, http.StatusSeeOther)
}
