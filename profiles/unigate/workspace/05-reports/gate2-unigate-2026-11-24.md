# Gate 2 — UniGate Go 重写代码交付门控

日期：2026-11-24（Asia/Shanghai）。判定：**WARN**（全部 CRITICAL 通过；4 项 WARN/边界需主会话确认后交付）。

依据：ADR-048 Java→Go 全量重写；引擎 validation Gate 2 + engine-rules GR-01~06 / ER-04；激活栈 go-vue3 backend-rules G-01~G-08（`.claude/rules/backend-rules.md`）。**异栈适配**：validation 通用 BE-01~34 / FE-xxx 清单属另一套 Java+Vue 框架，本次不适用，未执行、不声称通过。前端本次按契约冻结口径沿用上次 Gate 2（2026-10-21，Java，PASS）结论，不重扫前端规则；但工作区发现 5 个未提交前端外观改动，见 W-4。

扫描范围：`backend/cmd/unigate` + `backend/internal/{api,service,store,contract,errcode,config}` 的全部非 `_test` Go 源码（36 个源文件，另有 15 个测试文件）。遗留 Java 树（`backend/pom.xml`、`backend/src`、`backend/target`）不在本次 Go 门控扫描范围。

---

## 1. 冻结契约校验

| 项 | 结果 |
|----|------|
| `profiles/unigate/workspace/02-detailed/api/openapi-unigate.json` SHA256 | `cc185ea09d0f90ad1843e5716e7c5843566767ccce6e367dc4f26e667de6612b` == 冻结值 **PASS** |
| `git diff` / `git status` 该文件 | 空（自 14d226a 提交后未变） **PASS** |
| 后端内嵌副本 `backend/internal/contract/openapi-unigate.json`（go:embed） | SHA256 与冻结值逐字节一致 **PASS** |
| 前端 `frontend/src/generated/contract.json` `sourceSha256` | == 冻结值；85 操作 name/method/path 与 x-operation-index 全等；299 schema 除文档性 `description` 外逐字段一致；29 枚举池全等 **PASS** |

## 2. G-01~G-08 逐项结果（go-vue3 backend-rules §8）

