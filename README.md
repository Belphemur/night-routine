# Night Routine Scheduler

A Go application that manages night routine scheduling between two parents, with Google Calendar integration for automated event creation.

## Screenshots

![Setup Screen](docs/screenshots/Setup.png)
_Initial setup screen where you connect to Google Calendar_

![Calendar Selection](docs/screenshots/Calendar_Selection.png)
_Select which calendar to use for night routine events_

## Quick Start with Docker

Pre-built multi-architecture Docker images (supporting both amd64 and arm64) are available in the GitHub Container Registry:

```bash
# Pull the latest release
docker pull ghcr.io/belphemur/night-routine:latest

# Run the container
docker run \
  -e GOOGLE_OAUTH_CLIENT_ID=your-client-id \
  -e GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret \
  -e PORT=8080 \
  -e CONFIG_FILE=/app/config/routine.toml \
  -v /path/to/config:/app/config \
  -v /path/to/data:/app/data \
  -p 8080:8080 \
  ghcr.io/belphemur/night-routine:latest
```

_Note: These images are signed using Sigstore Cosign and include SBOM attestations for enhanced security._

Available tags:

- `latest`: Most recent release
- `vX.Y.Z` (e.g., `v1.0.0`): Specific versions

## Quick Start with Docker Compose

For easier self-hosting, you can use the provided `docker-compose.yml` file:

```bash
# Downlaod file
https://github.com/Belphemur/night-routine/blob/main/docker-compose.yml

# Create the config directory
mkdir -p config
cp configs/routine.toml config/

# Edit the configuration file
nano config/routine.toml

# Edit docker-compose.yml to set your environment variables
nano docker-compose.yml

# Start the service
docker-compose up -d
```

This will create the necessary directories for configuration and data persistence, and start the application in the background.

## Features

- Fair schedule distribution between parents
- Google Calendar integration with OAuth2
- Configurable parent availability
- Automated scheduling with daily/weekly/monthly updates
- Manual schedule sync on application startup (enabled by default)
- Webhook endpoint for manual assignment overrides (e.g., via Google Calendar updates)
- Structured logging using [zerolog](https://github.com/rs/zerolog) with configurable levels
- Persistent storage using SQLite:
  - Assignment history and fairness tracking
  - OAuth2 tokens and refresh tokens
  - Selected Google Calendar ID
- Docker containerization
- Multi-architecture support (amd64, arm64)

## First-Time Setup

1. Start the application
2. Visit the URL specified in your config's `app_url`
3. Click "Connect Google Calendar" to start OAuth flow
4. Select which calendar to use for night routine events
5. The scheduler will now automatically create events

Note: Authentication tokens and calendar selection are stored in the SQLite database and persist between restarts. You only need to authenticate once unless you revoke access or delete the database

## Configuration

### Google Calendar Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google Calendar API
4. Create OAuth 2.0 credentials
5. Note your Client ID and Client Secret
6. Set up environment variables with the credentials

### Environment Variables

Set up the following environment variables for Google OAuth2:

```bash
# Required environment variables
GOOGLE_OAUTH_CLIENT_ID=your-client-id          # OAuth2 credentials
GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret  # OAuth2 credentials
CONFIG_FILE=configs/routine.toml               # Path to TOML configuration file

# Optional environment variables
PORT=8080                                      # Override port from TOML configuration
ENV=development                                # Set to "production" for JSON logging, otherwise pretty console logging

Note: Application URLs (for OAuth callback and webhooks) are now configured in the TOML file
under the [app] section. The OAuth2 callback URL is automatically constructed as "<app_url>/oauth/callback"
```

### Application Configuration

Create a `configs/routine.toml` file:

```toml
[app]
port = 8080                              # Port to listen on (can be overridden by PORT env var)
app_url = "http://localhost:8080"        # Internal application URL for OAuth and general routes
public_url = "http://localhost:8080"     # Public URL for webhooks and external integrations

[parents]
parent_a = "Parent1"                     # First parent name
parent_b = "Parent2"                     # Second parent name

[availability]
parent_a_unavailable = ["Wednesday"]     # Days when parent A can't do the routine
parent_b_unavailable = ["Monday"]        # Days when parent B can't do the routine

[schedule]
update_frequency = "weekly"              # How often to update the calendar
look_ahead_days = 30                     # How many days to schedule in advance

[service]
state_file = "data/state.db"            # SQLite database file for state tracking
log_level = "info"                      # Logging level (trace, debug, info, warn, error, fatal, panic)
# manual_sync_on_startup = true         # Optional: Perform sync on startup (defaults to true). Set to false to disable.
```

Note: In production environments:

- Set `app_url` to your internal/private application URL for OAuth callbacks and general routes
- Set `public_url` to your publicly accessible URL for webhooks from external services

## Logging

The application uses [zerolog](https://github.com/rs/zerolog) for structured logging.

- By default (or when `ENV=development`), logs are output to the console in a human-readable format.
- When `ENV=production`, logs are output as JSON to stdout.
- The log level can be configured using the `log_level` setting in `configs/routine.toml`.

## Override Night Routine (via Google Calendar Event Title)

You can manually override a scheduled night routine assignment directly in Google Calendar **for events scheduled today or in the future**. Overrides for past events will be ignored.

**How it works:**

1.  Find the specific future or current night routine event in your Google Calendar (e.g., `"[ParentA] 🌃👶Routine"`).
2.  Edit the event title and change the parent's name within the square brackets (e.g., change it to `"[ParentB] 🌃👶Routine"`).
3.  Save the event change in Google Calendar.

Google Calendar will send a notification to the application's webhook endpoint (`/api/webhook/calendar`). The application will then:

- Verify the notification.
- Detect the parent name change in the event title.
- Update its internal database record for that specific date to reflect the override (only if the date is today or in the future).
- Recalculate subsequent future assignments if needed to maintain fairness based on this manual change.
- Sync any recalculated assignments back to Google Calendar.

This keeps the application's schedule and fairness tracking accurate even with manual adjustments. For a detailed technical explanation, see the Webhook Handler section in `docs/architecture.md`.

## Development

### Prerequisites

- Go 1.24 or later
- SQLite3
- Google Calendar API credentials
- Docker (optional)

### Storage

The application uses SQLite for persistent storage:

```
data/
└── state.db  # SQLite database containing:
    ├── assignments           # Night routine assignments
    ├── oauth_tokens          # Google OAuth2 tokens
    ├── calendar_settings     # Selected calendar configuration
    └── notification_channels # Google Calendar webhook channels
```

### Running Tests

```bash
go test -v ./...
```

### Making a Release

1. Tag your commit:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
```

2. Push the tag:

```bash
git push origin v1.0.0
```

This will trigger the GitHub Actions release workflow, which will:

- Run tests
- Build binaries for multiple platforms
- Create a Docker image
- Create a GitHub release

## Security Notes

- OAuth2 credentials are handled via environment variables for security
- Tokens are securely stored in the SQLite database
- Use HTTPS in production environments (e.g., via a reverse proxy)
- Keep your environment variables secure
- Regularly update dependencies

## License

This project is open source and available under the [AGPLv3 License](LICENSE).
