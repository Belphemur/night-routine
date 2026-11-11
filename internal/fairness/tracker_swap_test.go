package fairness

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSwapAssignmentsSwapsParentsAndReason verifies that SwapAssignments swaps
// the parents of two existing parent assignments and that both rows read back
// with the decision reason provided to the swap.
func TestSwapAssignmentsSwapsParentsAndReason(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	require.NoError(t, err)

	dateA := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	dateB := time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC)

	assignmentA, err := tracker.RecordAssignment("Alice", dateA, false, DecisionReasonTotalCount)
	require.NoError(t, err)
	require.NotNil(t, assignmentA)

	assignmentB, err := tracker.RecordAssignment("Bob", dateB, false, DecisionReasonAlternating)
	require.NoError(t, err)
	require.NotNil(t, assignmentB)

	// Swap: Bob takes dateA's night, Alice takes dateB's night. The first
	// parent/date pair describes who is assigned where after the swap. The
	// swap is recorded as an override (explicit user intent, like the web UI
	// manual swap) so it survives schedule recalculation.
	updatedA, updatedB, err := tracker.SwapAssignments("Bob", dateA, "Alice", dateB, DecisionReasonManualSwap, true)
	require.NoError(t, err)
	require.NotNil(t, updatedA)
	require.NotNil(t, updatedB)

	// Returned records reflect the post-swap state.
	assert.Equal(t, "Bob", updatedA.Parent)
	assert.Equal(t, DecisionReasonManualSwap, updatedA.DecisionReason)
	assert.Equal(t, CaregiverTypeParent, updatedA.CaregiverType)
	assert.True(t, updatedA.Override)

	assert.Equal(t, "Alice", updatedB.Parent)
	assert.Equal(t, DecisionReasonManualSwap, updatedB.DecisionReason)
	assert.Equal(t, CaregiverTypeParent, updatedB.CaregiverType)
	assert.True(t, updatedB.Override)

	// Read both rows back from the database after the transaction committed.
	readBackA, err := tracker.GetAssignmentByDate(dateA)
	require.NoError(t, err)
	require.NotNil(t, readBackA)
	assert.Equal(t, assignmentA.ID, readBackA.ID, "dateA should keep its original assignment row")
	assert.Equal(t, "Bob", readBackA.Parent)
	assert.Equal(t, DecisionReasonManualSwap, readBackA.DecisionReason)
	assert.Equal(t, CaregiverTypeParent, readBackA.CaregiverType)
	assert.True(t, readBackA.Override)

	readBackB, err := tracker.GetAssignmentByDate(dateB)
	require.NoError(t, err)
	require.NotNil(t, readBackB)
	assert.Equal(t, assignmentB.ID, readBackB.ID, "dateB should keep its original assignment row")
	assert.Equal(t, "Alice", readBackB.Parent)
	assert.Equal(t, DecisionReasonManualSwap, readBackB.DecisionReason)
	assert.Equal(t, CaregiverTypeParent, readBackB.CaregiverType)
	assert.True(t, readBackB.Override)
}

// TestSwapAssignmentsWithoutOverride verifies the algorithm-internal smoothing
// usage: swapped rows are upserted as non-override parent assignments so the
// fairness algorithm stays free to rebalance them on future recalculations.
func TestSwapAssignmentsWithoutOverride(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	require.NoError(t, err)

	dateA := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	dateB := time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC)

	_, err = tracker.RecordAssignment("Alice", dateA, false, DecisionReasonTotalCount)
	require.NoError(t, err)
	_, err = tracker.RecordAssignment("Bob", dateB, false, DecisionReasonAlternating)
	require.NoError(t, err)

	updatedA, updatedB, err := tracker.SwapAssignments("Bob", dateA, "Alice", dateB, DecisionReasonDoubleConsecutiveSwap, false)
	require.NoError(t, err)
	require.NotNil(t, updatedA)
	require.NotNil(t, updatedB)

	assert.False(t, updatedA.Override)
	assert.False(t, updatedB.Override)
	assert.Equal(t, DecisionReasonDoubleConsecutiveSwap, updatedA.DecisionReason)
	assert.Equal(t, DecisionReasonDoubleConsecutiveSwap, updatedB.DecisionReason)

	readBackA, err := tracker.GetAssignmentByDate(dateA)
	require.NoError(t, err)
	require.NotNil(t, readBackA)
	assert.Equal(t, "Bob", readBackA.Parent)
	assert.False(t, readBackA.Override)
}
