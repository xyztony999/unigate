# Go 后端规则（backend-rules）

> **层级**：L2 技术栈 profile。激活含本栈的业务 profile 时由挂载机制生效。
> **分级**：🔴 CRITICAL = 违规即 Gate 2 FAIL，必须修复；🟡 WARNING = 需说明理由或修复。每条附可 grep 的检查方法（Gate 2 等价规则集）。
> **占位符**：`{service}` = 二进制/服务名（业务 profile 替换）；📎 标记处为通用示例，可按项目替换。

---

## 1. 编码规约

| 技术 | 版本 | 说明 |
|------|------|------|
| Go | 1.22+ | 标准库优先，引入依赖需说明理由 |
| gin | 1.x | HTTP 框架，handler 薄壳 |
| log/slog | 标准库 | 结构化日志，全链路 context |
| database/sql + 驱动 | — | 默认 modernc.org/sqlite（纯 Go 免 CGO）；MySQL 用 go-sql-driver |

### 1.1 语言风格

- 🔴 **禁止** 在 handler/service 中 `panic`（进程级致命场景除外，main 启动期允许）；错误一律 `error` 返回
- 🔴 错误包装必须 `%w`（保持 `errors.Is/As` 可解链）：`fmt.Errorf("load config: %w", err)`
- 🔴 **禁止** `_ = err` 或裸 `if err != nil { return err }` 无上下文直接透传（至少加一层 wrap）
- 🟡 **禁止** 非测试代码出现 `fmt.Println/Printf` 调试输出 → `grep -rn "fmt.Print" --include="*.go" | grep -v _test | grep -v fmt.Errorf`
- 🟡 导出符号必须有 doc comment；接口定义在**消费方**，命名 `-er` 后缀（`Store`、`Runner`）
- 🟡 context 为第一参数：所有 IO/DB/外部调用函数签名首参 `ctx context.Context`

### 1.2 并发

- 🔴 每个 goroutine 必须有退出路径（context 取消 / 生命周期归属明确）；后台任务由 main 统一编排，**禁止**库/service 层裸起 `go func()` → `grep -rn "go func" --include="*.go" | grep -v _test`（逐个核对退出路径）
- 🟡 共享状态用 channel 或 mutex，禁止 map 并发读写；`sync.WaitGroup` 配对 `defer wg.Done()`

---

## 2. 安全硬约束（本栈门禁核心）

> 来源：系统管理类服务的红线。每条 CRITICAL 必须进 Gate 2 扫描。

- 🔴 **禁止 shell 拼接执行**：外部命令一律 `exec.CommandContext(ctx, name, args...)` 参数数组；**禁止** `sh -c` / `bash -c` / `cmd /c` 拼接任何含用户输入的字符串
  → `grep -rn -- "-c\"" --include="*.go" | grep -iE "sh|bash|cmd"` 并逐个核对；`grep -rn "exec.Command(" --include="*.go"` 核对是否参数数组
- 🔴 **禁止 SQL 字符串拼接**：一律参数化 `?` 占位；表名/列名等标识符动态化必须走白名单映射
  → `grep -rnE "fmt\.Sprintf\(.*(SELECT|INSERT|UPDATE|DELETE)" --include="*.go"`（命中即人工核对）
- 🔴 **禁止硬编码密钥/口令**：凭据来自环境变量或配置文件，默认值不得含真实凭据
  → `grep -rnEi "(password|secret|token|apikey)\s*=\s*\"[^\"{}]{8,}\"" --include="*.go" | grep -v _test`
- 🔴 文件路径操作必须防逃逸：用户输入路径先 `filepath.EvalSymlinks` + 前缀白名单校验，再读写
- 🔴 handler 层所有输入必须服务端校验（binding tag + service 层语义校验），不信任前端
- 🟡 默认监听地址/端口不得写死公网 `0.0.0.0` 常量在业务代码中（由配置注入）

---

## 3. 目录结构

```
{BACKEND_ROOT}/
├─ cmd/{service}/main.go       # 入口：配置解析、依赖装配、优雅退出
├─ internal/
│  ├─ api/                     # gin 路由 + handler（薄壳：参数绑定→service→响应）
│  │  └─ middleware/           # 鉴权/审计/限流/恢复
│  ├─ service/                 # 业务逻辑（不感知 HTTP/传输层）
│  ├─ store/                   # 数据访问（repository，SQL 只在此层）
│  ├─ adapter/                 # 外部系统适配（shell/dbus/第三方，接口+实现）
│  └─ config/                  # 配置定义与加载
├─ pkg/                        # 可复用纯工具（无业务语义；没有就删掉此目录）
└─ web/dist/                   # 前端构建产物（go:embed 内嵌）
```

