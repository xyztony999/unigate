# UniGate Go 后端重写任务清单

> 创建时间：2026-11-24
> 语义事实源：backend/src/main/java/com/unigate/（Java，git 14d226a 可恢复）
> 冻结契约：profiles/unigate/workspace/02-detailed/api/openapi-unigate.json（sha256 cc185ea0…6612b，不改）
> 代码输出：backend/（module unigate/backend）
> 覆盖决策：ADR-047 —— PostgreSQL+pgx（$N 占位）、冻结包络/真实 HTTP 状态码、PORT 环境变量、不做 go:embed、移植密码重置。

## 数据层

- [x] T01 · 迁移与 DDL + module 骨架
  - go.mod/go.sum；internal/store/migrations/0001_unigate.sql（V1 字节一致）；migrate.go 内嵌迁移+版本表+advisory lock
  - internal/contract/openapi-unigate.json（sha256 cc185ea0… 字节一致，go:embed 加载）

## 代码生成

- [x] T02 · store 层（pgx，SQL 仅此层）
  - crypto.go/keyvault.go/password.go 字节级复刻；store.go（Rows/Row/Exec/Scope/seal/unseal/page/selectorPage/Event/Version/Audit/JSONReady）
  - tx.go（READ COMMITTED/REPEATABLE READ 只读+5s statement_timeout）；identity/event/party/integration/operation/worker_repo.go 全量参数化 $N
- [ ] T03 · service 层（业务语义，不感知 HTTP）— 进行中
  - 已完成：errcode（canonical 映射，internal/errcode）、service.go（Req 五角色 grant/view/action/fields）、identity.go（session/newSession/Bootstrap/Authenticate/CSRF/Origin/cookie/reset/views）
  - 剩余：identity handle（Login/Logout/SessionMe/PasswordChange/Account*/Grant*）、security attempts、event、party、gate、integration、notification、provider、workers、dispatch（幂等/重放/审计）
- [ ] T04 · api 层（gin，85 操作）
  - 冻结包络 {code,message,data,requestId}、真实状态码、二进制导出、multipart、安全响应头
- [ ] T05 · middleware
  - 安全头/请求 ID/recovery/上传限制；鉴权与 CSRF 在 service authenticate（逐语义复刻）

## 测试与验证

- [ ] T06 · 测试
  - 表驱动单元：contract 校验、crypto/keyvault/password 字节兼容、normalize/safeCell/partyLink、canonical 错误码
  - PG 集成（UNIGATE_TEST_DB_URL/USER/PASSWORD，独立 schema unigate_test）：移植 WorkflowIntegrationTest/IntegrationWorkflowTest/PasswordResetTest/ProviderAdapterTest 覆盖
- [ ] T07 · Gate 2 合规扫描 G-01~G-08（必过）

## 收尾

- [ ] T08 · go build/vet/test 全绿后移除 Java Maven 结构（保留 DDL 于 Go 迁移目录），INDEX.md 写回