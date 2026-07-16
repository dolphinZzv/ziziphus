# Configuration Reference

Configuration is managed via [Viper](https://github.com/spf13/viper), supporting:

- **YAML file** (default: `config/config.yaml`)
- **Environment variables** (e.g. `JWT_SECRET` overrides `jwt.secret`)
- **File watching** — `viper.WatchConfig()` auto-reloads on file change
- **SIGHUP** — `kill -HUP <pid>` triggers a manual reload

## Hot-reload

Config sections marked **Hot-reload: ✅** can be changed at runtime without restarting the server.
Sections marked **❌** require a full restart — they affect startup-time initialization
(connection pools, crypto keys, etc.).

---

## `server`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `port` | int | `8080` | HTTP listen port | ❌ |
| `allow_registration` | bool | `true` | Allow new account registration | ✅ |

---

## `postgres`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `dsn` | string | — | PostgreSQL connection DSN | ❌ |
| `max_conns` | int | `20` | Max connections in pool | ❌ |
| `migrations` | string | `internal/storage/db/migrations` | Path to SQL migration files | ❌ |

---

## `redis`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `addr` | string | — | Redis address (`host:port`) | ❌ |
| `password` | string | `""` | Redis password | ❌ |
| `db` | int | `0` | Redis database index | ❌ |

---

## `jwt`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `secret` | string | — | JWT signing key (≥ 32 chars). Can be overridden by **`JWT_SECRET`** env var | ❌ |
| `expire_hours` | int | `1` | Access token lifetime in hours | ❌ |
| `refresh_expire_hours` | int | `168` (7 days) | Refresh token lifetime in hours | ❌ |

---

## `snowflake`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `worker_id` | int | — | Unique worker ID for snowflake ID generation | ❌ |
| `start_time` | string (RFC3339) | — | Epoch start for snowflake timestamps | ❌ |

---

## `ratelimit`

### WebSocket rate limiter

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `msg_per_sec` | int | `30` | Max WebSocket messages per second per user | ✅ |
| `max_body_bytes` | int | `10240` (10 KB) | Max WebSocket message body size | ✅ |
| `burst_size` | int | `50` | Initial burst allowance for WebSocket | ✅ |

### HTTP API rate limiters

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `api_enabled` | bool | `true` | Enable all HTTP rate limiters. Set `false` during E2E tests | ✅ |
| `login_attempts` | int | `10` | Failed login attempts before lockout | ✅ |
| `login_window_min` | int | `1` | Window (minutes) for counting failures | ✅ |
| `login_lockout_min` | int | `5` | Lockout duration (minutes) after max failures | ✅ |
| `reg_per_window` | int | `5` | Max registrations per IP per window | ✅ |
| `reg_window_hour` | int | `1` | Registration counting window (hours) | ✅ |
| `global_rate` | int | `100` | Global per-IP rate limit (req/s) | ✅ |
| `global_burst` | int | `200` | Global per-IP burst allowance | ✅ |

---

## `storage`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `local_path` | string | `data/files` | Local filesystem path for file storage | ❌ |
| `base_url` | string | `/files` | URL prefix for file serving | ❌ |

---

## `smtp`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `host` | string | `""` | SMTP server hostname | ✅ |
| `port` | string | `587` | SMTP server port | ✅ |
| `user` | string | `""` | SMTP username | ✅ |
| `password` | string | `""` | SMTP password | ✅ |
| `from` | string | (same as `user`) | From address for outgoing emails | ✅ |

> SMTP settings use `atomic.Pointer` — the Mailer reads config atomically on every
> `send()` call. Changes via file watch or SIGHUP take effect immediately for new
> email sends. Existing SMTP connections are unaffected (each send opens a new connection).

---

## `announcement`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `enabled` | bool | `false` | Show global announcement to all users | ✅ |
| `title` | string | `""` | Announcement title | ✅ |
| `body` | string | `""` | Announcement body text | ✅ |
| `url` | string | `""` | Optional link URL | ✅ |

> The `/api/v1/announcement` endpoint reads directly from the config manager,
> so changes are reflected immediately without restart.

---

## `log`

| Key | Type | Default | Description | Hot-reload |
|-----|------|---------|-------------|:----------:|
| `level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` | ✅ |
| `file` | string | `""` | Log file path (empty = stdout only) | ❌ |
| `max_size` | int | `100` | Log rotation size in MB | ❌ |
| `max_age` | int | `7` | Days to retain old log files | ❌ |
| `max_backups` | int | `10` | Number of old log files to keep | ❌ |
| `compress` | bool | `true` | Compress rotated log files | ❌ |

> Only `level` supports hot-reload via `logger.SetLevel()`.
> File output settings require a restart because lumberjack is initialized once at startup.

---

## Environment variables

| Variable | Overrides | Description |
|----------|-----------|-------------|
| `JWT_SECRET` | `jwt.secret` | JWT signing key (takes precedence over YAML) |
