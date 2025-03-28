-- Add index on parent_name column to improve query performance when filtering by parent
CREATE INDEX IF NOT EXISTS idx_assignments_parent_name ON assignments(parent_name);