# Ziziphus

[![CI](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg)](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml) [![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://47.95.200.101:10011/)

> **Démo**: [http://47.95.200.101:10011/](http://47.95.200.101:10011/)

[Français](README.fr.md) | [English](../README.md) | [中文](README.zh.md) | [日本語](README.ja.md) | [Deutsch](README.de.md) | [Español](README.es.md) | [한국어](README.ko.md) | [Русский](README.ru.md)

Application de messagerie instantanée (IM) — un backend Go alimentant plusieurs frontends.

**Clients pris en charge, par priorité :**
1. 🌐 **React Web** — SPA complète
2. 🖥 **macOS** — application native SwiftUI
3. 📱 **iOS** — application native SwiftUI
4. 🤖 **Android** — à venir

## Structure du projet

| Répertoire | Description |
|------------|-------------|
| `server/` | Backend Go (API REST + WebSocket) |
| `server/web/` | Frontend Web (React + TypeScript + Vite) |
| `client/` | Client macOS / iOS (Swift + SwiftUI) |
| `deps/` | Dépendances locales |
| `bin/` | Artefacts de build |

## Stack Technique

- **Backend** : Go 1.26, PostgreSQL, Redis, JWT, WebSocket
- **Frontend Web** : React 19, TypeScript, Vite, Tailwind CSS 4, Zustand
- **Client natif** : Swift 6.3.2, SwiftUI, iOS 18+ / macOS 15+

---

## Installation et Exécution

### Prérequis

- Go 1.26+
- Node.js 22+
- Swift 6.3.2+ (client macOS)
- Xcode 16+ (client iOS)
- PostgreSQL 16+
- Redis 7+
- Docker (optionnel)

### Déploiement Docker (Recommandé)

#### Démarrage en un clic avec Docker Compose

```bash
# 1. Préparer le fichier de configuration
cp server/config/config.example.yaml server/config/config.yaml
# Modifier config.yaml selon vos besoins

# 2. Démarrer tous les services (PostgreSQL + Redis + Application)
docker compose up -d

# 3. Voir les logs
docker compose logs -f app

# 4. Arrêter
docker compose down
```

Ceci démarre trois conteneurs :

| Service | Image | Port |
|---------|-------|------|
| postgres | `postgres:16-alpine` | 5432 |
| redis | `redis:7-alpine` | 6379 |
| app | build local | 8080 |

Les données persistantes sont stockées dans des volumes Docker.

#### Utilisation de PostgreSQL / Redis Externes

Modifiez `server/config/config.yaml` :

```yaml
postgres:
  dsn: "postgres://user:pass@your-pg-host:5432/imdb?sslmode=require"

redis:
  addr: "your-redis-host:6379"
  password: "your-password"
```

Puis démarrez uniquement le conteneur de l'application :

```bash
docker compose up -d app
```

#### Construire l'Image Uniquement

```bash
# Build complet (Frontend Web + Backend Go)
docker build -t ziziphus:latest .

# Backend Go uniquement (nécessite npm run build au préalable)
docker build -t ziziphus:latest -f server/Dockerfile server/
```

#### Exécuter le Conteneur Manuellement

```bash
docker run -d \
  --name ziziphus \
  -p 8080:8080 \
  -v ./server/config/config.yaml:/app/config/config.yaml:ro \
  ziziphus:latest
```

#### Registre d'Images

À chaque push sur main, le CI construit et pousse l'image vers GitHub Container Registry :

```bash
docker pull ghcr.io/dolphinZzv/ziziphus:latest
```

### 1. Backend (Exécution depuis les Sources)

#### Configuration

```bash
# Copier le fichier de configuration
cp server/config/config.example.yaml server/config/config.yaml
# Modifier config.yaml — configurez au minimum PostgreSQL DSN et JWT secret
```

Configuration clé :

| Champ | Description | Valeur par défaut |
|-------|-------------|-------------------|
| `server.port` | Port d'écoute HTTP | `8080` |
| `postgres.dsn` | Chaîne de connexion PostgreSQL | `postgres://postgres@localhost:5432/imdb?sslmode=disable` |
| `redis.addr` | Adresse Redis | `localhost:6379` |
| `jwt.secret` | Clé de signature JWT (à changer en production) | `change-me-to-a-random-secret` |
| `jwt.expire_hours` | Durée de validité du token d'accès | `1` heure |
| `jwt.refresh_expire_hours` | Durée de validité du refresh token | `168` heures (7 jours) |
| `ratelimit.msg_per_sec` | Limite de débit des messages | `30` msg/s |
| `smtp.*` | Service email SMTP (pour les codes de vérification) | — |

#### Installer les Dépendances et Lancer

```bash
# Installer les dépendances Go
cd server && go mod download

# Construire et lancer (compile automatiquement le frontend Web + démarre le serveur)
make server

# Lancer uniquement le binaire pré-construit
bin/ziziphus -c server/config/config.yaml

# Arrêter
make server-stop
```

Le serveur API écoute sur `http://localhost:8080`.

### 2. Frontend Web (Mode Développement)

```bash
cd server/web
npm install
npm run dev
```

Le serveur de développement est à `http://localhost:5173`, l'API est proxyfiée vers `http://localhost:8080`.

Build de production :

```bash
cd server/web
npm run build
# La sortie est automatiquement copiée dans server/internal/webembed/dist/
# Compilez ensuite le binaire Go pour intégrer le frontend
```

### 3. Client macOS

```bash
# Assurez-vous que les dépendances locales sont installées
# deps/textual/ est un package Swift local

# Construire et lancer le client macOS
make macos
```

La première exécution génère automatiquement `Info.plist` et ouvre l'application. Pour reconstruire :

```bash
make macos-stop
make macos
```

### 4. Client iOS

```bash
# 1) Modifiez .env, définissez IOS_DEVICE sur le nom de votre appareil
# 2) Générez le projet Xcode
make xcodegen

# 3) Construisez et déployez sur l'appareil
make ios-deploy

# Ou ouvrez client/IMApp.xcodeproj dans Xcode
# Sélectionnez le schéma IMApp-iOS, ciblez votre appareil physique, Cmd+R
```

> Ce projet utilise uniquement des appareils réels (pas de simulateurs — voir `CLAUDE.md`).

### 5. Migrations de Base de Données

Les migrations s'exécutent automatiquement au démarrage (`db.RunMigrations`). Les scripts se trouvent dans :

```
server/internal/storage/db/migrations/
```

Pour exécuter manuellement :

```bash
psql -d imdb -f server/internal/storage/db/migrations/001_initial.sql
```

---

## Qualité du Code

```bash
# Lint du frontend Web
make lint-web

# Lint du backend Go
make lint-server

# Tous les lints
make lint
```

---

## Déploiement

### Déploiement à Distance en Un Clic

Modifiez le fichier `.env` :

| Variable | Description |
|----------|-------------|
| `SSH_HOST` | Adresse du serveur |
| `SSH_PORT` | Port SSH |
| `DEPLOY_PORT` | Port du service |
| `DEPLOY_USER` | Utilisateur SSH |
| `DEPLOY_PATH` | Chemin de déploiement |
| `DEPLOY_DSN` | Chaîne de connexion à la base de production |

Puis exécutez :

```bash
make deploy          # Construire et déployer sur le serveur distant (service systemd)
make deploy-status   # Vérifier le statut du service
make deploy-logs     # Voir les logs du service
```

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Frontend Web (React + Vite)                     │
│  server/web/                                     │
└──────────────┬──────────────────────────────────┘
               │ HTTP / WebSocket
┌──────────────▼──────────────────────────────────┐
│  Backend Go (ziziphus)                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ API      │ │ WebSocket │ │ Routage Messages │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ Session  │ │ Passerelle│ │ Stockage Fichiers│ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐                       │
│  │PostgreSQL│ │ Redis    │                       │
│  └──────────┘ └──────────┘                       │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  Client macOS / iOS (Swift + SwiftUI)             │
│  client/                                         │
└─────────────────────────────────────────────────┘
```

---

## Internationalisation (i18n)

Ziziphus prend en charge **8 langues** sur le frontend et le backend :

| Code | Langue | Constante Backend | Fichier Frontend |
|------|--------|-------------------|------------------|
| `zh` | Chinois simplifié | `LangZH` | `zh.json` |
| `en` | Anglais | `LangEN` | `en.json` |
| `ja` | Japonais | `LangJA` | `ja.json` |
| `fr` | Français | `LangFR` | `fr.json` |
| `de` | Allemand | `LangDE` | `de.json` |
| `es` | Espagnol | `LangES` | `es.json` |
| `ko` | Coréen | `LangKO` | `ko.json` |
| `ru` | Russe | `LangRU` | `ru.json` |

### Frontend

Utilise [i18next](https://www.i18next.com/) avec [react-i18next](https://react.i18next.com/). La préférence linguistique est stockée dans `localStorage` sous la clé `ziziphus_language`. Les fichiers de traduction se trouvent dans `server/web/src/i18n/{lang}.json`. Les bundles autres que le chinois sont chargés à la demande pour garder le bundle initial léger.

Le frontend envoie la langue sélectionnée au backend via l'en-tête HTTP `X-Language` sur chaque requête.

### Backend

Le package `server/pkg/i18n/` fournit :

- **Constantes de langue** (`LangZH`, `LangEN`, ...)
- **ParseLang()** — Accepte les codes de locale du navigateur (ex. `zh-CN`, `en-US`, `ja-JP`) et les normalise en une constante Lang prise en charge
- **DetectLanguage()** — Lit l'en-tête `X-Language` (préférence du frontend) avec repli sur l'en-tête `Accept-Language`, puis sur `LangZH`
- **T() / TWithLang()** — Traduction de chaînes avec paramètres positionnels (`{0}`, `{1}`)
- **Middleware HTTP** — Détecte la langue par requête et la stocke dans le contexte

Les messages de traduction sont répartis par fichier de langue :
```
pkg/i18n/messages.go          # Déclarations des clés de message + helper registerLang
pkg/i18n/{zh,en,ja,fr,de,es,ko,ru}.go  # Traductions de chaque langue
```

### Modèles d'Email

Les modèles d'email de vérification et de réinitialisation de mot de passe prennent également en charge les 8 langues. Les modèles sont intégrés à la compilation via `//go:embed` et se trouvent dans :

```
internal/auth/email_templates/
  verify_code_{lang}.html
  reset_password_{lang}.html
```

### Ajouter une Nouvelle Langue

**Backend :**
1. Ajouter une nouvelle constante `LangXX` dans `pkg/i18n/i18n.go`
2. Ajouter le mappage de locale dans `ParseLang()`
3. Créer `pkg/i18n/{lang}.go` avec `init() + registerLang()` pour toutes les clés de message
4. Ajouter le mappage `langToFrontendCode()` dans `internal/api/language.go`

**Frontend :**
1. Créer `server/web/src/i18n/{lang}.json` avec les paires clé-valeur traduites
2. Ajouter l'option de langue dans le sélecteur de langue des paramètres et de l'authentification
3. Mettre à jour le type `Language` et `resolveAutoLang()` dans le store UI

**Modèles d'Email :**
1. Copier un modèle existant (ex. `verify_code_en.html` → `verify_code_{lang}.html`)
2. Traduire le contenu textuel
3. Ajouter la directive `//go:embed` et enregistrer dans la map `emailTemplates` dans `internal/auth/mailer.go`
4. Ajouter les traductions des sujets

---

## Variables d'Environnement

Voir les fichiers `.env.example` :

- `server/config/config.example.yaml` — Modèle de configuration backend
- `server/web/.env.example` — Modèle de variables d'environnement frontend Web
- `.env` à la racine — Paramètres de déploiement (dans `.gitignore`, non commité)
