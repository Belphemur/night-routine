package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
)

// UnlockHandler manages unlocking of overridden assignments by removing the override flag,
// allowing them to be re-evaluated by the scheduler.
type UnlockHandler struct {
	*BaseHandler
	Tracker         fairness.TrackerInterface
	Scheduler       Scheduler.SchedulerInterface
	CalendarService calendar.CalendarService
	ConfigStore     config.ConfigStoreInterface
}

// NewUnlockHandler creates a new unlock handler
func NewUnlockHandler(baseHandler *BaseHandler, tracker fairness.TrackerInterface, sched Scheduler.SchedulerInterface, calSvc calendar.CalendarService, configStore config.ConfigStoreInterface) *UnlockHandler {
	return &UnlockHandler{
		BaseHandler:     baseHandler,
		Tracker:         tracker,
		Scheduler:       sched,
		CalendarService: calSvc,
		ConfigStore:     configStore,
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

	if !h.CheckAuthentication(r.Context(), handlerLogger) {
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

	assignment, err := h.Tracker.GetAssignmentByID(assignmentID)
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get assignment")
		http.Redirect(w, r, "/?error="+ErrCodeUnlockFailed, http.StatusSeeOther)
		return
	}

	if assignment == nil {
		handlerLogger.Warn().Msg("Assignment not found")
		http.Redirect(w, r, "/?error="+ErrCodeInvalidAssignmentID, http.StatusSeeOther)
		return
	}

	// Check if the assignment is actually overridden
	if !assignment.Override {
		handlerLogger.Warn().Bool("override", assignment.Override).Msg("Attempted to unlock non-overridden assignment")
		http.Redirect(w, r, "/?error="+ErrCodeNotOverridden, http.StatusSeeOther)
		return
	}

	if err := h.Tracker.UnlockAssignment(assignmentID); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to unlock assignment")
		http.Redirect(w, r, "/?error="+ErrCodeUnlockFailed, http.StatusSeeOther)
		return
	}

	handlerLogger.Info().Msg("Assignment unlocked successfully, triggering schedule recalculation")

	// Recalculate and sync the schedule so the calendar reflects the removal of the override.
	if err := h.recalculateSchedule(r.Context(), assignment.Date); err != nil {
		// Log but don't fail the redirect — the DB is already correct.
		handlerLogger.Error().Err(err).Msg("Failed to recalculate schedule after unlock")
	}

	http.Redirect(w, r, "/?success="+SuccessCodeAssignmentUnlocked, http.StatusSeeOther)
}

// recalculateSchedule regenerates and syncs the schedule starting from the given date.
func (h *UnlockHandler) recalculateSchedule(ctx context.Context, fromDate time.Time) error {
	return recalculateScheduleAndSync(
		ctx,
		h.logger,
		h.Tracker,
		h.Scheduler,
		h.CalendarService,
		h.ConfigStore,
		fromDate,
	)
}
