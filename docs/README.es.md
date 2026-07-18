# Ziziphus

[![CI](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg)](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml) [![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://47.95.200.101:10011/)

> **Demo**: [http://47.95.200.101:10011/](http://47.95.200.101:10011/)

[Español](README.es.md) | [English](../README.md) | [中文](README.zh.md) | [日本語](README.ja.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [한국어](README.ko.md) | [Русский](README.ru.md)

Una aplicación de mensajería instantánea (IM) — backend Go que impulsa múltiples frontends.

**Clientes compatibles, por prioridad:**
1. 🌐 **React Web** — SPA con todas las funciones
2. 🖥 **macOS** — aplicación nativa SwiftUI
3. 📱 **iOS** — aplicación nativa SwiftUI
4. 🤖 **Android** — próximamente

## Estructura del Proyecto

| Directorio | Descripción |
|------------|-------------|
| `server/` | Backend Go (API REST + WebSocket) |
| `server/web/` | Frontend Web (React + TypeScript + Vite) |
| `client/` | Cliente macOS / iOS (Swift + SwiftUI) |
| `deps/` | Dependencias locales |
| `bin/` | Artefactos de compilación |

## Stack Tecnológico

- **Backend**: Go 1.26, PostgreSQL, Redis, JWT, WebSocket
- **Frontend Web**: React 19, TypeScript, Vite, Tailwind CSS 4, Zustand
- **Cliente Nativo**: Swift 6.3.2, SwiftUI, iOS 18+ / macOS 15+

---

## Instalación y Ejecución

### Requisitos Previos

- Go 1.26+
- Node.js 22+
- Swift 6.3.2+ (cliente macOS)
- Xcode 16+ (cliente iOS)
- PostgreSQL 16+
- Redis 7+
- Docker (opcional)

### Despliegue con Docker (Recomendado)

#### Inicio con un solo clic usando Docker Compose

```bash
# 1. Preparar archivo de configuración
cp server/config/config.example.yaml server/config/config.yaml
# Edite config.yaml según sea necesario

# 2. Iniciar todos los servicios (PostgreSQL + Redis + Aplicación)
docker compose up -d

# 3. Ver registros
docker compose logs -f app

# 4. Detener
docker compose down
```

Esto inicia tres contenedores:

| Servicio | Imagen | Puerto |
|----------|--------|--------|
| postgres | `postgres:16-alpine` | 5432 |
| redis | `redis:7-alpine` | 6379 |
| app | compilación local | 8080 |

Los datos persistentes se almacenan en volúmenes Docker.

#### Usando PostgreSQL / Redis Externos

Edite `server/config/config.yaml`:

```yaml
postgres:
  dsn: "postgres://user:pass@your-pg-host:5432/imdb?sslmode=require"

redis:
  addr: "your-redis-host:6379"
  password: "your-password"
```

Luego inicie solo el contenedor de la aplicación:

```bash
docker compose up -d app
```

#### Solo Construir la Imagen

```bash
# Compilación completa (Frontend Web + Backend Go)
docker build -t ziziphus:latest .

# Solo Backend Go (requiere npm run build primero)
docker build -t ziziphus:latest -f server/Dockerfile server/
```

#### Ejecutar el Contenedor Manualmente

```bash
docker run -d \
  --name ziziphus \
  -p 8080:8080 \
  -v ./server/config/config.yaml:/app/config/config.yaml:ro \
  ziziphus:latest
```

#### Registro de Imágenes

En cada push a main, CI construye y envía la imagen a GitHub Container Registry:

```bash
docker pull ghcr.io/dolphinZzv/ziziphus:latest
```

### 1. Backend (Ejecución desde Código Fuente)

#### Configuración

```bash
# Copiar archivo de configuración
cp server/config/config.example.yaml server/config/config.yaml
# Edite config.yaml — configure al menos PostgreSQL DSN y JWT secret
```

Configuración clave:

| Campo | Descripción | Valor por defecto |
|-------|-------------|-------------------|
| `server.port` | Puerto de escucha HTTP | `8080` |
| `postgres.dsn` | Cadena de conexión PostgreSQL | `postgres://postgres@localhost:5432/imdb?sslmode=disable` |
| `redis.addr` | Dirección de Redis | `localhost:6379` |
| `jwt.secret` | Clave de firma JWT (cambiar en producción) | `change-me-to-a-random-secret` |
| `jwt.expire_hours` | Duración del token de acceso | `1` hora |
| `jwt.refresh_expire_hours` | Duración del token de actualización | `168` horas (7 días) |
| `ratelimit.msg_per_sec` | Límite de velocidad de mensajes | `30` msg/s |
| `smtp.*` | Servicio de correo SMTP (para códigos de verificación) | — |

#### Instalar Dependencias y Ejecutar

```bash
# Instalar dependencias Go
cd server && go mod download

# Compilar e iniciar (compila automáticamente el frontend Web + inicia el servidor)
make server

# Iniciar solo el binario precompilado
bin/ziziphus -c server/config/config.yaml

# Detener
make server-stop
```

El servidor API escucha en `http://localhost:8080`.

### 2. Frontend Web (Modo Desarrollo)

```bash
cd server/web
npm install
npm run dev
```

El servidor de desarrollo se inicia en `http://localhost:5173`, la API se proxy a `http://localhost:8080`.

Compilación para producción:

```bash
cd server/web
npm run build
# La salida se copia automáticamente a server/internal/webembed/dist/
# Luego compile el binario Go para incrustar el frontend
```

### 3. Cliente macOS

```bash
# Asegúrese de que las dependencias locales estén instaladas
# deps/textual/ es un paquete Swift local

# Compilar e iniciar el cliente macOS
make macos
```

La primera ejecución genera automáticamente `Info.plist` y abre la aplicación. Para recompilar:

```bash
make macos-stop
make macos
```

### 4. Cliente iOS

```bash
# 1) Edite .env, establezca IOS_DEVICE con el nombre de su dispositivo
# 2) Genere el proyecto Xcode
make xcodegen

# 3) Compile e implemente en el dispositivo
make ios-deploy

# O abra client/IMApp.xcodeproj en Xcode
# Seleccione el esquema IMApp-iOS, seleccione su dispositivo físico, Cmd+R
```

> Este proyecto utiliza solo dispositivos reales (sin simuladores — consulte `CLAUDE.md`).

### 5. Migraciones de Base de Datos

Las migraciones se ejecutan automáticamente al iniciar (`db.RunMigrations`). Los scripts están en:

```
server/internal/storage/db/migrations/
```

Para ejecutar manualmente:

```bash
psql -d imdb -f server/internal/storage/db/migrations/001_initial.sql
```

---

## Calidad del Código

```bash
# Lint del frontend Web
make lint-web

# Lint del backend Go
make lint-server

# Todos los lints
make lint
```

---

## Despliegue

### Despliegue Remoto con un Solo Clic

Edite el archivo `.env`:

| Variable | Descripción |
|----------|-------------|
| `SSH_HOST` | Dirección del servidor |
| `SSH_PORT` | Puerto SSH |
| `DEPLOY_PORT` | Puerto del servicio |
| `DEPLOY_USER` | Usuario SSH |
| `DEPLOY_PATH` | Ruta de despliegue |
| `DEPLOY_DSN` | Cadena de conexión a la base de datos de producción |

Luego ejecute:

```bash
make deploy          # Compilar e implementar en servidor remoto (servicio systemd)
make deploy-status   # Verificar estado del servicio
make deploy-logs     | Ver registros del servicio
```

---

## Arquitectura

```
┌─────────────────────────────────────────────────┐
│  Frontend Web (React + Vite)                     │
│  server/web/                                     │
└──────────────┬──────────────────────────────────┘
               │ HTTP / WebSocket
┌──────────────▼──────────────────────────────────┐
│  Backend Go (ziziphus)                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ API      │ │ WebSocket │ │ Enrutamiento Msg │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ Sesión   │ │ Pasarela  │ │ Almacenamiento   │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐                       │
│  │PostgreSQL│ │ Redis    │                       │
│  └──────────┘ └──────────┘                       │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  Cliente macOS / iOS (Swift + SwiftUI)            │
│  client/                                         │
└─────────────────────────────────────────────────┘
```

---

## Internacionalización (i18n)

Ziziphus admite **8 idiomas** tanto en el frontend como en el backend:

| Código | Idioma | Constante Backend | Archivo Frontend |
|--------|--------|-------------------|------------------|
| `zh` | Chino simplificado | `LangZH` | `zh.json` |
| `en` | Inglés | `LangEN` | `en.json` |
| `ja` | Japonés | `LangJA` | `ja.json` |
| `fr` | Francés | `LangFR` | `fr.json` |
| `de` | Alemán | `LangDE` | `de.json` |
| `es` | Español | `LangES` | `es.json` |
| `ko` | Coreano | `LangKO` | `ko.json` |
| `ru` | Ruso | `LangRU` | `ru.json` |

### Frontend

Utiliza [i18next](https://www.i18next.com/) con [react-i18next](https://react.i18next.com/). La preferencia de idioma se almacena en `localStorage` con la clave `ziziphus_language`. Los archivos de traducción se encuentran en `server/web/src/i18n/{lang}.json`. Los paquetes que no son chino se cargan bajo demanda para mantener pequeño el paquete inicial.

El frontend envía el idioma seleccionado al backend mediante el encabezado HTTP `X-Language` en cada solicitud.

### Backend

El paquete `server/pkg/i18n/` proporciona:

- **Constantes de idioma** (`LangZH`, `LangEN`, ...)
- **ParseLang()** — Acepta códigos de configuración regional del navegador (ej. `zh-CN`, `en-US`, `ja-JP`) y los normaliza a una constante Lang compatible
- **DetectLanguage()** — Lee el encabezado `X-Language` (preferencia del frontend) con respaldo al encabezado `Accept-Language`, luego a `LangZH`
- **T() / TWithLang()** — Traducción de cadenas con parámetros posicionales (`{0}`, `{1}`)
- **Middleware HTTP** — Detecta el idioma por solicitud y lo almacena en el contexto de la solicitud

Los mensajes de traducción están divididos por archivo de idioma:
```
pkg/i18n/messages.go          # Declaraciones de claves de mensaje + helper registerLang
pkg/i18n/{zh,en,ja,fr,de,es,ko,ru}.go  # Traducciones de cada idioma
```

### Plantillas de Correo Electrónico

Las plantillas de verificación de correo electrónico y restablecimiento de contraseña también admiten los 8 idiomas. Las plantillas se incrustan en tiempo de compilación mediante `//go:embed` y se encuentran en:

```
internal/auth/email_templates/
  verify_code_{lang}.html
  reset_password_{lang}.html
```

### Agregar un Nuevo Idioma

**Backend:**
1. Agregar una nueva constante `LangXX` en `pkg/i18n/i18n.go`
2. Agregar la asignación de configuración regional en `ParseLang()`
3. Crear `pkg/i18n/{lang}.go` con `init() + registerLang()` para todas las claves de mensaje
4. Agregar la asignación `langToFrontendCode()` en `internal/api/language.go`

**Frontend:**
1. Crear `server/web/src/i18n/{lang}.json` con pares clave-valor traducidos
2. Agregar la opción de idioma en el selector de idioma de configuración y autenticación
3. Actualizar el tipo `Language` y `resolveAutoLang()` en el store de la UI

**Plantillas de Correo:**
1. Copiar una plantilla existente (ej. `verify_code_en.html` → `verify_code_{lang}.html`)
2. Traducir el contenido del texto
3. Agregar la directiva `//go:embed` y registrar en el mapa `emailTemplates` en `internal/auth/mailer.go`
4. Agregar traducciones de asuntos

---

## Variables de Entorno

Consulte los archivos `.env.example`:

- `server/config/config.example.yaml` — Plantilla de configuración del backend
- `server/web/.env.example` — Plantilla de variables de entorno del frontend Web
- `.env` raíz — Parámetros de despliegue (en `.gitignore`, no se commitea)
