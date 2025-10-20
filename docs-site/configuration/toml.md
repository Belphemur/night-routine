# TOML Configuration

The main application configuration is stored in a TOML file. This file contains settings for the application, parents, scheduling, and service options.

## Configuration File Location

The configuration file path is specified via the `CONFIG_FILE` environment variable:

```bash
export CONFIG_FILE="configs/routine.toml"
```

## Complete Configuration Example

```toml
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
manual_sync_on_startup = true
```

## Configuration Sections

### `[app]` - Application Settings

#### `port`

**Type:** Integer  
**Required:** Yes  
**Default:** None

The port on which the web server listens.

```toml
[app]
port = 8080
```

!!! note "Port Override"
    Can be overridden by the `PORT` environment variable.

#### `app_url`

**Type:** String (URL)  
**Required:** Yes  
**Default:** None

The internal application URL used for OAuth callbacks and general routes.

```toml
[app]
app_url = "http://localhost:8080"
```

**Examples:**
- Development: `http://localhost:8080`
- Production (internal): `http://192.168.1.100:8080`
- Production (behind proxy): `https://night-routine.example.com`

!!! important "OAuth Configuration"
    The OAuth callback URL is automatically constructed as `<app_url>/oauth/callback`. This must match the authorized redirect URI in your Google Cloud Console.

#### `public_url`

**Type:** String (URL)  
**Required:** Yes  
**Default:** None

The publicly accessible URL for webhooks and external integrations.

```toml
[app]
public_url = "http://localhost:8080"
```

**Examples:**
- Development: `http://localhost:8080`
- Production: `https://night-routine.example.com`
- With ngrok: `https://abc123.ngrok.io`

!!! tip "Internal vs Public URLs"
    In production behind a reverse proxy:
    
    - `app_url`: Your internal network address
    - `public_url`: Your public domain name
    
    Example:
    ```toml
    app_url = "http://192.168.1.100:8080"
    public_url = "https://night-routine.example.com"
    ```

### `[parents]` - Parent Configuration

#### `parent_a` and `parent_b`

**Type:** String  
**Required:** Yes  
**Default:** None

Names of the two parents who will be assigned night routine duties.

```toml
[parents]
parent_a = "Alice"
parent_b = "Bob"
```

!!! warning "Validation"
    - Both names must be provided
    - Names must be different from each other
    - These names appear in calendar events as `[ParentName] 🌃👶Routine`

### `[availability]` - Availability Constraints

#### `parent_a_unavailable` and `parent_b_unavailable`

**Type:** Array of strings  
**Required:** No  
**Default:** `[]` (empty - always available)

Days of the week when each parent is unavailable for night routine duties.

```toml
[availability]
parent_a_unavailable = ["Wednesday", "Friday"]
parent_b_unavailable = ["Monday", "Thursday"]
```

**Valid day names:**
- `Monday`, `Tuesday`, `Wednesday`, `Thursday`, `Friday`, `Saturday`, `Sunday`

!!! info "Case Sensitive"
    Day names must be capitalized as shown above.

**Examples:**

=== "Weekend Parent Only"
    ```toml
    [availability]
    parent_a_unavailable = ["Saturday", "Sunday"]
    parent_b_unavailable = []
    ```

=== "Weekday Split"
    ```toml
    [availability]
    parent_a_unavailable = ["Monday", "Wednesday", "Friday"]
    parent_b_unavailable = ["Tuesday", "Thursday"]
    ```

=== "One Day Off Each"
    ```toml
    [availability]
    parent_a_unavailable = ["Wednesday"]
    parent_b_unavailable = ["Monday"]
    ```

### `[schedule]` - Scheduling Settings

#### `update_frequency`

**Type:** String  
**Required:** Yes  
**Default:** None  
**Valid values:** `daily`, `weekly`, `monthly`

How often the schedule should be automatically updated.

```toml
[schedule]
update_frequency = "weekly"
```

- **`daily`** - Updates every day
- **`weekly`** - Updates once per week
- **`monthly`** - Updates once per month

!!! tip "Recommendation"
    `weekly` is recommended for most users as it provides a good balance between keeping the schedule current and minimizing API calls.

#### `look_ahead_days`

**Type:** Integer  
**Required:** Yes  
**Default:** None  
**Range:** 1-365

Number of days in advance to schedule night routine assignments.

```toml
[schedule]
look_ahead_days = 30
```

