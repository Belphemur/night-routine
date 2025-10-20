# Docker Installation

Pre-built multi-architecture Docker images (supporting both amd64 and arm64) are available in the GitHub Container Registry.

## Quick Start

```bash
# Pull the latest release
docker pull ghcr.io/belphemur/night-routine:latest

# Run the container
docker run \
  -e GOOGLE_OAUTH_CLIENT_ID=your-client-id \
  -e GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret \
  -e PORT=8080 \
  -e CONFIG_FILE=/app/config/routine.toml \
  -v /path/to/config:/app/config \
  -v /path/to/data:/app/data \
  -p 8080:8080 \
  ghcr.io/belphemur/night-routine:latest
```

!!! warning "Security Notice"
    These images are signed using Sigstore Cosign and include SBOM attestations for enhanced security.

## Available Tags

The following image tags are available:

- **`latest`** - Most recent stable release
- **`vX.Y.Z`** - Specific version tags (e.g., `v1.0.0`, `v1.1.0`)

## Environment Variables

When running with Docker, you must provide the following environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `GOOGLE_OAUTH_CLIENT_ID` | Yes | OAuth2 Client ID from Google Cloud Console |
| `GOOGLE_OAUTH_CLIENT_SECRET` | Yes | OAuth2 Client Secret from Google Cloud Console |
| `CONFIG_FILE` | Yes | Path to TOML configuration file (e.g., `/app/config/routine.toml`) |
| `PORT` | No | Override port from TOML configuration (default: 8080) |
| `ENV` | No | Set to `production` for JSON logging, otherwise pretty console logging |

## Volume Mounts

Mount the following volumes for persistent data:

| Container Path | Purpose | Required |
|----------------|---------|----------|
| `/app/config` | Configuration files (TOML) | Yes |
| `/app/data` | SQLite database storage | Yes |

## Example with Custom Configuration

Create a configuration directory and run the container:

```bash
# Create directories
mkdir -p ~/night-routine/config ~/night-routine/data

# Create a configuration file
cat > ~/night-routine/config/routine.toml << 'EOF'
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

# Run the container
docker run -d \
  --name night-routine \
  -e GOOGLE_OAUTH_CLIENT_ID=your-client-id \
  -e GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret \
  -e CONFIG_FILE=/app/config/routine.toml \
  -v ~/night-routine/config:/app/config \
  -v ~/night-routine/data:/app/data \
  -p 8080:8080 \
  --restart unless-stopped \
  ghcr.io/belphemur/night-routine:latest
```

## Verifying Image Signatures

Images are signed with Cosign. To verify the signature:

```bash
# Install cosign
# See https://docs.sigstore.dev/cosign/installation/

# Verify the image signature
cosign verify ghcr.io/belphemur/night-routine:latest \
  --certificate-identity-regexp="https://github.com/Belphemur/night-routine" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## Multi-Architecture Support

The Docker images support multiple architectures:

- **linux/amd64** - x86_64 systems (most desktops, servers)
- **linux/arm64** - ARM64 systems (Raspberry Pi 4/5, Apple Silicon, AWS Graviton)

Docker will automatically pull the correct image for your platform.

## Container Logs

View container logs to monitor the application:

```bash
# View logs
docker logs night-routine

# Follow logs in real-time
docker logs -f night-routine

# View last 100 lines
docker logs --tail 100 night-routine
```

## Updating the Container

To update to a new version:

```bash
# Stop and remove the old container
docker stop night-routine
docker rm night-routine

# Pull the latest image
docker pull ghcr.io/belphemur/night-routine:latest

# Start a new container with the same configuration
docker run -d \
  --name night-routine \
  -e GOOGLE_OAUTH_CLIENT_ID=your-client-id \
  -e GOOGLE_OAUTH_CLIENT_SECRET=your-client-secret \
  -e CONFIG_FILE=/app/config/routine.toml \
  -v ~/night-routine/config:/app/config \
  -v ~/night-routine/data:/app/data \
  -p 8080:8080 \
  --restart unless-stopped \
  ghcr.io/belphemur/night-routine:latest
```

!!! tip "Data Persistence"
    Your data and configuration are safe in the mounted volumes and will be preserved across container updates.

## Next Steps

- [Configure the application](../configuration/toml.md)
- [Set up Google Calendar integration](../configuration/google-calendar.md)
- [Complete first-time setup](../user-guide/setup.md)
