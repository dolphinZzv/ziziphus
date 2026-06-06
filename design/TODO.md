# IM 系统待办事项

> 当前设计中已识别但暂未实现/需要后续完善的事项。
> 按发布阶段组织。

---

## Phase 1 — Server + macOS + iOS

### P1.1 Server 基础框架
- [ ] Go 项目结构搭建（cmd/internal/pkg 分层）
- [ ] 配置文件加载（config.yaml）
- [ ] PostgreSQL 连接初始化 + 迁移工具
- [ ] Redis 客户端初始化
- [ ] OTel SDK 集成（自动采集 HTTP/gRPC 链路、日志导出）
- [ ] Prometheus /metrics 端点暴露
- [ ] OTel Collector 配置
- [ ] Prometheus 部署和 scrape 配置
- [ ] Alertmanager 部署和告警规则配置

### P1.2 数据模型 + 建表
- [ ] Model 定义（User / Session / Message / Conversation / Member / Receipt）
- [ ] 建表迁移 SQL
- [ ] Snowflake ID 生成器

### P1.3 用户认证
- [ ] 注册 API
- [ ] 登录 API（返回 JWT Token）
- [ ] Token 验证中间件
- [ ] Token 刷新 API

### P1.4 Session 管理
- [ ] Session CRUD（创建/查询/删除）
- [ ] 用户 → 多 Session 映射
- [ ] 在线状态管理
- [ ] Session 过期清理

### P1.5 Gateway + WebSocket
- [ ] WebSocket 连接管理（升级/关闭/保活）
- [ ] 连接注册表（connID → Session）
- [ ] 消息帧收发（JSON 编解码）
- [ ] Ping/Pong 心跳
- [ ] 连接断开自动清理

### P1.6 P2P 消息
- [ ] 消息发送/接收
- [ ] P2P ConvID 确定性生成
- [ ] P2P 会话隐式创建
- [ ] 消息持久化
- [ ] 发送确认（Ack）
- [ ] 多终端扇出推送
- [ ] 增量消息同步（Sync）
- [ ] 离线消息存储与拉取
- [ ] 消息去重（client_seq）
- [ ] 已读回执

### P1.7 群聊
- [ ] 群创建 API
- [ ] 群成员管理（加人/踢人/退群/角色）
- [ ] 群消息发送（写扩散）
- [ ] 群消息路由（排除发送者）
- [ ] 系统消息（加人/退群通知）
- [ ] @mention 支持

### P1.8 macOS + iOS 客户端

#### 共享层 (IMCore Package)
- [ ] 抽取共享 Swift Package
- [ ] WebSocket 网络层（URLSessionWebSocket）
- [ ] 协议编解码
- [ ] 数据模型定义
- [ ] ViewModel 基础（ChatViewModel / ConversationListViewModel / LoginViewModel）
- [ ] HTTP API 客户端
- [ ] 本地缓存（CoreData）

#### macOS 特有
- [ ] SwiftUI 项目初始化
- [ ] 登录注册界面
- [ ] 会话列表页
- [ ] 聊天页面（消息气泡 + 输入框）
- [ ] 联系人列表
- [ ] 群创建/管理界面
- [ ] 多终端登录状态指示
- [ ] 菜单栏集成

#### iOS 特有
- [ ] SwiftUI 项目初始化
- [ ] 登录注册界面
- [ ] 会话列表页
- [ ] 聊天页面（消息气泡 + 键盘适配）
- [ ] 联系人列表
- [ ] 后台 WebSocket 策略（Background Task 保活）

### P1.9 联调与基础设施
- [ ] macOS ↔ Server 端到端联调
- [ ] P0 风险修复（见下方 P0 列表）
- [ ] 基础错误处理完善

---

## Phase 2 — Android + Web

### P2.1 Android 客户端
- [ ] Kotlin 项目初始化
- [ ] OkHttp WebSocket 网络层
- [ ] 协议编解码（按 Protocol 规范）
- [ ] 登录注册界面
- [ ] 会话列表（Jetpack Compose）
- [ ] 聊天页面
- [ ] 联系人 / 群管理
- [ ] 推送通知（FCM）

### P2.3 Web 客户端
- [ ] React + TypeScript 项目初始化
- [ ] WebSocket 连接封装
- [ ] 状态管理（Zustand / Redux）
- [ ] 登录注册页面
- [ ] 会话列表组件
- [ ] 聊天页面组件
- [ ] 响应式布局适配

### P2.4 推送通知（iOS APNs + Android FCM）
- [ ] Server 端 device token 注册/更新 API
- [ ] Server 端 APNs 转发（连接 Apple 推送网关）
- [ ] Server 端 FCM 转发
- [ ] iOS 端 APNs 注册和 token 上报
- [ ] Android 端 FCM 注册和 token 上报
- [ ] 离线推送到达率监控

### P2.5 文件传输
- [ ] 文件上传 API
- [ ] 对象存储接入（S3/MinIO）
- [ ] 图片/文件消息类型
- [ ] 缩略图生成
- [ ] 文件下载/预览

### P2.6 消息搜索
- [ ] Elasticsearch 索引搭建
- [ ] 消息写入同步索引
- [ ] 搜索 API（关键词/发送者/时间范围）
- [ ] 客户端搜索界面

---

## 技术债务与优化项

> 以下项按优先级排列，各阶段穿插完成。

### P0 — Phase 1 期间必须做

- [ ] **限流与防滥用**：
  - 消息发送频率限制（按用户/按 IP）
  - 消息体大小上限校验
  - 群创建频率限制、成员上限校验
  - API 整体限流中间件
- [ ] **消息去重完善**：多终端 client_seq 冲突处理 + 降级方案
- [ ] **离线消息持久化**：至少写 DB 一份完整记录，Redis 仅做推送缓冲
- [ ] **Session 状态一致性**：DB 为权威源，Redis 为缓存，断连时清理
- [ ] **user_seq 可靠性**：Redis crash 后从 DB 消息表重建 user_seq

### P1 — Phase 2 之前做

- [ ] **Grafana 面板搭建**：
  - WebSocket 连接数
  - 消息处理延迟 P50/P99
  - 推送队列深度
  - 在线用户数 / 消息量 QPS
- [ ] **推送队列持久化**：内存 channel 替换为 Redis Stream
- [ ] **告警规则完善**：根据实际运行数据调整阈值

### P2 — Phase 2 期间或之后做

- [ ] **Gateway 水平扩展**：
  - 多实例部署支持
  - `conn_id → gateway_addr` Redis 注册表
  - 跨 Gateway 消息转发
  - 需要至少 2 台服务器，Phase 1 单机不需要
- [ ] **群聊写扩散优化**：500 人以上群自动切换读扩散
- [ ] **消息编辑与撤回**：API + MsgRecall/MsgEdit 系统消息 + 客户端展示
- [ ] **端到端加密 (E2EE)**：P2P 会话加密、群聊密钥协商
- [ ] **消息已读回执聚合**：群聊已读列表、已读成员统计
- [ ] **连接平滑迁移**：Gateway 滚动升级、客户端无感重连
- [ ] **消息分表 + 归档**：按月分表、6 个月冷存储
