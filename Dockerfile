# ── Stage 1: Build ────────────────────────────────────────────────────────────
# Use the official Go image on Alpine for a minimal build environment.
FROM golang:1.22-alpine AS builder

# libvips-dev and build tools are required for the govips image-processing
# library. CGO must be enabled because govips calls into C via cgo.
RUN apk add --no-cache \
    vips-dev \
    gcc \
    musl-dev \
    pkgconf

WORKDIR /build

# Copy dependency manifests first so Docker caches the download layer
# separately from the source code. Re-downloads only when go.mod changes.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and compile.
# -ldflags="-s -w" strips debug symbols → ~30% smaller binary.
# CGO_ENABLED=1 is required for govips (libvips C bindings).
COPY . .
RUN CGO_ENABLED=1 GOOS=linux \
    go build -ldflags="-s -w" -o /go-sidecar-worker ./cmd/worker


# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
# Use a minimal Alpine image. Only the vips runtime lib (not the dev headers)
# is needed here — this keeps the final image under ~30 MB.
FROM alpine:3.19

RUN apk add --no-cache \
    vips \
    ca-certificates \
    tzdata

# Create a non-root user for security hardening.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /go-sidecar-worker .

# Transfer ownership to the non-root user.
RUN chown appuser:appgroup /app/go-sidecar-worker

USER appuser

# Health check: Docker will mark the container unhealthy if /health stops
# responding, and orchestrators (Swarm/K8s) can restart it automatically.
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

EXPOSE 8080

ENTRYPOINT ["/app/go-sidecar-worker"]
