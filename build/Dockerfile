# Build stage
FROM golang:1-alpine AS builder

# Add ARGs for build flags
ARG VERSION="unknown"
ARG COMMIT="unknown"
ARG DATE="unknown"

WORKDIR /workspace

# Install build dependencies
RUN apk add --no-cache build-base

# Build with appropriate flags for the target architecture
RUN --mount=type=bind,target=/workspace CGO_ENABLED=1 \
    go build -v -o /build/night-routine \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    ./cmd/night-routine

# Runtime stage
FROM alpine:latest

ENV ENV="production"

WORKDIR /app

ARG USER=default
ENV HOME=/app/

RUN apk add --no-cache ca-certificates

# add new user
RUN adduser -D $USER


# Copy binary from builder stage
COPY --from=builder /build/night-routine .

# Copy default configuration
COPY configs/routine.toml /app/config/routine.toml

RUN chown -R $USER:$USER -R /app/

USER $USER:$USER

# Expose port for health checks and metrics
EXPOSE 8080

# Volume for persistent state
VOLUME ["/app/data"]

# Set environment variables
ENV CONFIG_FILE=/app/config/routine.toml

CMD ["/app/night-routine"]