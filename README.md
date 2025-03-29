# Night Routine Scheduler

A Go application that manages night routine scheduling between two parents, with Google Calendar integration for automated event creation.

## Features

- Fair schedule distribution between parents
- Google Calendar integration with OAuth2
- Configurable parent availability
- Automated scheduling with daily/weekly/monthly updates
- Webhook endpoint for manual assignment overrides (e.g., via Google Calendar updates)
- Persistent storage using SQLite:
  - Assignment history and fairness tracking
  - OAuth2 tokens and refresh tokens
  - Selected Google Calendar ID
- Docker containerization
- Multi-architecture support (amd64, arm64)

## Prerequisites

- Go 1.24 or later
- SQLite3
- Google Calendar API credentials
- Docker (optional)

## Storage

The application uses SQLite for persistent storage:

```
data/
└── state.db  # SQLite database containing:
    ├── assignments     # Night routine assignments
    ├── oauth_tokens    # Google OAuth2 tokens
    └── calendar_settings # Selected calendar configuration
```

## Configuration

### Environment Variables

Set up the following environment variables for Google OAuth2:

```bash
# Required environment variables
GOOGLE_OAUTH_CLIENT_ID=your-client-id          # OAuth2 credentials
GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret  # OAuth2 credentials
PORT=8080                                      # Port for OAuth web interface and metrics
CONFIG_FILE=configs/routine.toml               # Path to TOML configuration file
APP_URL=http://localhost:8080                  # Application URL (defaults to http://localhost:<PORT>)

The OAuth2 callback URL is automatically constructed from APP_URL as "<APP_URL>/oauth/callback"
```

### Application Configuration

Create a `configs/routine.toml` file:

```toml
[parents]
parent_a = "Parent1"  # First parent name
parent_b = "Parent2"  # Second parent name

[availability]
parent_a_unavailable = ["Wednesday"]  # Days when parent A can't do the routine
parent_b_unavailable = ["Monday"]     # Days when parent B can't do the routine

[schedule]
update_frequency = "weekly"  # How often to update the calendar
look_ahead_days = 30        # How many days to schedule in advance

[service]
state_file = "data/state.db"  # SQLite database file for state tracking
```

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

## Google Calendar Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google Calendar API
4. Create OAuth 2.0 credentials
5. Note your Client ID and Client Secret
6. Set up environment variables with the credentials

## Building

### Local Build

```bash
go build -v ./cmd/night-routine
```

### Docker Build

```bash
docker build -t night-routine:latest .
```

## Running

### Local Run

```bash
# Set environment variables
export GOOGLE_OAUTH_CLIENT_ID=your-client-id
export GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
export GOOGLE_OAUTH_REDIRECT_URL=http://localhost:8080/oauth/callback
export PORT=8080
export CONFIG_FILE=configs/routine.toml

# Run the application
./night-routine
```

### Docker Run

```bash
docker run \
  -e GOOGLE_OAUTH_CLIENT_ID=your-client-id \
  -e GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret \
  -e PORT=8080 \
  -e CONFIG_FILE=/etc/night-routine/routine.toml \
  -e APP_URL=http://localhost:8080 \
  -v /path/to/configs:/etc/night-routine \
  -v /path/to/data:/var/lib/night-routine \
  -p 8080:8080 \
  night-routine:latest
```

## First-Time Setup

1. Start the application
2. Visit http://localhost:8080 (or your configured port)
3. Click "Connect Google Calendar" to start OAuth flow
4. Select which calendar to use for night routine events
5. The scheduler will now automatically create events

Note: Authentication tokens and calendar selection are stored in the SQLite database and persist between restarts. You only need to authenticate once unless you revoke access or delete the database.

## Development

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
- Use HTTPS in production environments
- Keep your environment variables secure
- Regularly update dependencies

## License

This project is open source and available under the [AGPLv3 License](LICENSE).
