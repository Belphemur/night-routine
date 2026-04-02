# internal/config

Application configuration loading and management with a 4-tier precedence system.

## Purpose

Loads configuration from TOML files + environment variables at startup, then merges with database-backed runtime configuration for settings that can be changed via the web UI.

## Precedence (lowest → highest)

1. Built-in defaults
2. TOML file (`configs/routine.toml`)
3. Legacy env vars (`PORT`, `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`)
4. `NR_*` env vars (e.g., `NR_PARENTS__PARENT_A=Alice`)

## Key Types

- `Config` — Root struct holding all configuration sections (`Parents`, `Availability`, `Schedule`, `Service`, `App`, `Credentials`, `OAuth`).
- `RuntimeConfig` — Merged file + database config, used at runtime.
- `ConfigStoreInterface` — Interface for database-backed config reads (implemented by `database.ConfigAdapter`).
- `ConfigLoader` — Interface bridging file-based and DB-based config.

## Key Functions

- `Load(path string) (*Config, error)` — Load from TOML with env overrides using koanf.
- `LoadRuntimeConfig(fileConfig, loader) (*RuntimeConfig, error)` — Merge file + DB config.

## File vs Database Config

| Static (file/env, never changes at runtime) | Dynamic (database, UI-configurable) |
|---------------------------------------------|-------------------------------------|
| OAuth credentials | Parent names |
| App URL / port | Availability (unavailable days) |
| State file path | Schedule frequency & lookahead |
| Log level | Calendar ID |

## Dependencies

- Uses: `internal/constants`, koanf, mapstructure, oauth2
- Used by: `cmd/night-routine`, `internal/database`, `internal/handlers`
