# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Copy Go module manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with cached dependencies
RUN CGO_ENABLED=1 GOOS=linux go build -v -o night-routine ./cmd/night-routine

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /workspace/night-routine .

# Copy default configuration
COPY configs/routine.toml /etc/night-routine/routine.toml

# Create directory for state persistence
RUN mkdir -p /var/lib/night-routine && \
    chown -R nonroot:nonroot /var/lib/night-routine

USER nonroot:nonroot

# Expose port for health checks and metrics
EXPOSE 8080

# Volume for persistent state
VOLUME ["/var/lib/night-routine"]

# Set environment variables
ENV CONFIG_FILE=/etc/night-routine/routine.toml
ENV STATE_DIR=/var/lib/night-routine

ENTRYPOINT ["/app/night-routine"]