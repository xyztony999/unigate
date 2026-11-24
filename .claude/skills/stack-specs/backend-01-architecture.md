# backend-01 — 分层架构（Go）

> **层级**：L2 技术栈 profile skills。配套规则：`rules/backend-rules.md` §3。

## 1. 工程布局

```
{BACKEND_ROOT}/
├─ cmd/{service}/main.go       # 唯一入口：配置加载 → 依赖装配 → 优雅退出
├─ internal/
│  ├─ api/                     # 传输层：gin handler（薄壳）
│  │  ├─ middleware/           # 鉴权/审计/限流/recovery
│  │  └─ errcode/              # 业务错误码集中定义
│  ├─ service/                 # 业务逻辑层（DTO 在此定义，不感知 HTTP）
│  ├─ store/                   # 数据访问层（repository 接口 + SQL 实现）
│  │  └─ migrations/           # DDL 与迁移脚本
│  ├─ adapter/                 # 外部系统适配：shell/dbus/第三方 API（接口在消费方定义）
│  └─ config/                  # 配置结构体 + 加载校验
├─ web/dist/                   # 前端产物（go:embed）
└─ embed.go                    # //go:embed web/dist
```

**依赖方向**（单向，禁止反向）：

```
cmd/{service} → api → service → store / adapter
                    └──────────→ config
```

- handler **薄壳**：绑定参数 → 调 service → 包络响应；无业务逻辑
- service 返回业务 error（含 errcode）；**禁止**返回 gin 上下文或 HTTP 概念
- store 接口定义在 service 包（消费方），实现在 store 包（Go 惯例：接口随消费者）
- adapter 同理：service 定义所需接口，adapter 实现后由 main 装配注入

## 2. 错误处理

```go
// errcode：业务错误集中定义
var ErrNotFound = errcode.New(40401, "resource not found")

// service：包装传递，保持 %w 链路
if err := s.store.Get(ctx, id); err != nil {
    if errors.Is(err, store.ErrNoRows) {
        return nil, ErrNotFound            // 业务语义转换
    }
    return nil, fmt.Errorf("get site: %w", err)   // 透传加上下文
}

// api handler：统一映射包络
var ec errcode.Error
if errors.As(err, &ec) { c.JSON(200, envelope(ec.Code(), ec.Msg(), nil)); return }
c.JSON(200, envelope(50000, "internal error", nil))   // 未知错误不外泄细节
```

- 🔴 `%w` 包装；`errors.Is/As` 判定；**禁止**字符串匹配错误文本
- 🔴 handler 层兜底：未知 error 不外泄内部细节（日志记全量，响应给脱敏信息）

## 3. context 传播

- 首参 `ctx context.Context` 贯穿 api → service → store/adapter
- gin 场景：`c.Request.Context()` 取请求级 ctx；后台任务用 `context.Background()` 派生 + `WithCancel`，由 main 持有
- SQL/HTTP/shell 调用全部用 `*Context` 变体（`QueryContext`/`http.NewRequestWithContext`/`exec.CommandContext`）

## 4. 依赖装配与优雅退出

```go
func main() {
    cfg, err := config.Load()                       // fail-fast
    if err != nil { log.Fatalf("load config: %v", err) }

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    db := store.MustOpen(cfg.DB)                    // 依赖自底向上装配
    svc := service.New(db)
    srv := api.NewServer(cfg.Addr, svc)

    go func() {                                     // 有退出路径的 goroutine
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            slog.ErrorContext(ctx, "http server", "err", err); stop()
        }
    }()

    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = srv.Shutdown(shutdownCtx)                   // 优雅退出：停入站 → 排空 → 关资源
    _ = db.Close()
}
```

- 🟡 全局单例**禁止**（`init()` 里连接资源、包级可变全局）；依赖显式构造注入
- 🔴 日志用 `slog`，handler 内 `slog.InfoContext(ctx, ...)`

## 5. 内嵌前端

```go
//go:embed all:web/dist
var distFS embed.FS
// gin 静态路由挂载（SPA fallback 到 index.html）
```

- 前端产物目录不存在时编译报错属预期——先构建前端再编后端；CI 中两步顺序固定
