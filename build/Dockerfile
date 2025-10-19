# Runtime stage
FROM alpine:latest

ENV ENV="production"

WORKDIR /app

ARG USER=default
ARG TARGETPLATFORM
ENV HOME=/app/

RUN apk add --no-cache ca-certificates

# add new user
RUN adduser -D $USER

# Copy pre-built binary from the platform-specific directory
COPY ${TARGETPLATFORM}/night-routine /app/night-routine

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