# GitHub Copilot Instructions for night-routine

## Project Overview

Night Routine Scheduler is a Go application that manages night routine scheduling between two parents with Google Calendar integration. The application features:
- Fair distribution algorithm for parent assignments
- Web-based settings UI with real-time updates
- SQLite database for configuration and tracking
- Google OAuth2 authentication and Calendar API integration
- Tailwind CSS v4 for UI styling

## Project Structure

```
night-routine/
├── cmd/night-routine/     # Application entry point
├── internal/              # Internal packages
│   ├── calendar/          # Google Calendar API integration
│   ├── config/            # Configuration management
│   ├── database/          # SQLite database layer
│   ├── fairness/          # Assignment scheduling algorithm
│   │   └── scheduler/     # Core scheduling logic
│   ├── handlers/          # HTTP handlers and web UI
│   │   ├── assets/        # CSS and static assets
│   │   └── templates/     # HTML templates
│   ├── constants/         # Application constants
│   ├── logging/           # Zerolog configuration
│   ├── signals/           # Signal handling
│   ├── token/             # OAuth token management
│   └── viewhelpers/       # Template helper functions
├── docs/                  # Architecture and planning docs
├── docs-site/             # MkDocs documentation site
└── configs/               # Configuration examples
```

## Code Quality Standards

### Formatting
- **Always run `go fmt`** on any Go files you modify before committing
- Use `gofmt -s` for simplified formatting where possible
- Ensure consistent formatting across all Go source files

### Linting
- **Always run `golangci-lint`** to check for code quality issues
- Run `golangci-lint run` to check the entire project
- Run `golangci-lint run ./path/to/package` to check specific packages
- Address all linting issues before committing code
- The project uses a `.golangci.yml` configuration file with custom settings

### Testing
- **Always run tests** before committing: `go test ./...`
- Tests are located alongside source files with `_test.go` suffix
- Use table-driven tests for multiple test cases
- Follow existing test patterns in the codebase
- Key test areas:
  - `internal/fairness/scheduler/` - Scheduling algorithm tests
  - `internal/handlers/` - HTTP handler tests
  - `internal/database/` - Database operation tests
  - `internal/config/` - Configuration tests

### Building Assets
- **Always run `go generate` before building** to generate CSS and other assets
- Run `go generate ./...` from the project root to generate all assets
- The CSS files are generated using Tailwind CSS v4 via pnpm
- Assets must be regenerated after any template or CSS changes
- The generate directive is in `internal/handlers/base_handler.go`
- Generated CSS is embedded in the binary via `//go:embed` directives

### Build Artifacts and Git
- **Never commit build artifacts** - they are gitignored
- Gitignored items include:
  - Binary: `night-routine` executable
  - Dependencies: `node_modules/`
  - Build output: `dist/`, `bin/`, `_output/`
  - Database files: `data/*.db*`, `test_*.db`
  - Documentation build: `site/`
  - IDE files: `.idea/`, `.vscode/*` (except specific settings)
  - Environment: `.env` file
- Use `.gitignore` to exclude additional temporary or generated files

### Build Process
1. Install Node.js dependencies: `pnpm install --frozen-lockfile`
2. Generate assets: `go generate ./...`
3. Build the application: `go build -o night-routine ./cmd/night-routine`
4. Run tests: `go test ./...`

### Using Go Language Server (gopls)

When working with Go code, **prefer using gopls (Go language server)** for navigation, analysis, and understanding the codebase:

- **Use gopls for code navigation** instead of grep/find when exploring Go code
- **Start gopls in async mode** for complex analysis tasks
- **Leverage gopls capabilities** for:
  - Finding symbol definitions and references
  - Understanding package APIs and dependencies
  - Analyzing cross-file dependencies
  - Searching for symbols across the workspace
  - Getting accurate diagnostics beyond linting

**Example workflows using gopls tools:**
> Note: These examples use gopls tool integration available in the Copilot environment, not direct CLI commands

- Search for a symbol across the workspace: `gopls-go_search` with query: "ScheduleAssignment"
- Get package API summary: `gopls-go_package_api` with packagePaths: ["github.com/belphemur/night-routine/internal/fairness"]
- Find all references to a symbol: `gopls-go_symbol_references` with file: "/path/to/file.go", symbol: "ProcessEvents"
- Get file context and dependencies: `gopls-go_file_context` with file: "/path/to/file.go"
- Check for build/parse errors: `gopls-go_diagnostics` with files: ["/path/to/file1.go", "/path/to/file2.go"]

