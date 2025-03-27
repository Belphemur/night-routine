# Build stage
FROM golang:1-alpine AS builder

WORKDIR /workspace

# Copy Go module manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .
RUN apk add build-base --no-cache
# Build with cached dependencies
RUN CGO_ENABLED=1 GOOS=linux go build -v -o night-routine ./cmd/night-routine

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /workspace/night-routine .
# Create directory for state persistence
RUN mkdir -p /config

# Copy default configuration
COPY configs/routine.toml /config/routine.toml

USER nonroot:nonroot

# Expose port for health checks and metrics
EXPOSE 8080

# Volume for persistent state
VOLUME ["/config"]

# Set environment variables
ENV CONFIG_FILE=/config/routine.toml

ENTRYPOINT ["/app/night-routine"]