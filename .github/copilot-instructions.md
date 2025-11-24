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

## Architecture Guidelines

### Database Layer
- All database operations go through `internal/database/`
- Use transactions for multi-step operations
- SQLite3 with CGO-free ncruces/go-sqlite3 driver
- Migrations handled via golang-migrate/migrate
- Configuration stored in database, not files

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

## Additional Guidelines
- Follow Go best practices and idioms
- Write clear, maintainable code
- Add comments for complex logic
- Ensure all tests pass before committing
- Keep dependencies up to date
- Use structured logging with zerolog
- Handle errors properly with context
