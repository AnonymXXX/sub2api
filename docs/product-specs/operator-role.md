# 数据观察员角色与权限

## 产品定义

`operator` 是面向日常运营观测的只读角色，中文名为“数据观察员”。它保留普通用户的基础能力，但不具备系统管理、配置修改、告警处置或数据清理权限。

## 权限矩阵

| 能力 | `admin` | `operator` | `user` |
| --- | --- | --- | --- |
| 普通用户基础能力 | 允许 | 允许 | 允许 |
| 管理仪表盘 `admin.dashboard.read` | 允许 | 只读 | 禁止 |
| 运维监控 `admin.ops.read` | 允许 | 只读 | 禁止 |
| 使用记录 `admin.usage.read` | 允许 | 只读 | 禁止 |
| 其他管理页和管理写操作 | 允许 | 禁止 | 禁止 |

## 行为约束

- 数据观察员默认首页为 `/admin/dashboard`，管理侧边栏只显示 `/admin/dashboard`、`/admin/ops` 和 `/admin/usage`。
- 数据观察员可查询仪表盘数据、运维监控数据、QPS WebSocket、使用记录、统计、搜索、筛选选项、错误详情和客户端 Excel 导出。Usage 筛选选项接口只返回所需的 `id/name` 字段。
- Dashboard 聚合回填、Ops 的所有写操作、Usage 清理任务和用户余额历史始终只允许 `admin`。
- `GET /api/v1/admin/ops/viewer-config` 只返回 Ops 开关、默认查询模式和 `{id,name,platform}` 分组选项，不得暴露完整系统设置。
- JWT 管理认证在每次请求时以数据库当前角色为准；Admin API Key 继续映射为完整管理员权限。
- 数据观察员需接受管理运营合规承诺。Backend Mode 允许已有数据观察员登录、刷新令牌和使用基础用户接口，但不放开注册或新用户创建。
- 任何时候都必须保留至少一名 `admin`；最后一名管理员不能降级为 `operator` 或 `user`。

## 变更原则

数据观察员权限采用显式允许列表。后续新增管理接口或页面时，默认仅 `admin` 可访问；只有在更新本规格和权限矩阵后，才能向 `operator` 开放。
