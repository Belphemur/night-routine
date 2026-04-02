# internal/handlers

HTTP request handlers and web UI: all routes, templates, and static assets.

## Purpose

Implements all HTTP endpoints for the web interface: calendar home page, settings management, statistics, OAuth flow, Google Calendar sync, webhook processing, and static asset serving.

## Handler Overview

| Handler | Routes | Purpose |
|---------|--------|---------|
| `BaseHandler` | (shared) | Template rendering, auth checks, page data |
| `HomeHandler` | `GET /` | Calendar month view with assignments |
| `OAuthHandler` | `GET /oauth/login`, `/oauth/callback` | Google OAuth2 flow |
| `CalendarHandler` | `GET /api/calendars`, `POST /calendar/select` | List and select calendars |
| `SettingsHandler` | `GET /settings`, `POST /api/settings/*` | Runtime config management |
| `StatisticsHandler` | `GET /statistics`, `GET /api/statistics/*` | Monthly stats per parent/babysitter |
| `UnlockHandler` | `POST /api/assignments/{id}/unlock` | Remove override from assignment |
| `AssignmentDetailsHandler` | `GET /api/assignments/{id}/details` | Show fairness calculation details |
| `SyncHandler` | `POST /api/sync` | Manually trigger Google Calendar sync |
| `WebhookHandler` | `POST /webhook/calendar` | Process Google Calendar push notifications |
| `StaticHandler` | `GET /css/*`, `/images/*`, `/logo` | CSS and images with ETag caching |

## Templates

Located in `templates/` (embedded via `//go:embed`):

- `layout.html` — Base layout with navigation bar
- `home.html` — Calendar grid with assignment cards (largest template)
- `settings.html` — Configuration forms
- `statistics.html` — Monthly statistics charts
- `calendars.html` — Calendar selection list

## Static Assets

Located in `assets/` (embedded via `//go:embed`):

- `css/tailwind.css` — Generated Tailwind CSS (regenerated via `go generate`)
- `images/` — Logo and icons

## CSS Generation

The `go generate` directive in `base_handler.go` triggers Tailwind CSS compilation:
```
//go:generate pnpm run build:css
```
Must be run after any template or CSS changes. Output is embedded in the binary.

## Key Patterns

- **Live config reads**: Handlers read configuration from the database on every request (no restart needed for changes).
- **Schedule recalculation**: After overrides, unlocks, or settings changes, handlers trigger `GenerateSchedule` + `SyncSchedule`.
- **ETag versioning**: CSS and logo files use content-based ETags for cache busting.

## Dependencies

- Uses: `internal/database`, `internal/token`, `internal/config`, `internal/calendar`, `internal/fairness`, `internal/viewhelpers`, `internal/logging`
- Used by: `cmd/night-routine` (route registration)
