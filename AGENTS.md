# Night Routine Scheduler

A Go application that automates night routine scheduling between two parents with Google Calendar integration and babysitter support.

**This is the root/hub file.** Every package keeps its own `AGENTS.md` with package-specific detail — read the one for the package you are touching before changing it.

```
cmd/night-routine/AGENTS.md
internal/
  config/AGENTS.md
  database/AGENTS.md
  fairness/AGENTS.md           (covers internal/fairness/scheduler too)
  calendar/AGENTS.md
  handlers/AGENTS.md
  token/AGENTS.md
  signals/AGENTS.md
  logging/AGENTS.md
  constants/AGENTS.md
  viewhelpers/AGENTS.md
```

## Architecture Overview

```
cmd/night-routine/     Entry point: bootstrap, service loop, signal handling
internal/
  ├── config/          Configuration: TOML + env (koanf) + database-backed runtime config
  ├── database/        SQLite layer: migrations, CRUD, config/token stores
  ├── fairness/        Assignment tracking + TrackerInterface; scheduler/ = schedule generation
  ├── calendar/        Google Calendar API: event sync, push notification channels
  ├── handlers/        HTTP handlers + web UI templates + embedded static assets
  ├── token/           OAuth2 token lifecycle management
  ├── signals/         Event bus: TokenSetup, CalendarSelected (maniartech/signals)
  ├── logging/         Zerolog-based structured logging
  ├── constants/       Shared enums and identifiers (NightRoutineIdentifier, StatsOrder)
  └── viewhelpers/     Calendar grid preparation for templates
configs/               Default TOML configuration (routine.toml)
docs/                  Internal architecture and planning docs
docs-site/             Public MkDocs documentation — do NOT touch when changing code docs
build/                 Dockerfile for multi-arch production images
```

## Data Flow

```
User action (web UI) or Google Calendar webhook (/api/webhook/calendar)
  → HTTP handler (internal/handlers)
    → Scheduler (internal/fairness/scheduler) — regenerate schedule
      → Tracker (internal/fairness) — persist assignment + decision details in DB
    → Calendar service (internal/calendar) — sync events to Google Calendar
```

Overrides, unlocks and settings changes flow through the same path; handlers trigger `GenerateSchedule` + `SyncSchedule` after mutating mutations.

## Key Conventions (global)

- **CGO-free**: Uses `modernc.org/sqlite` (pure Go, driver name `"sqlite"`). `CGO_ENABLED=0` always.
- **Error wrapping**: Always `fmt.Errorf("context: %w", err)`.
- **Logging**: `zerolog` only, via `logging.GetLogger("component")`. Never `fmt.Print`/`log.Print`.
- **Config**: File/env for static settings (OAuth, port, paths), database for UI-configurable settings (parents, availability, schedule, calendar ID). Handlers read config live from the DB — no restart needed.
- **Tests**: Table-driven tests; add a regression test for every bug fix.
- **Build**: `pnpm install` → `go generate ./...` → `go build`. `go generate` regenerates embedded Tailwind CSS (`internal/handlers/base_handler.go`) — required after template/CSS changes.
- **Commits**: Conventional commits (`fix(scope):`, `feat(scope):`, …). Releases are cut by semantic-release.
- **Migrations**: Embedded, numbered (`000001_*.up.sql`/`.down.sql`); never modify an existing migration, add a new one.

## Fairness Algorithm

Implemented in `internal/fairness/scheduler/scheduler.go` (`determineNextParent`). Decision cascade, first match wins:

0. **No prior parent assignments** — assign the parent with fewer (or equal) total assignments (`TotalCount`).
1. **Unavailability** — checked *before* the cascade: one parent unavailable on that weekday → assign the other; both unavailable → error.
2. **TotalCount** — parent with fewer total assignments.
3. **ConsecutiveLimit** — totals tied and last parent had ≥2 consecutive assignments → force switch.
4. **RecentCount** — totals tied, no streak → parent with fewer last-30-day assignments.
5. **Alternating** — default: alternate from the last parent.

`DecisionReason` values recorded per assignment: `TotalCount`, `RecentCount`, `ConsecutiveLimit`, `Alternating`, `Unavailability`, `Override`.

Babysitter assignments (`caregiver_type = 'babysitter'`, always `override`) are excluded from parent fairness calculations (streaks, stats, lastParent) and break the consecutive-run tracker.

## Details

See the package-level `AGENTS.md` files linked above; `internal/fairness/AGENTS.md` documents the tracker interface and babysitter rules in full.
