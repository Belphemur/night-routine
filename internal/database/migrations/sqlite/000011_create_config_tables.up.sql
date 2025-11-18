-- Create table for parent configuration
CREATE TABLE IF NOT EXISTS config_parents (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- Ensure only one row
    parent_a TEXT NOT NULL,
    parent_b TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (parent_a != parent_b)
);

-- Create table for availability configuration
CREATE TABLE IF NOT EXISTS config_availability (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent TEXT NOT NULL CHECK (parent IN ('parent_a', 'parent_b')),
    unavailable_day TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(parent, unavailable_day)
);

-- Create table for schedule configuration
CREATE TABLE IF NOT EXISTS config_schedule (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- Ensure only one row
    update_frequency TEXT NOT NULL CHECK (update_frequency IN ('daily', 'weekly', 'monthly')),
    look_ahead_days INTEGER NOT NULL CHECK (look_ahead_days > 0),
    past_event_threshold_days INTEGER NOT NULL DEFAULT 5 CHECK (past_event_threshold_days >= 0),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_config_availability_parent ON config_availability(parent);
