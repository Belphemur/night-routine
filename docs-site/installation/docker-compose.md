# Docker Compose Installation

For easier self-hosting, you can use Docker Compose to manage the Night Routine Scheduler.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) installed
- [Docker Compose](https://docs.docker.com/compose/install/) installed

## Quick Start

1. **Download the docker-compose.yml file:**

    ```bash
    wget https://raw.githubusercontent.com/Belphemur/night-routine/main/docker-compose.yml
    ```

    Or manually create a `docker-compose.yml` file with the following content:

    ```yaml
    version: '3.8'

    services:
      night-routine:
        image: ghcr.io/belphemur/night-routine:latest
        container_name: night-routine
        restart: unless-stopped
        ports:
          - "8080:8080"
        environment:
          - GOOGLE_OAUTH_CLIENT_ID=your-client-id-here
          - GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret-here
          - CONFIG_FILE=/app/config/routine.toml
          - ENV=production
        volumes:
          - ./config:/app/config
          - ./data:/app/data
    ```

2. **Create the configuration directory:**

    ```bash
    mkdir -p config data
    ```

3. **Create a configuration file:**

    Download the example configuration or create `config/routine.toml`:

    ```bash
    cat > config/routine.toml << 'EOF'
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
    EOF
    ```

4. **Edit the docker-compose.yml to set your Google OAuth credentials:**

    ```bash
    nano docker-compose.yml
    ```

    Update these lines with your actual credentials:
    ```yaml
    - GOOGLE_OAUTH_CLIENT_ID=your-actual-client-id
    - GOOGLE_OAUTH_CLIENT_SECRET=your-actual-client-secret
    ```

5. **Edit the configuration file to match your needs:**

    ```bash
    nano config/routine.toml
    ```

6. **Start the service:**

    ```bash
    docker-compose up -d
    ```

## Managing the Service

### View logs

```bash
# View all logs
docker-compose logs

# Follow logs in real-time
docker-compose logs -f

# View logs for the last hour
docker-compose logs --since 1h
```

### Stop the service

```bash
docker-compose stop
```

### Start the service

```bash
docker-compose start
```

### Restart the service

```bash
docker-compose restart
```

### Stop and remove containers

```bash
docker-compose down
```

!!! warning "Data Preservation"
    Using `docker-compose down` will stop and remove containers, but your data in the `./config` and `./data` directories will be preserved.

### Update to latest version

```bash
# Pull the latest image
docker-compose pull

# Restart with the new image
docker-compose up -d
```

## Production Deployment

For production deployments, consider the following enhancements:

### Using Environment Files

Create a `.env` file for your environment variables:

```bash
cat > .env << 'EOF'
GOOGLE_OAUTH_CLIENT_ID=your-client-id-here
GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret-here
CONFIG_FILE=/app/config/routine.toml
ENV=production
PORT=8080
EOF
```

Update your `docker-compose.yml` to use the environment file:

```yaml
version: '3.8'

services:
  night-routine:
    image: ghcr.io/belphemur/night-routine:latest
    container_name: night-routine
    restart: unless-stopped
    ports:
      - "${PORT}:8080"
    env_file:
      - .env
    volumes:
      - ./config:/app/config
      - ./data:/app/data
```

!!! danger "Security"
    Make sure to add `.env` to your `.gitignore` file to prevent committing sensitive credentials.

### Behind a Reverse Proxy

When deploying behind a reverse proxy (e.g., nginx, Traefik, Caddy), update your configuration:

=== "Nginx"

    ```nginx
    server {
        listen 80;
        server_name night-routine.example.com;

        location / {
            proxy_pass http://localhost:8080;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
    ```

=== "Traefik"

    ```yaml
    version: '3.8'

    services:
      night-routine:
        image: ghcr.io/belphemur/night-routine:latest
        container_name: night-routine
        restart: unless-stopped
        env_file:
          - .env
        volumes:
          - ./config:/app/config
          - ./data:/app/data
        labels:
          - "traefik.enable=true"
          - "traefik.http.routers.night-routine.rule=Host(`night-routine.example.com`)"
          - "traefik.http.routers.night-routine.entrypoints=websecure"
          - "traefik.http.routers.night-routine.tls.certresolver=letsencrypt"
          - "traefik.http.services.night-routine.loadbalancer.server.port=8080"
        networks:
          - traefik

    networks:
      traefik:
        external: true
    ```

=== "Caddy"

    ```
    night-routine.example.com {
        reverse_proxy localhost:8080
    }
    ```

Update your `config/routine.toml` to reflect the public URL:

```toml
[app]
port = 8080
app_url = "http://localhost:8080"  # Internal URL
public_url = "https://night-routine.example.com"  # Public URL for webhooks
```

## Directory Structure

After setup, your directory structure should look like:

```
.
├── docker-compose.yml
├── .env (optional)
├── config/
│   └── routine.toml
└── data/
    └── state.db (created automatically)
```

## Troubleshooting

### Container won't start

Check the logs:
```bash
docker-compose logs
```

### Permission issues

Ensure the `data` directory is writable:
```bash
chmod 755 data
```

### Cannot access the web interface

1. Check if the container is running:
    ```bash
    docker-compose ps
    ```

2. Verify port mapping:
    ```bash
    docker-compose port night-routine 8080
    ```

3. Check firewall rules if accessing from another machine

## Next Steps

- [Configure the application](../configuration/toml.md)
- [Set up Google Calendar integration](../configuration/google-calendar.md)
- [Complete first-time setup](../user-guide/setup.md)
