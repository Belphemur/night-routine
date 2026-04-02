-- Remove redundant babysitter_name column; parent_name already stores the display name for all caregiver types
ALTER TABLE assignments DROP COLUMN babysitter_name;
