# backend-03 — 常用模式（Go）

> **层级**：L2 技术栈 profile skills。📎 示例为通用参考。

## 1. 配置加载（fail-fast）

```go
type Config struct {
    Addr      string        // 监听地址
    DBPath    string
    SessionTTL time.Duration
}

func Load() (*Config, error) {
    cfg := &Config{ Addr: "127.0.0.1:0", SessionTTL: 30 * time.Minute }   // 安全默认值
    if v := os.Getenv("APP_ADDR"); v != "" { cfg.Addr = v }
    // env 覆盖 > 配置文件 > 默认值；加载后统一校验
    if cfg.SessionTTL <= 0 { return nil, errors.New("session ttl must be positive") }
    return cfg, nil
}
```

- 启动期校验，非法即退出；**禁止**业务代码运行期读裸 env（统一走 Config）

## 2. DDL 与类型映射

| 设计文档类型 | SQLite | MySQL | Go |
|-------------|--------|-------|----|
| 主键/ID | TEXT | VARCHAR(32) | string |
| 短文本 | TEXT | VARCHAR(n) | string |
| 布尔 | INTEGER (0/1) | TINYINT(1) | bool |
| 时间 | INTEGER (unix) / TEXT (RFC3339) | DATETIME | time.Time |
| 金额 | INTEGER（最小单位） | BIGINT / DECIMAL | int64 |
| JSON 结构 | TEXT（序列化） | JSON | struct + json tags |

迁移：`internal/store/migrations/NNN_{name}.sql` 递增编号 + 版本表记录；启动时按序执行未应用的迁移。外键在 SQLite 需 `PRAGMA foreign_keys = ON`（连接初始化执行）。

## 3. 参数化查询（store 层唯一 SQL 出口）

```go
func (r *SiteRepo) ListByStatus(ctx context.Context, status string) ([]Site, error) {
    rows, err := r.db.QueryContext(ctx,
        `SELECT id, name, status FROM sites WHERE status = ? ORDER BY name`, status)   // ✅ 参数化
    ...
}
```

- 🔴 **禁止** `fmt.Sprintf` 拼 SQL；标识符（表名/列名/排序字段）动态化必须白名单映射：

```go
var sortable = map[string]string{"name": "name", "created": "created_at"}   // 白名单
col, ok := sortable[req.SortBy]
if !ok { return nil, errcode.New(42201, "invalid sort field") }
```

## 4. 外部命令安全调用

```go
// ✅ 参数数组 + context + 超时
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()
out, err := exec.CommandContext(ctx, "nginx", "-t").CombinedOutput()
if err != nil { return fmt.Errorf("nginx -t: %w: %s", err, out) }

// ❌ 禁止：shell 拼接（哪怕"输入看起来安全"）
exec.Command("sh", "-c", "nginx -t "+cfgPath)
```

- 🔴 用户输入只能作为**独立参数**传入；需要 shell 特性的场景，先参数化拆解，确实无法拆解的走白名单模板
- 二进制路径用 `exec.LookPath` 解析，**禁止**信任外部传入可执行路径

## 5. 后台任务编排

- 周期任务（采集/轮询）：`time.Ticker` + ctx 取消；由 main 启动、优雅退出时停止
- 单飞（防并发重入）：`singleflight` 或互斥锁
- 任务 panic 隔离：goroutine 顶部 `defer recover()` 记日志，不让单个任务击穿进程

## 6. 测试模板（表驱动）

```go
func TestCreateSite(t *testing.T) {
    tests := []struct {
        name    string
        input   CreateSiteInput
        wantErr error
    }{
        {"valid", CreateSiteInput{Name: "a.example.com"}, nil},
        {"empty name", CreateSiteInput{}, errcode.ErrInvalidParam},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := newTestService(t)          // 内存 sqlite / httptest
            got, err := svc.Create(context.Background(), tt.input)
            if !errors.Is(err, tt.wantErr) { t.Fatalf("err = %v, want %v", err, tt.wantErr) }
            _ = got
        })
    }
}
```

- store 层测试用内存 SQLite（`:memory:`）；api 层测试用 `httptest.NewServer`；不 mock 数据库
- 核心业务包覆盖率目标 ≥70%；错误路径必须有用例