| # | 检查 | 结果 | 证据/人工核对 |
|---|------|------|----------------|
| G-01 | 无 shell 拼接 | **PASS** | `exec.Command*` 全仓 0 处；`sh/bash/cmd -c` 0 命中。本服务无外部命令调用面 |
| G-02 | 无 SQL 拼接 | **PASS** | `Sprintf+(SELECT|INSERT|UPDATE|DELETE)` 0 命中。store 层人工复核：全部 `$n` 参数化；唯一拼接 `store/store.go:554` `"select * from ug_event where id=$1"+lock`，`lock` 仅取固定常量 `" for update"`/`" for share"`/`""`（store.go:548-553），非用户输入，标识符无动态化 |
| G-03 | 无硬编码凭据 | **PASS** | grep 唯一命中 `service/notification.go:665`：`partyLink(base, query, change)` 以函数参数变量拼 URL fragment（`queryToken=`/`changeToken=`），非凭据字面量 → **误报确认**。运行时凭据全部来自环境变量（config.go：DBPassword/MasterKey/SMSToken/MailPassword/BootstrapPassword 均 `os.Getenv`，无默认值）；测试夹具为固定非生产向量（crypto_test.go:15、integration_test.go:39-42 有注释声明） |
| G-04 | SQL 只在 store 层 | **PASS** | 大写+小写两轮 grep：store 外 0 条 SQL。case-insensitive 复核命中仅为 Go `select {}` 语句（main.go:129/163、workers.go:26）与注释（workers.go:16）。分层依赖单向：service/store 不 import api，store 不 import service，api 无 `.Pool./.Query(/.Exec(` 直连 |
| G-05 | 无 panic 滥用 | **PASS** | 全仓仅 3 处，逐个核对均规则允许：① `api/middleware/middleware.go:93` 重抛 `http.ErrAbortHandler`（net/http 哨兵，令服务器中断连接的标准模式；其余 panic 由 Recovery 转冻结 500 包络）；② `store/crypto.go:37`、③ `store/store.go:189` `crypto/rand` 不可用不变式（受支持平台不可能失败）。handler/service 业务路径无 panic，错误一律 error 返回 |
| G-06 | goroutine 退出路径 | **PASS** | 仅 main.go 3 处 `go func`，库/service 层无裸起：① main.go:113 worker 循环——`w.run(ctx)`，ctx 源自 `signal.NotifyContext`，`wg.Add/Done` 配对，shutdown 先 `cancel()`；三个 worker（SourcePull/Lifecycle/Notification）均经 `runPollLoop`（workers.go:21-37、notification.go:717-718）`select ctx.Done()` 退出；② main.go:125 `srv.Serve`——经 `srv.Shutdown(15s)/Close` 退出，缓冲 chan 无泄漏；③ main.go:159 `waitWorkers`——cancel 后 worker 必返回，`wg.Wait` 完成即退出，10s 超时仅告警不阻塞进程退出。无 `time.AfterFunc/time.Tick` |
| G-07 | 编译+vet+test 全绿 | **PASS** | `go build ./...` OK；`go vet ./...` OK；`set -a && . ../.local/go-test.env && set +a && go test ./... -count=1`：6 包全 ok，**45 个测试函数 / 110 个含表驱动子测试全部 PASS，0 FAIL、0 SKIP**（真实 PostgreSQL 5433 / unigate_go_test）。补跑 `go test ./... -count=1 -race` 全绿（api 包 134s），无数据竞争 |
| G-08 | 用户可见消息无 AI 味 | **PASS** | emoji / 全角 `！` / 空洞夸赞词（太棒/恭喜/操作成功完成/这是一个…）grep 0 命中（notification.go:603 的正则命中为 `!=` 运算符误报）。人工通读全部 246 条中文字面量：错误消息说清原因与出路（如"剩余名额不足，整批未提交""原操作正在处理中，请沿用原编号查询""活动告知已更新，请重新确认"），无系统腔/翻译腔。INFO：成功包络 `message:"操作成功"` 为 Java 基线冻结 wire 常量（前端只判 `code==='OK'`），非本次新增文案 |

## 3. 引擎 GR 条目

| # | 检查 | 结果 | 证据 |
|---|------|------|------|
| GR-01 | 激活链规则扫描必执行 | **PASS** | 引擎 GR ⊕ go-vue3 G-01~08 逐项执行（本报告 §2）；前端沿用上次 Gate 2 结论（契约冻结、wire 零改动），未跑前端全量扫描——范围外口径已注明。W-4 的外观改动不属 wire 变更 |
| GR-02 | 编译命令（栈 manifest） | **PASS** | go-vue3 `profile.yaml` `stack.build.command = "go build ./... && go vet ./..."`，已执行通过 |
| GR-03 | 接口 E2E 补测试 | **WARN** | 全量重写视同全部接口新增：85 操作中 42 个有集成测试直接执行，43 个无（见 §5 W-1）。go-vue3 规则未设覆盖率数值 CRITICAL 门槛，按任务口径列 WARN/边界 |
| GR-04 | 测试代码编译+运行 | **PASS** | 测试编译通过且真实 PG 运行全绿（含 -race）；集成夹具与 Java @SpringBootTest 同构（UNIGATE_TEST_DB_URL 守卫、每测试 truncate ug_*、隔离管理员重置） |
| GR-05 | 前后端契约逐一对齐 | **PASS** | 见 §4 |
| GR-06 | 对齐检查先记录再交付 | **PASS** | 本报告即记录，先于交付标记生成 |

## 4. GR-05 契约对齐明细

