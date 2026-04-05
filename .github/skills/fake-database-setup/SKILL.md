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
-- Layout (offsets from today = start of the current month):
--   Day  1 (today-N): Alice  – Total Count
--   Day  2           : Bob   – Alternating
--   Day  3           : Alice – Total Count
--   Day  4           : Bob   – Consecutive Limit  (Alice had 2 consecutive)
--   Day  5           : Alice – Recent Count
--   Day  6           : Bob   – Total Count
--   Day  7           : Alice – Alternating
--   Day  8           : Bob   – Total Count
--   Day  9           : Alice – Consecutive Limit  (Bob had 2 consecutive)
--   Day 10           : Dawn  – Babysitter
--   Day 11           : Bob   – Total Count
--   Day 12           : Alice – Alternating
--   Day 13           : Alice – Total Count         (Alice still behind)
--   Day 14           : Bob   – Double Consecutive Swap (was Alice; AA BB → AB AB)
--   Day 15           : Alice – Double Consecutive Swap (was Bob;  AA BB → AB AB)
--   Day 16           : Bob   – Total Count
--   Day 17           : Alice – Override            (manually changed)
--   Day 18           : Bob   – Unavailability      (Alice unavailable)
--   Day 19           : Alice – Total Count
--   Day 20           : Bob   – Alternating
--   Day 21           : Alice – Total Count
--   Day 22           : Bob   – Consecutive Limit
--   Day 23           : Alice – Total Count
--   Day 24           : Emma  – Babysitter
--   Day 25           : Bob   – Recent Count
--   Day 26           : Alice – Total Count
--   Day 27           : Bob   – Alternating
--   Day 28           : Alice – Total Count
--   Day 29           : Bob   – Alternating
--   Day 30           : Alice – Recent Count
-- ============================================================

-- Compute start-of-month offset so data fills the current calendar month.
-- We use the day-of-month to anchor day 1 to the 1st of the current month.
-- e.g. if today is the 5th, day 1 is today-4 days, day 30 is today+25 days.