**When to use gopls:**
- Refactoring: Find all usages before renaming
- Understanding: Get package structure and dependencies
- Debugging: Check for build errors and type issues
- Navigation: Locate definitions and implementations
- Architecture: Analyze cross-file dependencies

### Development Workflow
1. **Understand the change**: Use gopls to explore related code
2. **Write or modify Go code**: Follow Go best practices and idioms
3. **Format the code**: Run `go fmt ./...`
4. **Check for issues**: Run `golangci-lint run`
5. **Fix any linting issues**: Address all reported problems
6. **Run tests**: Execute `go test ./...` to ensure nothing breaks
7. **Generate assets**: If templates/CSS changed, run `go generate ./...`
8. **Verify the build**: Build the application to ensure it compiles
9. **Commit changes**: Use semantic/conventional commit format

### Commit Messages
- **Always use semantic/conventional commits** format for all commits
- Follow the pattern: `type(scope): subject`
- Common types: `fix`, `feat`, `chore`, `docs`, `refactor`, `test`, `style`
- Common scopes: `handlers`, `calendar`, `fairness`, `database`, `config`, `ui`
- Examples:
  - `fix(handlers): correct cache header for static files`
  - `feat(calendar): add new sync endpoint`
  - `refactor(fairness): improve assignment algorithm`
  - `test(database): add tests for config store`
  - `chore(deps): update dependencies`
- **Never create separate "Initial plan" or "WIP" commits**
- When starting work, create the first commit with proper semantic format immediately
- Use `git commit --amend` to update commits as work progresses
- Final PR should contain meaningful, well-structured commits

### Testing and Screenshots
- **Check `docs/FAKE_DATABASE_SETUP.md`** when you need to create test data or take screenshots
- The application uses SQLite with migrations that must run first
- Never create database tables manually - let migrations run automatically
- Key schema details:
  - `oauth_tokens` table stores token data as JSONB in the `token_data` column
  - `calendar_settings` table uses `calendar_id` column (not `selected_calendar_id`)
  - `assignments` table includes `decision_reason` field for tracking assignment logic
- Always verify data insertion with SQL queries before running the application
- Use the documented process for creating demo databases with proper OAuth tokens
- **CRITICAL: Any UI changes MUST include screenshots**
  - Take screenshots showing before/after for UI modifications
  - Include screenshots in PR descriptions to demonstrate visual impact
  - Use Playwright or similar tools to capture UI states
  - Test on both desktop and mobile viewports when applicable

## Architecture Guidelines

### Database Layer
- All database operations go through `internal/database/`
- Use transactions for multi-step operations
- SQLite3 with CGO-free ncruces/go-sqlite3 driver
- Migrations handled via golang-migrate/migrate
- Configuration stored in database, not files
- **Migrations**:
  - Migration files in `internal/database/migrations/sqlite/`
  - Numbered sequentially: `000001_description.up.sql` and `000001_description.down.sql`
  - Always provide both up and down migrations
  - Never modify existing migrations - create new ones for changes
  - Migrations are embedded via `//go:embed migrations` and run automatically on startup
- **Database Schema**:
  - Use `JSONB` for storing structured data like OAuth tokens
  - Use `TIMESTAMP` with `DEFAULT CURRENT_TIMESTAMP` for temporal fields
  - Add indexes for frequently queried columns
  - Use foreign key constraints where appropriate

### API Integration
- Google Calendar API through `internal/calendar/`
- OAuth2 tokens managed by `internal/token/`
- Webhook support for real-time calendar updates

### Fairness Algorithm
- Core logic in `internal/fairness/scheduler/`
- Considers total assignments, recent patterns, and availability
- Provides transparent decision reasons for each assignment
- Tracks assignment history in database

### HTTP Handlers
- All web handlers in `internal/handlers/`
- Templates use Go's html/template
- Static assets served with proper caching headers
- Settings page provides real-time configuration updates
- **Template Patterns**:
  - Templates are embedded via `//go:embed templates/*.html`
  - Base layout in `templates/layout.html` provides common structure
  - Page-specific templates parsed on-demand and executed with layout
  - Use `BasePageData` struct for common page data (year, path, auth status)
  - Custom template functions defined in `funcMap` (e.g., `add`, `js`)
  - Render templates with `h.RenderTemplate(w, "page.html", data)`
- **Asset Versioning**:
  - CSS and logo assets use ETag versioning for cache busting
  - Version strings generated at build time and embedded in templates
  - Proper `Cache-Control` headers set for static assets

## Security Considerations
- Never commit secrets or credentials
- OAuth tokens stored securely in database
- Use environment variables for sensitive configuration
- Follow Go security best practices

