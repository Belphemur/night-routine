# Handlers Package — Agent Guide

This document provides guidance for AI agents working inside `internal/handlers/`.

## Package Overview

The `handlers` package contains all HTTP request handlers for the Night Routine Scheduler web UI and API. Each handler follows a consistent pattern:

- Embeds `*BaseHandler` for shared template rendering, authentication checking, and common state
- Registers its own routes via a `RegisterRoutes()` method
- Is instantiated once in `cmd/night-routine/main.go` and reused across requests

## Handler Inventory

| File | Purpose |
|---|---|
| `base_handler.go` | Shared template rendering, auth checks, `BasePageData` |
| `webhook_handler.go` | Processes Google Calendar webhook notifications; updates parent assignments |
| `settings_handler.go` | Settings page UI and form submission; triggers calendar sync on save |
| `home_handler.go` | Home / dashboard page |
| `calendar_handler.go` | Calendar selection and management |
| `sync_handler.go` | Manual schedule sync endpoint |
| `statistics_handler.go` | Statistics and fairness display |
| `assignment_details_handler.go` | Per-assignment detail view |
| `unlock_handler.go` | Unlock a locked assignment |
| `oauth_handler.go` | Google OAuth2 callback |
| `static_handler.go` | Serves embedded CSS / logo with ETag caching |
| `errors.go` | Error and success code constants for redirect URLs |

## Key Architecture Patterns

### BaseHandler Composition

All handlers embed `*BaseHandler`:

```go
type WebhookHandler struct {
    *BaseHandler
    // handler-specific fields
}
```

`BaseHandler` provides:
- `ConfigStore config.ConfigStoreInterface` — the **single source of truth** for all application configuration; reads schedule/parent/availability settings live from the database and returns the static OAuth config from the file/env config
- `TokenStore *database.TokenStore` — OAuth token and notification-channel storage
- `Tracker fairness.TrackerInterface` — assignment history
- `RenderTemplate(w, name, data)` — clones + executes templates from the embedded FS
- `CheckAuthentication(ctx, logger)` — validates the stored OAuth token

### Single Source of Truth — `ConfigStoreInterface`

`ConfigStoreInterface` (implemented by `database.ConfigAdapter`) is the **only** place handlers and services read configuration from. It provides:

- `GetSchedule()` — live DB read of `UpdateFrequency`, `LookAheadDays`, `PastEventThresholdDays`, `StatsOrder`
- `GetParents()` — live DB read of parent names
- `GetAvailability(parent)` — live DB read of unavailability days
- `GetOAuthConfig()` — returns the static `*oauth2.Config` (from environment / file config)

Never read from `RuntimeConfig` directly in handlers or services:

```go
// ✅ Correct – reads live from the database
_, lookAheadDays, thresholdDays, _, err := h.ConfigStore.GetSchedule()
oauthCfg := h.ConfigStore.GetOAuthConfig()

// ❌ Wrong – stale startup snapshot; does not reflect UI changes
thresholdDays := h.RuntimeConfig.Config.Schedule.PastEventThresholdDays
oauthCfg := h.RuntimeConfig.Config.OAuth
```

All handlers and functions that read any user-configurable setting follow this pattern:
- `WebhookHandler.processEvents` — reads `PastEventThresholdDays` live
- `WebhookHandler.recalculateSchedule` — reads `LookAheadDays` live
- `WebhookHandler.processEventChanges` — reads OAuth config live via `ConfigStore.GetOAuthConfig()`
- `OAuthHandler` — reads OAuth config via `BaseHandler.ConfigStore.GetOAuthConfig()`
- `SyncHandler.updateScheduleWithDate` — reads `LookAheadDays` live
- `main.updateSchedule` — reads `LookAheadDays` live
- `main` service loop ticker — reads `UpdateFrequency` live on every tick

### Template Rendering

Templates live in `templates/*.html` and are embedded via `//go:embed`. Use `RenderTemplate`:

```go
h.RenderTemplate(w, "page.html", PageData{
    BasePageData: h.NewBasePageData(r, isAuth),
    // page-specific fields
})
```

Always embed `BasePageData` as the first field in page data structs so layout variables (year, CSS ETag, auth state) are available to the layout template.

## Webhook Handler Deep Dive

`webhook_handler.go` handles `POST /api/webhook/calendar` — notifications sent by Google Calendar when events change.

