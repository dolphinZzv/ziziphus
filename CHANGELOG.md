# Changelog

All notable changes to PandaAI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Agent（智能体）系统**
  - 数据库迁移：
    - `008_add_user_uid.sql`：`users` 表新增 `uid` 列追踪 agent ownership（人类用户 `uid = id`，agent 用户 `uid = 创建者 id`）。
    - `009_add_wake_mode.sql`：`users` 表新增 `wake_mode` 字段（0 = 全部消息唤醒，1 = 仅 @mention 唤醒）。
    - `010_add_api_key.sql`：`users` 表新增 `api_key` 字段并建立 `idx_users_api_key` 唯一索引。
  - `pkg/model`：`User` 新增 `UID` / `WakeMode` / `APIKey` 字段；新增 `WakeMode` 枚举（`WakeModeAll` / `WakeModeMention`）；`ConvMember` 暴露 `UserType` 与 `WakeMode`。
  - `UserRepo`：新增 `CountAgents` / `ListAgents` / `UpdateAgent` / `DeleteAgent` / `GetByAPIKey` / `UpdateAgentAPIKey` 等 agent 专用方法。
  - `UserHandler` & 路由：新增 `GET/POST/PUT/DELETE /api/v1/users/me/agents` 与 `PUT /api/v1/users/me/agents/{agent_id}/regenerate-key` 接口。
  - `AuthMiddleware`：新增 `AuthMiddlewareWithAPIKey`，支持以 `sk-` 为前缀的 API Key 鉴权（JWT 失败时回退）。
  - `Message.Router`：当群成员为 agent 且 `wake_mode = mention` 时，仅在消息 `Mention` 列表命中才投递。
  - `pkg/model.Message`：`ContentType` 枚举新增 `ContentRecall(6)` / `ContentEdit(7)` / `ContentCustom(8)` / `ContentAgentTimeline(9)`。
- **Agent 时间线**
  - IMCore 新增 `AgentTimelineBody` 模型，支持 `thinking` / `toolCall` / `toolResult` / `response` 四类条目与 `parentMsgID` 增量追加。
  - IMCore `Utils/String+ImageURL`：新增 `isImageURL` 与 `extractImageURLs(baseURL:)`，从 markdown 中解析图片 URL。
  - 新增 `AgentTimelineView`（iOS 与 macOS），使用 `Textual` 渲染 markdown，支持条目展开/收起与可点击图片。
  - 新增 `IMCoreUnitTests/AgentTimelineBodyTests.swift`。
  - 新增 `example/agent_timeline_test.py`。
- **通用 UI 组件**
  - `CachedImage`（iOS / macOS）：带缓存的图片视图。
  - `ImageViewer`（iOS / macOS）：全屏图片查看器。
  - `UserCardView`（iOS）：用户资料卡。
  - `AccountSelectView`（iOS）：账号选择 / 登录入口。
  - `InputBarView` 重写：增加 `@` 提及成员选择、键盘 `FocusState`、键盘工具栏的“完成”按钮。
- **Health 端点**
  - `GET /health` 响应新增 `startup_time`（RFC3339）与 `git_commit`（来自 `pkg/version.GitCommit`）。
- **i18n**
  - 整理 `pkg/i18n/messages.go` 键值与缩进，新增若干 agent / 认证相关 key。
- **部署 / 配置**
  - `Makefile`：新增 `deploy` / `deploy-status` / `deploy-logs` 目标，支持 SSH 远程部署后端为 systemd 服务（`panda_ai.service`），构建时通过 `ldflags` 注入 `GitCommit`。
  - `Makefile` macOS 目标：`Info.plist` 插入 `NSAppTransportSecurity` 以允许任意加载。
  - `Makefile` 顶部支持 `-include .env` 注入部署变量。
  - `.gitignore`：新增 `.env` 忽略。

- **Web 前端 SPA**
  - 新增 `server/web/` 目录，基于 React/TypeScript/Vite 构建 Web 前端。
  - 新增 `server/webembed/embed.go`，将前端构建产物嵌入 Go 二进制，通过 SPA fallback 路由提供前端服务。
  - 新增 Playwright E2E 测试覆盖 auth、chat-ui、conversation-dialogs、layout、routing 等场景。
  - `Makefile server` 目标：启动后端前自动构建 web 前端（`cd server/web && npm run build`）。
  - `.gitignore`：忽略 `server/web/dist/`、`server/internal/webembed/dist/` 构建产物。
- **会话置顶**
  - 数据库迁移 `012_add_pin.sql`：`conv_members` 表新增 `pinned` 列。
  - 新增 `POST /api/v1/conversations/{conv_id}/pin` 与 `POST /api/v1/conversations/{conv_id}/unpin` 接口。
  - 会话列表查询按 `cm.pinned DESC` 排序，置顶会话优先展示。
  - `ConvListItem` 与 `ConvMember` 模型新增 `Pinned` 字段。
- **群公告**
  - 数据库迁移 `011_add_notice.sql`：`conversations` 表新增 `notice` 列。
  - `UpdateGroup` 接口支持 `notice` 字段更新（仅群主可操作）。
  - `GetDetail` 响应新增 `notice` 字段。
