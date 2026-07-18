# Ziziphus

[![CI](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg)](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://47.95.200.101:10011/)


[English](README.md) | [中文](docs/README.zh.md) | [日本語](docs/README.ja.md) | [Français](docs/README.fr.md) | [Deutsch](docs/README.de.md) | [Español](docs/README.es.md) | [한국어](docs/README.ko.md) | [Русский](docs/README.ru.md)

An instant messaging (IM) application — Go backend powering multiple frontends.

**Supported clients, by priority:**
1. 🌐 **React Web** — full-featured SPA
2. 🖥 **macOS** — native SwiftUI app
3. 📱 **iOS** — native SwiftUI app
4. 🤖 **Android** — (coming soon)

## Project Structure

| Directory | Description |
|-----------|-------------|
| `server/` | Go backend (REST API + WebSocket) |
| `server/web/` | Web frontend (React + TypeScript + Vite) |
| `client/` | macOS / iOS client (Swift + SwiftUI) |
| `deps/` | Local dependencies |
| `bin/` | Build artifacts |

## Tech Stack

- **Backend**: Go 1.26, PostgreSQL, Redis, JWT, WebSocket
- **Web Frontend**: React 19, TypeScript, Vite, Tailwind CSS 4, Zustand
- **Native Client**: Swift 6.3.2, SwiftUI, iOS 18+ / macOS 15+

---

## Installation & Running

### Prerequisites

- Go 1.26+
- Node.js 22+
- Swift 6.3.2+ (macOS client)
- Xcode 16+ (iOS client)
- PostgreSQL 16+
- Redis 7+
- Docker (optional)

### Docker Deployment (Recommended)

#### One-Click Start with Docker Compose

```bash
# 1. Prepare config file
cp server/config/config.example.yaml server/config/config.yaml
# Edit config.yaml as needed

# 2. Start all services (PostgreSQL + Redis + App)
docker compose up -d

# 3. View logs
docker compose logs -f app

# 4. Stop
docker compose down
```

This starts three containers:

| Service | Image | Port |
|---------|-------|------|
| postgres | `postgres:16-alpine` | 5432 |
| redis | `redis:7-alpine` | 6379 |
| app | local build | 8080 |

Persistent data is stored in Docker volumes.

#### Using External PostgreSQL / Redis

Edit `server/config/config.yaml`:

```yaml
postgres:
  dsn: "postgres://user:pass@your-pg-host:5432/imdb?sslmode=require"

redis:
  addr: "your-redis-host:6379"
  password: "your-password"
```

Then start only the app container:

```bash
docker compose up -d app
```

#### Build Image Only

```bash
# Full build (Web frontend + Go backend)
docker build -t ziziphus:latest .

# Go backend only (requires npm run build first)
docker build -t ziziphus:latest -f server/Dockerfile server/
```

#### Run Container Manually

```bash
docker run -d \
  --name ziziphus \
  -p 8080:8080 \
  -v ./server/config/config.yaml:/app/config/config.yaml:ro \
  ziziphus:latest
```

#### Image Registry

On every push to main, CI builds and pushes to GitHub Container Registry:

```bash
docker pull ghcr.io/dolphinZzv/ziziphus:latest
```

### 1. Backend (Source Code)

#### Configuration

```bash
cp server/config/config.example.yaml server/config/config.yaml
# Edit config.yaml — at minimum configure PostgreSQL DSN and JWT secret
```

Key configuration:

| Field | Description | Default |
|-------|-------------|---------|
| `server.port` | HTTP listen port | `8080` |
| `postgres.dsn` | PostgreSQL connection string | `postgres://postgres@localhost:5432/imdb?sslmode=disable` |
| `redis.addr` | Redis address | `localhost:6379` |
| `jwt.secret` | JWT signing key (change in production) | `change-me-to-a-random-secret` |
| `jwt.expire_hours` | Access token lifetime | `1` hour |
| `jwt.refresh_expire_hours` | Refresh token lifetime | `168` hours (7 days) |
| `ratelimit.msg_per_sec` | Message rate limit | `30` msg/sec |
| `smtp.*` | SMTP email service (for verification codes) | — |

#### Install & Run

```bash
# Install Go dependencies
cd server && go mod download

# Build and start (auto-compiles web frontend + starts server)
make server

# Start pre-built binary only
bin/ziziphus -c server/config/config.yaml

# Stop
make server-stop
```

API server listens on `http://localhost:8080`.

### 2. Web Frontend (Dev Mode)

```bash
cd server/web
npm install
npm run dev
```

Dev server at `http://localhost:5173`, API proxied to `http://localhost:8080`.

Production build:

```bash
cd server/web
npm run build
# Output is auto-copied to server/internal/webembed/dist/
# Then compile Go binary to embed the frontend
```

### 3. macOS Client

```bash
# Ensure local dependencies are installed
# deps/textual/ is a local Swift package

# Build & launch macOS client
make macos
```

First run auto-generates `Info.plist` and opens the app. To rebuild:

```bash
make macos-stop
make macos
```

### 4. iOS Client

```bash
# 1) Edit .env, set IOS_DEVICE to your device name
# 2) Generate Xcode project
make xcodegen

# 3) Build and deploy to device
make ios-deploy

# Or open client/IMApp.xcodeproj in Xcode
# Select IMApp-iOS scheme, target your device, Cmd+R
```

> This project uses real devices only (no simulators — see `CLAUDE.md`).

### 5. Database Migrations

Migrations run automatically on startup (`db.RunMigrations`). Scripts are in:

```
server/internal/storage/db/migrations/
```

To run manually:

```bash
psql -d imdb -f server/internal/storage/db/migrations/001_initial.sql
```

---

## Code Quality

```bash
# Web frontend lint
make lint-web

# Go backend lint
make lint-server

# All lint
make lint
```

---

## Deployment

### One-Click Remote Deploy

Edit `.env` file:

| Variable | Description |
|----------|-------------|
| `SSH_HOST` | Server address |
| `SSH_PORT` | SSH port |
| `DEPLOY_PORT` | Service port |
| `DEPLOY_USER` | SSH user |
| `DEPLOY_PATH` | Deploy path |
| `DEPLOY_DSN` | Production database DSN |

Then run:

```bash
make deploy          # Build & deploy (systemd service)
make deploy-status   # Check service status
make deploy-logs     # View service logs
```

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Web Frontend (React + Vite)                     │
│  server/web/                                     │
└──────────────┬──────────────────────────────────┘
               │ HTTP / WebSocket
┌──────────────▼──────────────────────────────────┐
│  Go Backend (ziziphus)                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ API      │ │ WebSocket │ │ Message Route    │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ Session  │ │ Gateway  │ │ File Storage     │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐                       │
│  │PostgreSQL│ │ Redis    │                       │
│  └──────────┘ └──────────┘                       │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  macOS / iOS Client (Swift + SwiftUI)            │
│  client/                                         │
└─────────────────────────────────────────────────┘
```

---

## Internationalization (i18n)

Ziziphus supports **8 languages** across frontend and backend:

| Code | Language | Backend Constant | Frontend File |
|------|----------|-----------------|---------------|
| `zh` | Chinese (Simplified) | `LangZH` | `zh.json` |
| `en` | English | `LangEN` | `en.json` |
| `ja` | Japanese | `LangJA` | `ja.json` |
| `fr` | French | `LangFR` | `fr.json` |
| `de` | German | `LangDE` | `de.json` |
| `es` | Spanish | `LangES` | `es.json` |
| `ko` | Korean | `LangKO` | `ko.json` |
| `ru` | Russian | `LangRU` | `ru.json` |

### Frontend

Uses [i18next](https://www.i18next.com/) with [react-i18next](https://react.i18next.com/). Language preference is stored in `localStorage` under key `ziziphus_language`. Translation files live in `server/web/src/i18n/{lang}.json`. Non-Chinese bundles are lazy-loaded on demand to keep the initial bundle small.

The frontend sends the selected language to the backend via the `X-Language` HTTP header on every request.

### Backend

The `server/pkg/i18n/` package provides:

- **Language constants** (`LangZH`, `LangEN`, ...)
- **ParseLang()** — Accepts browser locale codes (e.g. `zh-CN`, `en-US`, `ja-JP`) and normalizes to a supported Lang constant
- **DetectLanguage()** — Reads `X-Language` header (frontend preference) with fallback to `Accept-Language` header, then to `LangZH`
- **T() / TWithLang()** — Template-style string translation with positional parameters (`{0}`, `{1}`)
- **HTTP Middleware** — Detects language per-request and stores it in the request context

Translation messages are split per language file:
```
pkg/i18n/messages.go          # Message key declarations + registerLang helper
pkg/i18n/{zh,en,ja,fr,de,es,ko,ru}.go  # Each language's translations
```

### Email Templates

Email verification and password reset templates support all 8 languages as well. Templates are embedded at compile time via `//go:embed` and live in:

```
internal/auth/email_templates/
  verify_code_{lang}.html
  reset_password_{lang}.html
```

### Adding a New Language

**Backend:**
1. Add a new `LangXX` constant in `pkg/i18n/i18n.go`
2. Add locale mapping in `ParseLang()`
3. Create a new file `pkg/i18n/{lang}.go` with `init() + registerLang()` for all message keys
4. Add `langToFrontendCode()` mapping in `internal/api/language.go`

**Frontend:**
1. Create `server/web/src/i18n/{lang}.json` with translated key-value pairs
2. Add the language option to the frontend settings/auth language selector
3. Update `Language` type and `resolveAutoLang()` in the UI store

**Email Templates:**
1. Copy an existing template (e.g., `verify_code_en.html` → `verify_code_{lang}.html`)
2. Translate the text content
3. Add `//go:embed` directive and register in `emailTemplates` map in `internal/auth/mailer.go`
4. Add subject translations

---

## Environment Variables

See `.env.example` files:

- `server/config/config.example.yaml` — Backend config template
- `server/web/.env.example` — Web frontend env template
- Root `.env` — Deploy parameters (gitignored)
