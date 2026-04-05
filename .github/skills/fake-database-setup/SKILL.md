---
name: fake-database-setup
description: >
  Guide for setting up a fake/demo SQLite database with sample data for testing
  the Night Routine application and taking screenshots. Use this skill when you
  need to create test data, set up a demo database, take screenshots of the UI,
  or verify the application with sample assignments and calendar data.
---

# Setting Up a Fake Database for Testing and Screenshots

The application uses SQLite with migrations that must be applied in order. To create a working demo database, you need to:

1. Let the application create the database schema via migrations
2. Insert demo data that matches the schema
3. Configure OAuth tokens and calendar settings properly

## Step 1: Create Configuration File

Create a demo configuration file at `/tmp/night-routine-demo-config.toml`:

```toml
[parents]
parent_a = "Alice"
parent_b = "Bob"

[availability]
parent_a_unavailable = []
parent_b_unavailable = []

[schedule]
update_frequency = "disabled"
look_ahead_days = 14
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

## Step 2: Initialize Database with Migrations

Start the application once to let it create the database and run all migrations:

```bash
GOOGLE_OAUTH_CLIENT_ID="demo-client-id" \
GOOGLE_OAUTH_CLIENT_SECRET="demo-client-secret" \
CONFIG_FILE=/tmp/night-routine-demo-config.toml \
./night-routine
```

Stop the application after it starts successfully (Ctrl+C). The database schema is now properly initialized.

## Step 3: Insert Demo Data

The key tables and their schemas are:

### assignments table

```sql
CREATE TABLE assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_name TEXT NOT NULL,
    assignment_date TEXT NOT NULL,  -- Format: YYYY-MM-DD
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    override BOOLEAN DEFAULT 0 NOT NULL,
    google_calendar_event_id TEXT,
    caregiver_type TEXT NOT NULL DEFAULT 'parent',  -- 'parent' or 'babysitter'
    decision_reason TEXT  -- Can be: NULL, "Override", "Total Count", "Recent Count",
                          -- "Consecutive Limit", "Alternating", "Unavailability",
                          -- "Double Consecutive Swap"
);
```

### assignment_details table

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

### oauth_tokens table

```sql
CREATE TABLE oauth_tokens (
    id INTEGER PRIMARY KEY,
    token_data JSONB NOT NULL,  -- JSON string with OAuth token data
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### calendar_settings table

```sql
CREATE TABLE calendar_settings (
    id INTEGER PRIMARY KEY,
    calendar_id TEXT NOT NULL,  -- Note: column is 'calendar_id', not 'selected_calendar_id'
    calendar_name TEXT DEFAULT '',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Step 4: Populate with Sample Data

The demo data covers a full calendar month and showcases every decision reason in the
fairness algorithm, including **Double Consecutive Swap** (AA BB → AB AB). The assignments
also include two babysitter nights and one manually-overridden assignment.

**Decision reason coverage:**
| Reason                 | Description                                                                |
|------------------------|----------------------------------------------------------------------------|
| `Total Count`          | Parent A or B had fewer total assignments overall                          |
| `Alternating`          | Totals and recent counts were equal; maintained alternating pattern        |
| `Recent Count`         | Totals were tied; chosen parent had fewer assignments in the last 30 days  |
| `Consecutive Limit`    | Totals were tied; one parent had 2+ consecutive nights — force a switch   |
| `Double Consecutive Swap` | AA BB detected; boundary nights swapped to produce AB AB               |
| `Unavailability`       | One parent was unavailable; the other was assigned automatically           |
| `Override`             | Assignment was manually changed by a user in Google Calendar               |
| Babysitter             | A babysitter covered the night (counts as +1 shift for both parents)       |

Use SQLite to insert demo data (this covers an entire month — adjust offsets if needed):

```bash
sqlite3 /tmp/night-routine-demo.db <<'EOF'
-- ============================================================
-- Full-month demo data for Night Routine Scheduler
-- Covers all decision reasons including Double Consecutive Swap
--
-- Day  1: Alice – Total Count
-- Day  2: Bob   – Alternating
-- Day  3: Alice – Double Consecutive Swap  ← pair start (AA BB → AB AB)
-- Day  4: Bob   – Double Consecutive Swap  ← pair end
-- Day  5: Alice – Recent Count
-- Day  6: Bob   – Total Count
-- Day  7: Dawn  – Babysitter
-- Day  8: Alice – Total Count
-- Day  9: Bob   – Consecutive Limit   (Alice had 2 consecutive)
-- Day 10: Alice – Alternating
-- Day 11: Bob   – Total Count
-- Day 12: Alice – Recent Count
-- Day 13: Bob   – Total Count
-- Day 14: Alice – Override             (manually changed, locked)
-- Day 15: Bob   – Alternating
-- Day 16: Alice – Total Count
-- Day 17: Bob   – Unavailability       (Alice unavailable)
-- Day 18: Alice – Total Count
-- Day 19: Bob   – Alternating
-- Day 20: Alice – Total Count
-- Day 21: Emma  – Babysitter
-- Day 22: Bob   – Consecutive Limit   (Alice had 2 consecutive)
-- Day 23: Alice – Total Count
-- Day 24: Bob   – Alternating
-- Day 25: Alice – Total Count
-- Day 26: Bob   – Recent Count
-- Day 27: Alice – Total Count
-- Day 28: Bob   – Alternating
-- Day 29: Alice – Total Count
-- Day 30: Bob   – Alternating
-- ============================================================

INSERT INTO assignments (parent_name, assignment_date, override, caregiver_type, decision_reason, google_calendar_event_id) VALUES
('Alice', date('now', (1  - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d01'),
('Bob',   date('now', (2  - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d02'),
-- Double Consecutive Swap pair: days 3 & 4 were the boundary of an AA BB run that got swapped to AB AB
('Alice', date('now', (3  - strftime('%d','now')) || ' days'), 0, 'parent',     'Double Consecutive Swap','evt_d03'),
('Bob',   date('now', (4  - strftime('%d','now')) || ' days'), 0, 'parent',     'Double Consecutive Swap','evt_d04'),
('Alice', date('now', (5  - strftime('%d','now')) || ' days'), 0, 'parent',     'Recent Count',           'evt_d05'),
('Bob',   date('now', (6  - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d06'),
('Dawn',  date('now', (7  - strftime('%d','now')) || ' days'), 0, 'babysitter', NULL,                     'evt_d07'),
('Alice', date('now', (8  - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d08'),
('Bob',   date('now', (9  - strftime('%d','now')) || ' days'), 0, 'parent',     'Consecutive Limit',      'evt_d09'),
('Alice', date('now', (10 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d10'),
('Bob',   date('now', (11 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d11'),
('Alice', date('now', (12 - strftime('%d','now')) || ' days'), 0, 'parent',     'Recent Count',           'evt_d12'),
('Bob',   date('now', (13 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d13'),
('Alice', date('now', (14 - strftime('%d','now')) || ' days'), 1, 'parent',     'Override',               'evt_d14'),
('Bob',   date('now', (15 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d15'),
('Alice', date('now', (16 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d16'),
('Bob',   date('now', (17 - strftime('%d','now')) || ' days'), 0, 'parent',     'Unavailability',         'evt_d17'),
('Alice', date('now', (18 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d18'),
('Bob',   date('now', (19 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d19'),
('Alice', date('now', (20 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d20'),
('Emma',  date('now', (21 - strftime('%d','now')) || ' days'), 0, 'babysitter', NULL,                     'evt_d21'),
('Bob',   date('now', (22 - strftime('%d','now')) || ' days'), 0, 'parent',     'Consecutive Limit',      'evt_d22'),
('Alice', date('now', (23 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d23'),
('Bob',   date('now', (24 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d24'),
('Alice', date('now', (25 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d25'),
('Bob',   date('now', (26 - strftime('%d','now')) || ' days'), 0, 'parent',     'Recent Count',           'evt_d26'),
('Alice', date('now', (27 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d27'),
('Bob',   date('now', (28 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d28'),
('Alice', date('now', (29 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d29'),
('Bob',   date('now', (30 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d30');

-- ============================================================
-- Assignment details (stats snapshot at time of each assignment)
-- Skipped for: babysitter days (IDs 7, 21) and override day (ID 14).
-- ============================================================
INSERT INTO assignment_details (assignment_id, calculation_date, parent_a_name, parent_a_total_count, parent_a_last_30_days, parent_b_name, parent_b_total_count, parent_b_last_30_days) VALUES
-- Day  1: Alice, Total Count   (Alice=12 < Bob=14)
(1,  date('now', (1  - strftime('%d','now')) || ' days'), 'Alice', 12, 5,  'Bob', 14, 6),
-- Day  2: Bob,   Alternating   (equal totals & recent, previous=Alice → Bob)
(2,  date('now', (2  - strftime('%d','now')) || ' days'), 'Alice', 13, 6,  'Bob', 13, 6),
-- Day  3: Alice, Double Consecutive Swap (boundary swap: was originally Bob but swapped to Alice)
(3,  date('now', (3  - strftime('%d','now')) || ' days'), 'Alice', 13, 6,  'Bob', 14, 7),
-- Day  4: Bob,   Double Consecutive Swap (boundary swap: was originally Alice but swapped to Bob)
(4,  date('now', (4  - strftime('%d','now')) || ' days'), 'Alice', 14, 7,  'Bob', 14, 7),
-- Day  5: Alice, Recent Count  (totals tied=15, Alice has fewer last-30: 7 < 8)
(5,  date('now', (5  - strftime('%d','now')) || ' days'), 'Alice', 14, 7,  'Bob', 15, 8),
-- Day  6: Bob,   Total Count   (Bob=15 < Alice=16)
(6,  date('now', (6  - strftime('%d','now')) || ' days'), 'Alice', 16, 8,  'Bob', 15, 8),
-- Day  7: babysitter (Dawn) – no details
-- Day  8: Alice, Total Count   (babysitter +1 both; Alice=17 < Bob=18 after babysitter shift)
(8,  date('now', (8  - strftime('%d','now')) || ' days'), 'Alice', 17, 9,  'Bob', 18, 9),
-- Day  9: Bob,   Consecutive Limit (totals tied=18, Alice had 2 consecutive → switch to Bob)
(9,  date('now', (9  - strftime('%d','now')) || ' days'), 'Alice', 18, 9,  'Bob', 18, 9),
-- Day 10: Alice, Alternating   (totals tied=19, recent tied=9, previous=Bob → Alice)
(10, date('now', (10 - strftime('%d','now')) || ' days'), 'Alice', 18, 9,  'Bob', 19, 10),
-- Day 11: Bob,   Total Count   (Bob=19 < Alice=20)
(11, date('now', (11 - strftime('%d','now')) || ' days'), 'Alice', 20, 10, 'Bob', 19, 9),
-- Day 12: Alice, Recent Count  (totals tied=20, Alice has fewer last-30: 9 < 10)
(12, date('now', (12 - strftime('%d','now')) || ' days'), 'Alice', 19, 9,  'Bob', 20, 10),
-- Day 13: Bob,   Total Count   (Bob=20 < Alice=21 after Alice assigned day 12)
(13, date('now', (13 - strftime('%d','now')) || ' days'), 'Alice', 21, 10, 'Bob', 20, 10),
-- Day 14: Alice, Override – no details (manually changed, fairness not calculated)
-- Day 15: Bob,   Alternating   (totals tied=21, recent tied=11, previous=Alice → Bob)
(15, date('now', (15 - strftime('%d','now')) || ' days'), 'Alice', 21, 11, 'Bob', 21, 11),
-- Day 16: Alice, Total Count   (Alice=21 < Bob=22)
(16, date('now', (16 - strftime('%d','now')) || ' days'), 'Alice', 21, 11, 'Bob', 22, 12),
-- Day 17: Bob,   Unavailability (Alice unavailable; Bob assigned regardless of counts)
(17, date('now', (17 - strftime('%d','now')) || ' days'), 'Alice', 22, 12, 'Bob', 22, 12),
-- Day 18: Alice, Total Count   (Alice=22 < Bob=23)
(18, date('now', (18 - strftime('%d','now')) || ' days'), 'Alice', 22, 12, 'Bob', 23, 13),
-- Day 19: Bob,   Alternating   (totals tied=23, recent tied=13, previous=Alice → Bob)
(19, date('now', (19 - strftime('%d','now')) || ' days'), 'Alice', 23, 13, 'Bob', 23, 13),
-- Day 20: Alice, Total Count   (Alice=23 < Bob=24)
(20, date('now', (20 - strftime('%d','now')) || ' days'), 'Alice', 23, 13, 'Bob', 24, 14),
-- Day 21: babysitter (Emma) – no details
-- Day 22: Bob,   Consecutive Limit (babysitter +1 both; totals tied=25, Alice had 2 consecutive → Bob)
(22, date('now', (22 - strftime('%d','now')) || ' days'), 'Alice', 25, 15, 'Bob', 25, 15),
-- Day 23: Alice, Total Count   (Alice=25 < Bob=26)
(23, date('now', (23 - strftime('%d','now')) || ' days'), 'Alice', 25, 14, 'Bob', 26, 16),
-- Day 24: Bob,   Alternating   (totals tied=26, recent tied=15, previous=Alice → Bob)
(24, date('now', (24 - strftime('%d','now')) || ' days'), 'Alice', 26, 15, 'Bob', 26, 15),
-- Day 25: Alice, Total Count   (Alice=26 < Bob=27)
(25, date('now', (25 - strftime('%d','now')) || ' days'), 'Alice', 26, 15, 'Bob', 27, 16),
-- Day 26: Bob,   Recent Count  (totals tied=27, Bob has fewer last-30: 15 < 16)
(26, date('now', (26 - strftime('%d','now')) || ' days'), 'Alice', 27, 16, 'Bob', 27, 15),
-- Day 27: Alice, Total Count   (Alice=27 < Bob=28)
(27, date('now', (27 - strftime('%d','now')) || ' days'), 'Alice', 27, 16, 'Bob', 28, 16),
-- Day 28: Bob,   Alternating   (totals tied=28, recent tied=16, previous=Alice → Bob)
(28, date('now', (28 - strftime('%d','now')) || ' days'), 'Alice', 28, 17, 'Bob', 28, 16),
-- Day 29: Alice, Total Count   (Alice=28 < Bob=29)
(29, date('now', (29 - strftime('%d','now')) || ' days'), 'Alice', 28, 16, 'Bob', 29, 17),
-- Day 30: Bob,   Alternating   (totals tied=29, recent tied=17, previous=Alice → Bob)
(30, date('now', (30 - strftime('%d','now')) || ' days'), 'Alice', 29, 17, 'Bob', 29, 17);

-- Insert OAuth token (JSON format — expiry far in the future so GetValidToken succeeds without a network call)
INSERT OR REPLACE INTO oauth_tokens (id, token_data, updated_at)
VALUES (1, '{"access_token":"demo_token","token_type":"Bearer","refresh_token":"demo_refresh","expiry":"2099-12-31T23:59:59Z"}', datetime('now'));

-- Insert calendar settings
INSERT OR REPLACE INTO calendar_settings (id, calendar_id, calendar_name)
VALUES (1, 'demo-calendar@example.com', 'Family Night Routine');
EOF
```

## Step 5: Start Application with Demo Data

```bash
GOOGLE_OAUTH_CLIENT_ID="demo-client-id" \
GOOGLE_OAUTH_CLIENT_SECRET="demo-client-secret" \
CONFIG_FILE=/tmp/night-routine-demo-config.toml \
./night-routine
```

The application will now show the calendar with demo data at `http://localhost:8080`.

## Important Schema Notes

1. **calendar_settings table**: The column is named `calendar_id`, NOT `selected_calendar_id`
2. **oauth_tokens table**: Stores token data as JSONB in the `token_data` column, not individual columns
3. **assignments table**:
   - `caregiver_type` is `'parent'` or `'babysitter'` (not NULL — defaults to `'parent'`)
   - `decision_reason` values: `Override`, `Total Count`, `Recent Count`, `Consecutive Limit`,
     `Alternating`, `Unavailability`, `Double Consecutive Swap`
   - Override assignments have `override=1` and no entry in `assignment_details`
   - Babysitter assignments have `caregiver_type='babysitter'` and no entry in `assignment_details`
4. **assignment_details table**: Stores the fairness algorithm calculation data (parent stats) used
   when making non-override, non-babysitter assignments. This data is displayed in a modal when
   clicking on assignments in the UI.
5. **Double Consecutive Swap**: Always comes in pairs — two adjacent days where both have this
   reason. The boundary pair was swapped to break an AA BB back-to-back pattern.

## Common Mistakes

- **Wrong**: Trying to create tables manually before migrations run. **Correct**: Let the application run migrations first, then insert data.
- **Wrong**: Using wrong column names (e.g., `selected_calendar_id`). **Correct**: Check the schema with `.schema table_name` in sqlite3.
- **Wrong**: Inserting OAuth token with individual columns. **Correct**: Insert as JSON string in the `token_data` column.
- **Wrong**: Creating assignment details for override or babysitter assignments. **Correct**: Only create assignment details for non-override parent assignments (`override=0` and `caregiver_type='parent'`).
- **Wrong**: Omitting `caregiver_type` column. **Correct**: Always set it — use `'parent'` for parent assignments and `'babysitter'` for babysitter nights.

## Verifying Data

Check your data was inserted correctly:

```bash
# Count assignments
sqlite3 /tmp/night-routine-demo.db "SELECT COUNT(*) FROM assignments;"

# View parent/babysitter distribution
sqlite3 /tmp/night-routine-demo.db "SELECT parent_name, caregiver_type, COUNT(*) FROM assignments GROUP BY parent_name, caregiver_type;"

# View all decision reasons used
sqlite3 /tmp/night-routine-demo.db "SELECT decision_reason, COUNT(*) FROM assignments GROUP BY decision_reason ORDER BY COUNT(*) DESC;"

# Check assignment details exist
sqlite3 /tmp/night-routine-demo.db "SELECT COUNT(*) FROM assignment_details;"

# View assignment with its details (first 5)
sqlite3 /tmp/night-routine-demo.db "
SELECT a.id, a.parent_name, a.assignment_date, a.decision_reason,
       d.parent_a_total_count, d.parent_a_last_30_days,
       d.parent_b_total_count, d.parent_b_last_30_days
FROM assignments a
LEFT JOIN assignment_details d ON a.id = d.assignment_id
ORDER BY a.assignment_date
LIMIT 5;
"

# Verify Double Consecutive Swap pair
sqlite3 /tmp/night-routine-demo.db "
SELECT id, parent_name, assignment_date, decision_reason
FROM assignments WHERE decision_reason = 'Double Consecutive Swap'
ORDER BY assignment_date;
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
- Shows full month calendar view with all 30 days

### Mobile View
- Viewport: 375x667 (iPhone SE size)
- Shows weekly calendar with navigation buttons
- Mobile view uses client-side JavaScript to filter the full month data to show only one week at a time

### Suggested Screenshots

1. **Full month overview** — Shows the balanced calendar with all decision reasons visible in cells
2. **Double Consecutive Swap cells** — April 3–4 show the swap pair (both show "Double Consecutive Swap" on adjacent days)
3. **Babysitter night** — Click the Dawn (day 7) or Emma (day 21) cell
4. **Total Count modal** — Click day 1 (Alice, Total Count); modal shows Alice with fewer total assignments
5. **Recent Count modal** — Click day 5 or day 26; modal shows equal totals but assigned parent has fewer last-30
6. **Consecutive Limit modal** — Click day 9 or day 22; shows equal totals with one parent having 2+ consecutive nights
7. **Double Consecutive Swap modal** — Click day 3 or 4; shows explanation of the AA BB → AB AB swap
8. **Override cell** — Click day 14 (Alice, Override); unlock modal appears

### Testing the Assignment Details Modal

When taking screenshots, demonstrate the assignment details feature:

1. **Non-Override Assignments**: Click on any non-overridden parent assignment
   - A modal will appear showing the fairness algorithm calculations
   - Displays the calculation date and both parents' statistics (total count and last 30 days)
   - Shows the decision reason with a full explanation

2. **Override Assignments**: Click on an overridden assignment (shows "Override" decision reason)
   - The unlock modal will appear instead (has priority over details modal)
   - This allows users to unlock the override

3. **Babysitter Assignments**: Click on a babysitter cell (Dawn or Emma)
   - The details modal shows a note that babysitter nights are excluded from parent fairness totals

The assignment details modal helps users understand how the fairness algorithm made its decision by showing:
- The date when the calculation was performed
- Parent A's total assignments and last 30-day count at that time
- Parent B's total assignments and last 30-day count at that time
- An explanation of how these numbers were used in the decision
