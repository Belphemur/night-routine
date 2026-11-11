# internal/calendar

Google Calendar API integration: event sync, notification channels, calendar management.

## Purpose

Handles all interaction with Google Calendar: creating/updating night routine events, managing push notification channels for real-time change detection, and listing available calendars for user selection.

## Files

- `calendar.go` — Main `Service`: initialization, `SyncSchedule`, event formatting
- `notification.go` — Push notification channel lifecycle (setup/renew/stop/verify)
- `manager.go` — `Manager`: calendar listing (`GetCalendarList`), selection (`SelectCalendar`/`SelectCalendarWithName`), reading selected calendar ID
- `interface.go` — `CalendarService` interface

## Key Types

- `Service` — Main calendar service (authenticated via OAuth2 token from `internal/token`).
- `Manager` — Calendar selection/management operations (backed by `database.TokenStore` / calendar settings).
- `CalendarService` — Interface for dependency injection and testing.

## Key Operations

| Method                                           | Purpose                                              |
| ------------------------------------------------ | ---------------------------------------------------- |
| `Initialize(ctx)`                                | Authenticate with stored OAuth token                 |
| `IsInitialized()`                                | Whether the service has an authenticated client      |
| `SyncSchedule(ctx, assignments)`                 | Create/update/delete calendar events (concurrently per assignment) |
| `SetupNotificationChannel(ctx)`                  | Register/renew push notification channel with Google |
| `StopNotificationChannel(ctx, id, resourceID)`   | Unregister a notification channel                    |
| `StopAllNotificationChannels(ctx)`               | Unregister all stored channels                       |
| `VerifyNotificationChannel(ctx, id, resourceID)` | Check channel validity                               |
| `ListCalendars(ctx)`                             | List user's calendars for selection                  |

## Calendar Events

- Summary format: `[Name] 🌃👶Routine` (`formatEventSummary`) for both parents and babysitters.
- Description includes the decision reason and is marked with `constants.NightRoutineIdentifier` ("Night Routine").
- Private extended property `app = "Night Routine"` (`constants.NightRoutineIdentifier`) marks events as owned by this app; the webhook ignores events without it.
- Events store the Google Calendar event ID back in the `assignments` table via `Scheduler.UpdateGoogleCalendarEventID`.
- All-day events, reminders disabled.

## Notification Channels

- Google pushes change notifications to `<AppURL>/api/webhook/calendar`.
- Channels have expiration times and are renewed proactively.
- Channel metadata stored in the `notification_channels` database table.

## Dependencies

- Uses: `internal/database`, `internal/token`, `internal/config`, `internal/constants`, `internal/fairness`, `internal/fairness/scheduler`, `google.golang.org/api/calendar/v3`
- Used by: `cmd/night-routine`, `internal/handlers` (sync, webhook, calendar select)
