# Environment Variables

Night Routine Scheduler supports two styles of environment variable configuration:

1. **`NR_*` variables** (recommended) — full coverage of every setting, using a consistent naming convention
2. **Legacy variables** — `PORT`, `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET` — kept for backwards compatibility

When both styles are set for the same value, `NR_*` always takes precedence.

## Naming Convention for NR_* Variables

```
NR_<SECTION>__<FIELD>
```

- **Prefix:** `NR_`
- **Section/field separator:** `__` (double underscore — avoids ambiguity with underscores inside field names)
- **Both section and field are UPPERCASE** in the env var name

Example: the TOML path `app.port` maps to `NR_APP__PORT`.

## Complete Reference

### `[oauth]` — Google OAuth2 Credentials (no TOML equivalent)

| Env Var | Legacy Equivalent | Required | Description |
|---------|-------------------|----------|-------------|
| `NR_OAUTH__CLIENT_ID` | `GOOGLE_OAUTH_CLIENT_ID` | **Yes** | Google OAuth2 Client ID |
| `NR_OAUTH__CLIENT_SECRET` | `GOOGLE_OAUTH_CLIENT_SECRET` | **Yes** | Google OAuth2 Client Secret |

```bash
export NR_OAUTH__CLIENT_ID="123456789-abc.apps.googleusercontent.com"
export NR_OAUTH__CLIENT_SECRET="your-secret-here"
```

!!! danger "Security Warning"
    Never commit OAuth credentials to version control. Keep them in `.env` files or a secret manager.

!!! info "Getting Credentials"
    See the [Google Calendar Setup Guide](google-calendar.md) for instructions on obtaining these credentials.

### `[app]` — Application Server

| Env Var | TOML Key | Default | Description |
|---------|----------|---------|-------------|
| `NR_APP__PORT` | `app.port` | `8888` | HTTP server port |
| `NR_APP__APP_URL` | `app.app_url` | *(required)* | Internal application URL used for OAuth callbacks |
| `NR_APP__PUBLIC_URL` | `app.public_url` | *(required)* | Public-facing URL for webhooks and external integrations |

```bash
export NR_APP__PORT=8080
export NR_APP__APP_URL="https://night-routine.internal"
export NR_APP__PUBLIC_URL="https://night-routine.example.com"
```

### `[parents]` — Parent Names

| Env Var | TOML Key | Default | Description |
|---------|----------|---------|-------------|
| `NR_PARENTS__PARENT_A` | `parents.parent_a` | *(required)* | First parent name |
| `NR_PARENTS__PARENT_B` | `parents.parent_b` | *(required)* | Second parent name |

```bash
export NR_PARENTS__PARENT_A="Alice"
export NR_PARENTS__PARENT_B="Bob"
```

### `[availability]` — Unavailability Constraints

| Env Var | TOML Key | Default | Description |
|---------|----------|---------|-------------|
| `NR_AVAILABILITY__PARENT_A_UNAVAILABLE` | `availability.parent_a_unavailable` | `""` (always available) | Comma-separated days when parent A is unavailable |
| `NR_AVAILABILITY__PARENT_B_UNAVAILABLE` | `availability.parent_b_unavailable` | `""` (always available) | Comma-separated days when parent B is unavailable |

```bash
# Comma-separated day names; whitespace around commas is trimmed
export NR_AVAILABILITY__PARENT_A_UNAVAILABLE="Monday, Wednesday"
export NR_AVAILABILITY__PARENT_B_UNAVAILABLE="Friday"

# Empty string means always available
export NR_AVAILABILITY__PARENT_A_UNAVAILABLE=""
```

!!! info "Valid Day Names"
    `Monday`, `Tuesday`, `Wednesday`, `Thursday`, `Friday`, `Saturday`, `Sunday`

### `[schedule]` — Scheduling Parameters

| Env Var | TOML Key | Default | Description |
|---------|----------|---------|-------------|
| `NR_SCHEDULE__UPDATE_FREQUENCY` | `schedule.update_frequency` | *(required)* | `daily`, `weekly`, `monthly`, or `disabled` |
| `NR_SCHEDULE__LOOK_AHEAD_DAYS` | `schedule.look_ahead_days` | *(required)* | Days to schedule in advance |
| `NR_SCHEDULE__PAST_EVENT_THRESHOLD_DAYS` | `schedule.past_event_threshold_days` | `5` | Days in the past to accept manual event changes |
| `NR_SCHEDULE__STATS_ORDER` | `schedule.stats_order` | `desc` | Statistics page sort order: `desc` or `asc` |
| `NR_SCHEDULE__CALENDAR_ID` | `schedule.calendar_id` | *(optional)* | Google Calendar ID |

```bash
export NR_SCHEDULE__UPDATE_FREQUENCY="weekly"
export NR_SCHEDULE__LOOK_AHEAD_DAYS=30
export NR_SCHEDULE__PAST_EVENT_THRESHOLD_DAYS=7
```

### `[service]` — Service Settings

