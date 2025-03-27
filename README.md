# Night Routine Scheduler

A Go application that manages night routine scheduling between two parents, with Google Calendar integration for automated event creation.

## Features

- Fair schedule distribution between parents
- Google Calendar integration with OAuth2
- Configurable parent availability
- Automated scheduling with daily/weekly/monthly updates
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
calendar_id = "primary"      # Default Google Calendar ID (can be changed via UI)
look_ahead_days = 30        # How many days to schedule in advance

[service]
port = 8080                 # Port for OAuth web interface and metrics
state_file = "data/state.db"  # SQLite database file for state tracking

[google]
credentials_file = "configs/credentials.json"  # Google OAuth2 credentials
```

## Google Calendar Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google Calendar API
4. Create OAuth 2.0 credentials
5. Download the credentials and save as `configs/credentials.json`

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
./night-routine -config configs/routine.toml
```

### Docker Run

```bash
docker run -v /path/to/configs:/etc/night-routine \
  -v /path/to/data:/var/lib/night-routine \
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

## License

This project is open source and available under the [MIT License](LICENSE).
