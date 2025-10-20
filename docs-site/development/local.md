# Local Development

This guide covers setting up a local development environment for contributing to Night Routine Scheduler.

## Prerequisites

- **Go 1.25 or later** - [Download](https://golang.org/dl/)
- **Git** - [Download](https://git-scm.com/downloads)
- **SQLite** - Usually pre-installed on macOS/Linux
- **Google OAuth Credentials** - [Setup Guide](../configuration/google-calendar.md)
- **Code Editor** - VS Code, GoLand, or your preferred editor

## Initial Setup

### 1. Fork and Clone

```bash
# Fork the repository on GitHub
# Then clone your fork
git clone https://github.com/YOUR_USERNAME/night-routine.git
cd night-routine

# Add upstream remote
git remote add upstream https://github.com/Belphemur/night-routine.git
```

### 2. Install Dependencies

```bash
# Download Go dependencies
go mod download

# Verify dependencies
go mod verify
```

### 3. Configure Environment

Create a `.env` file for development:

```bash
cat > .env << 'EOF'
GOOGLE_OAUTH_CLIENT_ID=your-dev-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-dev-client-secret
CONFIG_FILE=configs/dev-routine.toml
ENV=development
EOF
```

Create a development configuration:

```bash
mkdir -p configs test-data

cat > configs/dev-routine.toml << 'EOF'
[app]
port = 8080
app_url = "http://localhost:8080"
public_url = "http://localhost:8080"

[parents]
parent_a = "DevParentA"
parent_b = "DevParentB"

[availability]
parent_a_unavailable = []
parent_b_unavailable = []

[schedule]
update_frequency = "daily"
look_ahead_days = 7
past_event_threshold_days = 1

[service]
state_file = "test-data/state.db"
log_level = "debug"
manual_sync_on_startup = true
EOF
```

### 4. Build and Run

```bash
# Build the application
go build -o night-routine ./cmd/night-routine/

# Run the application
source .env  # Load environment variables
./night-routine
```

## Development Workflow

### Using Air for Live Reload

[Air](https://github.com/air-verse/air) automatically rebuilds and restarts your application when code changes.

**Install Air:**
```bash
go install github.com/air-verse/air@latest
```

**Create `.air.toml`:**
```bash
cat > .air.toml << 'EOF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/night-routine"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "test-data", "docs", "docs-site"]
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
EOF
```

**Run with Air:**
```bash
source .env
air
```

Now your application will automatically rebuild when you save changes.

### IDE Setup

#### VS Code

**Recommended extensions:**
- Go (golang.go)
- Go Test Explorer
- SQLite Viewer
- Better TOML

**settings.json:**
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.formatTool": "gofmt",
  "go.testFlags": ["-v"],
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

**launch.json (debugging):**
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Night Routine",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/night-routine",
      "env": {
        "GOOGLE_OAUTH_CLIENT_ID": "${env:GOOGLE_OAUTH_CLIENT_ID}",
        "GOOGLE_OAUTH_CLIENT_SECRET": "${env:GOOGLE_OAUTH_CLIENT_SECRET}",
        "CONFIG_FILE": "configs/dev-routine.toml",
        "ENV": "development"
      }
    }
  ]
}
```

#### GoLand / IntelliJ IDEA

1. Open the project directory
2. GoLand should automatically detect it as a Go project
3. Set environment variables in Run Configuration:
   - Edit Configurations → Go Build
   - Add environment variables
4. Enable golangci-lint integration in Preferences → Tools → File Watchers

## Code Quality

### Linting

The project uses `golangci-lint` for comprehensive code linting.

**Install:**
```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Windows (using Scoop)
scoop install golangci-lint
```

**Run linter:**
```bash
golangci-lint run
```

**Auto-fix issues:**
```bash
golangci-lint run --fix
```

**Configuration:** See `.golangci.yml` in the repository root.

### Formatting

**Format code:**
```bash
# Format all files
gofmt -s -w .

# Check formatting without modifying
gofmt -s -l .
```

**Verify formatting (CI style):**
```bash
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
  echo "Code is not formatted!"
  gofmt -s -l .
  exit 1
fi
```

## Testing

### Running Tests

**Run all tests:**
```bash
go test -v ./...
```

**Run tests with coverage:**
```bash
go test -v -cover ./...
```

**Generate coverage report:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

**Run specific package:**
```bash
go test -v ./internal/fairness/...
```

**Run specific test:**
```bash
go test -v -run TestFairnessAlgorithm ./internal/fairness/
```

### Writing Tests

Tests should be placed in `*_test.go` files alongside the code they test.

**Example test structure:**
```go
package fairness

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestAssignmentLogic(t *testing.T) {
    t.Run("alternating pattern", func(t *testing.T) {
        // Test setup
        // Test execution
        // Assertions
        assert.Equal(t, expected, actual)
    })
    
    t.Run("unavailability", func(t *testing.T) {
        // ...
    })
}
```

### Test Coverage Goals

- **Overall:** 70%+ coverage
- **Core logic:** 80%+ coverage (fairness, scheduler)
- **Handlers:** 60%+ coverage (web handlers)
- **Utilities:** 50%+ coverage

## Debugging

### Using Delve Debugger

**Install Delve:**
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**Debug with Delve:**
```bash
# Debug main package
dlv debug ./cmd/night-routine

