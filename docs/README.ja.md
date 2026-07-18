# Ziziphus

[![CI](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml/badge.svg)](https://github.com/dolphinZzv/ziziphus/actions/workflows/ci.yml) [![Demo](https://img.shields.io/badge/demo-online-brightgreen)](http://47.95.200.101:10011/)

> **デモ**: [http://47.95.200.101:10011/](http://47.95.200.101:10011/)

[日本語](README.ja.md) | [English](../README.md) | [中文](README.zh.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Español](README.es.md) | [한국어](README.ko.md) | [Русский](README.ru.md)

Go バックエンドが複数フロントエンドを駆動するインスタントメッセージング（IM）アプリケーション。

**対応クライアント（優先順位順）：**
1. 🌐 **React Web** — フル機能 SPA
2. 🖥 **macOS** — ネイティブ SwiftUI アプリ
3. 📱 **iOS** — ネイティブ SwiftUI アプリ
4. 🤖 **Android** — 近日対応予定

## プロジェクト構造

| ディレクトリ | 説明 |
|-------------|------|
| `server/` | Go バックエンド（REST API + WebSocket） |
| `server/web/` | Web フロントエンド（React + TypeScript + Vite） |
| `client/` | macOS / iOS クライアント（Swift + SwiftUI） |
| `deps/` | ローカル依存関係 |
| `bin/` | ビルド成果物 |

## 技術スタック

- **バックエンド**: Go 1.26, PostgreSQL, Redis, JWT, WebSocket
- **Web フロントエンド**: React 19, TypeScript, Vite, Tailwind CSS 4, Zustand
- **ネイティブクライアント**: Swift 6.3.2, SwiftUI, iOS 18+ / macOS 15+

---

## インストールと実行

### 前提条件

- Go 1.26+
- Node.js 22+
- Swift 6.3.2+（macOS クライアント）
- Xcode 16+（iOS クライアント）
- PostgreSQL 16+
- Redis 7+
- Docker（オプション）

### Docker デプロイ（推奨）

#### Docker Compose でワンクリック起動

```bash
# 1. 設定ファイルを準備
cp server/config/config.example.yaml server/config/config.yaml
# 必要に応じて config.yaml を編集

# 2. 全サービスを起動（PostgreSQL + Redis + アプリ）
docker compose up -d

# 3. ログを表示
docker compose logs -f app

# 4. 停止
docker compose down
```

3 つのコンテナが起動します：

| サービス | イメージ | ポート |
|---------|----------|-------|
| postgres | `postgres:16-alpine` | 5432 |
| redis | `redis:7-alpine` | 6379 |
| app | ローカルビルド | 8080 |

データは Docker ボリュームに永続化され、再起動後も保持されます。

#### 外部 PostgreSQL / Redis を使用する場合

`server/config/config.yaml` を編集します：

```yaml
postgres:
  dsn: "postgres://user:pass@your-pg-host:5432/imdb?sslmode=require"

redis:
  addr: "your-redis-host:6379"
  password: "your-password"
```

その後、アプリコンテナのみを起動します：

```bash
docker compose up -d app
```

#### イメージのみビルド（起動しない）

```bash
# フルビルド（Web フロントエンド + Go バックエンド）
docker build -t ziziphus:latest .

# Go バックエンドのみ（事前に npm run build が必要）
docker build -t ziziphus:latest -f server/Dockerfile server/
```

#### 手動でコンテナを実行

```bash
docker run -d \
  --name ziziphus \
  -p 8080:8080 \
  -v ./server/config/config.yaml:/app/config/config.yaml:ro \
  ziziphus:latest
```

#### イメージレジストリ

main ブランチにプッシュするたびに、CI がイメージをビルドして GitHub Container Registry にプッシュします：

```bash
docker pull ghcr.io/dolphinZzv/ziziphus:latest
```

### 1. バックエンド（ソースコード実行）

#### 設定

```bash
# 設定ファイルをコピー
cp server/config/config.example.yaml server/config/config.yaml
# config.yaml を編集 — PostgreSQL DSN と JWT secret は最低限設定してください
```

主要設定項目：

| フィールド | 説明 | デフォルト値 |
|-----------|------|------------|
| `server.port` | HTTP リスニングポート | `8080` |
| `postgres.dsn` | PostgreSQL 接続文字列 | `postgres://postgres@localhost:5432/imdb?sslmode=disable` |
| `redis.addr` | Redis アドレス | `localhost:6379` |
| `jwt.secret` | JWT 署名キー（本番環境では変更してください） | `change-me-to-a-random-secret` |
| `jwt.expire_hours` | アクセストークンの有効期限 | `1` 時間 |
| `jwt.refresh_expire_hours` | リフレッシュトークンの有効期限 | `168` 時間（7 日間） |
| `ratelimit.msg_per_sec` | メッセージ送信レート制限 | `30` メッセージ/秒 |
| `smtp.*` | SMTP メールサービス（認証コード送信用） | — |

#### 依存関係のインストールと起動

```bash
# Go 依存関係のインストール
cd server && go mod download

# ビルドして起動（Web フロントエンドの自動コンパイル + サーバー起動）
make server

# ビルド済みバイナリのみ起動
bin/ziziphus -c server/config/config.yaml

# 停止
make server-stop
```

起動後、API サーバーは `http://localhost:8080` で待機します。

### 2. Web フロントエンド（開発モード）

```bash
cd server/web
npm install
npm run dev
```

開発サーバーは `http://localhost:5173` で起動し、API は `http://localhost:8080` にプロキシされます。

本番ビルド：

```bash
cd server/web
npm run build
# 出力は server/internal/webembed/dist/ に自動コピーされます
# その後 Go バイナリをコンパイルしてフロントエンドを埋め込みます
```

### 3. macOS クライアント

```bash
# ローカル依存関係がインストールされていることを確認
# deps/textual/ はローカルの Swift パッケージです

# macOS クライアントをビルドして起動
make macos
```

初回実行時に自動的に `Info.plist` が生成され、アプリが開きます。再ビルドする場合：

```bash
make macos-stop
make macos
```

### 4. iOS クライアント

```bash
# 1) .env ファイルを編集し、IOS_DEVICE を実際のデバイス名に設定
# 2) Xcode プロジェクトを生成
make xcodegen

# 3) 実機にビルドしてデプロイ
make ios-deploy

# または client/IMApp.xcodeproj を Xcode で開く
# IMApp-iOS スキームを選択し、実機をターゲットにして Cmd+R で実行
```

> このプロジェクトはシミュレーターを使用せず、実機のみでデプロイします（`CLAUDE.md` の取り決め）。

### 5. データベースマイグレーション

マイグレーションはサーバー起動時に自動実行されます（`db.RunMigrations`）。スクリプトの場所：

```
server/internal/storage/db/migrations/
```

手動で実行する場合：

```bash
psql -d imdb -f server/internal/storage/db/migrations/001_initial.sql
```

---

## コード品質

```bash
# Web フロントエンドの lint
make lint-web

# Go バックエンドの lint
make lint-server

# すべての lint
make lint
```

---

## デプロイ

### ワンクリックリモートデプロイ

`.env` ファイルを編集します：

| 変数 | 説明 |
|------|------|
| `SSH_HOST` | サーバーアドレス |
| `SSH_PORT` | SSH ポート |
| `DEPLOY_PORT` | サービスポート |
| `DEPLOY_USER` | SSH ユーザー |
| `DEPLOY_PATH` | デプロイパス |
| `DEPLOY_DSN` | 本番データベース接続文字列 |

その後実行：

```bash
make deploy          # ビルドしてリモートサーバーにデプロイ（systemd サービス）
make deploy-status   # サービス状態を確認
make deploy-logs     # サービスログを表示
```

---

## アーキテクチャ

```
┌─────────────────────────────────────────────────┐
│  Web フロントエンド (React + Vite)                │
│  server/web/                                     │
└──────────────┬──────────────────────────────────┘
               │ HTTP / WebSocket
┌──────────────▼──────────────────────────────────┐
│  Go バックエンド (ziziphus)                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ API 層   │ │ WebSocket │ │ メッセージルーティング│
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ セッション │ │ ゲートウェイ│ │ ファイルストレージ│ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
│  ┌──────────┐ ┌──────────┐                       │
│  │ PostgreSQL│ │ Redis    │                       │
│  └──────────┘ └──────────┘                       │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│  macOS / iOS クライアント (Swift + SwiftUI)        │
│  client/                                         │
└─────────────────────────────────────────────────┘
```

---

## 国際化 (i18n)

Ziziphus はフロントエンドとバックエンドで **8 言語** をサポートしています：

| コード | 言語 | バックエンド定数 | フロントエンドファイル |
|-------|------|----------------|---------------------|
| `zh` | 簡体字中国語 | `LangZH` | `zh.json` |
| `en` | 英語 | `LangEN` | `en.json` |
| `ja` | 日本語 | `LangJA` | `ja.json` |
| `fr` | フランス語 | `LangFR` | `fr.json` |
| `de` | ドイツ語 | `LangDE` | `de.json` |
| `es` | スペイン語 | `LangES` | `es.json` |
| `ko` | 韓国語 | `LangKO` | `ko.json` |
| `ru` | ロシア語 | `LangRU` | `ru.json` |

### フロントエンド

[i18next](https://www.i18next.com/) + [react-i18next](https://react.i18next.com/) を使用。言語設定は `localStorage`（キー: `ziziphus_language`）に保存されます。翻訳ファイルは `server/web/src/i18n/{lang}.json` にあります。中国語以外のバンドルはオンデマンドで遅延ロードされ、初期バンドルサイズを最小限に抑えます。

フロントエンドはすべてのリクエストで `X-Language` HTTP ヘッダーを使用して選択された言語をバックエンドに送信します。

### バックエンド

`server/pkg/i18n/` パッケージが提供する機能：

- **言語定数**（`LangZH`、`LangEN` など）
- **ParseLang()** — ブラウザのロケールコード（例：`zh-CN`、`en-US`、`ja-JP`）を受け入れ、サポートされている Lang 定数に正規化します
- **DetectLanguage()** — `X-Language` ヘッダー（フロントエンドの設定）を優先し、`Accept-Language` ヘッダー、最後に `LangZH` にフォールバック
- **T() / TWithLang()** — 位置パラメータ（`{0}`、`{1}`）をサポートするテンプレート形式の文字列翻訳
- **HTTP ミドルウェア** — リクエストごとに言語を検出してコンテキストに保存

翻訳メッセージは言語ごとにファイルが分割されています：
```
pkg/i18n/messages.go          # メッセージキーの宣言 + registerLang ヘルパー
pkg/i18n/{zh,en,ja,fr,de,es,ko,ru}.go  # 各言語の翻訳データ
```

### メールテンプレート

メール確認とパスワードリセットのテンプレートも全 8 言語に対応しています。テンプレートはコンパイル時に `//go:embed` で埋め込まれ、以下の場所にあります：

```
internal/auth/email_templates/
  verify_code_{lang}.html
  reset_password_{lang}.html
```

### 新しい言語の追加方法

**バックエンド：**
1. `pkg/i18n/i18n.go` に新しい `LangXX` 定数を追加
2. `ParseLang()` にロケールマッピングを追加
3. `pkg/i18n/{lang}.go` を作成し、`init() + registerLang()` ですべてのメッセージキーを登録
4. `internal/api/language.go` に `langToFrontendCode()` マッピングを追加

**フロントエンド：**
1. 翻訳済みキーと値のペアを含む `server/web/src/i18n/{lang}.json` を作成
2. 設定画面と認証画面の言語セレクターにその言語を追加
3. UI ストアの `Language` 型と `resolveAutoLang()` メソッドを更新

**メールテンプレート：**
1. 既存のテンプレートをコピー（例：`verify_code_en.html` → `verify_code_{lang}.html`）
2. テキストを翻訳
3. `internal/auth/mailer.go` に `//go:embed` ディレクティブを追加し、`emailTemplates` マップに登録
4. 件名の翻訳を追加

---

## 環境変数

`.env.example` ファイルを参照してください：

- `server/config/config.example.yaml` — バックエンド設定テンプレート
- `server/web/.env.example` — Web フロントエンド環境変数テンプレート
- ルートの `.env` — デプロイパラメータ（`.gitignore` に含まれているためコミットされません）
