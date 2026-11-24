# UniGate 后端 Go 重写交付报告（ADR-048）

日期：2026-11-24 ｜ 模式：/pipeline 增量（多代理）｜ 判定：Gate 2 WARN 放行（0 CRITICAL）

## 1. 交付范围

后端 Java/Spring Boot **全量重写为 Go**（gin + log/slog + pgx，单二进制 `cmd/unigate` + `internal/`），替换 `backend/`。动机：用户 2核2G/3Mbps 服务器 JVM 内存过大；Go 单二进制无 JVM、运行约 50MB。

**冻结不变**：85 操作 OpenAPI 契约（sha256 `cc185ea0…6612b`，字节级）、33 页 PageSpec、25 表 PostgreSQL DDL（内嵌迁移 `0001_unigate.sql`）。前端零改动（5 处外观修复 wire 零影响，已验证）。数据库保留面板 PostgreSQL，pgx `$N` 占位，FOR UPDATE/SKIP LOCKED/advisory lock/jsonb/timestamptz 全保留。

## 2. 验证

- **142 测试函数 / 41 集成测试，6 包全过，0 FAIL 0 SKIP**；`go build/vet` 干净；`-race -timeout 30m` 无 DATA RACE。
- 单元字节兼容：AES-GCM 密文布局（主密钥 `[0x01|IV|密文]`、活动 DEK `[0x02|36字节UUID|IV|密文]`+AAD）、bcrypt 域分离预哈希（`$ug$sha256-bcrypt$v1$`，非 Argon2id）、HMAC 已知答案、UTF-16 表头、RawJSON 嵌套回归。
- 集成覆盖全部操作组：身份/报名/核销/导入/账号授权/名单自助/凭证/导出交付清理/连接来源 worker/短信找回通知/并发竞争。mock provider（HTTPDoer）+ SMTP sink，零真实外网。
- service 语句覆盖 **69.9%**（Java 基线 87.59%；剩余防御分支记边界，go-vue3 无数值门槛）。
- G-01~G-08 合规全过；GR-05 前后端契约 85 路由全等。
- 活体启动验证：迁移/bootstrap/匿名会话/管理员登录/冻结包络/cookie/时间格式/worker 全正确。

## 3. 测试驱动修复的 5 个真实 bug

1. 事务 `SET statement_timeout=$1` → `set_config(...,$1,true)`（SET 不接受绑定参数）。
2. `errcode.Require` typed-nil（返回 `*Error`）→ 返回 `error`。
3. `RawJSON`（[]byte）嵌套被 base64 化 → `MarshalJSON` verbatim（导入 availableColumns 恒空根因）。
4. EventCreate 清理范围 map 随机迭代 → canonical 顺序（冻结池顺序）。
5. gateRecords kiosk NULL actor_id 渲染 "" → 省略可选键（uuid 格式）。

## 4. 部署适配（已完成）

CI（setup-go 按 go.mod、Linux 交叉编译 `CGO_ENABLED=0`）、deploy.sh/systemd/reset-admin.sh 改单二进制、nginx/env 统一 **PORT=8090**（服务器 8080 被占）、文档去 JRE、local.py 改 go run/test。Java Maven 结构（pom.xml/src/target）已移除，git `14d226a` 可恢复。

## 5. 交付边界（WARN，记录）

- W-1 覆盖广度：69.9%，剩余防御/错误分支未集成执行。
- W-3 浏览器 E2E/真机/现场容量/备份恢复：属**部署验收**，HTTP 集成替代 API 层，不作为本地门控证据。
- 线上服务器 `/opt/unigate/unigate.env` 与宝塔 nginx 的 8090 同步、GitHub repo variable `APP_PORT` 检查：部署时操作。

## 6. 后续建议

- 补齐剩余防御分支集成测试以逼近 Java 覆盖基线（可选）。
- 首次公网部署后：真机扫码/横屏 kiosk/现场容量彩排，备份恢复演练（密钥目录铁律：绝不随 DB 备份回滚）。
- 提交本次工作区改动（Go 后端 + 部署适配 + 前端外观修复 + 设计文档/报告）。