- **群组克隆**
  - 新增 `POST /api/v1/conversations/{conv_id}/clone` 接口，复制群组结构及成员（克隆者设为群主）。
- **图片即时缩放**
  - `GET /files/*` 端点支持 `?w=xxx&h=xxx` 查询参数，按指定尺寸 center-crop 缩放图片。
  - 引入 `golang.org/x/image` 依赖用于高质量缩放。
- **macOS @mention 提及**
  - `ChatTextView`：新增 `onMentionChanged` 回调，检测 `@` 输入并提取查询文本。
  - `InputBarView`：新增 `@` 按钮与内联提及弹出面板，支持按昵称/ID 过滤成员。
  - `ChatView`：透传群成员列表至 `InputBarView`，`onAppear` 时调用 `loadMembers()`。
- **macOS 图片缓存**
  - `AvatarView`：新增 `ImageCache` 单例，支持内存+磁盘两级缓存，SHA256 哈希磁盘路径，自动缩放至 256pt。
- **WebSocket 设备类型扩展**
  - `handler/ws.go`：新增 `web`、`android`、`windows` 设备类型识别。
- **推送载荷增强**
  - `MsgPushPayload` 新增 `SenderName` 字段，推送时携带发送者名称。
- **Agent 创建增强**
  - `CreateAgent` 自动设置 `Account` 为 `"agent_" + agentID`。
- **i18n**
  - `StringCatalog`：新增 `common.download` 本地化条目。
- **macOS 对话列表 UI**
  - `CreateGroupView`、`JoinGroupView`、`NewChatView`：统一 padding 为 `horizontal(16)` + `vertical(10)`。
  - Xcode 项目从自动生成 `Info.plist` 改为使用自定义 `macos/Resources/Info.plist`。

### Changed

- `pkg/model.Message` / `Conversation` / `Session` / `Snowflake` / `errors.go` 等多个结构体按 gofmt 对齐字段。
- `internal/api/conversation.go`：`ConvHandler.NewConvHandler` 构造体缩进修正。
- `internal/api/contact.go` / `conversation.go` 等：错误响应调用统一加空格 `Error(w, r, ...)`。
- `internal/api/file.go`：去掉文件末尾多余空行。
- `internal/api/user.go`：`BatchGet` 响应增加 `uid` 字段。
- `internal/api/router.go`：批量注册 agent 相关路由。
- `internal/storage/db/conversations.go`：
  - `ConvListItem` 新增 `Role` / `Mute` / `PartnerType` 字段（从 `conv_members` 提取后写入 `items`）。
  - P2P 会话显示名解析改为批量 SQL：优先使用 `contacts.nickname`，回退 `users.name`，最终回退到 `partnerID`。
- `internal/message/router.go`：路由决策时过滤 `UserAgent + WakeModeMention` 且未被 @ 的成员。
- `internal/auth/middleware.go` & `service.go`：格式对齐；Service 字段排序与 import 分组调整。
- `internal/message/push.go` / `sync.go` / `receipt.go`：结构体字段对齐 + import 分组微调。
- `client/Package.swift` & `client/project.yml`：新增 `Textual` 依赖（指向 `../deps/textual`）。
- `client/ios` & `client/macos` 多个 `Views`（Chat / ConversationList / Profile / Settings 等）按新模型字段与交互刷新。
- `CLAUDE.md`：补充“使用真机部署，不要使用模拟器”、“所有用户反馈都需要增加单元测试和 e2e 测试”。
- 集成已有但未推送的提交：
  - `9ac2084` feat: security fixes, file upload, group name editing, UI improvements
  - `3d49077` fix: restore menu with gear icon in macOS chat toolbar

- `internal/api/file.go`：
  - 重构 `ServeFile`：先读入内存再响应，支持缩放后设置 `Cache-Control: public, max-age=2592000`（30天）。
  - 提取 `contentTypeByExt()` / `isImageExt()` / `resizeImage()` 辅助函数。
  - 移除多余注释与变量重命名（`url` -> `fileURL`）。
- `Makefile deploy` 目标：
  - 部署前 `systemctl stop panda_ai`，部署后 `systemctl start panda_ai`（替代 restart）。
  - service 文件直接写入 `/etc/systemd/system/`。
  - 新增 `chmod +x` 远程二进制权限设置。
- `CLAUDE.md`：新增"web 所有用户反馈都需要补充 e2e 测试防止问题回归"。

### Removed

- `client/ios/Sources/IMApp/Views/Settings/AppInfoView.swift` 已删除（功能并入 `AppSettingsView`）。

### Security

- 新增 `AuthMiddlewareWithAPIKey` 鉴权路径，仅在 token 以 `sk-` 为前缀时回退到 `users.api_key` 查询；JWT 失败不再直接拒绝，便于 agent 程序化访问。
- agent `api_key` 字段建立唯一索引（`api_key != ''` 子句），避免空字符串冲突。

[Unreleased]: https://github.com/dolphinZzv/panda-ai/compare/62add0c...HEAD