INSERT INTO assignments (parent_name, assignment_date, override, caregiver_type, decision_reason, google_calendar_event_id) VALUES
-- Week 1
-- Note: date offset = (day_of_month - strftime('%d','now')); positive = future, negative = past
('Alice', date('now', (1  - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d01'),
('Bob',   date('now', (2  - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d02'),
('Alice', date('now', (3  - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d03'),
('Bob',   date('now', (4  - strftime('%d','now')) || ' days'), 0, 'parent',     'Consecutive Limit',      'evt_d04'),
('Alice', date('now', (5  - strftime('%d','now')) || ' days'), 0, 'parent',     'Recent Count',           'evt_d05'),
('Bob',   date('now', (6  - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d06'),
('Alice', date('now', (7  - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d07'),
-- Week 2
('Bob',   date('now', (8  - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d08'),
('Alice', date('now', (9  - strftime('%d','now')) || ' days'), 0, 'parent',     'Consecutive Limit',      'evt_d09'),
('Dawn',  date('now', (10 - strftime('%d','now')) || ' days'), 0, 'babysitter', NULL,                     'evt_d10'),
('Bob',   date('now', (11 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d11'),
('Alice', date('now', (12 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d12'),
('Alice', date('now', (13 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d13'),
-- Week 3 – Double Consecutive Swap pair (days 14 & 15)
-- Before swap: Alice Alice Bob Bob (AA BB). After swap: Alice Bob Alice Bob (AB AB).
-- Day 14 was originally Alice but swapped to Bob; day 15 was originally Bob but swapped to Alice.
('Bob',   date('now', (14 - strftime('%d','now')) || ' days'), 0, 'parent',     'Double Consecutive Swap','evt_d14'),
('Alice', date('now', (15 - strftime('%d','now')) || ' days'), 0, 'parent',     'Double Consecutive Swap','evt_d15'),
('Bob',   date('now', (16 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d16'),
('Alice', date('now', (17 - strftime('%d','now')) || ' days'), 1, 'parent',     'Override',               'evt_d17'),
('Bob',   date('now', (18 - strftime('%d','now')) || ' days'), 0, 'parent',     'Unavailability',         'evt_d18'),
-- Week 4
('Alice', date('now', (19 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d19'),
('Bob',   date('now', (20 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d20'),
('Alice', date('now', (21 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d21'),
('Bob',   date('now', (22 - strftime('%d','now')) || ' days'), 0, 'parent',     'Consecutive Limit',      'evt_d22'),
('Alice', date('now', (23 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d23'),
('Emma',  date('now', (24 - strftime('%d','now')) || ' days'), 0, 'babysitter', NULL,                     'evt_d24'),
('Bob',   date('now', (25 - strftime('%d','now')) || ' days'), 0, 'parent',     'Recent Count',           'evt_d25'),
('Alice', date('now', (26 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d26'),
('Bob',   date('now', (27 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d27'),
('Alice', date('now', (28 - strftime('%d','now')) || ' days'), 0, 'parent',     'Total Count',            'evt_d28'),
('Bob',   date('now', (29 - strftime('%d','now')) || ' days'), 0, 'parent',     'Alternating',            'evt_d29'),
('Alice', date('now', (30 - strftime('%d','now')) || ' days'), 0, 'parent',     'Recent Count',           'evt_d30');

-- ============================================================
-- Assignment details (stats snapshot at time of each assignment)
-- Skipped for: babysitter days (IDs 10, 24) and override day (ID 17).
-- Stats follow a realistic progression. Babysitter nights count as +1 for BOTH parents.
--
-- Key: "Total Count" → assigned parent has fewer total
--      "Alternating"  → both totals equal, both last-30 equal
--      "Recent Count" → totals equal, assigned parent has fewer last-30
--      "Consecutive Limit" / "Double Consecutive Swap" → totals equal (or close)
--      "Unavailability" → one parent unavailable regardless of counts
-- ============================================================
INSERT INTO assignment_details (assignment_id, calculation_date, parent_a_name, parent_a_total_count, parent_a_last_30_days, parent_b_name, parent_b_total_count, parent_b_last_30_days) VALUES
-- Day  1: Alice, Total Count   (Alice=12 < Bob=14)
(1,  date('now', (1  - strftime('%d','now')) || ' days'), 'Alice', 12, 5,  'Bob', 14, 6),
-- Day  2: Bob,   Alternating   (equal totals & recent, previous=Alice → Bob)
(2,  date('now', (2  - strftime('%d','now')) || ' days'), 'Alice', 13, 6,  'Bob', 13, 6),
-- Day  3: Alice, Total Count   (Alice=13 < Bob=14)
(3,  date('now', (3  - strftime('%d','now')) || ' days'), 'Alice', 13, 6,  'Bob', 14, 7),
-- Day  4: Bob,   Consecutive Limit  (totals tied=14, Alice had 2 consecutive → switch)
(4,  date('now', (4  - strftime('%d','now')) || ' days'), 'Alice', 14, 7,  'Bob', 14, 7),
-- Day  5: Alice, Recent Count  (totals tied=15, Alice has fewer last-30: 7 < 8)
(5,  date('now', (5  - strftime('%d','now')) || ' days'), 'Alice', 14, 7,  'Bob', 15, 8),
-- Day  6: Bob,   Total Count   (Bob=15 < Alice=16)
(6,  date('now', (6  - strftime('%d','now')) || ' days'), 'Alice', 16, 8,  'Bob', 15, 8),
-- Day  7: Alice, Alternating   (totals=16/16, recent=8/8, previous=Bob → Alice)
(7,  date('now', (7  - strftime('%d','now')) || ' days'), 'Alice', 15, 8,  'Bob', 16, 9),
-- Day  8: Bob,   Total Count   (Bob=16 < Alice=17)
(8,  date('now', (8  - strftime('%d','now')) || ' days'), 'Alice', 17, 9,  'Bob', 16, 8),
-- Day  9: Alice, Consecutive Limit  (totals tied=17, Bob had 2 consecutive → switch)
(9,  date('now', (9  - strftime('%d','now')) || ' days'), 'Alice', 17, 9,  'Bob', 17, 9),
-- Day 10: babysitter (Dawn) – no details
-- Day 11: Bob,   Total Count   (babysitter night added +1 to both; Bob=18 < Alice=19)
(11, date('now', (11 - strftime('%d','now')) || ' days'), 'Alice', 19, 10, 'Bob', 18, 9),
-- Day 12: Alice, Alternating   (totals=19/19, recent=10/10, previous=Bob → Alice)
(12, date('now', (12 - strftime('%d','now')) || ' days'), 'Alice', 18, 9,  'Bob', 19, 10),
-- Day 13: Alice, Total Count   (Alice=19 < Bob=20; still catching up)
(13, date('now', (13 - strftime('%d','now')) || ' days'), 'Alice', 19, 10, 'Bob', 20, 11),
-- Day 14: Bob,   Double Consecutive Swap  (was Alice; AA BB pattern → swapped to Bob)
--         Before swap: Alice(13) Alice(14) Bob(15) Bob(16) → AA BB
--         After swap:  Alice(13) Bob(14)   Alice(15) Bob(16) → AB AB
(14, date('now', (14 - strftime('%d','now')) || ' days'), 'Alice', 20, 11, 'Bob', 20, 11),
-- Day 15: Alice, Double Consecutive Swap  (was Bob; paired swap with day 14)
(15, date('now', (15 - strftime('%d','now')) || ' days'), 'Alice', 20, 11, 'Bob', 21, 12),
-- Day 16: Bob,   Total Count   (Bob=21 < Alice=22 after swap resolution)
(16, date('now', (16 - strftime('%d','now')) || ' days'), 'Alice', 22, 12, 'Bob', 21, 11),
-- Day 17: Alice, Override – no details (manually changed, fairness not calculated)
-- Day 18: Bob,   Unavailability  (Alice unavailable; Bob assigned regardless of counts)
(18, date('now', (18 - strftime('%d','now')) || ' days'), 'Alice', 22, 12, 'Bob', 22, 12),
-- Day 19: Alice, Total Count   (Alice=22 < Bob=23)
(19, date('now', (19 - strftime('%d','now')) || ' days'), 'Alice', 22, 12, 'Bob', 23, 13),
-- Day 20: Bob,   Alternating   (totals tied=23, recent tied=13, previous=Alice → Bob)
(20, date('now', (20 - strftime('%d','now')) || ' days'), 'Alice', 23, 13, 'Bob', 23, 13),
-- Day 21: Alice, Total Count   (Alice=23 < Bob=24)
(21, date('now', (21 - strftime('%d','now')) || ' days'), 'Alice', 23, 13, 'Bob', 24, 14),
-- Day 22: Bob,   Consecutive Limit  (totals tied=24, Alice had 2 consecutive → switch)
(22, date('now', (22 - strftime('%d','now')) || ' days'), 'Alice', 24, 14, 'Bob', 24, 14),
-- Day 23: Alice, Total Count   (Alice=24 < Bob=25)
(23, date('now', (23 - strftime('%d','now')) || ' days'), 'Alice', 24, 14, 'Bob', 25, 15),
-- Day 24: babysitter (Emma) – no details
-- Day 25: Bob,   Recent Count  (babysitter +1 both; totals tied=26, Bob has fewer last-30: 15 < 16)
(25, date('now', (25 - strftime('%d','now')) || ' days'), 'Alice', 26, 16, 'Bob', 26, 15),
-- Day 26: Alice, Total Count   (Alice=26 < Bob=27)
(26, date('now', (26 - strftime('%d','now')) || ' days'), 'Alice', 26, 15, 'Bob', 27, 16),
-- Day 27: Bob,   Alternating   (totals tied=27, recent tied=16, previous=Alice → Bob)
(27, date('now', (27 - strftime('%d','now')) || ' days'), 'Alice', 27, 16, 'Bob', 27, 16),
-- Day 28: Alice, Total Count   (Alice=27 < Bob=28)
(28, date('now', (28 - strftime('%d','now')) || ' days'), 'Alice', 27, 16, 'Bob', 28, 17),
-- Day 29: Bob,   Alternating   (totals tied=28, recent tied=17, previous=Alice → Bob)
(29, date('now', (29 - strftime('%d','now')) || ' days'), 'Alice', 28, 17, 'Bob', 28, 17),
-- Day 30: Alice, Recent Count  (totals tied=29, Alice has fewer last-30: 17 < 18)
(30, date('now', (30 - strftime('%d','now')) || ' days'), 'Alice', 28, 17, 'Bob', 29, 18);

-- Insert OAuth token (JSON format)
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
2. **Double Consecutive Swap cells** — Navigate to days 14–15 to show the swap pair (both show "Double Consecutive Swap" reason in adjacent cells)
3. **Babysitter night** — Click the Dawn (day 10) or Emma (day 24) cell; modal shows babysitter note
4. **Total Count modal** — Click day 1 (Alice, Total Count); modal shows Alice with fewer total assignments
5. **Recent Count modal** — Click day 5 (Alice) or day 30 (Alice); modal shows equal totals but Alice fewer last-30
6. **Consecutive Limit modal** — Click day 4 or 9 or 22; shows equal totals with one parent having 2+ consecutive nights
7. **Double Consecutive Swap modal** — Click day 14 or 15; shows explanation of the AA BB → AB AB swap
8. **Override cell** — Click day 17 (Alice, Override); unlock modal appears

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
