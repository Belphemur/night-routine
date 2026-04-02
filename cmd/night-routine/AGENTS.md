# cmd/night-routine

Application entry point and main service loop.

## Purpose

Bootstraps all application components, starts the HTTP server, and runs the scheduling loop.

## Startup Sequence

1. Initialize logging (dev vs production based on `ENV`)
2. Load configuration (TOML file + environment variable overrides)
3. Create SQLite database + run migrations
4. Seed database config from TOML (first run only)
5. Initialize services: TokenManager, Fairness Tracker, Scheduler, Calendar Service
6. Register all HTTP handlers on the router
7. Start HTTP server
8. Register signal listeners (TokenSetup → init calendar, CalendarSelected → setup notifications)
9. Optionally run manual sync on startup

## Main Loop

- Ticks every minute
- Reads `UpdateFrequency` and `LookAheadDays` live from the database (no restart needed)
- When interval has elapsed: generates schedule → syncs to Google Calendar
- Handles graceful shutdown via context cancellation

## Dependencies

Uses all `internal/` packages. This is the only package that imports and wires everything together.
