package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/rs/zerolog"
)

// AssignmentDetailsHandler handles requests for assignment fairness calculation details
type AssignmentDetailsHandler struct {
	*BaseHandler
	Tracker         fairness.TrackerInterface
	Scheduler       Scheduler.SchedulerInterface
	CalendarService calendar.CalendarService
	ConfigStore     config.ConfigStoreInterface
}

// NewAssignmentDetailsHandler creates a new assignment details handler
func NewAssignmentDetailsHandler(baseHandler *BaseHandler, tracker fairness.TrackerInterface, sched Scheduler.SchedulerInterface, calSvc calendar.CalendarService, configStore config.ConfigStoreInterface) *AssignmentDetailsHandler {
	return &AssignmentDetailsHandler{
		BaseHandler:     baseHandler,
		Tracker:         tracker,
		Scheduler:       sched,
		CalendarService: calSvc,
		ConfigStore:     configStore,
	}
}

// RegisterRoutes registers assignment details related routes
func (h *AssignmentDetailsHandler) RegisterRoutes() {
	http.HandleFunc("/api/assignment-details", h.handleGetAssignmentDetails)
	http.HandleFunc("/api/assignment-babysitter", h.handleSetAssignmentBabysitter)
	http.HandleFunc("/api/assignment-swap", h.handleSwapAssignments)
}

// AssignmentDetailsResponse represents the JSON response for assignment details
type AssignmentDetailsResponse struct {
	AssignmentID      int64  `json:"assignment_id"`
	CalculationDate   string `json:"calculation_date"`
	DecisionReason    string `json:"decision_reason"`
	CaregiverType     string `json:"caregiver_type"`
	ParentName        string `json:"parent_name,omitempty"`
	ParentAName       string `json:"parent_a_name"`
	ParentATotalCount int    `json:"parent_a_total_count"`
	ParentALast30Days int    `json:"parent_a_last_30_days"`
	ParentBName       string `json:"parent_b_name"`
	ParentBTotalCount int    `json:"parent_b_total_count"`
	ParentBLast30Days int    `json:"parent_b_last_30_days"`
}

// handleGetAssignmentDetails handles GET requests for assignment details
func (h *AssignmentDetailsHandler) handleGetAssignmentDetails(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleGetAssignmentDetails").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling get assignment details request")

	if r.Method != http.MethodGet {
		handlerLogger.Warn().Msg("Invalid method for get assignment details request")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.CheckAuthentication(r.Context(), handlerLogger) {
		handlerLogger.Warn().Msg("Unauthenticated access attempt to assignment details")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode unauthorized response")
		}
		return
	}

	assignmentIDStr := r.URL.Query().Get("assignment_id")
	if assignmentIDStr == "" {
		handlerLogger.Warn().Msg("No assignment_id provided")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Missing assignment_id parameter"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode bad request response")
		}
		return
	}

	assignmentID, err := strconv.ParseInt(assignmentIDStr, 10, 64)
	if err != nil {
		handlerLogger.Error().Err(err).Str("assignment_id_str", assignmentIDStr).Msg("Invalid assignment ID format")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid assignment_id format"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode bad request response")
		}
		return
	}

	handlerLogger = handlerLogger.With().Int64("assignment_id", assignmentID).Logger()
	handlerLogger.Debug().Msg("Fetching assignment and details")

	// First get the assignment to retrieve the decision reason
	assignment, err := h.Tracker.GetAssignmentByID(assignmentID)
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get assignment")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve assignment"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode error response")
		}
		return
	}

	if assignment == nil {
		handlerLogger.Debug().Msg("Assignment not found")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Assignment details not found"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode not found response")
		}
		return
	}

	// Then get the assignment details
	details, err := h.Tracker.GetAssignmentDetails(assignmentID)
	if err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to get assignment details")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve assignment details"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode error response")
		}
		return
	}

	if details == nil {
		if assignment.CaregiverType == fairness.CaregiverTypeBabysitter {
			response := AssignmentDetailsResponse{
				AssignmentID:   assignment.ID,
				DecisionReason: assignment.DecisionReason.String(),
				CaregiverType:  assignment.CaregiverType.String(),
				ParentName:     assignment.Parent,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				handlerLogger.Error().Err(err).Msg("Failed to encode babysitter details response")
			}

			handlerLogger.Info().Msg("Returned babysitter assignment details without fairness snapshot")
			return
		}

		handlerLogger.Debug().Msg("No details found for assignment")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Assignment details not found"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode not found response")
		}
		return
	}

	response := AssignmentDetailsResponse{
		AssignmentID:      details.AssignmentID,
		CalculationDate:   details.CalculationDate.Format("2006-01-02"),
		DecisionReason:    assignment.DecisionReason.String(),
		CaregiverType:     assignment.CaregiverType.String(),
		ParentAName:       details.ParentAName,
		ParentATotalCount: details.ParentATotalCount,
		ParentALast30Days: details.ParentALast30Days,
		ParentBName:       details.ParentBName,
		ParentBTotalCount: details.ParentBTotalCount,
		ParentBLast30Days: details.ParentBLast30Days,
	}
	if assignment.CaregiverType == fairness.CaregiverTypeBabysitter {
		response.ParentName = assignment.Parent
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to encode response")
	}

	handlerLogger.Info().Msg("Successfully returned assignment details")
}

