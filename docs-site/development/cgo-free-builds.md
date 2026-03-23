# CGO-Free Builds

Night Routine is built as a fully **CGO-free** Go application. This is an intentional architectural decision that simplifies builds, deployments, and cross-compilation.

## Why Avoid CGO?

CGO enables Go programs to call C libraries, but introduces significant complexity:

| Concern | With CGO | Without CGO |
|---------|----------|-------------|
| **Cross-compilation** | Requires C cross-compiler toolchains | Native `GOOS`/`GOARCH` flags |
| **Static binaries** | Complex linking, may need musl/glibc | Fully static by default |
| **Build reproducibility** | Depends on system C libraries | Deterministic Go toolchain |
| **Docker images** | Needs build dependencies (gcc, libc-dev) | Minimal `scratch` or `distroless` images |
| **CI/CD pipelines** | Extra setup for each target platform | Simple matrix builds |
| **Security surface** | C code may have memory safety issues | Go's memory safety guarantees |

!!! tip "Rule of Thumb"
    Always prefer pure Go libraries over CGO-dependent alternatives when a well-maintained option exists. CGO should only be used as a last resort when no pure Go alternative provides the required functionality.

## SQLite: `modernc.org/sqlite`

The most common reason Go projects enable CGO is for SQLite support. The popular `mattn/go-sqlite3` library wraps the C SQLite library via CGO, requiring a C compiler for every build.

Night Routine uses **[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)** instead — a pure Go translation of the SQLite C source code using the [CCGO](https://pkg.go.dev/modernc.org/ccgo/v4) compiler.

### Key Features

- **100% Go** — No C compiler needed, no CGO dependency
- **Full SQLite compatibility** — Passes the SQLite test suite
- **Cross-platform** — Works on all Go-supported platforms (linux, darwin, windows; amd64, arm64)
- **Database-level compatible** — Reads and writes standard SQLite database files
- **WAL mode support** — Full Write-Ahead Logging support for concurrency

### Driver Registration

The driver registers itself as `"sqlite"` with Go's `database/sql` package:

```go
import (
    "database/sql"
    _ "modernc.org/sqlite" // Register the pure Go SQLite driver
)

func main() {
    db, err := sql.Open("sqlite", "file:data/state.db?cache=private&mode=rwc")
    // ...
}
```

### Connection String Format

The driver uses standard SQLite URI format:

```
file:<path>?<params>
```

**Supported URI parameters:**

| Parameter | Values | Description |
|-----------|--------|-------------|
| `mode` | `ro`, `rw`, `rwc`, `memory` | Access mode |
| `cache` | `shared`, `private` | Cache mode |
| `immutable` | `1` | Read-only, no locking |

Additional settings (journal mode, foreign keys, busy timeout, etc.) are applied via `PRAGMA` commands after the connection is established.

### Migration from Other Drivers

If migrating from another SQLite driver (e.g., `mattn/go-sqlite3` or `ncruces/go-sqlite3`):

1. **Replace the import** — Change the driver import to `_ "modernc.org/sqlite"`
2. **Update the driver name** — Change `sql.Open("sqlite3", ...)` to `sql.Open("sqlite", ...)`
3. **Database files are compatible** — No data migration needed, SQLite file format is universal
4. **PRAGMAs work the same** — All standard SQLite PRAGMA commands are supported

## Build Configuration

CGO is explicitly disabled in all build contexts:

=== "Local Build"

    ```bash
    CGO_ENABLED=0 go build -o night-routine ./cmd/night-routine
    ```

=== "GoReleaser"

    ```yaml
    builds:
      - env:
          - CGO_ENABLED=0
    ```

=== "GitHub Actions CI"

    ```yaml
    - name: Build Go binary
      run: CGO_ENABLED=0 go build -o night-routine ./cmd/night-routine
    ```

=== "Dockerfile"

    ```dockerfile
    ENV CGO_ENABLED=0
    RUN go build -o /app/night-routine ./cmd/night-routine
    ```

## Verifying CGO-Free Builds

To confirm a binary has no CGO dependencies:

```bash
# Build the binary
CGO_ENABLED=0 go build -o night-routine ./cmd/night-routine

# Check for dynamic linking (should show "not a dynamic executable")
ldd night-routine

# Or check the Go build info
go version -m night-routine | grep CGO
```

## Next Steps

- [Database Structure](../architecture/database.md) — Schema and migration details
- [Local Development](local.md) — Development environment setup
- [Contributing](contributing.md) — How to contribute to the project
