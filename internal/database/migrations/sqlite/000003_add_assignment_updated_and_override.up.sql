-- Creating a new migration to add updated_at and override columns to assignments table
ALTER TABLE assignments ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE assignments ADD COLUMN override BOOLEAN DEFAULT 0 NOT NULL;

-- Create a trigger to update the updated_at column when a row is updated
CREATE TRIGGER IF NOT EXISTS assignments_update_trigger
AFTER UPDATE ON assignments
FOR EACH ROW
BEGIN
    UPDATE assignments SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;