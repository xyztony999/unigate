# backend-02 — API 设计规约（api_spec）

> **层级**：L2 技术栈 profile skills。引擎 design skill D3 阶段引用本文件（`{stack.api_spec}`）。
> 所有 REST/WS 端点设计与 handler 生成以此为唯一约定。

## 1. 响应包络

所有 REST 接口 HTTP 状态码统一 200（传输层健康由中间件/网关表达），业务结果用包络表达：

```json
{ "code": 0, "msg": "ok", "data": { } }
```

| 字段 | 说明 |
|------|------|
| code | 0 = 成功；非 0 = 业务错误码（见 §2） |
| msg | 人读信息；错误时给可操作的提示 |
| data | 成功时为业务数据；失败时 null |

## 2. 错误码分段

| 段 | 范围 | 含义 |
|----|------|------|
| 0 | 0 | 成功 |
| 4xxxx | 40000-49999 | 客户端错误（参数/权限/状态冲突） |
| 5xxxx | 50000-59999 | 服务端错误 |

细分约定：`401xx` 未认证、`403xx` 未授权、`404xx` 不存在、`409xx` 状态冲突、`422xx` 参数校验失败、`500xx` 内部错误、`503xx` 依赖不可用。错误码集中定义在 `internal/api/errcode`，**禁止** handler 内裸写数字。

## 3. 路径与命名

- REST 资源路径：`/api/v1/{module}/{resource}`；集合 GET / 单体 GET|PUT|DELETE 按 REST 语义
- 动作类端点（非 CRUD 语义，如 reload/restart）：`POST /api/v1/{module}/{resource}/{id}/{action}`
- 资源名复数小写连字符（`/api/v1/sites/ssl-certs`）；query 参数 camelCase
- **禁止**路径含动词混排 CRUD（`GET /api/v1/getSiteList` ❌）

## 4. 请求与响应字段

- 字段 camelCase JSON；时间 RFC3339（UTC）；枚举传输 string code
- 分页请求：`pageNum`（1 起）/ `pageSize`（默认 20，上限 100）；分页响应：`{ "records": [], "total": 0 }`
- **禁止** `list/data/items` 等异名列表字段；前端类型以本规约对齐
- 写操作幂等：PUT 幂等；POST 非幂等操作支持 `Idempotency-Key` 头（可选增强）
- 批量删除 `POST /api/v1/{module}/{resource}/batch-delete`，body `{ "ids": [] }`

## 5. 校验

- handler 层 binding tag（`binding:"required,max=128"`）做格式校验
- service 层做语义校验（存在性/状态/权限），错误映射 errcode
- **不信任前端**：所有输入服务端再校验；路径类参数必须过防逃逸校验后才可触达文件系统

## 6. WebSocket 端点

- 路径：`/api/v1/ws/{channel}`；升级前必须过鉴权中间件（Cookie 会话）
- 消息帧：`{ "type": "...", "payload": {...} }`；type 枚举集中定义
- 服务端主动推送（监控指标/日志跟踪）与客户端心跳（ping/pong 30s）约定：
  - 客户端 60s 无 pong → 服务端断开
  - 会话过期 → 服务端发 `{ "type": "expired" }` 后关闭

## 7. OpenAPI 对齐

- D3 设计文档中每个接口给出：方法、路径、请求/响应 JSON 示例、错误码列表
- handler 生成（backend-orchestrator T04）逐接口对照设计文档，**禁止**自行增删字段
