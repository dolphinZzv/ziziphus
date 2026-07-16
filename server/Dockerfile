# Build stage
FROM golang:1.26-bookworm AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and embedded frontend dist
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o ziziphus ./cmd/ziziphus/

# ── Runtime stage ──────────────────────────────────────────────
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Non-root user
RUN groupadd -r appuser && useradd -r -g appuser -d /app appuser

WORKDIR /app

# Binary
COPY --from=builder /build/ziziphus .

# Migrations (needed at runtime)
COPY --from=builder /build/internal/storage/db/migrations/ ./internal/storage/db/migrations/

# Config template — mount a real config at runtime
COPY --from=builder /build/config/config.example.yaml ./config/config.example.yaml

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

VOLUME ["/app/config"]

ENTRYPOINT ["./ziziphus", "-c", "config/config.yaml"]
