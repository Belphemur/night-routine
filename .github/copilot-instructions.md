# GitHub Copilot Instructions for night-routine

## Code Quality Standards

When working with Go code in this repository, always follow these practices:

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

### Building Assets
- **Always run `go generate` before building** to generate CSS and other assets
- Run `go generate ./...` from the project root to generate all assets
- The CSS files are generated using Tailwind CSS v4
- Assets must be regenerated after any template or CSS changes

### Workflow
1. Write or modify Go code
2. Run `go fmt` to format the code
3. Run `golangci-lint run` to check for issues
4. Fix any issues reported by golangci-lint
5. If templates or CSS were modified, run `go generate ./...` to rebuild assets
6. Commit the changes

### Commit Messages
- **Always use semantic/conventional commits** format for all commits
- Follow the pattern: `type(scope): subject`
- Common types: `fix`, `feat`, `chore`, `docs`, `refactor`, `test`, `style`
- Examples:
  - `fix(handlers): correct cache header for static files`
  - `feat(calendar): add new sync endpoint`
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

### Additional Guidelines
- Follow Go best practices and idioms
- Write clear, maintainable code
- Add comments for complex logic
- Ensure all tests pass before committing
- Keep dependencies up to date