| 项 | 后端事实 | 前端事实 | 结果 |
|----|----------|----------|------|
| 路由注册数 | `api/router.go:50-62` 遍历 `contract.Operations` 逐一注册，重复 method+path 即启动报错；`router_test.go:34` 断言注册数 == 85 且 method/path 全匹配 | `contract.json` 85 操作 | **85 == 85 PASS** |
| 成功包络 | `execute.go:255-272`：`{code:"OK", message:"操作成功", data, requestId}` + 操作真实状态码（200，异步 202） | `api.ts:34` 判 `response.ok && result.code==='OK'`，取 `result.data/message/requestId/status` | PASS |
| 错误包络 | `middleware.go:127-141`：真实 HTTP 状态 + 字符串错误码 + `data:{}` + requestId；`errcode.go` canonical 映射复刻 ApiException（IDEMPOTENCY_PAYLOAD_MISMATCH 强制 400 等），`errcode_test.go` 100% 覆盖 | `api.ts:5-7,34` ApiError(code:string, message, status, requestId)；mutation 5xx/OPERATION_IN_PROGRESS 标记 uncertain | PASS |
| 未知路由 | `router.go:63-72` NoRoute/NoMethod → 404 `RESOURCE_NOT_FOUND` 冻结包络 + 安全头（router_test 实测） | fetch `redirect:'error'`；后端 RedirectTrailingSlash/FixedPath=false | PASS |
| 分页 | `store/store.go:442-468` Page → `{items,total,page,pageSize}`（默认 page=1/size=20，上限 100）；SelectorPage → `{items,nextCursor}` | 12 个 *Page schema（EventPage/PartyPage/GrantPage/AccountPage/ReviewPage/ImportRowPage/AdmissionPage/AuditPage/JobPage/DeliveryPage/ConnectionPage/GateCandidatePage 等）含 page/pageSize，由同一冻结 schema 生成类型 | PASS |
| Cookie | `service/identity.go:32-49`：`ug_staff` → `SameSite=Lax`；其余（ug_device 等）→ `SameSite=Strict`；Path=/; Max-Age=43200; HttpOnly; Secure 由配置 | `api.ts:31` `credentials:'include'` | PASS |
| schema/枚举 | 内嵌冻结 OpenAPI：299 schema、29 枚举池 | contract.json 299 schema（仅去除文档性 description）、29 枚举全等 | PASS |
| 二进制下载 | `execute.go:276-285` ExportDownload 流式 + Content-Disposition | `api.ts:32` 非 JSON 响应取 blob | PASS |

## 5. 覆盖评估（对照 Java 基线 gate2-unigate-2026-10-21）

### 测试通过数

- Go 本次：**45 个测试函数 / 110 个含子测试，0 失败 0 跳过**；`-race` 全量复跑全绿。包分布：api 10（含 router 1 + 集成 9）、config 2、contract 2、errcode 4、service 4、store 23。
- Java 基线：47 测试（真实 PG，含并发竞争）；前端 42 测试（本次未重跑，沿用基线）。

### 单元字节兼容/回归套件（全部通过）

| 域 | 测试 | 说明 |
|----|------|------|
| crypto | 7 | Seal 布局冻结（0x01+12B IV+128bit tag，独立 AES-GCM 解密验证）、HMAC 已知答案、防篡改/版本拒绝、签名往返、Token 43 字符 |
| keyvault | 6 | DEK 布局、跨活动拒绝、撤销墓碑、短/篡改拒绝、非法 UUID |
| password | 6 | Unicode 预哈希已知答案（"密码Pässw0rd!"）、$ug$ 前缀+cost、200 字符长密码预哈希绕过 72 字节、旧裸 bcrypt 兼容、旧哈希 >72 字节拒绝 |
| contract | 2 | 真实冻结契约校验（枚举/$ref/required/additionalProperties）+ 分支表驱动 |
| errcode | 4 | canonical 映射全表、typed-nil 回归、AsError 解链 |
| safeCell/UTF-16 | 4 | 公式注入拦截、表头校验、UTF-16 长度、Java 时间单元格 |
| NormalizeDBURL | 2 | JDBC→pgx 归一化 + 非法输入 |
| RawJSON | 1 | 嵌套 map/slice 中 jsonb 原文逐字节序列化回归（防 base64 腐蚀） |
| jsontime | 3 | UTC/RFC3339、JSONReady 时间转换 |

