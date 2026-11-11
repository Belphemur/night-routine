# internal/handlers

HTTP request handlers and web UI: all routes, templates, and static assets.

## Purpose

Implements all HTTP endpoints for the web interface: home calendar page, settings, statistics, OAuth flow, Google Calendar sync, webhook processing, unlock/override, assignment details, and static asset serving. Each handler registers its own routes via a `RegisterRoutes()` method called from `cmd/night-routine/main.go`.

## Handler Overview (actual routes)

| Handler | Routes | Purpose |
|---------|--------|---------|
| `BaseHandler` | (shared) | Template rendering, auth checks, page data |
| `HomeHandler` | `GET /` | Calendar month view with assignments |
| `OAuthHandler` | `GET /auth`, `GET /oauth/callback` | Google OAuth2 flow |
| `CalendarHandler` | `GET /calendars` | List and select Google Calendars |
| `SettingsHandler` | `GET /settings`, `POST /settings/update` | Runtime config management |
| `StatisticsHandler` | `GET /statistics` | Monthly stats per parent/babysitter |
| `SyncHandler` | `GET /sync`, `POST /api/sync` | Manual sync (page + API) |
| `UnlockHandler` | `POST /unlock` | Unlock / override an assignment |
| `AssignmentDetailsHandler` | `GET /api/assignment-details`, `POST /api/assignment-babysitter` | Fairness decision details; set babysitter |
| `WebhookHandler` | `POST /api/webhook/calendar` | Process Google Calendar push notifications |
| `StaticHandler` | `GET /static/css/tailwind.css`, `/favicon.ico`, `/static/images/favicon.png`, `/static/images/logo.png` | CSS and images with ETag caching |

Route-level files in the package: `home_handler.go`, `oauth_handler.go`, `calendar_handler.go`, `settings_handler.go`, `statistics_handler.go`, `sync_handler.go`, `unlock_handler.go`, `assignment_details_handler.go`, `webhook_handler.go`, `static_handler.go`, plus `schedule_recalculation.go` (shared regenerate+sync logic) and `errors.go`.

## Templates

Located in `templates/` (embedded via `//go:embed`):

- `layout.html` — Base layout with navigation bar
- `home.html` — Calendar grid with assignment cards (largest template)
- `settings.html` — Configuration forms
- `statistics.html` — Monthly statistics
- `calendars.html` — Calendar selection list

## Static Assets

Located in `assets/` (embedded via `//go:embed`):

- `css/input.css` → `css/tailwind.css` — Generated Tailwind CSS (minified)
- `images/` — Logo and favicon

## CSS Generation

`//go:generate pnpm run build:css` in `base_handler.go` runs Tailwind (`input.css` → `tailwind.css --minify`). Must be re-run after any template or CSS input change; output is embedded in the binary.

## Key Patterns

- **Live config reads**: Handlers read configuration from the database on every request (no restart needed for changes). `configAdapter` is passed to handlers so schedule settings always come from the DB.
- **Schedule recalculation**: After overrides, unlocks, babysitter changes or settings updates, handlers trigger `GenerateSchedule` + `SyncSchedule` (see `schedule_recalculation.go`).
- **ETag versioning**: Tailwind CSS and logo use content-based ETags for cache busting.

## Tests

`home_handler_test.go`, `settings_handler_test.go`, `statistics_handler_test.go`, `unlock_handler_test.go`, `webhook_handler_test.go`, `static_handler_test.go`, `assignment_details_handler_test.go` (table-driven, httptest).

## Dependencies

- Uses: `internal/database`, `internal/token`, `internal/config`, `internal/calendar`, `internal/fairness`, `internal/viewhelpers`, `internal/logging`, `internal/signals`
- Used by: `cmd/night-routine` (handler construction + route registration)
