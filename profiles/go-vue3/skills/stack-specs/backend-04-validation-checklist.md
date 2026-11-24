# backend-04 — 合规验证清单（后端）

> **层级**：L2 技术栈 profile skills。与 `rules/backend-rules.md` §7 对应；编排 agent T07 / Gate 2 扫描以此为清单。
> 执行方式：先跑 grep 快扫，命中项人工核对上下文；全部通过才算合规。

## CRITICAL（一项不合规即 FAIL）

- [ ] **C1 无 shell 拼接**：`grep -rnE "exec\.Command(Weak)?Context?\(.+\"(-c|/c)\"" --include="*.go"` 0 命中；且逐个核对所有 `exec.Command*` 调用为参数数组，用户输入未进入 shell 字符串
- [ ] **C2 无 SQL 拼接**：`grep -rnE "Sprintf\(.+(SELECT|INSERT|UPDATE|DELETE)" --include="*.go"` 0 命中；store 层抽查参数化 `?` 占位；动态标识符走白名单
- [ ] **C3 无硬编码凭据**：`grep -rnEi "(password|secret|token|apikey)\s*=\s*\"[^\"{}]{8,}\"" --include="*.go" | grep -v _test` 0 命中
- [ ] **C4 SQL 只在 store 层**：`grep -rlnE "(SELECT|INSERT|UPDATE|DELETE) " --include="*.go" | grep -v internal/store | grep -v _test` 0 命中
- [ ] **C5 无 panic 滥用**：`grep -rn "panic(" --include="*.go" | grep -v _test` 命中项逐个核对（仅 main 启动期 fail-fast / 真不变式）
- [ ] **C6 goroutine 有退出路径**：`grep -rn "go func" --include="*.go" | grep -v _test` 逐个核对（ctx 取消 / 生命周期归属 / panic 隔离）
- [ ] **C7 输入服务端校验**：所有 handler 有 binding tag；service 层语义校验；路径类参数过防逃逸（EvalSymlinks + 前缀白名单）
- [ ] **C8 编译与测试全绿**：`go build ./... && go vet ./... && go test ./...` 通过

## WARNING（需说明理由或修复）

- [ ] W1 无非测试代码 `fmt.Print*` 调试输出
- [ ] W2 无 `log.Print*` 混用（统一 slog）
- [ ] W3 错误包装带上下文（无裸 `return err` 大段透传；抽 3 个热点路径核对 `%w`）
- [ ] W4 context 首参贯穿（抽查 service/store/adapter 函数签名）
- [ ] W5 无包级可变全局 / init() 内资源连接
- [ ] W6 时间 UTC 存储、RFC3339 传输；金额非 float
- [ ] W7 日志无敏感字段（密码/token 明文）
- [ ] W8 导出符号有 doc comment

## 记录格式

```
扫描时间：{ts}
CRITICAL：C1 ✅ / C2 ✅ / ... （不合规项附文件:行号与修复建议）
WARNING：W1 ✅ / ...（豁免项附理由）
结论：PASS / FAIL
```
