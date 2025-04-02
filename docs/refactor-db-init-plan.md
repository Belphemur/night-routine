# Refactoring Database Initialization Plan

**Objective:** Refactor the database initialization in `internal/database` to correctly apply all `SQLiteOptions` using a combination of connection string parameters (for supported options) and explicit `PRAGMA` commands (for others). The primary constructor `New` will accept `SQLiteOptions` directly, and PRAGMA application errors will cause the connection to fail.

**Plan Details:**

1.  **Refactor `internal/database/database.go`**:

    - Modify the `New` function signature to accept `opts SQLiteOptions` instead of `connectionString string`.
    - Inside `New(opts SQLiteOptions)`:
      - Call `opts.buildConnectionString()` to generate the base connection string (containing only URI-supported parameters).
      - Call `sql.Open("sqlite3", connStr)` to establish the initial connection.
      - If `sql.Open` succeeds, call a new private helper function `applyPragmas(conn *sql.DB, opts SQLiteOptions, logger zerolog.Logger) error`.
      - **Error Handling:** If `applyPragmas` returns an error, log it, close the `conn`, and return `nil, err`.
      - If `conn.Ping()` fails after opening and applying PRAGMAs, log the error, close the connection, and return `nil, err`.
      - If all steps succeed, return the `*DB` struct and `nil` error.
    - **Remove** the now redundant `NewWithOptions` function.

2.  **Implement `applyPragmas` Helper Function (in `database.go`)**:

    - This function will receive the `*sql.DB` connection, `SQLiteOptions`, and `zerolog.Logger`.
    - It will execute `PRAGMA` statements for each relevant option not handled by the connection string (e.g., `Journal`, `BusyTimeout`, `ForeignKeys`, `Synchronous`, `CacheSize`).
    - Convert boolean options to 0/1 and handle enum types correctly.
    - Log each PRAGMA being applied.
    - If any `conn.Exec()` call for a PRAGMA returns an error, log the specific failure and immediately return that error.
    - If all PRAGMAs are applied successfully, return `nil`.

3.  **Refactor `internal/database/connection_string.go`**:

    - Modify the `buildConnectionString` method:
      - **Remove** the logic for setting parameters not supported directly in the connection string URI (e.g., `_journal_mode`, `_busy_timeout`, `_foreign_keys`, `_synchronous`, `_cache_size`).
      - **Keep** the logic for parameters that _are_ supported via the URI (e.g., `mode`, `cache`, `immutable`).

4.  **Update Tests**:
    - Modify tests in `internal/database/database_test.go` to use the new `New(opts)` signature and expect correct PRAGMA settings.
    - Modify tests in `internal/database/sqlite_options_test.go` if they relied on the old `buildConnectionString` behavior for removed parameters.
    - Modify tests in `internal/database/connection_string_test.go` to verify that `buildConnectionString` now only includes the URI-supported parameters.

**Mermaid Diagram:**

```mermaid
sequenceDiagram
    participant Client
    participant New(opts)
    participant buildConnectionString
    participant sql.Open
    participant applyPragmas
    participant conn.Exec()

    Client->>New(opts): Call with SQLiteOptions
    New(opts)->>buildConnectionString: Get connection string (URI params only)
    buildConnectionString-->>New(opts): Return connStr
    New(opts)->>sql.Open: Open DB connection with connStr
    sql.Open-->>New(opts): Return *sql.DB conn, error
    alt sql.Open Failed
        New(opts)-->>Client: Return nil, error
    else sql.Open Succeeded
        New(opts)->>applyPragmas: Call with conn, opts, logger
        loop For each relevant PRAGMA in opts
            applyPragmas->>conn.Exec(): Execute PRAGMA command (e.g., PRAGMA journal_mode=?)
            conn.Exec()-->>applyPragmas: Return error (if any)
            opt PRAGMA Failed
                 applyPragmas-->>New(opts): Return error
                 New(opts)->>conn.Close(): Close connection
                 New(opts)-->>Client: Return nil, error
                 Note right of New(opts): Stop processing
            end
        end
        applyPragmas-->>New(opts): Return nil (all PRAGMAs OK)
        New(opts)->>conn.Ping(): Verify connection
        alt Ping Failed
            New(opts)->>conn.Close(): Close connection
            New(opts)-->>Client: Return nil, error
        else Ping Succeeded
            New(opts)-->>Client: Return *DB, nil
        end
    end
```
