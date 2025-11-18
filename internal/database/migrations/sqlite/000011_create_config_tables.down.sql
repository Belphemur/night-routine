-- Drop indexes
DROP INDEX IF EXISTS idx_config_availability_parent;

-- Drop tables in reverse order
DROP TABLE IF EXISTS config_schedule;
DROP TABLE IF EXISTS config_availability;
DROP TABLE IF EXISTS config_parents;
