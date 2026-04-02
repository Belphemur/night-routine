-- Remove babysitter-related fields from assignments
ALTER TABLE assignments DROP COLUMN babysitter_name;
ALTER TABLE assignments DROP COLUMN caregiver_type;
