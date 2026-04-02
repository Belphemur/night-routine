# internal/database

SQLite data layer: connection management, schema migrations, and CRUD operations.

## Purpose

All persistent storage goes through this package. Uses pure-Go `modernc.org/sqlite` (CGO-free, driver name `"sqlite"`). Schema is managed via numbered migration files embedded in the binary.

## Key Types

- `DB` — Wraps `*sql.DB` with migration support and transaction helpers.
- `SQLiteOptions` — Connection configuration (WAL journal, shared cache, busy timeout, etc.).
- `TokenStore` — OAuth token CRUD (save/get/clear).
- `ConfigStore` — Runtime configuration CRUD (parents, availability, schedule).
- `ConfigAdapter` — Bridges `ConfigStore` to `config.ConfigStoreInterface` (adds static OAuth config).
- `ConfigSeeder` — Seeds initial database config from TOML file on first run.
- `NotificationChannel` — Google Calendar push notification channel records.

## Database Schema (key tables)

| Table | Purpose |
|-------|---------|
| `assignments` | Night routine assignments (parent, date, override, caregiver_type, babysitter_name, decision_reason, google_calendar_event_id) |
| `assignment_details` | Fairness calculation snapshots for each assignment |
| `oauth_tokens` | OAuth2 token storage (JSONB) |
| `calendar_settings` | Selected Google Calendar ID |
| `notification_channels` | Google Calendar push notification registrations |
| `config_parents` | Parent names (A and B) |
| `config_availability` | Per-parent unavailable days |
| `config_schedule` | Schedule settings (frequency, lookahead, stats order) |

## Migrations

- Located in `migrations/sqlite/` (embedded via `//go:embed`)
- Numbered sequentially: `000001_description.up.sql` / `.down.sql`
- **Never** modify existing migrations; always create new ones
- Run automatically on startup via `MigrateDatabase()`

## Key Functions

- `New(opts SQLiteOptions) (*DB, error)` — Open connection with PRAGMAs.
- `MigrateDatabase()` — Run embedded migrations.
- `WithTransaction(ctx, fn)` — Execute function in a transaction.

## Dependencies

- Uses: `modernc.org/sqlite`, `golang-migrate/migrate`
- Used by: `cmd/night-routine`, `internal/fairness`, `internal/token`, `internal/handlers`
