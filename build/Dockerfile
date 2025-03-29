# Build stage
FROM golang:1 AS builder

# Add ARGs for build flags
ARG VERSION="unknown"
ARG COMMIT="unknown"
ARG DATE="unknown"

WORKDIR /workspace

# Install build dependencies
RUN apt-get update && apt-get install -y build-essential && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy Go module manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with appropriate flags for the target architecture
RUN CGO_ENABLED=1 \
    go build -v -o night-routine \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    ./cmd/night-routine

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

ENV ENV="production"

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /workspace/night-routine .

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