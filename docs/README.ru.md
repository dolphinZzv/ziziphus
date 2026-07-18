# Ziziphus

[![CI](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg)](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![Backend](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg?query=branch:main+job:"Backend+\(test+%2B+lint\)")](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![Frontend](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg?query=branch:main+job:"Frontend+\(lint+%2B+build\)")](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![E2E](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg?query=branch:main+job:"E2E+tests")](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![Container](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg?query=branch:main+job:"Container+\(build+%2B+smoke+%2B+push\)")](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/dolphinZzv/ziziphus/branch/main/graph/badge.svg)](https://codecov.io/gh/dolphinZzv/ziziphus)
[![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://47.95.200.101:10011/)

> **Демо**: [http://47.95.200.101:10011/](http://47.95.200.101:10011/)

[Русский](README.ru.md) | [English](../README.md) | [中文](README.zh.md) | [日本語](README.ja.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Español](README.es.md) | [한국어](README.ko.md)

Приложение для обмена мгновенными сообщениями (IM) — бэкенд на Go, обслуживающий несколько фронтендов.

**Поддерживаемые клиенты (в порядке приоритета):**
1. 🌐 **React Web** — полнофункциональное SPA
2. 🖥 **macOS** — нативное приложение SwiftUI
3. 📱 **iOS** — нативное приложение SwiftUI
4. 🤖 **Android** — скоро

## Структура проекта

| Директория | Описание |
|------------|----------|
| `server/` | Бэкенд Go (REST API + WebSocket) |
| `server/web/` | Веб-фронтенд (React + TypeScript + Vite) |
| `client/` | Клиент macOS / iOS (Swift + SwiftUI) |
| `deps/` | Локальные зависимости |
| `bin/` | Артефакты сборки |

## Технологический стек

- **Бэкенд**: Go 1.26, PostgreSQL, Redis, JWT, WebSocket
- **Веб-фронтенд**: React 19, TypeScript, Vite, Tailwind CSS 4, Zustand
- **Нативные клиенты**: Swift 6.3.2, SwiftUI, iOS 18+ / macOS 15+

---

## Установка и запуск

### Требования

- Go 1.26+
- Node.js 22+
- Swift 6.3.2+ (клиент macOS)
- Xcode 16+ (клиент iOS)
- PostgreSQL 16+
- Redis 7+
- Docker (опционально)

### Развёртывание с Docker (рекомендуется)

#### Запуск одним кликом с Docker Compose

```bash
# 1. Подготовить файл конфигурации
cp server/config/config.example.yaml server/config/config.yaml
# Отредактируйте config.yaml по необходимости

# 2. Запустить все сервисы (PostgreSQL + Redis + Приложение)
docker compose up -d

# 3. Просмотр логов
docker compose logs -f app

# 4. Остановить
docker compose down
```

Это запустит три контейнера:

| Сервис | Образ | Порт |
|--------|-------|------|
| postgres | `postgres:16-alpine` | 5432 |
| redis | `redis:7-alpine` | 6379 |
| app | локальная сборка | 8080 |

Постоянные данные хранятся в томах Docker.

#### Использование внешних PostgreSQL / Redis

Отредактируйте `server/config/config.yaml`:

```yaml
postgres:
  dsn: "postgres://user:pass@your-pg-host:5432/imdb?sslmode=require"

redis:
  addr: "your-redis-host:6379"
  password: "your-password"
```

Затем запустите только контейнер приложения:

```bash
docker compose up -d app
```

#### Только сборка образа (без запуска)

```bash
# Полная сборка (Веб-фронтенд + Go бэкенд)
docker build -t ziziphus:latest .

# Только Go бэкенд (требуется предварительный npm run build)
docker build -t ziziphus:latest -f server/Dockerfile server/
```

#### Запуск контейнера вручную

```bash
docker run -d \
  --name ziziphus \
  -p 8080:8080 \
  -v ./server/config/config.yaml:/app/config/config.yaml:ro \
  ziziphus:latest
```

#### Реестр образов

При каждом пуше в main, CI собирает образ и отправляет его в GitHub Container Registry:

```bash
docker pull ghcr.io/dolphinZzv/ziziphus:latest
```

### 1. Бэкенд (запуск из исходного кода)

#### Конфигурация

```bash
# Скопировать файл конфигурации
cp server/config/config.example.yaml server/config/config.yaml
# Отредактируйте config.yaml — как минимум настройте PostgreSQL DSN и JWT secret
```

Ключевые настройки:

| Поле | Описание | По умолчанию |
|------|----------|-------------|
| `server.port` | HTTP порт для прослушивания | `8080` |
| `postgres.dsn` | Строка подключения к PostgreSQL | `postgres://postgres@localhost:5432/imdb?sslmode=disable` |
| `redis.addr` | Адрес Redis | `localhost:6379` |
| `jwt.secret` | Ключ подписи JWT (измените в продакшене) | `change-me-to-a-random-secret` |
| `jwt.expire_hours` | Срок действия токена доступа | `1` час |
| `jwt.refresh_expire_hours` | Срок действия токена обновления | `168` часов (7 дней) |
| `ratelimit.msg_per_sec` | Лимит отправки сообщений | `30` сообщ/с |
| `smtp.*` | SMTP сервис электронной почты (для кодов подтверждения) | — |

#### Установка зависимостей и запуск

```bash
# Установка Go зависимостей
cd server && go mod download

# Сборка и запуск (автоматически компилирует веб-фронтенд + запускает сервер)
make server

# Запуск только предварительно собранного бинарника
bin/ziziphus -c server/config/config.yaml

# Остановка
make server-stop
```

API сервер слушает на `http://localhost:8080`.

### 2. Веб-фронтенд (режим разработки)

```bash
cd server/web
npm install
npm run dev
```

Сервер разработки запускается на `http://localhost:5173`, API проксируется на `http://localhost:8080`.

Продакшен-сборка:

```bash
cd server/web
npm run build
# Вывод автоматически копируется в server/internal/webembed/dist/
# Затем скомпилируйте Go бинарник для встраивания фронтенда
```

### 3. Клиент macOS

```bash
# Убедитесь, что локальные зависимости установлены
# deps/textual/ — локальный Swift пакет

# Собрать и запустить клиент macOS
make macos
```

При первом запуске автоматически генерируется `Info.plist` и открывается приложение. Для пересборки:

```bash
make macos-stop
make macos
```

### 4. Клиент iOS

```bash
# 1) Отредактируйте .env, установите IOS_DEVICE на имя вашего устройства
# 2) Сгенерируйте проект Xcode
make xcodegen

# 3) Соберите и разверните на устройстве
make ios-deploy

# Или откройте client/IMApp.xcodeproj в Xcode
# Выберите схему IMApp-iOS, выберите физическое устройство, Cmd+R
```

> Этот проект использует только реальные устройства (без симуляторов — см. `CLAUDE.md`).

### 5. Миграции базы данных

Миграции выполняются автоматически при запуске (`db.RunMigrations`). Скрипты находятся в:

```
server/internal/storage/db/migrations/
```

Для ручного выполнения:

```bash
psql -d imdb -f server/internal/storage/db/migrations/001_initial.sql
```

---

## Качество кода

```bash
# Линтинг веб-фронтенда
make lint-web

# Линтинг Go бэкенда
make lint-server

# Весь линтинг
make lint
```

---

## Развёртывание

### Удалённое развёртывание одним кликом

Отредактируйте файл `.env`:

| Переменная | Описание |
|------------|----------|
| `SSH_HOST` | Адрес сервера |
| `SSH_PORT` | Порт SSH |
| `DEPLOY_PORT` | Порт сервиса |
| `DEPLOY_USER` | Пользователь SSH |
| `DEPLOY_PATH` | Путь развёртывания |
| `DEPLOY_DSN` | Строка подключения к продакшен-базе |

Затем выполните:

```bash
make deploy          # Собрать и развернуть на удалённом сервере (systemd сервис)
make deploy-status   # Проверить статус сервиса
make deploy-logs     | Просмотр логов сервиса
```

---

## Архитектура

```
┌─────────────────────────────────────────────────┐
│  Веб-фронтенд (React + Vite)                     │
│  server/web/                                     │
└──────────────┬──────────────────────────────────┘
               │ HTTP / WebSocket
┌──────────────▼──────────────────────────────────┐
│  Go Бэкенд (ziziphus)                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ API      │ │ WebSocket │ │ Маршрутизация    │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ Сессия   │ │ Шлюз     │ │ Хранилище файлов │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐                       │
│  │PostgreSQL│ │ Redis    │                       │
│  └──────────┘ └──────────┘                       │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  Клиент macOS / iOS (Swift + SwiftUI)             │
│  client/                                         │
└─────────────────────────────────────────────────┘
```

---

## Интернационализация (i18n)

Ziziphus поддерживает **8 языков** на фронтенде и бэкенде:

| Код | Язык | Константа бэкенда | Файл фронтенда |
|-----|------|-------------------|----------------|
| `zh` | Китайский (упрощённый) | `LangZH` | `zh.json` |
| `en` | Английский | `LangEN` | `en.json` |
| `ja` | Японский | `LangJA` | `ja.json` |
| `fr` | Французский | `LangFR` | `fr.json` |
| `de` | Немецкий | `LangDE` | `de.json` |
| `es` | Испанский | `LangES` | `es.json` |
| `ko` | Корейский | `LangKO` | `ko.json` |
| `ru` | Русский | `LangRU` | `ru.json` |

### Фронтенд

Использует [i18next](https://www.i18next.com/) + [react-i18next](https://react.i18next.com/). Языковые предпочтения сохраняются в `localStorage` под ключом `ziziphus_language`. Файлы переводов находятся в `server/web/src/i18n/{lang}.json`. Пакеты, отличные от китайского, загружаются по требованию, чтобы минимизировать начальный размер бандла.

Фронтенд отправляет выбранный язык на бэкенд через HTTP-заголовок `X-Language` в каждом запросе.

### Бэкенд

Пакет `server/pkg/i18n/` предоставляет:

- **Константы языков** (`LangZH`, `LangEN`, ...)
- **ParseLang()** — Принимает коды локали браузера (например, `zh-CN`, `en-US`, `ja-JP`) и нормализует их в поддерживаемую константу Lang
- **DetectLanguage()** — Читает заголовок `X-Language` (настройка фронтенда) с запасным вариантом на заголовок `Accept-Language`, затем на `LangZH`
- **T() / TWithLang()** — Шаблонный перевод строк с позиционными параметрами (`{0}`, `{1}`)
- **HTTP Middleware** — Определяет язык для каждого запроса и сохраняет его в контексте

Сообщения перевода разделены по файлам языков:
```
pkg/i18n/messages.go          # Объявление ключей сообщений + вспомогательная registerLang
pkg/i18n/{zh,en,ja,fr,de,es,ko,ru}.go  # Переводы каждого языка
```

### Шаблоны писем

Шаблоны писем для подтверждения email и сброса пароля также поддерживают все 8 языков. Шаблоны встраиваются во время компиляции через `//go:embed` и находятся в:

```
internal/auth/email_templates/
  verify_code_{lang}.html
  reset_password_{lang}.html
```

### Добавление нового языка

**Бэкенд:**
1. Добавьте новую константу `LangXX` в `pkg/i18n/i18n.go`
2. Добавьте сопоставление локали в `ParseLang()`
3. Создайте `pkg/i18n/{lang}.go` с `init() + registerLang()` для всех ключей сообщений
4. Добавьте сопоставление `langToFrontendCode()` в `internal/api/language.go`

**Фронтенд:**
1. Создайте `server/web/src/i18n/{lang}.json` с переведёнными парами ключ-значение
2. Добавьте опцию языка в селектор языка на страницах настроек и аутентификации
3. Обновите тип `Language` и метод `resolveAutoLang()` в сторе UI

**Шаблоны писем:**
1. Скопируйте существующий шаблон (например, `verify_code_en.html` → `verify_code_{lang}.html`)
2. Переведите текстовое содержание
3. Добавьте директиву `//go:embed` и зарегистрируйте в map `emailTemplates` в `internal/auth/mailer.go`
4. Добавьте переводы тем писем

---

## Переменные окружения

Смотрите файлы `.env.example`:

- `server/config/config.example.yaml` — Шаблон конфигурации бэкенда
- `server/web/.env.example` — Шаблон переменных окружения веб-фронтенда
- Корневой `.env` — Параметры развёртывания (в `.gitignore`, не коммитится)
