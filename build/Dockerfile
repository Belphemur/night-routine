# Build stage
FROM --platform=$BUILDPLATFORM golang:1-alpine AS builder

ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Add ARGs for build flags
ARG VERSION="unknown"
ARG COMMIT="unknown"
ARG DATE="unknown"

WORKDIR /workspace

# Install build dependencies
RUN apk add --no-cache build-base

# For cross-compilation with SQLite
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    apk add --no-cache gcc-aarch64-linux-musl musl-dev; \
    fi

# Copy Go module manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with appropriate flags for the target architecture
RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -v -o night-routine \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    ./cmd/night-routine

# Runtime stage
FROM --platform=$TARGETPLATFORM gcr.io/distroless/static-debian12:nonroot

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