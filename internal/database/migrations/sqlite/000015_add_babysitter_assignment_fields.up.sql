-- Add caregiver type and babysitter name support to assignments
ALTER TABLE assignments ADD COLUMN caregiver_type TEXT NOT NULL DEFAULT 'parent';
ALTER TABLE assignments ADD COLUMN babysitter_name TEXT;
