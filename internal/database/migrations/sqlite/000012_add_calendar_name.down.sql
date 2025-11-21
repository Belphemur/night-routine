-- SQLite doesn't support DROP COLUMN easily, so we recreate the table
CREATE TABLE calendar_settings_backup (
    id INTEGER PRIMARY KEY,
    calendar_id TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO calendar_settings_backup (id, calendar_id, updated_at)
SELECT id, calendar_id, updated_at FROM calendar_settings;

DROP TABLE calendar_settings;

ALTER TABLE calendar_settings_backup RENAME TO calendar_settings;