!!! example "Use Cases"
    - **7 days** - Weekly planners
    - **14 days** - Bi-weekly planners
    - **30 days** - Monthly planners (recommended)
    - **90 days** - Quarterly planners

#### `past_event_threshold_days`

**Type:** Integer  
**Required:** No  
**Default:** 5  
**Range:** 0-30

Number of days in the past to accept manual event changes via Google Calendar.

```toml
[schedule]
past_event_threshold_days = 5
```

This setting controls the window for manual overrides:

- **`0`** - No past changes accepted (only future events)
- **`5`** - Accept changes up to 5 days ago (default)
- **`14`** - Accept changes up to 2 weeks ago
- **`30`** - Accept changes up to 1 month ago

!!! info "Purpose"
    Prevents old assignments from being accidentally modified, which could affect fairness calculations.

### `[service]` - Service Settings

#### `state_file`

**Type:** String (file path)  
**Required:** Yes  
**Default:** None

Path to the SQLite database file for persistent storage.

```toml
[service]
state_file = "data/state.db"
```

**Path handling:**
- Relative paths are resolved from the working directory
- Absolute paths are used as-is
- Parent directories must exist

!!! tip "Docker Deployment"
    Use a path inside a mounted volume to persist data:
    ```toml
    state_file = "data/state.db"
    ```
    With volume mount: `-v ./data:/app/data`

#### `log_level`

**Type:** String  
**Required:** No  
**Default:** `info`  
**Valid values:** `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

Logging verbosity level.

```toml
[service]
log_level = "info"
```

**Level descriptions:**

| Level | Description | Use Case |
|-------|-------------|----------|
| `trace` | Very detailed logging | Deep debugging |
| `debug` | Detailed debug info | Development, troubleshooting |
| `info` | General information | Production (recommended) |
| `warn` | Warning messages | Production |
| `error` | Error messages only | Production (minimal logging) |
| `fatal` | Fatal errors only | Not recommended |
| `panic` | Panic level only | Not recommended |

!!! warning "Performance"
    Lower log levels (trace, debug) can impact performance and generate large log files.

#### `manual_sync_on_startup`

**Type:** Boolean  
**Required:** No  
**Default:** `true`

Whether to perform a schedule synchronization when the application starts.

```toml
[service]
manual_sync_on_startup = true
```

- **`true`** - Sync schedule on startup (default)
- **`false`** - Don't sync on startup

!!! info "When to Disable"
    You might want to set this to `false` if:
    
    - You restart the application frequently
    - You want to control syncs manually
    - You're testing and don't want API calls on every restart

## Validation

The application validates the configuration on startup. Common validation errors:

### Missing Required Fields

```
Error: missing required configuration: parents.parent_a
```

**Solution:** Add the missing field to your configuration.

### Invalid Values

```
Error: invalid update_frequency: 'biweekly' (must be daily, weekly, or monthly)
```

**Solution:** Use one of the valid values.

### Same Parent Names

```
Error: parent_a and parent_b must have different names
```

**Solution:** Ensure parent names are unique.

### Invalid URL Format

```
Error: app_url must be a valid URL (e.g., http://localhost:8080)
```

**Solution:** Provide a complete URL with protocol.

## Complete Examples

### Basic Configuration

```toml
[app]
port = 8080
app_url = "http://localhost:8080"
public_url = "http://localhost:8080"

[parents]
parent_a = "Alice"
parent_b = "Bob"

[availability]
parent_a_unavailable = []
parent_b_unavailable = []

[schedule]
update_frequency = "weekly"
look_ahead_days = 30

[service]
state_file = "data/state.db"
log_level = "info"
```

### Production Configuration

```toml
[app]
port = 8080
app_url = "http://192.168.1.100:8080"
public_url = "https://night-routine.example.com"

[parents]
parent_a = "Parent One"
parent_b = "Parent Two"

[availability]
parent_a_unavailable = ["Wednesday"]
parent_b_unavailable = ["Monday"]

[schedule]
update_frequency = "weekly"
look_ahead_days = 30
past_event_threshold_days = 7

[service]
state_file = "/var/lib/night-routine/state.db"
log_level = "warn"
manual_sync_on_startup = false
```

### Development Configuration

```toml
[app]
port = 3000
app_url = "http://localhost:3000"
public_url = "http://localhost:3000"

[parents]
parent_a = "TestParentA"
parent_b = "TestParentB"

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
```

## Next Steps

- [Set up environment variables](environment.md)
- [Configure Google Calendar](google-calendar.md)
- [Complete first-time setup](../user-guide/setup.md)
