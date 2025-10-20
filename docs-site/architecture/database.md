# Database Structure

Night Routine Scheduler uses SQLite for persistent storage with WAL (Write-Ahead Logging) mode for better concurrency.

## Database Location

The database file location is configured in `routine.toml`:

```toml
[service]
state_file = "data/state.db"
```

## Database Features

- **SQLite 3** - Lightweight, serverless database
- **WAL Mode** - Write-Ahead Logging for better concurrency
- **Foreign Key Constraints** - Data integrity enforcement
- **Automatic Migrations** - Schema updates on application startup
- **Incremental Auto-Vacuum** - Automatic database maintenance

## Schema

### Tables

#### `assignments`

Stores night routine assignment history and fairness tracking.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PRIMARY KEY | Auto-incrementing ID |
| `date` | TEXT UNIQUE NOT NULL | ISO date (YYYY-MM-DD) |
| `parent` | TEXT NOT NULL | Parent name (Parent A or Parent B) |
| `reason` | TEXT NOT NULL | Decision reason |
| `created_at` | TEXT NOT NULL | Creation timestamp |
| `updated_at` | TEXT NOT NULL | Last update timestamp |

**Indexes:**
- Primary key on `id`
- Unique index on `date`
- Index on `parent` for fast lookups

**Decision Reasons:**
- `Unavailability` - One parent was unavailable
- `Total Count` - Balance total assignment counts
- `Recent Count` - Balance recent assignments
- `Consecutive Limit` - Avoid too many consecutive assignments
- `Alternating` - Maintain alternating pattern
- `Override` - Manual change via Google Calendar

#### `oauth_tokens`

Stores Google OAuth2 access and refresh tokens.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PRIMARY KEY | Always 1 (single row table) |
| `access_token` | TEXT NOT NULL | OAuth2 access token |
| `refresh_token` | TEXT NOT NULL | OAuth2 refresh token |
| `token_type` | TEXT NOT NULL | Token type (usually "Bearer") |
| `expiry` | TEXT NOT NULL | Token expiration timestamp |
| `created_at` | TEXT NOT NULL | Creation timestamp |
| `updated_at` | TEXT NOT NULL | Last update timestamp |

**Notes:**
- Only one row exists (enforced by application logic)
- Tokens are automatically refreshed when expired
- Access tokens expire after ~1 hour
- Refresh tokens are long-lived (until revoked)

#### `calendar_settings`

Stores selected Google Calendar configuration.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PRIMARY KEY | Always 1 (single row table) |
| `calendar_id` | TEXT NOT NULL | Google Calendar ID |
| `calendar_name` | TEXT NOT NULL | Human-readable calendar name |
| `created_at` | TEXT NOT NULL | Creation timestamp |
| `updated_at` | TEXT NOT NULL | Last update timestamp |

**Notes:**
- Only one row exists (single calendar selection)
- `calendar_id` format: email or calendar ID from Google
- `calendar_name` is for display purposes

#### `notification_channels`

Manages Google Calendar webhook notification channels.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PRIMARY KEY | Auto-incrementing ID |
| `channel_id` | TEXT UNIQUE NOT NULL | Google notification channel ID |
| `resource_id` | TEXT NOT NULL | Google resource ID |
| `expiration` | TEXT NOT NULL | Channel expiration timestamp |
| `created_at` | TEXT NOT NULL | Creation timestamp |
| `updated_at` | TEXT NOT NULL | Last update timestamp |

**Indexes:**
- Primary key on `id`
- Unique index on `channel_id`

**Notes:**
- Channels expire and must be renewed (typically every 7-30 days)
- Application automatically manages channel lifecycle
- Multiple channels may exist during renewal periods

## Migrations

