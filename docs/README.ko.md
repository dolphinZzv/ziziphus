# Ziziphus

[![CI](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg)](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml) [![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://47.95.200.101:10011/)

> **데모**: [http://47.95.200.101:10011/](http://47.95.200.101:10011/)

[한국어](README.ko.md) | [English](../README.md) | [中文](README.zh.md) | [日本語](README.ja.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Español](README.es.md) | [Русский](README.ru.md)

Go 백엔드가 여러 프론트엔드를 구동하는 인스턴트 메시징(IM) 애플리케이션입니다.

**지원 클라이언트 (우선순위 순):**
1. 🌐 **React Web** — 완전한 기능의 SPA
2. 🖥 **macOS** — 네이티브 SwiftUI 앱
3. 📱 **iOS** — 네이티브 SwiftUI 앱
4. 🤖 **Android** — 곧 지원 예정

## 프로젝트 구조

| 디렉토리 | 설명 |
|----------|------|
| `server/` | Go 백엔드 (REST API + WebSocket) |
| `server/web/` | 웹 프론트엔드 (React + TypeScript + Vite) |
| `client/` | macOS / iOS 클라이언트 (Swift + SwiftUI) |
| `deps/` | 로컬 의존성 |
| `bin/` | 빌드 결과물 |

## 기술 스택

- **백엔드**: Go 1.26, PostgreSQL, Redis, JWT, WebSocket
- **웹 프론트엔드**: React 19, TypeScript, Vite, Tailwind CSS 4, Zustand
- **네이티브 클라이언트**: Swift 6.3.2, SwiftUI, iOS 18+ / macOS 15+

---

## 설치 및 실행

### 사전 요구사항

- Go 1.26+
- Node.js 22+
- Swift 6.3.2+ (macOS 클라이언트)
- Xcode 16+ (iOS 클라이언트)
- PostgreSQL 16+
- Redis 7+
- Docker (선택 사항)

### Docker 배포 (권장)

#### Docker Compose 원클릭 시작

```bash
# 1. 설정 파일 준비
cp server/config/config.example.yaml server/config/config.yaml
# 필요에 따라 config.yaml 편집

# 2. 모든 서비스 시작 (PostgreSQL + Redis + 앱)
docker compose up -d

# 3. 로그 보기
docker compose logs -f app

# 4. 중지
docker compose down
```

세 개의 컨테이너가 시작됩니다:

| 서비스 | 이미지 | 포트 |
|--------|--------|------|
| postgres | `postgres:16-alpine` | 5432 |
| redis | `redis:7-alpine` | 6379 |
| app | 로컬 빌드 | 8080 |

데이터는 Docker 볼륨에 영구 저장됩니다.

#### 외부 PostgreSQL / Redis 사용

`server/config/config.yaml` 편집:

```yaml
postgres:
  dsn: "postgres://user:pass@your-pg-host:5432/imdb?sslmode=require"

redis:
  addr: "your-redis-host:6379"
  password: "your-password"
```

그런 다음 앱 컨테이너만 시작:

```bash
docker compose up -d app
```

#### 이미지만 빌드 (시작하지 않음)

```bash
# 전체 빌드 (웹 프론트엔드 + Go 백엔드)
docker build -t ziziphus:latest .

# Go 백엔드만 (사전에 npm run build 필요)
docker build -t ziziphus:latest -f server/Dockerfile server/
```

#### 컨테이너 수동 실행

```bash
docker run -d \
  --name ziziphus \
  -p 8080:8080 \
  -v ./server/config/config.yaml:/app/config/config.yaml:ro \
  ziziphus:latest
```

#### 이미지 레지스트리

main 브랜치에 푸시할 때마다 CI가 이미지를 빌드하여 GitHub Container Registry에 푸시합니다:

```bash
docker pull ghcr.io/dolphinZzv/ziziphus:latest
```

### 1. 백엔드 (소스 코드 실행)

#### 설정

```bash
# 설정 파일 복사
cp server/config/config.example.yaml server/config/config.yaml
# config.yaml 편집 — 최소 PostgreSQL DSN과 JWT secret은 설정해야 합니다
```

주요 설정 항목:

| 필드 | 설명 | 기본값 |
|------|------|--------|
| `server.port` | HTTP 수신 포트 | `8080` |
| `postgres.dsn` | PostgreSQL 연결 문자열 | `postgres://postgres@localhost:5432/imdb?sslmode=disable` |
| `redis.addr` | Redis 주소 | `localhost:6379` |
| `jwt.secret` | JWT 서명 키 (프로덕션에서는 변경 필요) | `change-me-to-a-random-secret` |
| `jwt.expire_hours` | 액세스 토큰 만료 시간 | `1` 시간 |
| `jwt.refresh_expire_hours` | 리프레시 토큰 만료 시간 | `168` 시간 (7일) |
| `ratelimit.msg_per_sec` | 메시지 전송 속도 제한 | `30` 개/초 |
| `smtp.*` | SMTP 이메일 서비스 (인증 코드 전송용) | — |

#### 의존성 설치 및 실행

```bash
# Go 의존성 설치
cd server && go mod download

# 빌드 및 시작 (웹 프론트엔드 자동 컴파일 + 서버 시작)
make server

# 미리 빌드된 바이너리만 시작
bin/ziziphus -c server/config/config.yaml

# 중지
make server-stop
```

API 서버가 `http://localhost:8080`에서 수신 대기합니다.

### 2. 웹 프론트엔드 (개발 모드)

```bash
cd server/web
npm install
npm run dev
```

개발 서버는 `http://localhost:5173`에서 실행되며, API는 `http://localhost:8080`으로 프록시됩니다.

프로덕션 빌드:

```bash
cd server/web
npm run build
# 출력물이 server/internal/webembed/dist/로 자동 복사됩니다
# 그런 다음 Go 바이너리를 컴파일하여 프론트엔드를 임베드합니다
```

### 3. macOS 클라이언트

```bash
# 로컬 의존성이 설치되어 있는지 확인
# deps/textual/은 로컬 Swift 패키지입니다

# macOS 클라이언트 빌드 및 시작
make macos
```

첫 실행 시 `Info.plist`가 자동 생성되고 앱이 열립니다. 다시 빌드하려면:

```bash
make macos-stop
make macos
```

### 4. iOS 클라이언트

```bash
# 1) .env 파일 편집, IOS_DEVICE를 기기 이름으로 설정
# 2) Xcode 프로젝트 생성
make xcodegen

# 3) 실제 기기에 빌드 및 배포
make ios-deploy

# 또는 Xcode에서 client/IMApp.xcodeproj 열기
# IMApp-iOS 스킴 선택, 실제 기기 선택, Cmd+R 실행
```

> 이 프로젝트는 시뮬레이터를 사용하지 않고 실제 기기만 사용합니다 (`CLAUDE.md` 참조).

### 5. 데이터베이스 마이그레이션

마이그레이션은 서버 시작 시 자동 실행됩니다 (`db.RunMigrations`). 스크립트 위치:

```
server/internal/storage/db/migrations/
```

수동 실행:

```bash
psql -d imdb -f server/internal/storage/db/migrations/001_initial.sql
```

---

## 코드 품질

```bash
# 웹 프론트엔드 린트
make lint-web

# Go 백엔드 린트
make lint-server

# 전체 린트
make lint
```

---

## 배포

### 원클릭 원격 배포

`.env` 파일 편집:

| 변수 | 설명 |
|------|------|
| `SSH_HOST` | 서버 주소 |
| `SSH_PORT` | SSH 포트 |
| `DEPLOY_PORT` | 서비스 포트 |
| `DEPLOY_USER` | SSH 사용자 |
| `DEPLOY_PATH` | 배포 경로 |
| `DEPLOY_DSN` | 프로덕션 데이터베이스 연결 문자열 |

그런 다음 실행:

```bash
make deploy          # 빌드 및 원격 서버에 배포 (systemd 서비스)
make deploy-status   # 서비스 상태 확인
make deploy-logs     # 서비스 로그 보기
```

---

## 아키텍처

```
┌─────────────────────────────────────────────────┐
│  웹 프론트엔드 (React + Vite)                     │
│  server/web/                                     │
└──────────────┬──────────────────────────────────┘
               │ HTTP / WebSocket
┌──────────────▼──────────────────────────────────┐
│  Go 백엔드 (ziziphus)                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ API      │ │ WebSocket │ │ 메시지 라우팅     │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ 세션     │ │ 게이트웨이│ │ 파일 스토리지     │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐                       │
│  │PostgreSQL│ │ Redis    │                       │
│  └──────────┘ └──────────┘                       │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  macOS / iOS 클라이언트 (Swift + SwiftUI)         │
│  client/                                         │
└─────────────────────────────────────────────────┘
```

---

## 국제화 (i18n)

Ziziphus는 프론트엔드와 백엔드에서 **8개 언어**를 지원합니다:

| 코드 | 언어 | 백엔드 상수 | 프론트엔드 파일 |
|------|------|-------------|----------------|
| `zh` | 중국어 간체 | `LangZH` | `zh.json` |
| `en` | 영어 | `LangEN` | `en.json` |
| `ja` | 일본어 | `LangJA` | `ja.json` |
| `fr` | 프랑스어 | `LangFR` | `fr.json` |
| `de` | 독일어 | `LangDE` | `de.json` |
| `es` | 스페인어 | `LangES` | `es.json` |
| `ko` | 한국어 | `LangKO` | `ko.json` |
| `ru` | 러시아어 | `LangRU` | `ru.json` |

### 프론트엔드

[i18next](https://www.i18next.com/) + [react-i18next](https://react.i18next.com/) 사용. 언어 설정은 `localStorage`(키: `ziziphus_language`)에 저장됩니다. 번역 파일은 `server/web/src/i18n/{lang}.json`에 있습니다. 중국어 외 언어 팩은 필요 시 지연 로드되어 초기 번들 크기를 최소화합니다.

프론트엔드는 모든 요청에서 `X-Language` HTTP 헤더를 통해 선택된 언어를 백엔드에 전송합니다.

### 백엔드

`server/pkg/i18n/` 패키지 제공 기능:

- **언어 상수** (`LangZH`, `LangEN` 등)
- **ParseLang()** — 브라우저 로케일 코드(예: `zh-CN`, `en-US`, `ja-JP`)를 지원되는 Lang 상수로 정규화
- **DetectLanguage()** — `X-Language` 헤더(프론트엔드 설정) 우선, `Accept-Language` 헤더로 폴백, 마지막으로 `LangZH`로 폴백
- **T() / TWithLang()** — 위치 매개변수(`{0}`, `{1}`)를 지원하는 템플릿 방식 문자열 번역
- **HTTP 미들웨어** — 요청별로 언어를 감지하여 컨텍스트에 저장

번역 메시지는 언어별로 파일이 분할되어 있습니다:
```
pkg/i18n/messages.go          # 메시지 키 선언 + registerLang 헬퍼
pkg/i18n/{zh,en,ja,fr,de,es,ko,ru}.go  # 각 언어의 번역 데이터
```

### 이메일 템플릿

이메일 확인 및 비밀번호 재설정 템플릿도 8개 언어를 모두 지원합니다. 템플릿은 컴파일 시 `//go:embed`로 임베드되며 다음 위치에 있습니다:

```
internal/auth/email_templates/
  verify_code_{lang}.html
  reset_password_{lang}.html
```

### 새 언어 추가 방법

**백엔드:**
1. `pkg/i18n/i18n.go`에 새 `LangXX` 상수 추가
2. `ParseLang()`에 로케일 매핑 추가
3. `pkg/i18n/{lang}.go` 생성, `init() + registerLang()`으로 모든 메시지 키 등록
4. `internal/api/language.go`에 `langToFrontendCode()` 매핑 추가

**프론트엔드:**
1. 번역된 키-값 쌍이 포함된 `server/web/src/i18n/{lang}.json` 생성
2. 설정 및 인증 페이지의 언어 선택기에 해당 언어 옵션 추가
3. UI 스토어에서 `Language` 타입과 `resolveAutoLang()` 메서드 업데이트

**이메일 템플릿:**
1. 기존 템플릿 복사 (예: `verify_code_en.html` → `verify_code_{lang}.html`)
2. 텍스트 내용 번역
3. `internal/auth/mailer.go`에 `//go:embed` 지시문 추가 및 `emailTemplates` 맵에 등록
4. 제목 번역 추가

---

## 환경 변수

`.env.example` 파일 참조:

- `server/config/config.example.yaml` — 백엔드 설정 템플릿
- `server/web/.env.example` — 웹 프론트엔드 환경 변수 템플릿
- 루트 `.env` — 배포 파라미터 (`.gitignore`에 포함되어 커밋되지 않음)
