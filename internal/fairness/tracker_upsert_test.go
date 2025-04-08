package fairness

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestUpsertBehavior specifically tests the ON CONFLICT behavior of RecordAssignment
func TestUpsertBehavior(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	// Create a common date to use for all assignments to trigger conflicts
	date := time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC)

	// Step 1: Create initial assignment
	assignment1, err := tracker.RecordAssignment("Alice", date, false, "Initial Assignment")
	assert.NoError(t, err)
	assert.NotNil(t, assignment1)
	assert.Equal(t, "Alice", assignment1.Parent)
	assert.Equal(t, "Initial Assignment", assignment1.DecisionReason)
	assert.False(t, assignment1.Override)
	initialID := assignment1.ID

	// Step 2: Update the same date with a different parent (should update existing record)
	assignment2, err := tracker.RecordAssignment("Bob", date, false, "Updated Assignment")
	assert.NoError(t, err)
	assert.NotNil(t, assignment2)
	assert.Equal(t, initialID, assignment2.ID, "ID should remain the same on conflict") // Confirm same record
	assert.Equal(t, "Bob", assignment2.Parent, "Parent should be updated")
	assert.Equal(t, "Updated Assignment", assignment2.DecisionReason, "Decision reason should be updated")
	assert.False(t, assignment2.Override)

	// Step 3: Update with override flag set to true
	assignment3, err := tracker.RecordAssignment("Charlie", date, true, "Overridden Assignment")
	assert.NoError(t, err)
	assert.NotNil(t, assignment3)
	assert.Equal(t, initialID, assignment3.ID, "ID should remain the same on conflict")
	assert.Equal(t, "Charlie", assignment3.Parent, "Parent should be updated")
	assert.Equal(t, "Overridden Assignment", assignment3.DecisionReason, "Decision reason should be updated")
	assert.True(t, assignment3.Override, "Override flag should be updated")

	// Step 4: Update back to false for override
	assignment4, err := tracker.RecordAssignment("Alice", date, false, "Final Assignment")
	assert.NoError(t, err)
	assert.NotNil(t, assignment4)
	assert.Equal(t, initialID, assignment4.ID, "ID should remain the same on conflict")
	assert.Equal(t, "Alice", assignment4.Parent, "Parent should be updated")
	assert.Equal(t, "Final Assignment", assignment4.DecisionReason, "Decision reason should be updated")
	assert.False(t, assignment4.Override, "Override flag should be updated to false")

	// Step 5: Verify that we only have ONE record for this date in the database
	// (to ensure we're truly updating, not inserting)
	var count int
	err = db.Conn().QueryRow(`SELECT COUNT(*) FROM assignments WHERE assignment_date = ?`, date.Format(dateFormat)).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "There should be exactly one assignment record for this date")
}

// TestConflictPreservesMetadata tests that the UPSERT preserves metadata like creation timestamp
func TestConflictPreservesMetadata(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tracker, err := New(db)
	assert.NoError(t, err)

	date := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)

	// Create initial assignment
	assignment1, err := tracker.RecordAssignment("Alice", date, false, "Initial")
	assert.NoError(t, err)
	initialCreatedAt := assignment1.CreatedAt

	// Small delay to ensure timestamps would be different if re-created
	time.Sleep(1 * time.Second)

	// Update the same assignment
	assignment2, err := tracker.RecordAssignment("Bob", date, true, "Updated")
	assert.NoError(t, err)

	// Check that created_at is preserved but updated_at is changed
	assert.Equal(t, initialCreatedAt.Unix(), assignment2.CreatedAt.Unix(), "CreatedAt timestamp should be preserved")
	assert.True(t, assignment2.UpdatedAt.After(initialCreatedAt), "UpdatedAt should be newer than original CreatedAt")

	// Verify record was updated not re-created
	assert.Equal(t, assignment1.ID, assignment2.ID, "Record ID should not change")
}
