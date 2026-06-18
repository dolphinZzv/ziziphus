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

### Removed

- `client/ios/Sources/IMApp/Views/Settings/AppInfoView.swift` 已删除（功能并入 `AppSettingsView`）。

### Security

- 新增 `AuthMiddlewareWithAPIKey` 鉴权路径，仅在 token 以 `sk-` 为前缀时回退到 `users.api_key` 查询；JWT 失败不再直接拒绝，便于 agent 程序化访问。
- agent `api_key` 字段建立唯一索引（`api_key != ''` 子句），避免空字符串冲突。

[Unreleased]: https://github.com/dolphinZzv/panda-ai/compare/62add0c...HEAD
