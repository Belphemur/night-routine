# internal/logging

Centralized structured logging using [zerolog](https://github.com/rs/zerolog).

## Purpose

Provides a unified logging interface for all application components with environment-aware formatting (pretty console for development, JSON for production).

## Key API

- `Initialize(isDevelopment bool)` — Sets up the global logger format.
- `GetLogger(component string) zerolog.Logger` — Returns a component-scoped logger (e.g., `logging.GetLogger("scheduler")`).
- `SetLogLevel(level string)` — Dynamically changes verbosity at runtime.

## Conventions

- **Never** use `fmt.Print` or `log.Print`. Always use `zerolog` via `GetLogger`.
- Chain context fields for structured output: `logger.Info().Str("key", "val").Msg("message")`.
- Log levels: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`.

## Dependencies

- Uses: `github.com/rs/zerolog`
- Used by: every other internal package