# Set breakpoint
(dlv) break main.main
(dlv) break internal/fairness/fairness.go:42

# Continue execution
(dlv) continue

# Step through code
(dlv) next
(dlv) step

# Inspect variables
(dlv) print variableName
(dlv) locals

# Exit
(dlv) quit
```

**Attach to running process:**
```bash
# Get PID
ps aux | grep night-routine

# Attach debugger
dlv attach <pid>
```

### Debug Logging

Set log level to debug or trace for detailed output:

```toml
[service]
log_level = "debug"  # or "trace"
```

**Log output shows:**
- HTTP requests and responses
- Database queries
- Fairness calculations
- Google Calendar API calls
- Webhook processing

## Database Development

### Inspecting the Database

```bash
# Open database in SQLite CLI
sqlite3 test-data/state.db

# View schema
.schema

# Query data
SELECT * FROM assignments ORDER BY date DESC LIMIT 10;
SELECT * FROM oauth_tokens;
SELECT * FROM calendar_settings;

# Exit
.quit
```

### Database Migrations

**Creating a new migration:**

1. Create up and down migration files:
   ```bash
   # In internal/database/migrations/
   touch 000003_my_migration.up.sql
   touch 000003_my_migration.down.sql
   ```

2. Write the migration:
   ```sql
   -- 000003_my_migration.up.sql
   ALTER TABLE assignments ADD COLUMN new_field TEXT;
   CREATE INDEX idx_assignments_new_field ON assignments(new_field);
   ```

   ```sql
   -- 000003_my_migration.down.sql
   DROP INDEX IF EXISTS idx_assignments_new_field;
   ALTER TABLE assignments DROP COLUMN new_field;
   ```

3. Test the migration:
   ```bash
   # Migrations run automatically on app startup
   ./night-routine
   ```

**Note:** Migrations are embedded in the binary via `//go:embed` directives.

## Common Development Tasks

### Testing Google Calendar Integration

For local development without hitting Google APIs, you can:

1. **Use test credentials** in `.env`
2. **Mock the Calendar API** in tests
3. **Use a dedicated test calendar** in your Google account

### Testing Webhooks Locally

Google Calendar webhooks require a publicly accessible URL. Options:

**Option 1: ngrok**
```bash
# Install ngrok
brew install ngrok  # macOS

# Start ngrok tunnel
ngrok http 8080

# Update public_url in config
public_url = "https://abc123.ngrok.io"
```

**Option 2: localhost.run**
```bash
# No installation needed
ssh -R 80:localhost:8080 localhost.run

# Use the provided URL in public_url
```

**Option 3: Skip webhooks**
- Set `manual_sync_on_startup = false`
- Use "Sync Now" button instead of webhooks

### Resetting Development Database

```bash
# Stop the application
# Delete the database
rm test-data/state.db test-data/state.db-*

# Restart - fresh database will be created
./night-routine
```

## Contribution Guidelines

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Pass `golangci-lint` checks
- Write meaningful commit messages
- Add comments for exported functions

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new assignment reason type
fix: correct consecutive limit calculation
docs: update configuration guide
test: add tests for fairness algorithm
refactor: simplify webhook handling
```

### Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Add tests
5. Run linters and tests
6. Commit with conventional commit messages
7. Push to your fork
8. Open a pull request

### PR Checklist

- [ ] Tests pass locally
- [ ] Linters pass (`golangci-lint run`)
- [ ] Code is formatted (`gofmt`)
- [ ] Added tests for new features
- [ ] Updated documentation if needed
- [ ] Commit messages follow conventions

## Troubleshooting

### Build Errors

**Module issues:**
```bash
go mod tidy
go mod verify
```

**Stale cache:**
```bash
go clean -cache -modcache -testcache
go mod download
```

### Test Failures

**Database locks:**
- Ensure only one test instance runs at a time
- Use separate test databases: `state_test.db`
- Close database connections in tests

**Time-dependent tests:**
- Mock time in tests
- Use relative dates instead of absolute
- Account for timezone differences

## Next Steps

- [Run tests](testing.md)
- [Read contributing guidelines](contributing.md)
- [Understand the architecture](../architecture/overview.md)
