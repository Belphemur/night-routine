package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/belphemur/night-routine/internal/calendar"
	"github.com/belphemur/night-routine/internal/config"
	"github.com/belphemur/night-routine/internal/fairness"
	Scheduler "github.com/belphemur/night-routine/internal/fairness/scheduler"
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
