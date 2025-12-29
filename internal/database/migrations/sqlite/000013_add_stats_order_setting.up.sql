-- Add stats_order setting to config_schedule table
-- Default to 'desc' (showing current month first)
-- Valid values: 'desc', 'asc'
ALTER TABLE config_schedule ADD COLUMN stats_order TEXT NOT NULL DEFAULT 'desc' CHECK (stats_order IN ('desc', 'asc'));
