package fairness

// DecisionReason represents the reason for a parent assignment decision
type DecisionReason string

const (
	// DecisionReasonUnavailability represents that a parent was assigned because other parent was unavailable
	DecisionReasonUnavailability DecisionReason = "Unavailability"
	// DecisionReasonTotalCount represents that a parent was assigned due to having fewer total assignments
	DecisionReasonTotalCount DecisionReason = "Total Count"
	// DecisionReasonRecentCount represents that a parent was assigned due to having fewer recent assignments
	DecisionReasonRecentCount DecisionReason = "Recent Count"
	// DecisionReasonConsecutiveLimit represents that a parent was assigned to avoid too many consecutive assignments
	DecisionReasonConsecutiveLimit DecisionReason = "Consecutive Limit"
	// DecisionReasonAlternating represents that a parent was assigned to maintain alternating pattern
	DecisionReasonAlternating DecisionReason = "Alternating"
	// DecisionReasonOverride represents that the assignment was manually overridden
	DecisionReasonOverride DecisionReason = "Override"
)

// String returns the string representation of the DecisionReason
func (d DecisionReason) String() string {
	return string(d)
}
