# Environment Variables

Night Routine Scheduler uses environment variables for sensitive data and deployment-specific configuration.

## Required Variables

These environment variables **must** be set for the application to run:

### `GOOGLE_OAUTH_CLIENT_ID`

**Type:** String  
**Required:** Yes

Your Google OAuth 2.0 Client ID obtained from the Google Cloud Console.

```bash
export GOOGLE_OAUTH_CLIENT_ID="123456789-abcdefghijklmnop.apps.googleusercontent.com"
```

!!! info "Getting Credentials"
    See the [Google Calendar Setup Guide](google-calendar.md) for instructions on obtaining these credentials.

### `GOOGLE_OAUTH_CLIENT_SECRET`

**Type:** String  
**Required:** Yes

Your Google OAuth 2.0 Client Secret obtained from the Google Cloud Console.

```bash
export GOOGLE_OAUTH_CLIENT_SECRET="your-secret-here"
```

!!! danger "Security Warning"
    Never commit this value to version control. Keep it secure and rotate it if compromised.

### `CONFIG_FILE`

**Type:** String (file path)  
**Required:** Yes

Path to your TOML configuration file.

```bash
export CONFIG_FILE="configs/routine.toml"
```

In Docker:
```bash
export CONFIG_FILE="/app/config/routine.toml"
```

## Optional Variables

### `PORT`

**Type:** Integer  
**Required:** No  
**Default:** Value from TOML configuration

Override the port specified in your TOML configuration.

```bash
export PORT=8080
```

!!! tip "Use Case"
    Useful in containerized environments where you want to specify the port at runtime without modifying the configuration file.

### `ENV`

**Type:** String  
**Required:** No  
**Default:** `development`  
**Values:** `development` | `production`

Controls logging format:

- **`development`** - Pretty, human-readable console logs with colors
- **`production`** - Structured JSON logs for log aggregation systems

```bash
# Development (pretty logs)
export ENV=development

# Production (JSON logs)
export ENV=production
```

**Example development output:**
```
2024-01-15T10:30:45Z INF Starting Night Routine Scheduler
2024-01-15T10:30:45Z INF Connecting to database file=data/state.db
```

**Example production output:**
```json
{"level":"info","time":"2024-01-15T10:30:45Z","message":"Starting Night Routine Scheduler"}
{"level":"info","time":"2024-01-15T10:30:45Z","file":"data/state.db","message":"Connecting to database"}
```

## Setting Environment Variables

### Linux/macOS

=== "Command Line"

    ```bash
    export GOOGLE_OAUTH_CLIENT_ID="your-client-id"
    export GOOGLE_OAUTH_CLIENT_SECRET="your-client-secret"
    export CONFIG_FILE="configs/routine.toml"
    export ENV=production
    ```

=== ".env File"

    Create a `.env` file:
    ```bash
    GOOGLE_OAUTH_CLIENT_ID=your-client-id
    GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
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
    Environment="GOOGLE_OAUTH_CLIENT_ID=your-client-id"
    Environment="GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret"
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
    set GOOGLE_OAUTH_CLIENT_ID=your-client-id
    set GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
    set CONFIG_FILE=configs\routine.toml
    set ENV=production
    ```

=== "PowerShell"

    ```powershell
    $env:GOOGLE_OAUTH_CLIENT_ID="your-client-id"
    $env:GOOGLE_OAUTH_CLIENT_SECRET="your-client-secret"
    $env:CONFIG_FILE="configs\routine.toml"
    $env:ENV="production"
    ```

=== "System Environment Variables"

    1. Open System Properties → Advanced → Environment Variables
    2. Add new user or system variables
    3. Restart your terminal/application

### Docker

=== "docker run"

    ```bash
    docker run \
      -e GOOGLE_OAUTH_CLIENT_ID=your-client-id \
      -e GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret \
      -e CONFIG_FILE=/app/config/routine.toml \
      -e ENV=production \
      ghcr.io/belphemur/night-routine:latest
    ```

=== "docker-compose.yml"

    ```yaml
    version: '3.8'
    services:
      night-routine:
        image: ghcr.io/belphemur/night-routine:latest
        environment:
          - GOOGLE_OAUTH_CLIENT_ID=your-client-id
          - GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
          - CONFIG_FILE=/app/config/routine.toml
          - ENV=production
    ```

=== ".env file (Docker Compose)"

    Create `.env`:
    ```
    GOOGLE_OAUTH_CLIENT_ID=your-client-id
    GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret
    CONFIG_FILE=/app/config/routine.toml
    ENV=production
    ```

    Reference in `docker-compose.yml`:
    ```yaml
    version: '3.8'
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
- **HashiCorp Vault** - For centralized secret management
- **AWS Secrets Manager** - For AWS deployments
- **Azure Key Vault** - For Azure deployments

### Rotate Credentials Regularly

- Rotate OAuth secrets every 90 days
- Use different credentials for different environments
- Monitor for unauthorized access

### Restrict Permissions

- Limit file permissions on `.env` files:
    ```bash
    chmod 600 .env
    ```
- Run the application with a non-root user
- Use principle of least privilege for OAuth scopes

## Troubleshooting

### Variable Not Set Errors

If you see errors about missing environment variables:

1. Verify the variable is exported:
    ```bash
    echo $GOOGLE_OAUTH_CLIENT_ID
    ```

2. Check for typos in variable names

3. Ensure variables are exported before running the application

### OAuth Authentication Fails

1. Verify credentials are correct in Google Cloud Console
2. Check for extra whitespace in environment variables
3. Ensure redirect URIs match your configuration

### Configuration File Not Found

1. Verify `CONFIG_FILE` path is absolute or relative to working directory
2. Check file exists and is readable
3. Ensure proper file permissions

## Next Steps

- [Configure TOML settings](toml.md)
- [Set up Google Calendar](google-calendar.md)
- [Complete first-time setup](../user-guide/setup.md)