### Processing Pipeline

1. **Validate channel** — look up the channel ID in `TokenStore`; reject unknown IDs
2. **Check expiry** — renew the notification channel if it expires within 7 days
3. **Filter sync messages** — return 200 immediately for `resource_state: sync`
4. **Fetch recent events** — list events updated in the last 2 minutes via the Calendar API; OAuth client obtained via `ConfigStore.GetOAuthConfig()`
5. **Process events** — `processEvents` iterates events:
   - Skip cancelled events, non–Night-Routine events
   - Extract parent name from summary format `[ParentName] 🌃👶Routine`
   - Find the matching `Assignment` by Google Calendar event ID
   - Skip if parent name is unchanged
   - **Read `PastEventThresholdDays` live from `ConfigStore`** — reject the update if the assignment is older than the threshold
   - Call `Scheduler.UpdateAssignmentParent` and then `recalculateSchedule`
6. **Recalculate schedule** — `recalculateSchedule` regenerates future assignments from the changed date and syncs them back to Google Calendar; **reads `LookAheadDays` live from `ConfigStore`**

### Why ConfigStore for All Settings

All settings the user can change via the UI (schedule, parents, availability) are stored in the database. Reading from `ConfigStore` on every request ensures the latest value is used immediately — no app restart required. The OAuth config (static, from environment) is also served through `ConfigStore.GetOAuthConfig()` so handlers have a single dependency.

## Testing Patterns

Tests follow two styles depending on complexity:

### Unit tests with mocks (preferred for logic isolation)

```go
mockConfigStore := new(MockConfigStore)
mockConfigStore.On("GetSchedule").Return("daily", 7, 5, constants.StatsOrderDesc, nil)
mockConfigStore.On("GetOAuthConfig").Return((*oauth2.Config)(nil))

handler := &WebhookHandler{
    BaseHandler: &BaseHandler{
        Tracker:     mockTracker,
        ConfigStore: mockConfigStore,
    },
    ConfigStore: mockConfigStore,
    // ...
}
```

Use `.Maybe()` on optional expectations (calls that only happen on certain code paths):

```go
mockConfigStore.On("GetSchedule").Maybe().Return(...)
```

### Integration tests with a real SQLite database

```go
db, _ := database.New(database.NewDefaultOptions(filepath.Join(t.TempDir(), "test.db")))
db.MigrateDatabase()
configStore, _ := database.NewConfigStore(db)
configStore.SaveSchedule("daily", 7, 5, constants.StatsOrderDesc)

configAdapter := database.NewConfigAdapter(configStore, nil) // nil OAuth in tests

handler := &WebhookHandler{
    BaseHandler: &BaseHandler{
        Tracker:     tracker,
        ConfigStore: configAdapter,
    },
    ConfigStore: configAdapter,
}
```

Integration tests are preferred when testing database interactions or the dynamic-config behaviour (verifying that a `SaveSchedule` call is immediately visible to the handler without restart).

### Mock types defined in this package

All mock types are defined in `webhook_handler_test.go` and are available to all tests in the package:

- `MockTracker` — `fairness.TrackerInterface`
- `MockScheduler` — `Scheduler.SchedulerInterface`
- `MockCalendarService` — `calendar.CalendarService`
- `MockConfigStore` — `config.ConfigStoreInterface`

## Common Mistakes to Avoid

1. **Reading any config from `RuntimeConfig` in handlers or services** — always use `ConfigStore.GetSchedule()`, `ConfigStore.GetParents()`, `ConfigStore.GetAvailability()`, or `ConfigStore.GetOAuthConfig()`.
2. **Forgetting to add `ConfigStore` to integration test handler structs** — the handler will panic if `ConfigStore` is nil.
3. **Adding a new field to `BaseHandler` instead of a specific handler** — `BaseHandler` is shared; put handler-specific state in the concrete handler struct.
4. **Not providing both `up` and `down` migration files** when adding a new config column — see `internal/database/migrations/sqlite/`.

## Build & Test Commands

```bash
# Format
go fmt ./internal/handlers/...

# Lint
golangci-lint run ./internal/handlers/...

# Test (all handlers)
go test ./internal/handlers/ -v

# Test (specific test)
go test ./internal/handlers/ -run TestWebhookHandler_DynamicConfigReading -v
```