### 集成链路（真实 PG 5433 + httptest 真实 HTTP，全部通过）

身份（bootstrap/login/session/me/改密撤销会话/一次性重置临时密码+强制改密/未知账号拒绝）、报名（PublicEvent/SubmitRegistration/MyParty/MyCredential/PartyExchange/RegistrationResult）、核销（TicketResolve/GateCommit/原 ID 重放幂等/GateLookup/Records/Operation/Correct/WalkIn/Increase/kiosk 模式+审核守卫）、导入（BatchUpload/Preview/Rows/DuplicateResolve/Commit 原子提交/错误行/策略漂移+阶段守卫）、Statistics（itGateStats 入场不变量）、grant 字段投影部分（name 掩码 O**、无 phone grant 隐藏手机号）。

### 覆盖率（`go test ./... -coverpkg=./internal/...` 全测试并集，语句覆盖）

| 包 | 覆盖 | 备注 |
|----|------|------|
| internal/api | 69.2%（110/159） | 传输壳 |
| internal/api/middleware | 72.0%（36/50） | |
| internal/contract | 94.5%（120/127） | |
| internal/errcode | 100%（37/37） | |
| internal/store | 62.2%（515/828） | crypto/keyvault/password 高覆盖；repo 由集成驱动 |
| internal/service | **34.0%（1542/4535）** | 对照 Java Service 87.59% 显著偏低 |
| internal/config | 15.0%（9/60） | Load 未单测（集成间接走 NormalizeDBURL） |
| internal 合计 | 40.9%（2369/5796） | Java 基线 Service 87.59% / Controller 78.48% |

go-vue3 backend-rules §7 要求核心包表驱动测试（已满足），未设覆盖率数值 CRITICAL 门槛；通用 validation 的 Service 80%/Controller 60% 表属异栈清单，不作强制判定，仅以 Java 基线作对照参考。

### WARN 与边界

- **W-1 覆盖广度缺口（WARN，需裁决）**：85 操作中 43 个无集成测试执行——整条导出/交付/清理链（ExportCreate/ExportDetail/ExportDownload/DeliveryList/DeliveryRecord/CleanupDetail/CleanupRetry）、连接与来源管理及其后台 worker（ConnectionSave/Authorize/Callback/Revoke、SourceSave/SourcePull、JobList/JobDetail/JobRetry、EmailResend）、账号与授权管理（AccountList/AccountUpdate、GrantList/GrantRevoke、Logout）、名单自助变更/取消（MyPartyChange/MyPartyCancel/PartyChange/PartyCancel）、短信/找回验证（SmsStart/SmsVerify/RecoveryStart/RecoveryVerify）、CredentialReissue/Revoke、EventList/EventUpdate/EventOptions 等。service 语句覆盖 34.0% vs Java 87.59%。Java 基线 SW-03 要求的"任务/导出/交付/清理、来源/预览/去重/提交"完整链路测试未移植到 Go 集成层。worker 退出路径仅经 G-06 代码核对，无运行态测试。
- **W-2 并发竞争回归未移植（WARN/边界）**：Java 基线的末名额竞争、16 并发读取、24 会话/匿名读取、多标签 CSRF 轮换在 Go 测试中无对应（测试代码无 WaitGroup/t.Parallel/并发 goroutine）。服务端约束（行锁 FOR UPDATE、版本守卫、幂等摘要）已移植并被顺序用例（重放幂等、冲突 409） exercised；`-race` 对已执行路径无竞争告警，但不等于竞争语义回归。
- **W-3 浏览器 E2E 未对 Go 二进制重跑（边界）**：基线 8 组浏览器证据（kiosk 布局、扫码枪重获焦、丢弃响应重试、Storage 无 PII 等）针对 Java jar。本次以 HTTP 级集成测试替代 API 层验证；真实浏览器/真机/现场容量仍属部署验收，不作为本地代码门控通过证据。
- **W-4 前端工作区非零改动（WARN，需裁决）**：与"前端本次零改动"前提不符——5 个未提交外观改动（`frontend/src/App.vue`、`components/ResultDisplay.vue`、`style.css`、`views/LoginView.vue`、`views/ScannerView.vue`，合计 11+/11-）：去除 eyebrow 装饰与"欢迎回来/欢迎到场"问候、空状态文案补出路（ER-04 方向）、sticky 首列背景修复。**wire 零影响**：`src/lib/api.ts` 与 `src/generated/contract.json` 均未变，无操作/字段/包络改动。本次未重跑前端 build/test。另有未跟踪 `frontend/.mimosa/` 工具目录。需主会话决定：随交付提交（应补前端类型检查/构建验证）或暂存剥离。
- **INFO-1**：`cmd/unigate/main.go:178` `fmt.Printf` 为一次性 `--reset-password-for` CLI 模式的有意 stdout 输出（复刻 Java reset-admin.sh 行为，非调试残留），🟡 风格规则例外，理由成立。
- **INFO-2**：遗留 Java 树仍在 backend/（pom.xml/src/target/unigate.exe）；`IdentityService.java` 有 +28 行未提交改动（resetPasswordFor runner，与 Go runReset 对齐）。不在 Go 扫描范围；交付打包应仅含 Go 二进制，Java 残留建议主会话另行处置。
- **INFO-3**：成功包络 message "操作成功" 为冻结 wire 常量（Java  parity），非新增 AI 味文案。