type setBabysitterRequest struct {
	AssignmentID   int64  `json:"assignment_id"`
	BabysitterName string `json:"babysitter_name"`
}

func (h *AssignmentDetailsHandler) handleSetAssignmentBabysitter(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleSetAssignmentBabysitter").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling set assignment babysitter request")

	if r.Method != http.MethodPost {
		handlerLogger.Warn().Msg("Invalid method for set assignment babysitter request")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.CheckAuthentication(r.Context(), handlerLogger) {
		handlerLogger.Warn().Msg("Unauthenticated access attempt to set babysitter")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode unauthorized response")
		}
		return
	}

	var req setBabysitterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to decode set babysitter payload")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"}); encErr != nil {
			handlerLogger.Error().Err(encErr).Msg("Failed to encode bad request response")
		}
		return
	}

	req.BabysitterName = strings.TrimSpace(req.BabysitterName)
	if req.AssignmentID <= 0 || req.BabysitterName == "" {
		handlerLogger.Warn().Int64("assignment_id", req.AssignmentID).Msg("Invalid assignment id or babysitter name")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "assignment_id and babysitter_name are required"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode validation error response")
		}
		return
	}

	const maxBabysitterNameLen = 80
	if len(req.BabysitterName) > maxBabysitterNameLen {
		handlerLogger.Warn().Int("name_len", len(req.BabysitterName)).Msg("Babysitter name exceeds maximum length")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "babysitter_name exceeds maximum length"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode validation error response")
		}
		return
	}

	assignment, err := h.Tracker.GetAssignmentByID(req.AssignmentID)
	if err != nil {
		handlerLogger.Error().Err(err).Int64("assignment_id", req.AssignmentID).Msg("Failed to get assignment")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve assignment"}); encErr != nil {
			handlerLogger.Error().Err(encErr).Msg("Failed to encode server error response")
		}
		return
	}

	if assignment == nil {
		handlerLogger.Warn().Int64("assignment_id", req.AssignmentID).Msg("Assignment not found")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Assignment not found"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode not found response")
		}
		return
	}

	// Enforce the same past-event threshold used by the webhook handler to prevent
	// modification of historical assignments that should remain fixed for fairness.
	_, _, thresholdDays, _, schedErr := h.ConfigStore.GetSchedule()
	if schedErr != nil {
		handlerLogger.Error().Err(schedErr).Msg("Failed to get schedule configuration for threshold check")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate assignment date"}); encErr != nil {
			handlerLogger.Error().Err(encErr).Msg("Failed to encode server error response")
		}
		return
	}

	now := time.Now()
	thresholdDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -thresholdDays)
	y, m, d := assignment.Date.Date()
	assignmentDate := time.Date(y, m, d, 0, 0, 0, 0, now.Location())

	if assignmentDate.Before(thresholdDate) {
		handlerLogger.Warn().
			Int("threshold_days", thresholdDays).
			Str("assignment_date", assignmentDate.Format("2006-01-02")).
			Msg("Rejecting babysitter assignment for past assignment outside threshold")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Assignment is too far in the past to modify"}); err != nil {
			handlerLogger.Error().Err(err).Msg("Failed to encode threshold error response")
		}
		return
	}

	if err := h.Tracker.UpdateAssignmentToBabysitter(req.AssignmentID, req.BabysitterName, true); err != nil {
		handlerLogger.Error().Err(err).Int64("assignment_id", req.AssignmentID).Msg("Failed to update assignment to babysitter")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Failed to set babysitter"}); encErr != nil {
			handlerLogger.Error().Err(encErr).Msg("Failed to encode server error response")
		}
		return
	}

	// Keep calendar and future assignments coherent after introducing a babysitter override.
	if err := h.recalculateSchedule(r.Context(), assignment.Date); err != nil {
		handlerLogger.Error().Err(err).Int64("assignment_id", req.AssignmentID).Msg("Failed to recalculate schedule after setting babysitter")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to encode success response")
	}
}