The application uses [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations.

### Migration Files

Migrations are embedded in the application binary and located at:

```
internal/database/migrations/
├── 000001_initial_schema.up.sql
├── 000001_initial_schema.down.sql
├── 000002_add_indexes.up.sql
├── 000002_add_indexes.down.sql
└── ...
```

### Automatic Migration

Migrations run automatically on application startup:

1. Database connection is established
2. Current schema version is checked
3. Pending migrations are applied in order
4. Application starts normally

**Log output:**
```
INF Connecting to database file=data/state.db
INF Running database migrations
INF Migration applied version=1 name=initial_schema
INF Migration applied version=2 name=add_indexes
INF Database ready
```

### Manual Migration (Advanced)

You can manually check or migrate using the Go migrate CLI:

```bash
# Install migrate
go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Check current version
migrate -path internal/database/migrations -database "sqlite3://data/state.db" version

# Apply all pending migrations
migrate -path internal/database/migrations -database "sqlite3://data/state.db" up

# Rollback last migration
migrate -path internal/database/migrations -database "sqlite3://data/state.db" down 1
```

## WAL Mode

The database uses Write-Ahead Logging (WAL) mode for improved concurrency.

### Benefits

- **Better Concurrency** - Readers don't block writers
- **Faster Writes** - Append-only writes to WAL file
- **Atomic Transactions** - Guaranteed consistency
- **Crash Recovery** - Automatic recovery from crashes

### File Structure

When using WAL mode, you'll see three files:

```
data/
├── state.db        # Main database file
├── state.db-shm    # Shared memory file
└── state.db-wal    # Write-ahead log file
```

!!! warning "Backup Considerations"
    When backing up the database, include all three files or use the SQLite backup API to ensure consistency.

## Querying the Database

### Using SQLite CLI

```bash
# Open the database
sqlite3 data/state.db

# View schema
.schema

# Query assignments
SELECT date, parent, reason FROM assignments ORDER BY date DESC LIMIT 10;

# Count assignments by parent
SELECT parent, COUNT(*) as count FROM assignments GROUP BY parent;

# View OAuth token expiry
SELECT expiry FROM oauth_tokens;
```

### Common Queries

**Get current month's assignments:**
```sql
SELECT date, parent, reason 
FROM assignments 
WHERE date >= date('now', 'start of month')
  AND date < date('now', '+1 month', 'start of month')
ORDER BY date;
```

**Count assignments by parent (last 30 days):**
```sql
SELECT parent, COUNT(*) as count
FROM assignments
WHERE date >= date('now', '-30 days')
GROUP BY parent;
```

**Find all manual overrides:**
```sql
SELECT date, parent
FROM assignments
WHERE reason = 'Override'
ORDER BY date DESC;
```

**Check token expiration:**
```sql
SELECT 
  datetime(expiry) as expires_at,
  datetime(expiry, 'localtime') as expires_local,
  (julianday(expiry) - julianday('now')) * 24 as hours_remaining
FROM oauth_tokens;
```

## Data Integrity

### Foreign Key Constraints

Foreign key constraints are enabled to maintain referential integrity:

```sql
PRAGMA foreign_keys = ON;
```

### Transaction Safety

All database operations use transactions to ensure atomic updates:

```go
tx, err := db.Begin()
// ... perform operations
tx.Commit() // or tx.Rollback() on error
```

### Unique Constraints

- Assignments: One assignment per date
- OAuth tokens: Single token set
- Calendar settings: Single calendar selection
- Notification channels: Unique channel IDs

## Performance Considerations

### Indexes

Strategic indexes for common queries:

- **assignments.date** - Fast date lookups
- **assignments.parent** - Quick parent filtering
- **notification_channels.channel_id** - Fast channel lookups

### Query Optimization

The application uses prepared statements for:

- Inserting new assignments
- Updating existing assignments
- Checking assignment counts
- Token refresh operations

### Connection Pooling

SQLite doesn't support true connection pooling, but the application:

- Reuses a single connection
- Uses WAL mode for concurrency
- Implements proper locking

## Backup and Restore

### Manual Backup

```bash
# Stop the application first
# Then copy all database files
cp data/state.db data/state.db.backup
cp data/state.db-shm data/state.db-shm.backup 2>/dev/null
cp data/state.db-wal data/state.db-wal.backup 2>/dev/null
```

### Online Backup (SQLite Backup API)

```bash
# Using sqlite3 CLI
sqlite3 data/state.db ".backup data/state.db.backup"
```

### Automated Backup Script

```bash
#!/bin/bash
BACKUP_DIR="/path/to/backups"
DATE=$(date +%Y%m%d_%H%M%S)
sqlite3 data/state.db ".backup ${BACKUP_DIR}/state_${DATE}.db"
# Keep only last 30 days
find ${BACKUP_DIR} -name "state_*.db" -mtime +30 -delete
```

### Restore from Backup

```bash
# Stop the application
# Replace the database file
cp data/state.db.backup data/state.db
# Remove WAL files to force checkpoint
rm -f data/state.db-shm data/state.db-wal
# Restart the application
```

## Maintenance

### Vacuum

WAL mode uses automatic vacuuming, but you can manually optimize:

```sql
-- Check database size
SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size();

-- Checkpoint WAL
PRAGMA wal_checkpoint(TRUNCATE);

-- Vacuum (reclaim space)
VACUUM;

-- Analyze (update statistics)
ANALYZE;
```

### Integrity Check

```sql
-- Check database integrity
PRAGMA integrity_check;

-- Quick check
PRAGMA quick_check;

-- Check foreign keys
PRAGMA foreign_key_check;
```

## Troubleshooting

### Database Locked Errors

**Symptom:** `database is locked` errors

**Causes:**
- Multiple processes accessing the database
- Long-running transaction
- WAL checkpoint in progress

**Solutions:**
1. Ensure only one instance is running
2. Check for hung processes
3. WAL mode should prevent most locking issues

### Corruption

**Symptom:** Integrity check fails

**Solutions:**
1. Restore from backup
2. Try `.recover` command in sqlite3 CLI
3. If minor corruption, `.dump` and `.restore`

### Performance Degradation

**Symptom:** Slow queries

**Solutions:**
1. Run `ANALYZE` to update statistics
2. Checkpoint the WAL: `PRAGMA wal_checkpoint(TRUNCATE)`
3. Consider `VACUUM` if database has grown large
4. Check for missing indexes

## Next Steps

- [Learn about assignment logic](assignment-logic.md)
- [Understand the architecture](overview.md)
- [Explore development setup](../development/local.md)
