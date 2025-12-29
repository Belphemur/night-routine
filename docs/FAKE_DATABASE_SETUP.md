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
    decision_reason TEXT  -- Can be: NULL, "Override", "Total Count", "Recent Count", "Consecutive Limit", "Alternating", "Unavailability"
);
```

#### assignment_details table
```sql
CREATE TABLE assignment_details (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    assignment_id INTEGER NOT NULL,
    calculation_date TEXT NOT NULL,  -- Format: YYYY-MM-DD
    parent_a_name TEXT NOT NULL,
    parent_a_total_count INTEGER NOT NULL,
    parent_a_last_30_days INTEGER NOT NULL,
    parent_b_name TEXT NOT NULL,
    parent_b_total_count INTEGER NOT NULL,
    parent_b_last_30_days INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (assignment_id) REFERENCES assignments(id) ON DELETE CASCADE
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
-- Insert assignments (alternating between parents with various decision reasons)
INSERT INTO assignments (parent_name, assignment_date, override, decision_reason, google_calendar_event_id) VALUES
('Alice', date('now', '-10 days'), 0, 'Total Count', 'event_-10'),
('Bob', date('now', '-9 days'), 0, 'Alternating', 'event_-9'),
('Alice', date('now', '-8 days'), 0, 'Recent Count', 'event_-8'),
('Bob', date('now', '-7 days'), 0, 'Total Count', 'event_-7'),
('Alice', date('now', '-6 days'), 1, 'Override', 'event_-6'),  -- Override example (no details stored)
('Bob', date('now', '-5 days'), 0, 'Consecutive Limit', 'event_-5'),
('Alice', date('now', '-4 days'), 0, 'Total Count', 'event_-4'),
('Bob', date('now', '-3 days'), 0, 'Alternating', 'event_-3'),
('Alice', date('now', '-2 days'), 0, 'Total Count', 'event_-2'),
('Bob', date('now', '-1 days'), 0, 'Recent Count', 'event_-1'),
('Alice', date('now'), 0, 'Total Count', 'event_0'),
('Bob', date('now', '+1 days'), 0, 'Alternating', 'event_1'),
('Alice', date('now', '+2 days'), 0, 'Total Count', 'event_2'),
('Bob', date('now', '+3 days'), 0, 'Total Count', 'event_3'),
('Alice', date('now', '+4 days'), 0, 'Recent Count', 'event_4'),
('Bob', date('now', '+5 days'), 0, 'Alternating', 'event_5');

-- Insert assignment details for non-override assignments
-- These show the fairness algorithm calculations at the time of assignment
-- Note: assignment_id corresponds to the id from the assignments table above
INSERT INTO assignment_details (assignment_id, calculation_date, parent_a_name, parent_a_total_count, parent_a_last_30_days, parent_b_name, parent_b_total_count, parent_b_last_30_days) VALUES
(1, date('now', '-10 days'), 'Alice', 5, 3, 'Bob', 7, 4),   -- Assignment 1
(2, date('now', '-9 days'), 'Alice', 6, 4, 'Bob', 7, 4),    -- Assignment 2
(3, date('now', '-8 days'), 'Alice', 6, 3, 'Bob', 8, 5),    -- Assignment 3
(4, date('now', '-7 days'), 'Alice', 7, 4, 'Bob', 8, 5),    -- Assignment 4
-- Assignment 5 is override, so no details stored
(6, date('now', '-5 days'), 'Alice', 8, 4, 'Bob', 9, 5),    -- Assignment 6
(7, date('now', '-4 days'), 'Alice', 8, 4, 'Bob', 10, 6),   -- Assignment 7
(8, date('now', '-3 days'), 'Alice', 9, 5, 'Bob', 10, 6),   -- Assignment 8
(9, date('now', '-2 days'), 'Alice', 9, 5, 'Bob', 11, 7),   -- Assignment 9
(10, date('now', '-1 days'), 'Alice', 10, 6, 'Bob', 11, 7), -- Assignment 10
(11, date('now'), 'Alice', 10, 6, 'Bob', 12, 8),            -- Assignment 11
(12, date('now', '+1 days'), 'Alice', 11, 7, 'Bob', 12, 8), -- Assignment 12
(13, date('now', '+2 days'), 'Alice', 11, 7, 'Bob', 13, 9), -- Assignment 13
(14, date('now', '+3 days'), 'Alice', 12, 8, 'Bob', 13, 9), -- Assignment 14
(15, date('now', '+4 days'), 'Alice', 12, 8, 'Bob', 14, 10),-- Assignment 15
(16, date('now', '+5 days'), 'Alice', 13, 9, 'Bob', 14, 10);-- Assignment 16

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
3. **assignments table**: The `decision_reason` field shows why an assignment was made (Override, Total Count, Recent Count, Consecutive Limit, Alternating, Unavailability)
4. **assignment_details table**: Stores the fairness algorithm calculation data (parent stats) used when making non-override assignments. This data is displayed in a modal when clicking on assignments in the UI.

### Common Mistakes

❌ **Wrong**: Trying to create tables manually before migrations run
✅ **Correct**: Let the application run migrations first, then insert data

❌ **Wrong**: Using wrong column names (e.g., `selected_calendar_id`)
✅ **Correct**: Check the schema with `.schema table_name` in sqlite3

❌ **Wrong**: Inserting OAuth token with individual columns
✅ **Correct**: Insert as JSON string in the `token_data` column

❌ **Wrong**: Creating assignment details for override assignments
✅ **Correct**: Only create assignment details for non-override assignments (where override=0)

### Verifying Data

Check your data was inserted correctly:

```bash
# Count assignments
sqlite3 /tmp/night-routine-demo.db "SELECT COUNT(*) FROM assignments;"

# View parent distribution
sqlite3 /tmp/night-routine-demo.db "SELECT parent_name, COUNT(*) FROM assignments GROUP BY parent_name;"

# Check assignment details exist
sqlite3 /tmp/night-routine-demo.db "SELECT COUNT(*) FROM assignment_details;"

# View assignment with its details
sqlite3 /tmp/night-routine-demo.db "
SELECT a.id, a.parent_name, a.assignment_date, a.decision_reason, 
       d.parent_a_total_count, d.parent_a_last_30_days,
       d.parent_b_total_count, d.parent_b_last_30_days
FROM assignments a
LEFT JOIN assignment_details d ON a.id = d.assignment_id
LIMIT 5;
"

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

### Testing the Assignment Details Modal

When taking screenshots, demonstrate the new assignment details feature:

1. **Non-Override Assignments**: Click on any non-overridden assignment (shows regular background colors)
   - A modal will appear showing the fairness algorithm calculations
   - Displays the calculation date and both parents' statistics (total count and last 30 days)
   
2. **Override Assignments**: Click on an overridden assignment (shows "Override" decision reason)
   - The unlock modal will appear instead (has priority over details modal)
   - This allows users to unlock the override

The assignment details modal helps users understand how the fairness algorithm made its decision by showing:
- The date when the calculation was performed
- Parent A's total assignments and last 30-day count at that time
- Parent B's total assignments and last 30-day count at that time
- An explanation of how these numbers were used in the decision
