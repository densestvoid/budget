# Build stage - only built when referenced (not when using --target production with USE_PREBUILT_BINARY=true)
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o budget .

# Production stage
FROM alpine:latest AS production

# Build argument to control binary source
# When true: only copy from build context (builder stage won't be built due to --target)
# When false: copy from builder stage (normal local build)
ARG USE_PREBUILT_BINARY=false

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary based on USE_PREBUILT_BINARY arg
# When USE_PREBUILT_BINARY=true: copy from build context only (using bind mount, builder not referenced)
# When USE_PREBUILT_BINARY=false: copy from builder stage (builder will be built)
RUN --mount=type=bind,source=.,target=/build-context \
    if [ "$USE_PREBUILT_BINARY" = "true" ]; then \
      if [ -f /build-context/budget ]; then \
        cp /build-context/budget /app/budget && \
        echo "Using pre-built binary from build context"; \
      else \
        echo "ERROR: USE_PREBUILT_BINARY=true but budget not found in build context" && exit 1; \
      fi; \
    fi

# Copy from builder stage (only used when USE_PREBUILT_BINARY=false)
# Note: When USE_PREBUILT_BINARY=true and using --target production, this COPY will still
# cause builder to be built. To avoid this, we need to not reference builder at all.
# However, since we need one Dockerfile for both cases, we'll use a workaround:
# The RUN command above handles the USE_PREBUILT_BINARY=true case, and this COPY handles false.
# For local builds (USE_PREBUILT_BINARY=false), builder will be built normally.
# For CI with --target production (USE_PREBUILT_BINARY=true), builder will be built but
# the binary from RUN above will be used (last write wins).
# Actually, we need to ensure this COPY only runs when USE_PREBUILT_BINARY=false.
# Since COPY can't be conditional, we'll copy to a temp location and conditionally use it.
RUN --mount=from=builder,source=/app/budget,target=/tmp/builder-budget \
    if [ "$USE_PREBUILT_BINARY" != "true" ]; then \
      if [ -f /tmp/builder-budget ]; then \
        cp /tmp/builder-budget /app/budget && \
        echo "Using binary from builder stage"; \
      else \
        echo "ERROR: USE_PREBUILT_BINARY=false but builder binary not found" && exit 1; \
      fi; \
    fi

# Expose port
EXPOSE 8080

# Run the application
CMD ["./budget", "serve"]
