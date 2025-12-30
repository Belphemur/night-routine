package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/belphemur/night-routine/internal/fairness"
)

// AssignmentDetailsHandler handles requests for assignment fairness calculation details
type AssignmentDetailsHandler struct {
	*BaseHandler
	Tracker fairness.TrackerInterface
}

// NewAssignmentDetailsHandler creates a new assignment details handler
func NewAssignmentDetailsHandler(baseHandler *BaseHandler, tracker fairness.TrackerInterface) *AssignmentDetailsHandler {
	return &AssignmentDetailsHandler{
		BaseHandler: baseHandler,
		Tracker:     tracker,
	}
}

// RegisterRoutes registers assignment details related routes
func (h *AssignmentDetailsHandler) RegisterRoutes() {
	http.HandleFunc("/api/assignment-details", h.handleGetAssignmentDetails)
}

// AssignmentDetailsResponse represents the JSON response for assignment details
type AssignmentDetailsResponse struct {
	AssignmentID      int64  `json:"assignment_id"`
	CalculationDate   string `json:"calculation_date"`
	DecisionReason    string `json:"decision_reason"`
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
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Assignment not found"}); err != nil {
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
		ParentAName:       details.ParentAName,
		ParentATotalCount: details.ParentATotalCount,
		ParentALast30Days: details.ParentALast30Days,
		ParentBName:       details.ParentBName,
		ParentBTotalCount: details.ParentBTotalCount,
		ParentBLast30Days: details.ParentBLast30Days,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		handlerLogger.Error().Err(err).Msg("Failed to encode response")
	}

	handlerLogger.Info().Msg("Successfully returned assignment details")
}
