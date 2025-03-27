-- Remove the trigger first
DROP TRIGGER IF EXISTS assignments_update_trigger;

-- Drop the columns using ALTER TABLE (requires SQLite 3.35.0+)
ALTER TABLE assignments DROP COLUMN updated_at;
ALTER TABLE assignments DROP COLUMN override;