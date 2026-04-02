# internal/constants

Shared constants and enums used across multiple packages.

## Purpose

Provides application-wide identifiers and validated enum types to avoid magic strings.

## Key Exports

- `NightRoutineIdentifier = "Night Routine"` — Marks calendar events as owned by this app.
- `StatsOrder` — Enum for statistics display order (`"desc"` or `"asc"`), validated via `IsValid()` and `ParseStatsOrder()`.

## Dependencies

- Uses: none (foundational package)
- Used by: `internal/config`, `internal/handlers`, `internal/calendar`