type swapAssignmentsRequest struct {
	AssignmentID int64  `json:"assignment_id"`
	SwapWithDate string `json:"swap_with_date"`
}

// containsString reports whether the slice contains the given value.
func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// writeJSONError writes a JSON error response with the given status code.
func (h *AssignmentDetailsHandler) writeJSONError(w http.ResponseWriter, logger zerolog.Logger, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		logger.Error().Err(err).Msg("Failed to encode error response")
	}
}

func (h *AssignmentDetailsHandler) handleSwapAssignments(w http.ResponseWriter, r *http.Request) {
	handlerLogger := h.logger.With().Str("handler", "handleSwapAssignments").Logger()
	handlerLogger.Info().Str("method", r.Method).Msg("Handling swap assignments request")

	if r.Method != http.MethodPost {
		handlerLogger.Warn().Msg("Invalid method for swap assignments request")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// CSRF mitigation: a cross-site attacker can only submit "simple" requests
	// (text/plain, form encodings) without a CORS preflight. Requiring the
	// application/json media type forces any cross-origin call through
	// preflight, which this server never answers, so forged cross-site POSTs
	// are rejected by the browser before they reach the handler.
	contentType := r.Header.Get("Content-Type")
	mediaType, _, parseErr := mime.ParseMediaType(contentType)
	if parseErr != nil || mediaType != "application/json" {
		handlerLogger.Warn().Str("content_type", contentType).Msg("Rejecting swap request with unsupported content type")
		h.writeJSONError(w, handlerLogger, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	if !h.CheckAuthentication(r.Context(), handlerLogger) {
		handlerLogger.Warn().Msg("Unauthenticated access attempt to swap assignments")
		h.writeJSONError(w, handlerLogger, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req swapAssignmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to decode swap assignments payload")
		h.writeJSONError(w, handlerLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AssignmentID <= 0 || strings.TrimSpace(req.SwapWithDate) == "" {
		handlerLogger.Warn().Int64("assignment_id", req.AssignmentID).Msg("Invalid assignment id or swap date")
		h.writeJSONError(w, handlerLogger, http.StatusBadRequest, "assignment_id and swap_with_date are required")
		return
	}

	targetDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.SwapWithDate))
	if err != nil {
		handlerLogger.Warn().Str("swap_with_date", req.SwapWithDate).Msg("Invalid swap_with_date format")
		h.writeJSONError(w, handlerLogger, http.StatusBadRequest, "swap_with_date must be a valid date in YYYY-MM-DD format")
		return
	}

	assignment, err := h.Tracker.GetAssignmentByID(req.AssignmentID)
	if err != nil {
		handlerLogger.Error().Err(err).Int64("assignment_id", req.AssignmentID).Msg("Failed to get assignment")
		h.writeJSONError(w, handlerLogger, http.StatusInternalServerError, "Failed to retrieve assignment")
		return
	}

	if assignment == nil {
		handlerLogger.Warn().Int64("assignment_id", req.AssignmentID).Msg("Assignment not found")
		h.writeJSONError(w, handlerLogger, http.StatusNotFound, "Assignment not found")
		return
	}

	// Enforce the same past-event threshold used by the babysitter and webhook
	// handlers: neither leg of the swap may be outside the threshold.
	_, _, thresholdDays, _, schedErr := h.ConfigStore.GetSchedule()
	if schedErr != nil {
		handlerLogger.Error().Err(schedErr).Msg("Failed to get schedule configuration for threshold check")
		h.writeJSONError(w, handlerLogger, http.StatusInternalServerError, "Failed to validate assignment dates")
		return
	}

	now := time.Now()
	thresholdDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -thresholdDays)

	y, m, d := assignment.Date.Date()
	sourceDate := time.Date(y, m, d, 0, 0, 0, 0, now.Location())

	// Normalize the parsed target date to midnight in the same location as
	// the source date so equality and Before comparisons compare calendar
	// dates, not wall-clock instants across time zones (time.Parse returns
	// UTC).
	ty, tm, td := targetDate.Date()
	targetDate = time.Date(ty, tm, td, 0, 0, 0, 0, now.Location())

	if sourceDate.Before(thresholdDate) {
		handlerLogger.Warn().
			Int("threshold_days", thresholdDays).
			Str("assignment_date", sourceDate.Format("2006-01-02")).
			Msg("Rejecting swap for past assignment outside threshold")
		h.writeJSONError(w, handlerLogger, http.StatusBadRequest, "Assignment is too far in the past to modify")
		return
	}

	if targetDate.Before(thresholdDate) {
		handlerLogger.Warn().
			Int("threshold_days", thresholdDays).
			Str("swap_with_date", targetDate.Format("2006-01-02")).
			Msg("Rejecting swap with target date outside threshold")
		h.writeJSONError(w, handlerLogger, http.StatusBadRequest, "Swap target date is too far in the past to modify")
		return
	}

	if sourceDate.Equal(targetDate) {
		handlerLogger.Warn().Str("swap_with_date", targetDate.Format("2006-01-02")).Msg("Swap target date equals the assignment date")
		h.writeJSONError(w, handlerLogger, http.StatusBadRequest, "Cannot swap an assignment with itself")
		return
	}

	targetAssignment, err := h.Tracker.GetAssignmentByDate(targetDate)
	if err != nil {
		handlerLogger.Error().Err(err).Str("swap_with_date", targetDate.Format("2006-01-02")).Msg("Failed to get target assignment")
		h.writeJSONError(w, handlerLogger, http.StatusInternalServerError, "Failed to retrieve target assignment")
		return
	}

	if targetAssignment == nil {
		handlerLogger.Warn().Str("swap_with_date", targetDate.Format("2006-01-02")).Msg("Target assignment not found")
		h.writeJSONError(w, handlerLogger, http.StatusNotFound, "No assignment exists on the target date")
		return
	}

	if assignment.CaregiverType != fairness.CaregiverTypeParent || targetAssignment.CaregiverType != fairness.CaregiverTypeParent {
		handlerLogger.Warn().
			Int64("assignment_id", req.AssignmentID).
			Str("source_type", assignment.CaregiverType.String()).
			Str("target_type", targetAssignment.CaregiverType.String()).
			Msg("Rejecting swap involving non-parent assignment")
		h.writeJSONError(w, handlerLogger, http.StatusBadRequest, "Both assignments must be parent assignments to swap")
		return
	}

	// A swap between two nights that already belong to the same parent would
	// change nothing yet still lock both rows as overrides, freezing them
	// against future fairness recalculations. Refuse it instead.
	if assignment.Parent == targetAssignment.Parent {
		handlerLogger.Warn().
			Int64("assignment_id", req.AssignmentID).
			Str("parent", assignment.Parent).
			Msg("Rejecting swap between two nights assigned to the same parent")
		h.writeJSONError(w, handlerLogger, http.StatusBadRequest, "Both nights are already assigned to the same parent")
		return
	}

	// After the swap the source date carries the target assignment's parent and
	// vice versa. Verify neither parent is unavailable on the date it receives.
	parentA, parentB, parentsErr := h.ConfigStore.GetParents()
	if parentsErr != nil {
		handlerLogger.Error().Err(parentsErr).Msg("Failed to get parent names")
		h.writeJSONError(w, handlerLogger, http.StatusInternalServerError, "Failed to validate parent availability")
		return
	}

	// After the swap the source date carries the target assignment's parent
	// and the target date carries the source assignment's parent — verify the
	// post-swap pairing, mirroring the scheduler's double-consecutive swap.
	if err := h.verifySwapAvailability(targetAssignment.Parent, sourceDate, assignment.Parent, targetDate, parentA, parentB, &handlerLogger); err != nil {
		if apiErr, ok := err.(*swapAvailabilityError); ok {
			h.writeJSONError(w, handlerLogger, http.StatusBadRequest, apiErr.Error())
		} else {
			h.writeJSONError(w, handlerLogger, http.StatusInternalServerError, "Failed to validate parent availability")
		}
		return
	}

	// SwapAssignments(parentA, dateA, ...) assigns parentA to dateA and parentB
	// to dateB, mirroring the scheduler's double-consecutive smoothing. The
	// swap is recorded as an override (explicit user intent) so the
	// recalculation below respects it instead of re-deriving both days.
	_, _, swapErr := h.Tracker.SwapAssignments(
		targetAssignment.Parent, sourceDate,
		assignment.Parent, targetDate,
		fairness.DecisionReasonManualSwap, true,
	)
	if swapErr != nil {
		handlerLogger.Error().Err(swapErr).
			Int64("assignment_id", req.AssignmentID).
			Str("swap_with_date", targetDate.Format("2006-01-02")).
			Msg("Failed to swap assignments")
		h.writeJSONError(w, handlerLogger, http.StatusInternalServerError, "Failed to swap assignments")
		return
	}

	// Keep calendar and future assignments coherent after the manual swap.
	// Recalculate from the EARLIER of the two dates so the swapped target
	// assignment itself is covered by the regeneration and calendar sync when
	// the user swapped with an earlier night.
	fromDate := sourceDate
	if targetDate.Before(fromDate) {
		fromDate = targetDate
	}
	if err := h.recalculateSchedule(r.Context(), fromDate); err != nil {
		handlerLogger.Error().Err(err).Int64("assignment_id", req.AssignmentID).Msg("Failed to recalculate schedule after swapping assignments")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to encode success response")
	}

	handlerLogger.Info().Msg("Successfully swapped assignments")
}

// swapAvailabilityError signals a user-facing validation failure (a parent is
// unavailable on a date they would receive in the swap).
type swapAvailabilityError struct {
	parent        string
	formattedDate string
}

func (e *swapAvailabilityError) Error() string {
	return fmt.Sprintf("%s is unavailable on %s", e.parent, e.formattedDate)
}

// verifySwapAvailability checks that the parent receiving each date in the swap
// is not unavailable on that date, mirroring the scheduler's
// isParentAvailableOnDate day-of-week constraint logic. A non-nil non-API error
// indicates a configuration lookup failure, not a validation rejection.
func (h *AssignmentDetailsHandler) verifySwapAvailability(parentOnSourceDate string, sourceDate time.Time, parentOnTargetDate string, targetDate time.Time, parentA, parentB string, logger *zerolog.Logger) error {
	type availabilityCheck struct {
		parent     string
		date       time.Time
		roleColumn string
	}

	roleFor := func(parent string) string {
		if parent == parentA {
			return "parent_a"
		}
		return "parent_b"
	}

	checks := []availabilityCheck{
		{parent: parentOnSourceDate, date: sourceDate, roleColumn: roleFor(parentOnSourceDate)},
		{parent: parentOnTargetDate, date: targetDate, roleColumn: roleFor(parentOnTargetDate)},
	}

	for _, check := range checks {
		unavailableDays, err := h.ConfigStore.GetAvailability(check.roleColumn)
		if err != nil {
			logger.Error().Err(err).Str("parent_role", check.roleColumn).Msg("Failed to get parent availability")
			return fmt.Errorf("failed to get availability for %s: %w", check.roleColumn, err)
		}
		dayOfWeek := check.date.Format("Monday")
		if containsString(unavailableDays, dayOfWeek) {
			logger.Warn().
				Str("parent", check.parent).
				Str("date", check.date.Format("2006-01-02")).
				Str("day", dayOfWeek).
				Msg("Swap rejected: parent unavailable on target date")
			return &swapAvailabilityError{parent: check.parent, formattedDate: check.date.Format("2006-01-02")}
		}
	}

	return nil
}

func (h *AssignmentDetailsHandler) recalculateSchedule(ctx context.Context, fromDate time.Time) error {
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
