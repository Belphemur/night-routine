-- Add index on google_calendar_event_id column to improve query performance
CREATE INDEX IF NOT EXISTS idx_assignments_gcal_event_id ON assignments(google_calendar_event_id);