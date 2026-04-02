# internal/signals

Application event system for decoupled inter-component communication.

## Purpose

Provides a publish/subscribe event bus so components can react to state changes without direct coupling. Uses the `github.com/maniartech/signals` library.

## Signals

| Signal | Emitted By | Listened By | Trigger |
|--------|-----------|-------------|---------|
| `TokenSetup` | `token.TokenManager` | `main.go` | OAuth token saved or cleared |
| `CalendarSelected` | `handlers.CalendarHandler` | `main.go` | User selects a Google Calendar |

## Key Functions

- `EmitTokenSetup(ctx, success bool)` — Notify that token state changed.
- `EmitCalendarSelected(ctx, calendarID string)` — Notify that calendar was selected.
- `OnTokenSetup(handler)` — Register listener for token events.
- `OnCalendarSelected(handler)` — Register listener for calendar selection events.

## Dependencies

- Uses: `github.com/maniartech/signals`
- Used by: `internal/token`, `internal/handlers`, `cmd/night-routine`