| Env Var | TOML Key | Default | Description |
|---------|----------|---------|-------------|
| `NR_SERVICE__STATE_FILE` | `service.state_file` | *(required)* | Path to SQLite database file |
| `NR_SERVICE__LOG_LEVEL` | `service.log_level` | `info` | Log level: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic` |
| `NR_SERVICE__MANUAL_SYNC_ON_STARTUP` | `service.manual_sync_on_startup` | `true` | Sync schedule on startup if a token exists |

```bash
export NR_SERVICE__STATE_FILE="/var/lib/night-routine/state.db"
export NR_SERVICE__LOG_LEVEL="warn"
export NR_SERVICE__MANUAL_SYNC_ON_STARTUP="false"
```

## Meta Variables (no NR_* equivalent)

| Env Var | Required | Description |
|---------|----------|-------------|
| `CONFIG_FILE` | Yes | Path to TOML configuration file |
| `ENV` | No | `production` (JSON logs) or anything else (pretty console logs). Default: dev mode |

```bash
export CONFIG_FILE="/app/config/routine.toml"
export ENV=production
```

## Legacy Variables

The following variables are kept for backwards compatibility. They are equivalent
to their `NR_*` counterparts but have **lower precedence** — if both are set, `NR_*` wins.

| Legacy Var | NR_* Equivalent |
|------------|-----------------|
| `PORT` | `NR_APP__PORT` |
| `GOOGLE_OAUTH_CLIENT_ID` | `NR_OAUTH__CLIENT_ID` |
| `GOOGLE_OAUTH_CLIENT_SECRET` | `NR_OAUTH__CLIENT_SECRET` |

## Configuration Precedence (highest to lowest)

```
NR_* env vars
  ↑ higher priority
Legacy env vars (PORT, GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET)
  ↑
TOML file values
  ↑
Built-in defaults
```

## Setting Environment Variables

### Linux/macOS

=== "Shell export"

    ```bash
    export NR_OAUTH__CLIENT_ID="your-client-id"
    export NR_OAUTH__CLIENT_SECRET="your-client-secret"
    export CONFIG_FILE="configs/routine.toml"
    export ENV=production
    ```

=== ".env File"

    Create a `.env` file:
    ```bash
    NR_OAUTH__CLIENT_ID=your-client-id
    NR_OAUTH__CLIENT_SECRET=your-client-secret
    CONFIG_FILE=configs/routine.toml
    ENV=production
    ```

    Then source it:
    ```bash
    source .env
    ```

=== "systemd Service"

    Create `/etc/systemd/system/night-routine.service`:
    ```ini
    [Unit]
    Description=Night Routine Scheduler
    After=network.target

    [Service]
    Type=simple
    User=night-routine
    WorkingDirectory=/opt/night-routine
    Environment="NR_OAUTH__CLIENT_ID=your-client-id"
    Environment="NR_OAUTH__CLIENT_SECRET=your-client-secret"
    Environment="CONFIG_FILE=/opt/night-routine/configs/routine.toml"
    Environment="ENV=production"
    ExecStart=/opt/night-routine/night-routine
    Restart=on-failure

    [Install]
    WantedBy=multi-user.target
    ```

### Windows

=== "Command Prompt"

    ```cmd
    set NR_OAUTH__CLIENT_ID=your-client-id
    set NR_OAUTH__CLIENT_SECRET=your-client-secret
    set CONFIG_FILE=configs\routine.toml
    set ENV=production
    ```

=== "PowerShell"

    ```powershell
    $env:NR_OAUTH__CLIENT_ID="your-client-id"
    $env:NR_OAUTH__CLIENT_SECRET="your-client-secret"
    $env:CONFIG_FILE="configs\routine.toml"
    $env:ENV="production"
    ```

### Docker

=== "docker run"

    ```bash
    docker run \
      -e NR_OAUTH__CLIENT_ID=your-client-id \
      -e NR_OAUTH__CLIENT_SECRET=your-client-secret \
      -e CONFIG_FILE=/app/config/routine.toml \
      -e ENV=production \
      ghcr.io/belphemur/night-routine:latest
    ```

=== "docker-compose.yml"

    ```yaml
    services:
      night-routine:
        image: ghcr.io/belphemur/night-routine:latest
        environment:
          - NR_OAUTH__CLIENT_ID=your-client-id
          - NR_OAUTH__CLIENT_SECRET=your-client-secret
          - CONFIG_FILE=/app/config/routine.toml
          - ENV=production
    ```

=== ".env file (Docker Compose)"

    Create `.env`:
    ```
    NR_OAUTH__CLIENT_ID=your-client-id
    NR_OAUTH__CLIENT_SECRET=your-client-secret
    CONFIG_FILE=/app/config/routine.toml
    ENV=production
    ```

    Reference in `docker-compose.yml`:
    ```yaml
    services:
      night-routine:
        image: ghcr.io/belphemur/night-routine:latest
        env_file:
          - .env
    ```

## Security Best Practices

### Never Commit Secrets

Add to `.gitignore`:
```
.env
*.env
secrets/
```

### Use Secret Management

For production deployments, consider using:

- **Docker Secrets** - For Docker Swarm
- **Kubernetes Secrets** - For Kubernetes deployments
- **HashiCorp Vault** - For centralised secret management
- **AWS Secrets Manager** - For AWS deployments
- **Azure Key Vault** - For Azure deployments

### Rotate Credentials Regularly

- Rotate OAuth secrets every 90 days
- Use different credentials for different environments
- Monitor for unauthorised access

### Restrict Permissions

- Limit file permissions on `.env` files:
    ```bash
    chmod 600 .env
    ```
- Run the application with a non-root user
- Use the principle of least privilege for OAuth scopes

## Troubleshooting

### Variable Not Set Errors

If you see errors about missing environment variables:

1. Verify the variable is exported:
    ```bash
    echo $NR_OAUTH__CLIENT_ID
    ```

2. Check for typos in variable names — the double underscore `__` separator is easy to miss

3. Ensure variables are exported before running the application

### OAuth Authentication Fails

1. Verify credentials are correct in Google Cloud Console
2. Check for extra whitespace in environment variables
3. Ensure redirect URIs match your `app_url` configuration

### Configuration File Not Found

1. Verify `CONFIG_FILE` path is absolute or relative to working directory
2. Check file exists and is readable
3. Ensure proper file permissions

## Next Steps

- [Configure TOML settings](toml.md)
- [Set up Google Calendar](google-calendar.md)
- [Complete first-time setup](../user-guide/setup.md)
