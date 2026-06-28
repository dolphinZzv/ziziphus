# 会话文件目录 设计文档

## 设计

每个会话有一个共享文件目录：
- `files` 表加 `conv_id VARCHAR(64)` 列
- 上传时关联会话 → `POST /api/v1/files/upload` 加 `conv_id` 参数  
- 会话文件列表 → `GET /api/v1/conversations/{conv_id}/files`
- 删除文件 → `DELETE /api/v1/conversations/{conv_id}/files/{file_id}`
- 文件服务 → `GET /files/{file_id}`（已有）
- UI 入口：聊天工具栏 → 文件目录面板

## 数据模型

```sql
ALTER TABLE files ADD COLUMN conv_id VARCHAR(64) NOT NULL DEFAULT '';
CREATE INDEX idx_files_conv_id ON files(conv_id);
```

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/files/upload | 已有，+conv_id |
| GET | /api/v1/conversations/{conv_id}/files | 新增，列表 |
| DELETE | /api/v1/conversations/{conv_id}/files/{file_id} | 新增，删除 |

## UI

聊天工具栏 "📁" 按钮 → 面板：文件列表（名称/大小/上传者/时间）+ 上传按钮 + 下载/删除