## 6. 总结

**判定：WARN**

- CRITICAL 违规：**0**（G-01~G-08 全 PASS；GR-01/02/04/05/06 PASS；冻结契约 sha256 与 git diff 校验 PASS）
- 测试：45 函数 / 110 含子测试全过，0 失败 0 跳过；build/vet/-race 全绿
- WARN/边界：**4**（W-1 覆盖广度、W-2 并发回归未移植、W-3 浏览器 E2E 未对 Go 重跑、W-4 前端未提交外观改动）
- 需主会话裁决项：
  1. W-1：接受覆盖缺口为交付边界，或要求补齐导出/交付/清理、连接/来源+worker、账号/授权管理等 43 操作的集成测试后再交付；
  2. W-4：5 个前端外观改动的处置（提交并补前端验证 / 暂存剥离）；
  3. W-2/W-3：并发竞争与浏览器/真机验收作为本地代码交付边界的确认。

按引擎判定语义，WARN 项经用户确认后可继续交付；确认记录应写入本报告。本报告不替代部署验收（真实平台账号、安卓硬件、现场容量、备份恢复另行验收，口径同基线）。

## 7. 复现入口

```bash
cd backend
go build ./... && go vet ./...
set -a && . ../.local/go-test.env && set +a
go test ./... -count=1            # 45 函数 / 110 含子测试
go test ./... -count=1 -race      # 竞争检测复跑
go test ./... -count=1 -coverpkg=./internal/... -coverprofile=cover.out  # 并集覆盖
sha256sum ../profiles/unigate/workspace/02-detailed/api/openapi-unigate.json
# 期望 cc185ea09d0f90ad1843e5716e7c5843566767ccce6e367dc4f26e667de6612b
```

## 8. 覆盖补齐重判（2026-11-24 续，主会话）

**用户裁决**：补齐集成覆盖再交付（否决"先交付记录边界"）。

**补齐批次**（复用 HTTP 集成骨架，mock provider/SMTP sink 零外网）：
- c-1 账号/授权管理 6 测试（Account CRUD/最后管理员守卫/强制改密/Grant 选择器/授予/撤销/最后负责人/负责人范围/READER 投影/过期）
- c-2 名单自助/凭证/Logout 5 测试（自助变更取消/守卫/REVIEW 增员差额/CredentialReissue/Revoke/旋转/Logout）
- c-3 导出/交付/清理 3 测试（受控导出 CSV/XLSX/交付登记披露密封/清理六范围/BACKUPS 诚实失败/ARCHIVED 先于物理清理）
- c-4a 连接/来源/SourcePull worker 7 测试（连接 CRUD/OAuth/资源字段/来源拉取/重试阶梯/resultRef 续导入，mock HTTPDoer）
- c-4b 短信/找回/通知 5 测试（SmsStart/Verify/暴力计数独立事务/Recovery 匹配/凭证旋转/EmailResend 投递/失败隔离，SMTP sink+SMS mock）
- c-5 并发竞争 6 测试（末名额竞争/并发读快照/多标签 CSRF 轮换/同 operationId 并发幂等，-race）

