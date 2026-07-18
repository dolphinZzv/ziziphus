# Ziziphus

[![CI](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg)](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml) [![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://47.95.200.101:10011/)

> **Demo**: [http://47.95.200.101:10011/](http://47.95.200.101:10011/)

[Deutsch](README.de.md) | [English](../README.md) | [中文](README.zh.md) | [日本語](README.ja.md) | [Français](README.fr.md) | [Español](README.es.md) | [한국어](README.ko.md) | [Русский](README.ru.md)

Eine Instant-Messaging (IM) Anwendung — Go-Backend mit mehreren Frontends.

**Unterstützte Clients (nach Priorität):**
1. 🌐 **React Web** — voll ausgestattete SPA
2. 🖥 **macOS** — native SwiftUI-App
3. 📱 **iOS** — native SwiftUI-App
4. 🤖 **Android** — in Kürze

## Projektstruktur

| Verzeichnis | Beschreibung |
|-------------|-------------|
| `server/` | Go-Backend (REST API + WebSocket) |
| `server/web/` | Web-Frontend (React + TypeScript + Vite) |
| `client/` | macOS / iOS-Client (Swift + SwiftUI) |
| `deps/` | Lokale Abhängigkeiten |
| `bin/` | Build-Artefakte |

## Technologie-Stack

- **Backend**: Go 1.26, PostgreSQL, Redis, JWT, WebSocket
- **Web-Frontend**: React 19, TypeScript, Vite, Tailwind CSS 4, Zustand
- **Native Client**: Swift 6.3.2, SwiftUI, iOS 18+ / macOS 15+

---

## Installation und Ausführung

### Voraussetzungen

- Go 1.26+
- Node.js 22+
- Swift 6.3.2+ (macOS-Client)
- Xcode 16+ (iOS-Client)
- PostgreSQL 16+
- Redis 7+
- Docker (optional)

### Docker-Bereitstellung (Empfohlen)

#### Ein-Klick-Start mit Docker Compose

```bash
# 1. Konfigurationsdatei vorbereiten
cp server/config/config.example.yaml server/config/config.yaml
# config.yaml nach Bedarf bearbeiten

# 2. Alle Dienste starten (PostgreSQL + Redis + App)
docker compose up -d

# 3. Logs anzeigen
docker compose logs -f app

# 4. Stoppen
docker compose down
```

Dies startet drei Container:

| Dienst | Image | Port |
|--------|-------|------|
| postgres | `postgres:16-alpine` | 5432 |
| redis | `redis:7-alpine` | 6379 |
| app | Lokaler Build | 8080 |

Persistente Daten werden in Docker-Volumes gespeichert.

#### Externe PostgreSQL / Redis verwenden

Bearbeiten Sie `server/config/config.yaml`:

```yaml
postgres:
  dsn: "postgres://user:pass@your-pg-host:5432/imdb?sslmode=require"

redis:
  addr: "your-redis-host:6379"
  password: "your-password"
```

Starten Sie dann nur den App-Container:

```bash
docker compose up -d app
```

#### Nur Image erstellen (nicht starten)

```bash
# Vollständiger Build (Web-Frontend + Go-Backend)
docker build -t ziziphus:latest .

# Nur Go-Backend (erfordert vorher npm run build)
docker build -t ziziphus:latest -f server/Dockerfile server/
```

#### Container manuell ausführen

```bash
docker run -d \
  --name ziziphus \
  -p 8080:8080 \
  -v ./server/config/config.yaml:/app/config/config.yaml:ro \
  ziziphus:latest
```

#### Image-Registrierung

Bei jedem Push auf main erstellt CI das Image und pusht es an GitHub Container Registry:

```bash
docker pull ghcr.io/dolphinZzv/ziziphus:latest
```

### 1. Backend (Quellcode-Ausführung)

#### Konfiguration

```bash
# Konfigurationsdatei kopieren
cp server/config/config.example.yaml server/config/config.yaml
# config.yaml bearbeiten — mindestens PostgreSQL DSN und JWT Secret konfigurieren
```

Wichtige Konfiguration:

| Feld | Beschreibung | Standardwert |
|------|-------------|-------------|
| `server.port` | HTTP-Listener-Port | `8080` |
| `postgres.dsn` | PostgreSQL-Verbindungszeichenfolge | `postgres://postgres@localhost:5432/imdb?sslmode=disable` |
| `redis.addr` | Redis-Adresse | `localhost:6379` |
| `jwt.secret` | JWT-Signaturschlüssel (in Produktion ändern) | `change-me-to-a-random-secret` |
| `jwt.expire_hours` | Access-Token-Gültigkeit | `1` Stunde |
| `jwt.refresh_expire_hours` | Refresh-Token-Gültigkeit | `168` Stunden (7 Tage) |
| `ratelimit.msg_per_sec` | Nachrichten-Ratenbegrenzung | `30` Nachrichten/s |
| `smtp.*` | SMTP-E-Mail-Dienst (für Verifizierungscodes) | — |

#### Abhängigkeiten installieren und starten

```bash
# Go-Abhängigkeiten installieren
cd server && go mod download

# Bauen und starten (kompiliert automatisch Web-Frontend + startet Server)
make server

# Nur vorcompiliertes Binary starten
bin/ziziphus -c server/config/config.yaml

# Stoppen
make server-stop
```

Der API-Server lauscht auf `http://localhost:8080`.

### 2. Web-Frontend (Entwicklungsmodus)

```bash
cd server/web
npm install
npm run dev
```

Der Entwicklungs-Server läuft auf `http://localhost:5173`, die API wird an `http://localhost:8080` weitergeleitet.

Produktions-Build:

```bash
cd server/web
npm run build
# Ausgabe wird automatisch nach server/internal/webembed/dist/ kopiert
# Dann Go-Binary compilieren, um das Frontend einzubetten
```

### 3. macOS-Client

```bash
# Sicherstellen, dass lokale Abhängigkeiten installiert sind
# deps/textual/ ist ein lokales Swift-Paket

# macOS-Client bauen und starten
make macos
```

Beim ersten Start wird `Info.plist` automatisch generiert und die App geöffnet. Zum Neuerstellen:

```bash
make macos-stop
make macos
```

### 4. iOS-Client

```bash
# 1) .env bearbeiten, IOS_DEVICE auf Ihren Gerätenamen setzen
# 2) Xcode-Projekt generieren
make xcodegen

# 3) Auf dem Gerät bauen und bereitstellen
make ios-deploy

# Oder client/IMApp.xcodeproj in Xcode öffnen
# IMApp-iOS-Schema auswählen, Zielgerät auswählen, Cmd+R
```

> Dieses Projekt verwendet nur echte Geräte (keine Simulatoren — siehe `CLAUDE.md`).

### 5. Datenbank-Migrationen

Migrationen werden beim Start automatisch ausgeführt (`db.RunMigrations`). Skripte befinden sich in:

```
server/internal/storage/db/migrations/
```

Zur manuellen Ausführung:

```bash
psql -d imdb -f server/internal/storage/db/migrations/001_initial.sql
```

---

## Code-Qualität

```bash
# Web-Frontend Lint
make lint-web

# Go-Backend Lint
make lint-server

# Alle Lints
make lint
```

---

## Bereitstellung

### Ein-Klick-Fernbereitstellung

Bearbeiten Sie die `.env`-Datei:

| Variable | Beschreibung |
|----------|-------------|
| `SSH_HOST` | Serveradresse |
| `SSH_PORT` | SSH-Port |
| `DEPLOY_PORT` | Dienst-Port |
| `DEPLOY_USER` | SSH-Benutzer |
| `DEPLOY_PATH` | Bereitstellungspfad |
| `DEPLOY_DSN` | Produktionsdatenbank-Verbindungszeichenfolge |

Dann ausführen:

```bash
make deploy          # Bauen und auf entfernten Server bereitstellen (systemd-Dienst)
make deploy-status   # Dienststatus prüfen
make deploy-logs     # Dienst-Logs anzeigen
```

---

## Architektur

```
┌─────────────────────────────────────────────────┐
│  Web-Frontend (React + Vite)                     │
│  server/web/                                     │
└──────────────┬──────────────────────────────────┘
               │ HTTP / WebSocket
┌──────────────▼──────────────────────────────────┐
│  Go-Backend (ziziphus)                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ API      │ │ WebSocket │ │ Nachrichten-Route│ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ Sitzung  │ │ Gateway  │ │ Dateispeicher    │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐                       │
│  │PostgreSQL│ │ Redis    │                       │
│  └──────────┘ └──────────┘                       │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  macOS / iOS-Client (Swift + SwiftUI)             │
│  client/                                         │
└─────────────────────────────────────────────────┘
```

---

## Internationalisierung (i18n)

Ziziphus unterstützt **8 Sprachen** auf Frontend und Backend:

| Code | Sprache | Backend-Konstante | Frontend-Datei |
|------|---------|-------------------|----------------|
| `zh` | Chinesisch (Vereinfacht) | `LangZH` | `zh.json` |
| `en` | Englisch | `LangEN` | `en.json` |
| `ja` | Japanisch | `LangJA` | `ja.json` |
| `fr` | Französisch | `LangFR` | `fr.json` |
| `de` | Deutsch | `LangDE` | `de.json` |
| `es` | Spanisch | `LangES` | `es.json` |
| `ko` | Koreanisch | `LangKO` | `ko.json` |
| `ru` | Russisch | `LangRU` | `ru.json` |

### Frontend

Verwendet [i18next](https://www.i18next.com/) mit [react-i18next](https://react.i18next.com/). Die Spracheinstellung wird im `localStorage` unter dem Schlüssel `ziziphus_language` gespeichert. Übersetzungsdateien befinden sich in `server/web/src/i18n/{lang}.json`. Nicht-chinesische Sprachpakete werden bei Bedarf nachgeladen, um die anfängliche Paketgröße klein zu halten.

Das Frontend sendet die ausgewählte Sprache über den `X-Language` HTTP-Header bei jeder Anfrage an das Backend.

### Backend

Das Paket `server/pkg/i18n/` bietet:

- **Sprachkonstanten** (`LangZH`, `LangEN`, ...)
- **ParseLang()** — Akzeptiert Browser-Locale-Codes (z. B. `zh-CN`, `en-US`, `ja-JP`) und normalisiert sie zu einer unterstützten Lang-Konstante
- **DetectLanguage()** — Liest den `X-Language`-Header (Frontend-Einstellung) mit Rückfall auf den `Accept-Language`-Header, dann auf `LangZH`
- **T() / TWithLang()** — Vorlagenbasierte Zeichenkettenübersetzung mit Positionsparametern (`{0}`, `{1}`)
- **HTTP-Middleware** — Erkennt die Sprache pro Anfrage und speichert sie im Anfragekontext

Übersetzungsnachrichten sind nach Sprache in Dateien aufgeteilt:
```
pkg/i18n/messages.go          # Nachrichtenschlüssel-Deklarationen + registerLang-Helfer
pkg/i18n/{zh,en,ja,fr,de,es,ko,ru}.go  # Übersetzungen jeder Sprache
```

### E-Mail-Vorlagen

E-Mail-Verifizierungs- und Passwort-Zurücksetzungs-Vorlagen unterstützen ebenfalls alle 8 Sprachen. Die Vorlagen werden zur Compile-Zeit über `//go:embed` eingebettet und befinden sich in:

```
internal/auth/email_templates/
  verify_code_{lang}.html
  reset_password_{lang}.html
```

### Hinzufügen einer Neuen Sprache

**Backend:**
1. Neue `LangXX`-Konstante in `pkg/i18n/i18n.go` hinzufügen
2. Locale-Zuordnung in `ParseLang()` hinzufügen
3. Neue Datei `pkg/i18n/{lang}.go` mit `init() + registerLang()` für alle Nachrichtenschlüssel erstellen
4. `langToFrontendCode()`-Zuordnung in `internal/api/language.go` hinzufügen

**Frontend:**
1. `server/web/src/i18n/{lang}.json` mit übersetzten Schlüssel-Wert-Paaren erstellen
2. Sprachoption in den Frontend-Einstellungen und im Authentifizierungs-Sprachselektor hinzufügen
3. `Language`-Typ und `resolveAutoLang()` im UI-Store aktualisieren

**E-Mail-Vorlagen:**
1. Vorhandene Vorlage kopieren (z. B. `verify_code_en.html` → `verify_code_{lang}.html`)
2. Textinhalt übersetzen
3. `//go:embed`-Direktive hinzufügen und in der `emailTemplates`-Map in `internal/auth/mailer.go` registrieren
4. Betreff-Übersetzungen hinzufügen

---

## Umgebungsvariablen

Siehe `.env.example`-Dateien:

- `server/config/config.example.yaml` — Backend-Konfigurationsvorlage
- `server/web/.env.example` — Web-Frontend-Umgebungsvariablenvorlage
- Root `.env` — Bereitstellungsparameter (in `.gitignore`, wird nicht commitet)
