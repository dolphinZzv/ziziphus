# ── Web build stage ────────────────────────────────────────────
FROM node:22-bookworm-slim AS webbuilder

WORKDIR /web

COPY server/web/package.json server/web/package-lock.json ./
RUN npm ci

COPY server/web/ ./
RUN mkdir -p /internal/webembed && npm run build

# ── Go build stage ─────────────────────────────────────────────
FROM golang:1.26-bookworm AS gobuilder

WORKDIR /build

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./

# Inject web dist from the webbuilder stage into the embed directory
COPY --from=webbuilder /web/dist/ ./internal/webembed/dist/

# Static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o ziziphus ./cmd/ziziphus/

# ── Runtime stage ──────────────────────────────────────────────
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r appuser && useradd -r -g appuser -d /app appuser

WORKDIR /app

COPY --from=gobuilder /build/ziziphus .
COPY --from=gobuilder /build/internal/storage/db/migrations/ ./internal/storage/db/migrations/
COPY --from=gobuilder /build/config/config.example.yaml ./config/config.example.yaml

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

VOLUME ["/app/config"]

ENTRYPOINT ["./ziziphus", "-c", "config/config.yaml"]
