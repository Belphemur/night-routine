# Local Development Installation

For development or running from source, you can build and run Night Routine Scheduler locally.

## Prerequisites

- **Go 1.25 or later** - [Download Go](https://golang.org/dl/)
- **Git** - [Download Git](https://git-scm.com/downloads)
- **Google Calendar API Credentials** - [Get credentials](../configuration/google-calendar.md)

## Installation Steps

### 1. Clone the Repository

```bash
git clone https://github.com/Belphemur/night-routine.git
cd night-routine
```

### 2. Create Configuration

Copy the example configuration:

```bash
# Create a configuration directory if it doesn't exist
mkdir -p configs

# Copy or create the configuration file
cat > configs/my-routine.toml << 'EOF'
[app]
port = 8080
app_url = "http://localhost:8080"
public_url = "http://localhost:8080"

[parents]
parent_a = "Parent1"
parent_b = "Parent2"

[availability]
parent_a_unavailable = ["Wednesday"]
parent_b_unavailable = ["Monday"]

[schedule]
update_frequency = "weekly"
look_ahead_days = 30
past_event_threshold_days = 5

[service]
state_file = "data/state.db"
log_level = "info"
EOF
```

### 3. Set Environment Variables

Create a `.env` file or export the variables directly:

```bash
export GOOGLE_OAUTH_CLIENT_ID=your-client-id
export GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
export CONFIG_FILE=configs/my-routine.toml
export ENV=development
```

Or create a `.env` file:

```bash
cat > .env << 'EOF'
GOOGLE_OAUTH_CLIENT_ID=your-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
CONFIG_FILE=configs/my-routine.toml
ENV=development
EOF
```

Then source it:
```bash
source .env
```

### 4. Download Dependencies

```bash
go mod download
```

### 5. Build the Application

```bash
go build -o night-routine ./cmd/night-routine/
```

### 6. Run the Application

```bash
./night-routine
```

The application will start and be available at `http://localhost:8080` (or your configured port).

## Development Workflow

### Using Air for Live Reload

For a better development experience, use [Air](https://github.com/air-verse/air) for automatic reloading:

1. **Install Air:**

    ```bash
    go install github.com/air-verse/air@latest
    ```

2. **Create `.air.toml` configuration:**

    ```toml
    root = "."
    testdata_dir = "testdata"
    tmp_dir = "tmp"

    [build]
      args_bin = []
      bin = "./tmp/main"
      cmd = "go build -o ./tmp/main ./cmd/night-routine"
      delay = 1000
      exclude_dir = ["assets", "tmp", "vendor", "testdata", "data", "docs", "docs-site"]
      exclude_file = []
      exclude_regex = ["_test.go"]
      exclude_unchanged = false
      follow_symlink = false
      full_bin = ""
      include_dir = []
      include_ext = ["go", "tpl", "tmpl", "html"]
      include_file = []
      kill_delay = "0s"
      log = "build-errors.log"
      poll = false
      poll_interval = 0
      rerun = false
      rerun_delay = 500
      send_interrupt = false
      stop_on_error = false

    [color]
      app = ""
      build = "yellow"
      main = "magenta"
      runner = "green"
      watcher = "cyan"

    [log]
      main_only = false
      time = false

    [misc]
      clean_on_exit = false

    [screen]
      clear_on_rebuild = false
      keep_scroll = true
    ```

3. **Run with Air:**

    ```bash
    air
    ```

Now your application will automatically rebuild and restart when you make changes to the code.

### Running Tests

Run all tests:

```bash
go test -v ./...
```

Run tests with coverage:

```bash
go test -v -cover ./...
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Run specific package tests:

```bash
go test -v ./internal/fairness/...
```

### Linting

The project uses `golangci-lint` for code linting:

1. **Install golangci-lint:**

    ```bash
    # macOS
    brew install golangci-lint

    # Linux
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

    # Windows
    # Download from https://github.com/golangci/golangci-lint/releases
    ```

2. **Run the linter:**

    ```bash
    golangci-lint run
    ```

### Code Formatting

Format your code before committing:

```bash
# Format all Go files
gofmt -s -w .

# Verify formatting
gofmt -s -l .
```

## Directory Structure

After setup, your project structure will look like:

```
night-routine/
├── cmd/
│   └── night-routine/          # Main application entry point
├── internal/                    # Internal packages
│   ├── calendar/               # Google Calendar integration
│   ├── config/                 # Configuration management
│   ├── database/               # Database operations
│   ├── fairness/               # Fairness algorithm
│   ├── scheduler/              # Scheduling logic
│   └── web/                    # Web interface
├── configs/                     # Configuration files
│   └── my-routine.toml
├── data/                        # Runtime data (created automatically)
│   └── state.db
├── docs/                        # Additional documentation
├── .env                         # Environment variables (gitignored)
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
└── README.md
```

## Debugging

### Using Delve Debugger

Install and use [Delve](https://github.com/go-delve/delve) for debugging:

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug ./cmd/night-routine

# Or attach to running process
dlv attach <pid>
```

### Enabling Debug Logging

Update your configuration to enable debug logging:

```toml
[service]
log_level = "debug"  # or "trace" for even more detail
```

### Common Issues

#### Port Already in Use

If port 8080 is already in use:

1. Change the port in your `routine.toml`:
    ```toml
    [app]
    port = 8081
    ```

2. Or set the `PORT` environment variable:
    ```bash
    export PORT=8081
    ```

#### OAuth Callback Issues

Make sure your Google OAuth2 redirect URI matches your local URL:

- In development: `http://localhost:8080/oauth/callback`
- Update in [Google Cloud Console](https://console.cloud.google.com/)

#### Database Locked Errors

If you encounter database locked errors:

1. Ensure only one instance is running
2. Check file permissions on the `data` directory
3. The database uses WAL mode which should prevent most locking issues

## Building for Production

Build a production-ready binary:

```bash
# Build with optimizations and version information
VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

CGO_ENABLED=0 go build \
  -ldflags="-s -w -X 'main.version=${VERSION}' -X 'main.commit=${COMMIT}' -X 'main.date=${DATE}'" \
  -o night-routine \
  ./cmd/night-routine
```

The resulting binary will be statically linked and can be deployed anywhere.

## Next Steps

- [Configure the application](../configuration/toml.md)
- [Set up Google Calendar integration](../configuration/google-calendar.md)
- [Learn about the architecture](../architecture/overview.md)
- [Read the development guide](../development/local.md)