**最终验证**：6 包全过；**142 测试函数 / 41 集成测试，0 FAIL 0 SKIP**；`go build/vet` 干净；`-race -timeout 30m` 无 DATA RACE（api 包 race 下约 14min，超默认 10m 超时，故复跑须 `-timeout 30m`；CI 用非 race `go test` 不受影响）。

**覆盖率**：service 语句覆盖 **34% → 69.9%**（api 集成 69.6% + service 单元合并，`-coverpkg=./internal/service`）。对照 Java 基线 87.59% 仍有差距，剩余主要为防御性/错误分支；go-vue3 栈无数值覆盖 CRITICAL 门槛，记为边界而非 FAIL。

**测试驱动修复的 5 个真实实现 bug**（均已修+回归测试）：
1. `store/tx.go` 事务 `SET statement_timeout=$1`（SET 不接受绑定参数）→ `set_config('statement_timeout',$1,true)`。
2. `errcode.Require` 返回 `*Error` 致 typed-nil（合法输入误判错误）→ 返回 `error`。
3. `store.RawJSON`（[]byte）嵌套 map 被 json.Marshal base64 化（导入行 raw 信封损坏、availableColumns 恒空）→ 加 `MarshalJSON` verbatim。
4. `service/event.go` EventCreate 用 Go map 随机迭代播种清理范围（首次 CleanupDetail 顺序不确定，违反冻结池顺序）→ canonical 遍历。
5. `service/gate.go` gateRecords 将 kiosk 核销 NULL actor_id 渲染为 ""（违反 AdmissionView.actorId uuid 格式）→ store.Str 省略可选键。

**WARN 项更新**：
- W-1 覆盖广度：部分缓解（69.9%，全 85 操作组+并发已覆盖），剩余防御分支记边界。
- W-2 并发竞争回归：**已补齐解除**（c-5，-race 净）。
- W-3 浏览器 E2E/真机：仍属部署验收边界（HTTP 集成替代 API 层；真机/现场容量/备份恢复另行验收）。
- W-4 前端 5 处外观改动：**已验证**（42 测试+build 通过，wire 零影响），随交付提交。

**交付状态**：Java Maven 结构（pom.xml/src/target）已移除（git 14d226a 可恢复）；backend/ 为纯 Go（cmd/unigate + internal/，内嵌迁移 0001_unigate.sql）；部署/dev 工具链已适配 Go 单二进制（CI setup-go 按 go.mod、Linux 交叉编译、systemd/reset/nginx 统一 PORT=8090、文档去 JRE、local.py 改 go run/test）。

**重判定：WARN**（0 CRITICAL；余 W-1 覆盖边界、W-3 部署验收边界）。按用户"继续并记录"策略放行本地代码交付。本报告不替代部署验收。

**契约基线同步注记（rebase 整合）**：本报告正文的冻结校验针对 Go 重写时的契约快照 sha256 `cc185ea0…`。整合并行会话 ADR-047 域重划分（远端 896a831）后，仓库契约采用重划分基线 sha256 `11bbf80b…`（API v0.4）。两者经逐结构比对：85 操作名/路径/方法、299 schema 结构、29 组枚举池完全一致，唯一差异为 `x-aa-submodules` 域标签（33 处）与描述文字——wire 契约功能上 100% 相同。Go 内嵌契约（backend/internal/contract/openapi-unigate.json）与前端 generated（contract.json/api-types.ts）已同步至 `11bbf80b…`，同步后 Go 6 包测试（含 41 集成）与前端 build 复跑全绿。Go 重写 ADR 编号因与重划分 ADR-047 撞号，改编号为 **ADR-048**。