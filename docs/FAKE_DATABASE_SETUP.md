# Setting Up a Fake Database for Testing and Screenshots

This guide explains how to properly set up a fake database with demo data for testing the Night Routine application and taking screenshots.

## Overview

The application uses SQLite with migrations that must be applied in order. To create a working demo database, you need to:

1. Let the application create the database schema via migrations
2. Insert demo data that matches the schema
3. Configure OAuth tokens and calendar settings properly

## Step-by-Step Instructions

### 1. Create Configuration File

Create a demo configuration file (e.g., `/tmp/night-routine-demo-config.toml`):

```toml
[parents]
parent_a = "Alice"
parent_b = "Bob"

[availability]
parent_a_unavailable = []
parent_b_unavailable = []

[schedule]
update_frequency = "weekly"
look_ahead_days = 7
past_event_threshold_days = 5

[service]
state_file = "/tmp/night-routine-demo.db"
log_level = "info"
manual_sync_on_startup = false

[app]
port = 8080
app_url = "http://localhost:8080"
public_url = "http://localhost:8080"

[calendar]
time_zone = "America/New_York"

[fairness]
lookback_days = 90
```

### 2. Initialize Database with Migrations

Start the application once to let it create the database and run all migrations:

```bash
GOOGLE_OAUTH_CLIENT_ID="demo-client-id" \
GOOGLE_OAUTH_CLIENT_SECRET="demo-client-secret" \
CONFIG_FILE=/tmp/night-routine-demo-config.toml \
./night-routine
```

Stop the application after it starts successfully (Ctrl+C). The database schema is now properly initialized.

### 3. Insert Demo Data

Now insert demo assignment data. The key tables are:

#### assignments table
```sql
CREATE TABLE assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_name TEXT NOT NULL,
    assignment_date TEXT NOT NULL,  -- Format: YYYY-MM-DD
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    override BOOLEAN DEFAULT 0 NOT NULL,
    google_calendar_event_id TEXT,
    decision_reason TEXT  -- Can be: NULL, "Override", "Fairness", "Total Count", etc.
);
```

#### oauth_tokens table
```sql
CREATE TABLE oauth_tokens (
    id INTEGER PRIMARY KEY,
    token_data JSONB NOT NULL,  -- JSON string with OAuth token data
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### calendar_settings table
```sql
CREATE TABLE calendar_settings (
    id INTEGER PRIMARY KEY,
    calendar_id TEXT NOT NULL,  -- Note: column is 'calendar_id', not 'selected_calendar_id'
    calendar_name TEXT DEFAULT '',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 4. Populate with Sample Data

Use SQLite to insert demo data:

```bash
sqlite3 /tmp/night-routine-demo.db <<'EOF'
-- Insert assignments (alternating between parents with some overrides)
INSERT INTO assignments (parent_name, assignment_date, override, decision_reason, google_calendar_event_id) VALUES
('Alice', date('now', '-60 days'), 0, NULL, 'event_-60'),
('Bob', date('now', '-59 days'), 0, NULL, 'event_-59'),
-- ... more assignments ...
('Alice', date('now', '-5 days'), 1, 'Override', 'event_-5'),  -- Override example
('Bob', date('now', '-17 days'), 0, 'Fairness', 'event_-17'), -- Fairness reason
('Alice', date('now'), 0, NULL, 'event_0'),
('Bob', date('now', '+1 days'), 0, NULL, 'event_1'),
-- ... more future assignments ...
('Alice', date('now', '+30 days'), 0, NULL, 'event_30');

-- Insert OAuth token (JSON format)
INSERT OR REPLACE INTO oauth_tokens (id, token_data, updated_at)
VALUES (1, '{"access_token":"demo_token","token_type":"Bearer","refresh_token":"demo_refresh","expiry":"2025-12-31T23:59:59Z"}', datetime('now'));

-- Insert calendar settings
INSERT OR REPLACE INTO calendar_settings (id, calendar_id, calendar_name)
VALUES (1, 'demo-calendar@example.com', 'Family Night Routine');
EOF
```

### 5. Start Application with Demo Data

```bash
GOOGLE_OAUTH_CLIENT_ID="demo-client-id" \
GOOGLE_OAUTH_CLIENT_SECRET="demo-client-secret" \
CONFIG_FILE=/tmp/night-routine-demo-config.toml \
./night-routine
```

The application will now show the calendar with demo data at `http://localhost:8080`.

## Important Notes

### Schema Differences to Watch For

1. **calendar_settings table**: The column is named `calendar_id`, NOT `selected_calendar_id`
2. **oauth_tokens table**: Stores token data as JSONB in the `token_data` column, not individual columns
3. **assignments table**: The `decision_reason` field shows why an assignment was made (Override, Fairness, Total Count, etc.)

### Common Mistakes

❌ **Wrong**: Trying to create tables manually before migrations run
✅ **Correct**: Let the application run migrations first, then insert data

❌ **Wrong**: Using wrong column names (e.g., `selected_calendar_id`)
✅ **Correct**: Check the schema with `.schema table_name` in sqlite3

❌ **Wrong**: Inserting OAuth token with individual columns
✅ **Correct**: Insert as JSON string in the `token_data` column

### Verifying Data

Check your data was inserted correctly:

```bash
# Count assignments
sqlite3 /tmp/night-routine-demo.db "SELECT COUNT(*) FROM assignments;"

# View parent distribution
sqlite3 /tmp/night-routine-demo.db "SELECT parent_name, COUNT(*) FROM assignments GROUP BY parent_name;"

# Check OAuth token
sqlite3 /tmp/night-routine-demo.db "SELECT * FROM oauth_tokens;"

# Check calendar settings
sqlite3 /tmp/night-routine-demo.db "SELECT * FROM calendar_settings;"
```

## Taking Screenshots

Once the demo database is set up, use a browser or automated tool to take screenshots:

### Desktop View
- Default viewport: 1280x720 or larger
- Shows full month calendar view

### Mobile View  
- Viewport: 375x667 (iPhone SE size)
- Shows weekly calendar with navigation buttons

The mobile view uses client-side JavaScript to filter the full month data to show only one week at a time.
