-- Add 'disabled' as a valid update_frequency value.
-- SQLite does not support DROP CONSTRAINT, so the table must be recreated.
CREATE TABLE config_schedule_new (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    update_frequency TEXT NOT NULL CHECK (update_frequency IN ('daily', 'weekly', 'monthly', 'disabled')),
    look_ahead_days INTEGER NOT NULL CHECK (look_ahead_days > 0),
    past_event_threshold_days INTEGER NOT NULL DEFAULT 5 CHECK (past_event_threshold_days >= 0),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    stats_order TEXT NOT NULL DEFAULT 'desc' CHECK (stats_order IN ('desc', 'asc'))
);

INSERT INTO config_schedule_new SELECT * FROM config_schedule;

DROP TABLE config_schedule;

ALTER TABLE config_schedule_new RENAME TO config_schedule;