## Documentation
- Architecture docs in `docs/` directory
- User documentation in `docs-site/` (MkDocs)
- Add comments for complex logic and public APIs
- Update relevant docs when changing functionality

## Coding Conventions

### Error Handling
- **Always wrap errors** with context using `fmt.Errorf` with `%w` verb
- Examples from codebase:
  ```go
  return fmt.Errorf("failed to get valid token: %w", err)
  return fmt.Errorf("failed to create calendar service: %w", err)
  ```
- This preserves error chains and enables `errors.Is()` and `errors.As()`
- Use structured logging to log errors with context before returning
- Handle errors at appropriate levels - don't ignore them

### Interfaces
- The codebase uses interfaces for dependency injection and testability
- Key interfaces:
  - `CalendarService` in `internal/calendar/interface.go`
  - `SchedulerInterface` in `internal/fairness/scheduler/interface.go`
  - `TrackerInterface` in `internal/fairness/interface.go`
  - `ConfigStoreInterface` in `internal/config/runtime_config.go`
- When creating new components, define interfaces for external dependencies
- Place interface definitions near their primary usage, not in separate files unless shared widely

### Logging
- Use `zerolog` for all logging throughout the application
- Get a component-specific logger: `logger := logging.GetLogger("component-name")`
- Use appropriate log levels:
  - `Debug()` - Detailed diagnostic information
  - `Info()` - General informational messages
  - `Warn()` - Warning messages for recoverable issues
  - `Error()` - Error messages for failures
- Chain context fields for structured logging:
  ```go
  logger.Info().Str("version", version).Msg("Starting application")
  logger.Error().Err(err).Msg("Failed to connect")
  ```

### Comments
- Add comments for:
  - **All exported functions, types, and constants** (Go convention)
  - Complex algorithms or non-obvious logic
  - Public APIs and interfaces
  - Important architectural decisions
- Follow Go doc comment conventions:
  - Start with the name of the item being documented
  - Use complete sentences
  - Example: `// New creates a new database connection using the provided options.`
- Avoid obvious comments that just restate the code

### Dependencies
- Use `go mod` for dependency management
- Keep dependencies up to date with Renovate (configured in `renovate.json`)
- Run `go mod verify` to verify dependencies
- Run `go mod tidy` to clean up unused dependencies
- Pin specific versions for stability

## Environment and Configuration

### Environment Variables
- `GOOGLE_OAUTH_CLIENT_ID` - Google OAuth client ID (required)
- `GOOGLE_OAUTH_CLIENT_SECRET` - Google OAuth client secret (required)
- `CONFIG_FILE` - Path to TOML configuration file (default: `configs/routine.toml`)
- `PORT` - HTTP server port (default: `8080`)
- `APP_URL` - Internal application URL
- `PUBLIC_URL` - Public-facing URL for OAuth callbacks
- `ENV` - Environment mode: `development` or `production`

### Configuration Files
- TOML configuration files in `configs/` directory
- Configuration is seeded from TOML file on first run, then stored in database
- Runtime configuration managed through web UI
- Never commit `.env` files (they're gitignored)

## Development Environment

### Local Development
- Use `ENV=development` for verbose debug logging with console output
- Use `.env` file for local environment variables (see `.env.example`)
- Database file typically stored in `data/` directory (gitignored)
- Hot reload not supported - rebuild after changes

### Docker Development
- Multi-stage Dockerfile in `build/Dockerfile`
- Pre-built images available: `ghcr.io/belphemur/night-routine:latest`
- Docker Compose configuration in `docker-compose.yml`
- Images support both `amd64` and `arm64` architectures
- Dev container configuration in `.devcontainer/devcontainer.json`

### CI/CD Pipeline
- GitHub Actions workflows in `.github/workflows/`
- **CI workflow** (`ci.yml`):
  - Lint: runs `go fmt` and `golangci-lint`
  - Test: runs tests with race detection and coverage
  - Build: generates assets and builds binary
- **Release workflow** (`release.yml`):
  - Triggered by semantic-release
  - Multi-platform Docker builds
  - Binary releases with GoReleaser
- **Docs workflow** (`docs.yml`):
  - Builds and deploys MkDocs documentation

## Additional Guidelines
- Follow Go best practices and idioms
- Write clear, maintainable code
- Ensure all tests pass before committing
- Use structured logging with zerolog for all output
- Never use `fmt.Print` or `log.Print` - use zerolog instead
- Keep the codebase DRY (Don't Repeat Yourself)
- Prefer composition over inheritance
- Write self-documenting code with clear names
