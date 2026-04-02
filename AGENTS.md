# Night Routine Scheduler

A Go application that automates night routine scheduling between two parents with Google Calendar integration and babysitter support.

## Architecture Overview

```
cmd/night-routine/     Entry point: bootstrap, service loop, signal handling
internal/
  ├── config/          Configuration: TOML + env + database-backed runtime config
  ├── database/        SQLite layer: migrations, CRUD, config/token stores
  ├── fairness/        Core scheduling algorithm + assignment tracking
  │   └── scheduler/   Schedule generation with fairness rules
  ├── calendar/        Google Calendar API: event sync, notification channels
  ├── handlers/        HTTP handlers + web UI templates + static assets
  ├── token/           OAuth2 token lifecycle management
  ├── signals/         Event bus: TokenSetup, CalendarSelected
  ├── logging/         Zerolog-based structured logging
  ├── constants/       Shared enums and identifiers
  └── viewhelpers/     Calendar grid preparation for templates
configs/               Default TOML configuration
docs/                  Internal architecture and planning docs
docs-site/             Public MkDocs documentation
build/                 Dockerfile for multi-arch production images
```

## Data Flow

```
User action (web UI / Google Calendar edit)
  → HTTP handler (internal/handlers)
    → Fairness tracker (internal/fairness) — update assignment in DB
    → Scheduler (internal/fairness/scheduler) — regenerate schedule
    → Calendar service (internal/calendar) — sync to Google Calendar
```

## Key Conventions

- **CGO-free**: Uses `modernc.org/sqlite` (pure Go). `CGO_ENABLED=0` always.
- **Error wrapping**: Always use `fmt.Errorf("context: %w", err)`.
- **Logging**: `zerolog` only, via `logging.GetLogger("component")`.
- **Config**: File/env for static settings, database for UI-configurable settings.
- **Tests**: Table-driven tests, regression tests for every bug fix.
- **Build**: `pnpm install` → `go generate ./...` → `go build`.
- **Commits**: Conventional commits (`fix(scope):`, `feat(scope):`, etc.).

## Fairness Algorithm

Decision cascade (first match wins):

1. **Unavailability** — One parent unavailable → assign other
2. **TotalCount** — Parent with fewer total assignments
3. **ConsecutiveLimit** — If tied, force switch after 2+ consecutive same-parent days
4. **RecentCount** — If tied and no streak, parent with fewer last-30-day assignments
5. **Alternating** — Default: alternate from last parent

Babysitter assignments are **always excluded** from parent fairness calculations.
