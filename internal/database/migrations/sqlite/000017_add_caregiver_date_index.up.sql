-- Add composite index to improve performance of stats and fairness queries that filter by caregiver_type and assignment_date
CREATE INDEX IF NOT EXISTS idx_assignments_caregiver_date ON assignments(caregiver_type, assignment_date DESC);