- 分层依赖方向单向：`api → service → store/adapter`；**禁止**反向引用与跨层直连（handler 不直接碰 store）
- 🔴 SQL 语句只允许出现在 `internal/store/` → `grep -rlnE "(SELECT|INSERT|UPDATE|DELETE) " --include="*.go" | grep -v internal/store | grep -v _test`
- 🟡 `internal/` **禁止**被外部模块 import（Go 工具链强制，无需检查）；`pkg/` 保持无业务语义

---

## 4. HTTP/WS 约定

- handler 薄壳：绑定参数（`ShouldBindJSON` + binding tag）→ 调 service → 统一响应包络；**禁止**在 handler 写业务逻辑
- 统一响应包络：`{"code": 0, "msg": "ok", "data": ...}`；业务错误码集中在 `internal/api/errcode`（或同层）定义
- 路由分组挂中间件：鉴权 → 审计 → 限流，顺序固定；公开端点（登录/健康检查）单独分组显式声明
- WebSocket：升级前必须过鉴权中间件；连接持有 ctx，会话过期即断
- 🟡 时间字段统一 UTC 存储、RFC3339 传输；金额用整数最小单位或 decimal，**禁止** float

---

## 5. 配置与日志

- 配置：结构体 + 加载函数（env 覆盖 > 配置文件 > 默认值），启动期一次性校验，非法即 fail-fast
- 🔴 日志必须 slog 结构化：`slog.InfoContext(ctx, "msg", "key", val)`；**禁止** `log.Println` 混用 → `grep -rn "log\.Print" --include="*.go" | grep -v slog | grep -v _test`
- 🟡 日志不得输出敏感字段（密码/token 明文）；错误日志必须带请求上下文（trace/request id）

---

## 6. 面向用户的错误消息（去 AI 味）

> 机制源：engine-rules.md **ER-04**。范围 = 返回给前端展示的错误消息与业务提示（`msg`/`message` 字段）；日志（slog）不受此约束——日志写给开发者，可含技术细节。

- 错误消息**说清原因与出路**，用户视角："机构名称已存在" ✓；"数据校验异常" ✗；"发生未知错误" ✗（应给可行动的信息或引导联系支持的上下文）
- 禁 emoji / 感叹号堆叠 / 空洞夸赞 / 系统腔·翻译腔（"操作成功完成"→"已保存"；"这是一个非法的参数"→"参数 regionCode 不合法"）
- 错误消息不直出内部术语与堆栈（内部错误细节进日志，对外给可读原因 + trace id 供排查）
- 自查：grep 返回体文案字段中 emoji 与 `！` → 0 命中；逐条读是否说清原因与下一步（Gate 2 扫描见 §8 G-08）

---

## 7. 构建与门禁

| 门禁 | 命令 | 说明 |
|------|------|------|
| 编译 | `go build ./...` | Gate 2 必过 |
| 静态检查 | `go vet ./...` | Gate 2 必过 |
| 单元测试 | `go test ./... -race` | 核心包必须有表驱动测试 |
| 漏洞扫描 | `govulncheck ./...` | CI 周期跑，CRITICAL 阻塞发布 |

- 前端产物内嵌：`//go:embed web/dist` 的 embed.go 变更后必须先构建前端再编译后端
- 交叉编译以 CGO_ENABLED=0 为准（依赖选型必须纯 Go 兼容）

---

## 8. Gate 2 扫描清单（CRITICAL 汇总）

| # | 检查 | grep 方法 | 判定 |
|---|------|----------|------|
| G-01 | 无 shell 拼接 | `grep -rnE "exec\.Command(Weak)?Context?\(.+\"(-c|/c)\"" --include="*.go"` + 人工核对全部 `exec.Command` | 0 命中 |
| G-02 | 无 SQL 拼接 | `grep -rnE "Sprintf\(.+(SELECT|INSERT|UPDATE|DELETE)" --include="*.go"` | 0 命中（store 层人工复核参数化） |
| G-03 | 无硬编码凭据 | `grep -rnEi "(password|secret|token)\s*=\s*\"[^\"]{8,}\"" --include="*.go" | grep -v _test` | 0 命中 |
| G-04 | SQL 只在 store 层 | 见 §3 | 0 命中 |
| G-05 | 无 panic 滥用 | `grep -rn "panic(" --include="*.go" | grep -v _test` | 仅 main 启动期/不可能不变式，逐个核对 |
| G-06 | goroutine 有退出路径 | 见 §1.2 | 逐个核对 |
| G-07 | 编译+vet+test 全绿 | `go build ./... && go vet ./... && go test ./...` | 全绿 |
| G-08 | 用户可见错误消息无 AI 味（无 emoji/`！`，说清原因与出路） | `grep -rnE "msg\]|message\]|Msg\][^,]*[！!]|🎉|✨|太棒" --include="*.go" | grep -v _test` + 逐条人工读 | 0 命中（见 §6） |
