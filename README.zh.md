# Ziziphus

[![codecov](https://codecov.io/gh/dolphinZzv/ziziphus/branch/main/graph/badge.svg)](https://codecov.io/gh/dolphinZzv/ziziphus)
[![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://ziziphus.siciv.space:10011/)

> **演示地址**: [http://ziziphus.siciv.space:10011/](http://ziziphus.siciv.space:10011/)

[中文](README.zh.md) | [English](README.md)

即时通讯（IM）应用 — Go 后端驱动多端客户端。

**客户端支持优先级：**
1. 🌐 **React Web** — 功能完整的 SPA
2. 🖥 **macOS** — 原生 SwiftUI 应用
3. 📱 **iOS** — 原生 SwiftUI 应用
4. 🤖 **Android** — 即将支持

## 项目结构

| 目录 | 说明 |
|------|------|
| `server/` | Go 后端服务（REST API + WebSocket） |
| `server/web/` | Web 前端（React + TypeScript + Vite） |
| `client/` | macOS / iOS 客户端（Swift + SwiftUI） |
| `deps/` | 本地依赖 |
| `bin/` | 编译产物 |

## 技术栈

- **后端**: Go 1.26, PostgreSQL, Redis, JWT, WebSocket
- **Web 前端**: React 19, TypeScript, Vite, Tailwind CSS 4, Zustand
- **原生客户端**: Swift 6.3.2, SwiftUI, iOS 18+ / macOS 15+

---

## 安装与运行

### 环境要求

- Go 1.26+
- Node.js 22+
- Swift 6.3.2+ (macOS 客户端需要)
- Xcode 16+ (iOS 客户端需要)
- PostgreSQL 16+
- Redis 7+
- Docker (可选)

### Docker 部署（推荐）

#### 使用 Docker Compose 一键启动

```bash
# 1. 准备配置文件
cp server/config/config.example.yaml server/config/config.yaml
# 编辑 config.yaml，可按需修改端口、密码等

# 2. 启动全部服务（PostgreSQL + Redis + 应用）
docker compose up -d

# 3. 查看日志
docker compose logs -f app

# 4. 停止
docker compose down
```

这会自动启动三个容器：

| 服务 | 镜像 | 端口 |
|------|------|------|
| postgres | `postgres:16-alpine` | 5432 |
| redis | `redis:7-alpine` | 6379 |
| app | 本地构建 | 8080 |

数据持久化在 Docker volume 中，重启不丢失。

#### 连接外部 PostgreSQL / Redis

编辑 `server/config/config.yaml`，将地址改为外部服务：

```yaml
postgres:
  dsn: "postgres://user:pass@your-pg-host:5432/imdb?sslmode=require"

redis:
  addr: "your-redis-host:6379"
  password: "your-password"
```

然后只启动应用容器：

```bash
docker compose up -d app
```

#### 仅构建镜像（不启动）

```bash
# 完整构建（含 Web 前端 + Go 后端）
docker build -t ziziphus:latest .

# 仅 Go 后端（需要预先 npm run build 前端）
docker build -t ziziphus:latest -f server/Dockerfile server/
```

#### 手动运行容器

```bash
docker run -d \
  --name ziziphus \
  -p 8080:8080 \
  -v ./server/config/config.yaml:/app/config/config.yaml:ro \
  ziziphus:latest
```

#### 镜像注册表

每次推送到 main 分支，CI 自动构建并推送镜像到 GitHub Container Registry：

```bash
docker pull ghcr.io/dolphinZzv/ziziphus:latest
```

### 1. 后端服务（源码运行）

#### 配置

```bash
# 复制配置文件
cp server/config/config.example.yaml server/config/config.yaml
# 按需编辑 config.yaml，至少配置 PostgreSQL DSN 和 JWT secret
```

配置项说明：

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `server.port` | HTTP 监听端口 | `8080` |
| `postgres.dsn` | PostgreSQL 连接串 | `postgres://postgres@localhost:5432/imdb?sslmode=disable` |
| `redis.addr` | Redis 地址 | `localhost:6379` |
| `jwt.secret` | JWT 签名密钥（生产环境请更换） | `change-me-to-a-random-secret` |
| `jwt.expire_hours` | Access Token 有效期 | `1` 小时 |
| `jwt.refresh_expire_hours` | Refresh Token 有效期 | `168` 小时（7 天） |
| `ratelimit.msg_per_sec` | 消息发送频率限制 | `30` 条/秒 |
| `smtp.*` | SMTP 邮件服务（用于验证码等） | — |

#### 安装依赖并启动

```bash
# 安装 Go 依赖
cd server && go mod download

# 构建并启动（自动编译 Web 前端 + 启动服务）
make server

# 仅启动已编译的二进制
bin/ziziphus -c server/config/config.yaml

# 停止服务
make server-stop
```

启动后 API 服务监听在 `http://localhost:8080`。

### 2. Web 前端（开发模式）

```bash
cd server/web
npm install
npm run dev
```

开发服务器启动于 `http://localhost:5173`，API 默认代理到 `http://localhost:8080`。

生产构建：

```bash
cd server/web
npm run build
# 构建产物自动复制到 server/internal/webembed/dist/
# 随后编译 Go 二进制即可嵌入前端
```

### 3. macOS 客户端

```bash
# 确保已安装本地依赖
# deps/textual/ 为本地 Swift 包依赖

# 构建并启动 macOS 客户端
make macos
```

首次运行会自动生成 `Info.plist`，并打开应用。如需清除后重新构建：

```bash
make macos-stop
make macos
```

### 4. iOS 客户端

```bash
# 1) 编辑 .env 文件，设置 IOS_DEVICE 为你的真机名称
# 2) 生成 Xcode 项目
make xcodegen

# 3) 编译并部署到真机
make ios-deploy

# 也可手动通过 Xcode 打开 client/IMApp.xcodeproj
# 选择 IMApp-iOS scheme，目标选真机，Cmd+R 运行
```

> 本项目不使用模拟器，请使用真机部署（`CLAUDE.md` 约定）。

### 5. 数据库迁移

数据库迁移在服务启动时自动执行（`db.RunMigrations`）。迁移脚本位于：

```
server/internal/storage/db/migrations/
```

如需手动执行：

```bash
psql -d imdb -f server/internal/storage/db/migrations/001_initial.sql
```

---

## 代码质量

```bash
# Web 前端 lint
make lint-web

# Go 后端 lint
make lint-server

# 全部 lint
make lint
```

---

## 部署

### 一键远程部署

编辑 `.env` 文件，配置以下变量：

| 变量 | 说明 |
|------|------|
| `SSH_HOST` | 服务器地址 |
| `SSH_PORT` | SSH 端口 |
| `DEPLOY_PORT` | 服务端口 |
| `DEPLOY_USER` | SSH 用户 |
| `DEPLOY_PATH` | 部署路径 |
| `DEPLOY_DSN` | 生产数据库连接串 |

然后执行：

```bash
make deploy          # 构建并部署到远程服务器（systemd 服务）
make deploy-status   # 查看服务状态
make deploy-logs     # 查看服务日志
```

---

## 架构概览

```
┌─────────────────────────────────────────────────┐
│  Web 前端 (React + Vite)                         │
│  server/web/                                     │
└──────────────┬──────────────────────────────────┘
               │ HTTP / WebSocket
┌──────────────▼──────────────────────────────────┐
│  Go 后端 (ziziphus)                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ API 层   │ │ WebSocket │ │ 消息路由 / 推送   │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ 会话管理  │ │ 网关     │ │ 文件存储          │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐                       │
│  │ PostgreSQL│ │ Redis    │                       │
│  └──────────┘ └──────────┘                       │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  macOS / iOS 客户端 (Swift + SwiftUI)             │
│  client/                                         │
└─────────────────────────────────────────────────┘
```

---

## 环境变量

参考 `.env.example` 文件：

- `server/config/config.example.yaml` — 后端配置模版
- `server/web/.env.example` — Web 前端环境变量模版
- 项目根 `.env` — 部署参数（已加入 `.gitignore`，不会提交）
