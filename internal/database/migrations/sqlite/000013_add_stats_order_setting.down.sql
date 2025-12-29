-- SQLite doesn't support DROP COLUMN directly in all versions
-- We need to recreate the table without the column
CREATE TABLE config_schedule_backup (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    update_frequency TEXT NOT NULL CHECK (update_frequency IN ('daily', 'weekly', 'monthly')),
    look_ahead_days INTEGER NOT NULL CHECK (look_ahead_days > 0),
    past_event_threshold_days INTEGER NOT NULL DEFAULT 5 CHECK (past_event_threshold_days >= 0),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO config_schedule_backup (id, update_frequency, look_ahead_days, past_event_threshold_days, created_at, updated_at)
SELECT id, update_frequency, look_ahead_days, past_event_threshold_days, created_at, updated_at
FROM config_schedule;

DROP TABLE config_schedule;

ALTER TABLE config_schedule_backup RENAME TO config_schedule;
