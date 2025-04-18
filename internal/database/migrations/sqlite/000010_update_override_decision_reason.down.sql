-- This is a no-op down migration since we can't determine the original decision_reason values
-- The decision_reason column will remain, but we'll clear the values for overridden assignments
UPDATE assignments
SET decision_reason = NULL
WHERE override = 1 AND decision_reason = 'Override';