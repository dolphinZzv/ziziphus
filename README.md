# Ziziphus

[![CI](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg)](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![Backend](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg?query=branch:main+job:"Backend+\(test+%2B+lint\)")](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![Frontend](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg?query=branch:main+job:"Frontend+\(lint+%2B+build\)")](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![E2E](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg?query=branch:main+job:"E2E+tests")](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![Container](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg?query=branch:main+job:"Container+\(build+%2B+smoke+%2B+push\)")](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/dolphinZzv/ziziphus/branch/main/graph/badge.svg)](https://codecov.io/gh/dolphinZzv/ziziphus)
[![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://ziziphus.siciv.space:10011/)

> **Demo**: [http://ziziphus.siciv.space:10011/](http://ziziphus.siciv.space:10011/)

[English](README.md) | [中文](README.zh.md)

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

## Environment Variables

See `.env.example` files:

- `server/config/config.example.yaml` — Backend config template
- `server/web/.env.example` — Web frontend env template
- Root `.env` — Deploy parameters (gitignored)
