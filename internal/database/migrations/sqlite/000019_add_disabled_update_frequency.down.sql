-- Revert: remove 'disabled' from valid update_frequency values.
-- Rows using 'disabled' are reset to 'daily' before the constraint is restored.
UPDATE config_schedule SET update_frequency = 'daily' WHERE update_frequency = 'disabled';

CREATE TABLE config_schedule_new (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    update_frequency TEXT NOT NULL CHECK (update_frequency IN ('daily', 'weekly', 'monthly')),
    look_ahead_days INTEGER NOT NULL CHECK (look_ahead_days > 0),
    past_event_threshold_days INTEGER NOT NULL DEFAULT 5 CHECK (past_event_threshold_days >= 0),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    stats_order TEXT NOT NULL DEFAULT 'desc' CHECK (stats_order IN ('desc', 'asc'))
);

INSERT INTO config_schedule_new SELECT * FROM config_schedule;

DROP TABLE config_schedule;

ALTER TABLE config_schedule_new RENAME TO config_schedule;
