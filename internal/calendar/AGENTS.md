# internal/calendar

Google Calendar API integration: event sync, notification channels, calendar management.

## Purpose

Handles all interaction with Google Calendar: creating/updating night routine events, managing push notification channels for real-time change detection, and listing available calendars for user selection.

## Key Types

- `Service` — Main calendar service (authenticated via OAuth2 token).
- `CalendarService` — Interface for dependency injection and testing.

## Key Operations

| Method                                           | Purpose                                              |
| ------------------------------------------------ | ---------------------------------------------------- |
| `Initialize(ctx)`                                | Authenticate with stored OAuth token                 |
| `SyncSchedule(ctx, assignments)`                 | Create/update/delete calendar events for assignments |
| `SetupNotificationChannel(ctx)`                  | Register push notification channel with Google       |
| `StopNotificationChannel(ctx, id, resourceID)`   | Unregister notification channel                      |
| `VerifyNotificationChannel(ctx, id, resourceID)` | Check channel validity                               |
| `ListCalendars(ctx)`                             | List user's calendars for selection                  |

## Calendar Events

- Title format: `[Name] 🌃👶Routine` (for both parents and babysitters)
- Private extended property `app = "night-routine"` marks events as owned by this app
- Events store the Google Calendar event ID back in the `assignments` table

## Notification Channels

- Google pushes change notifications to `/api/webhook/calendar`
- Channels have expiration times and are renewed proactively
- Channel metadata stored in `notification_channels` database table

## Dependencies

- Uses: `internal/database`, `internal/token`, `internal/config`, `internal/fairness/scheduler`, `google.golang.org/api/calendar/v3`
- Used by: `cmd/night-routine`, `internal/handlers` (sync, webhook)
