# UniGate 领域架构重划分提案

日期：2026-11-11；状态：提案待评审（未采纳，未执行级联变更）；前置：Gate 2 已通过（2026-10-21，ADR-046）。

动机：用户确认「设计与实现脱节」——设计侧 7 上下文/7 聚合根与实现侧平铺单包 20 文件（6554 行）不对应。本提案以已交付实现为证据源，全面重划 BA → AA → 领域模型 → 代码结构四个层面，先出方案后定落地深度。

## 1. 取证摘要（实现侧事实）

来源：backend/src/main/java/com/unigate 全部 20 文件、V1__unigate.sql（25 表）、openapi-unigate.json（85 操作）。

### 1.1 实际运行结构

```
ApiController（611行：契约校验+幂等 ug_operation+审计+统计寄生 reports()）
  → 6 领域服务 + 2 worker → Store（SQL+每活动密钥加密）
EventService(364) ←无领域依赖叶子
IdentityService(658) → SecurityAttempts/PasswordHasher
PartyService(723) → Event/Identity
GateService(476) → Party/Identity
IntegrationService(1704) → Party/ProviderAdapter/Notification   ← 顶层聚合
NotificationService(466) → Party
SourcePullWorker(243) → ProviderAdapter；LifecycleWorker(35) → Integration
```

无循环依赖；依赖方向 Integration > Party > Event/Identity。

### 1.2 边界异味（文件:行 段为证据）

| # | 异味 | 证据 |
|---|---|---|
| E1 | IntegrationService 吞下 6 个上下文：连接/OAuth、来源+导入全流程、平台任务、导出、交付登记、数据清理 | IntegrationService L56-1699；另兼 dispatch 转发 Sms/Recovery（L78-79） |
| E2 | PartyService 混 7 类职责：资格+凭证+自助会话+现场扩容（WalkIn/Increase，URL 在 /gate 下）+手机验证核销+邮件入队+名单读模型 | PartyService L113-695 |
| E3 | 统计寄生在控制器：Statistics 实时聚合 ug_party/ug_admission/ug_outbox | ApiController.reports L468-521；另 PartyService.list L666-695 与 IntegrationService 导出各有一份过滤聚合 |
| E4 | 多服务共写表（事务边界弥散）：ug_job、ug_challenge 各 4 写方；ug_event、ug_party、ug_session、ug_event_grant 各 3 写方；ug_credential、ug_device、ug_operation、ug_import_batch/row、ug_cleanup 各 2 写方 | V1 迁移+各服务 SQL；PartyService.allocate L101-111 写 ug_event.allocated_count；EventService L141-160 预建 ug_cleanup；NotificationService.verify L190-200 越界吊销凭证/会话 |
| E5 | 幂等两套：ApiController 通用幂等壳 + GateService.GateOperation 检票侧重放 | ApiController L145-244；GateService L289-326 |
| E6 | 设计的异步 ExportJob 未实现：导出同步内联完成（newJob 直接 RUNNING→SUCCEEDED） | IntegrationService L1215-1280 |
| E7 | ug_outbox 有写无发布：仅统计 SQL 反读 | PartyService L622-628；ApiController L604-610 |

### 1.3 URL 域（85 操作的天然聚类）

identity 13、event 8、public 13、enrollment 8、gate 13、integration 7、intake 9、platform 5、delivery 3、report 4、lifecycle 2（API §4.2.1-4.2.85）。URL 空间已勾勒出比 Java 服务更细的边界。

## 2. 现划分问题清单

| # | 层面 | 问题 | 依据 |
|---|---|---|---|
| P1 | 领域模型 | ActivityAdmission 过重：Event 规则、Party 资格、入场核销是三个生命周期，实现为三个服务（Event/Party/Gate），设计塞进一个上下文 | D2 模型「本域信息」表 vs §1.1 |
| P2 | 领域模型 | Reporting 无独立写模型，实际是三处各自实现的查询（控制器统计/名单 list/归档匿名快照），不构成边界 | E3；匿名快照在 IntegrationService.runCleanup L1557-1579 |
| P3 | 领域模型 | Identity/Connection 合并上下文，实现里 Identity(658行)与 Connection(Integration L186-322+ProviderAdapter 420行) 完全不搭界；DeviceSession 归 Identity 但终端生命周期实现在 GateService L84-176 | D2 §业务域与上下文 vs §1.1 |
| P4 | 领域模型 | 幂等/审计/任务队列/挑战/发件箱（平台过程底座）无对应上下文，寄生在 ApiController 与 IntegrationService | E1/E3/E4/E5 |
| P5 | BA | L3「凭证交付」独立成关键业务，但凭证是 RegistrationParty 聚合内实体（设计自认），实现由 PartyService 一并承担，凭证操作散布在 /public 与 /enrollment | D2 领域对象表 AdmissionCredential 行；§1.3 |
| P6 | BA | L3「现场准备与控制」在代码中仅是 Event 状态机转移+终端状态，与核销同属一个现场会话概念（BA 术语表现有「现场会话」），拆两个 L3 使 AA 核心域碎化 | EventService.EventTransition；BA 附录B |
| P7 | AA | 「连接与任务」模块把外部连接（OAuth 秘密保管）与平台任务（队列执行）捆绑；实现中任务队列被 4 个服务直写 | AA 2.4.2；E4 |
| P8 | 代码 | 平铺单包：模块边界只存在于文件名；6554 行 20 文件中 IntegrationService 1704 行不可维护 | §1.1 |

## 3. 新划分提案

### 3.1 BA：4 L2 / 6 L3 / 17 L4（阶段不动，L3 两处合并）

| L2 | L3（新） | L4（不变，仅归属调整） | 变化 |
|---|---|---|---|
| 筹备 | 活动规则设定 | 活动基础信息设定、报名及留存规则设定（2） | 不变 |
| 报名与名单准备 | 报名及名单接入 | 内置报名接收、外部名单接收与核对（2） | 原「报名及名单接收」更名，范围不变 |
| 报名与名单准备 | 资格、名额与凭证 | 资格审核与名额确认、报名人数变更与取消、状态查询与领票、邮件通知与找票协助（4） | **合并**原 2.2.2「资格与名额管理」+2.2.3「凭证交付」 |
| 现场入场 | 现场检票与控制 | 现场就绪与开启、暂停与恢复、组码识别及分批核销、现场增额及补录、无效凭证、重复及误检处理（5） | **合并**原 2.3.1「现场准备与控制」+2.3.2「入场核销与异常处置」 |
| 收尾与留存 | 结果统计与交付 | 现场封存与结果核对、统计查看与明细交付（2） | 不变 |
| 收尾与留存 | 留存与清理 | 明细到期处置、外部副本处置记录（2） | 不变 |

L4 总数保持 17；业务范围、角色、术语表不变。合并依据：P5（凭证=Party 聚合内实体，同一服务同一事务边界）、P6（现场会话是单一业务概念）。

### 3.2 AA：DDD 归类调整（模块数 8→6 业务 + 3 平台，叶节点仍 23）

| 分层 | 功能模块（新） | 变化说明 |
|---|---|---|
| 核心业务域 | 资格、名额与凭证；现场检票与控制（2 模块 9 子模块） | 两大不变量（名额占用、核销唯一性）所在；合并后核心域更聚焦 |
| 支撑业务域 | 活动规则设定；报名及名单接入；结果统计与交付；留存与清理（4 模块 8 子模块） | 活动规则设定维持支撑（规则为约束输入，不变量在核心域；EventStatus 6 值状态机为控制手段） |
| 基础数据域 | 无独立模块（维持） | 枚举随公共规约 |
| 平台支撑域 | 身份与授权（账号与活动授权、终端绑定与会话）；外部连接（连接凭据保管）；平台运行（任务与队列、操作幂等与结果、审计持久化）（3 模块 6 子模块） | **拆分**原「连接与任务」：连接归外部连接，任务/幂等/审计归平台运行；平台子模块总数 6 不变 |

### 3.3 领域模型：8 上下文 + 1 平台内核（替换原 7 上下文）

| 上下文（新） | 分层 | 聚合根 | 表（所有权） | URL 域 / 操作数 | 实现锚点 |
|---|---|---|---|---|---|
| EventAdmin 活动与规则 | 支撑 | Event | ug_event | event 8 | EventService |
| Enrollment 报名与资格 | **核心** | RegistrationParty（Review/Credential/SourceLink 为内实体） | ug_party、ug_party_review、ug_credential、ug_source_link | public 9 + enrollment 8 = 17 | PartyService 核心 |
| Gate 现场检票 | **核心** | Device、Admission（追加事实+更正） | ug_device、ug_admission、ug_admission_correction | gate 13 | GateService + PartyService 的 WalkIn/Increase（应迁入） |
| Intake 名单接入 | 支撑 | ImportBatch（Row 过程数据） | ug_import_batch、ug_import_row、ug_source | intake 9 | IntegrationService 导入段 + SourcePullWorker |
| Connection 外部连接 | 通用 | SourceConnection（ConnectionEvent 实体） | ug_connection、ug_connection_event | integration 7 | IntegrationService 连接段 + ProviderAdapter |
| Delivery 通知与交付 | 支撑 | ExportJob（通知尝试/交付登记为过程） | ug_export、ug_delivery | delivery 3 + 验证找回 4 + report 4 = 11 | NotificationService + IntegrationService 导出/交付段 |
| Lifecycle 留存与清理 | 支撑 | CleanupRun（DeletionLedger 实体） | ug_cleanup、ug_deletion_ledger | lifecycle 2 | runCleanup + LifecycleWorker |
| Identity 身份与访问 | 通用 | Account（Session/EventGrant 实体） | ug_account、ug_session、ug_event_grant | identity 13 | IdentityService + Context |
| 平台内核（横切底座，非业务上下文） | 平台 | Operation/Job/Challenge/Outbox 为过程记录，无业务聚合 | ug_operation、ug_audit、ug_job、ug_challenge、ug_outbox + 统计读模型 | platform 5 + Statistics 1 | ApiController 幂等/审计壳、任务队列、挑战发放 |

操作数合计 85 ✓。与原 7 上下文的映射：ActivityAdmission 拆为 EventAdmin+Enrollment+Gate（P1）；Reporting 解散——统计查询归各上下文自有读侧+Delivery 汇总视图，归档匿名快照归 Lifecycle（P2）；Identity/Connection 拆开，DeviceSession 改名 Device 归 Gate（P3）；新增平台内核承接原无主能力（P4）。

**跨上下文事务契约（保留现状、显式化）**：核销/更正/现场增额在同一 RC 事务内更新 Party 余额与 Admission 事实（Gate↔Enrollment）；容量分配同事务更新 ug_event.allocated_count（Enrollment→EventAdmin）；这两条沿用 D2「子流程 I/O 契约」表，作为上下文间唯一允许的共写白名单，其余共写（E4 清单）改为经属主服务方法。

### 3.4 代码结构：模块化单体目标包

```
com.unigate
├── api/        ApiController（瘦身：仅路由/契约/鉴权/幂等壳；统计迁出）
├── platform/   幂等 Operation、审计、Job 队列、Challenge、Outbox、统计读模型
├── shared/     Store、Crypto、EventKeyVault、Contract、ApiErrors/ApiException
├── identity/   IdentityService、Context、SecurityAttempts、PasswordHasher
├── event/      EventService
├── enrollment/ PartyService（报名/资格/凭证/审核/自助）
├── gate/       GateService + WalkIn/Increase（自 PartyService 迁入）
├── intake/     导入+来源配置+SourcePullWorker
├── connection/ 连接/OAuth + ProviderAdapter
├── delivery/   NotificationService + 导出/交付登记
└── lifecycle/  runCleanup + LifecycleWorker + DeletionLedger
```

20 文件迁移映射：ApiController→api（统计段→platform）；IdentityService/Context/SecurityAttempts/PasswordHasher→identity；EventService→event；PartyService→enrollment（GateWalkIn/GateIncrease→gate）；GateService→gate；IntegrationService 拆 5 份→intake/connection/delivery(导出交付)/lifecycle/platform(任务段)；NotificationService→delivery；SourcePullWorker→intake；LifecycleWorker→lifecycle；Store/Crypto/EventKeyVault/Contract/ApiErrors/ApiException→shared；其余随属主。

**边界修复清单（重划分顺带修正，按优先级）**：
- R1（随迁移必做）：WalkIn/Increase 迁入 gate；统计自 ApiController 迁入 platform 读模型。
- R2（小改）：NotificationService.verify 恢复流程改为调 Enrollment 凭证契约（不再直写 ug_credential/ug_session）；SecurityAttempts 保留（经属主方法改 ug_device 计数）；EventCreate 预建 ug_cleanup 改调 lifecycle。
- R3（可后置）：幂等双轨合并（GateOperation 重放统一走平台内核）；ug_outbox 发布器或删除该表；导出转真异步（补 E6）。

## 4. 影响面与级联变更清单

| 触及物 | 变更 | 量级 |
|---|---|---|
| BA business-arch.md | v0.4→v0.5：L3 8→6（两处合并），L4 17、附录、依赖表同步 | 小 |
| AA application-arch.md | v0.4→v0.5：模块 8+3→6+3，调用关系重标（模块编码按「删除不复用」规则重排） | 中 |
| FL function-list.md | v1.0→v1.1：43 项功能全部重挂新模块 ID（条目数不变） | 中（机械） |
| D2 领域模型 | v1.0→v2.0：上下文 7→8+内核、聚合根 7→8（+Account、+Device，DeviceSession 更名迁移） | 中 |
| API api-unigate.md / openapi | v0.3→v0.4：仅 §1.3 模块 ID 与 §5.2 追溯表重挂；路径/DTO/枚举 **零变更**（URL 域本就与新上下文一致） | 小 |
| data-model-unigate.md | §1 聚合归属表更新；DDL 不变 | 小 |
| 枚举池（29 组） | 不变（名称未动） | 无 |
| 代码 | 若采纳代码深度：包重构+R1-R3+测试回归（47 后端+42 前端）+Gate 2 复查；若仅文档深度：代码不动 | 大 / 无 |
| Gate 1 锚点 | ER-G1-01（8/17/23/43）、DS-G1-03（7 聚合根）、19 份产物 SHA256 失效，需按新基线重跑 | 中 |
| Gate 2 锚点 | 仅在改代码时失效（16 跨栈+浏览器 8 组重跑）；纯文档变更不影响 | 视深度 |
| INDEX.md / CHANGELOG.md / ADR | INDEX 状态注记更新；CHANGELOG 按日条目追加；采纳时记 **ADR-047**（当前最新 ADR-046） | 小 |

## 5. 决策点（待用户拍板）

| # | 决策 | 选项 | 建议 |
|---|---|---|---|
| D-1 | BA 合并幅度 | a) 两处 L3 合并（本提案）；b) 仅合并凭证入资格、保留现场两个 L3（7 L3）；c) BA 不动（8 L3），仅重划上下文 | a——与聚合/服务边界一致；c 会使 AA 模块与上下文长期错位 |
| D-2 | 活动规则设定分层 | a) 维持支撑业务域（本提案）；b) 升核心（EventStatus 6 值状态机驱动全部写入窗口） | a——不变量与业务价值在名额/核销，规则是约束输入 |
| D-3 | 终端归属 | a) Device 聚合归 Gate、终端授权留在平台「身份与授权」（本提案）；b) 终端整体升业务模块（BA 需 +1 L4） | a——避免 BA 层级膨胀 |
| D-4 | Reporting 处置 | a) 解散为各上下文读侧+归档快照归 Lifecycle（本提案）；b) 保留独立读模型上下文并补物化 | a——现实从未有独立写模型；b 需新增开发量 |
| D-5 | 落地深度 | a) 仅文档（BA/AA/FL/D2/API 重挂+ADR-047+Gate1 重跑）；b) 文档+代码包重构（+R1/R2+Gate2 复查）；c) 文档+代码+R3 全量（含异步导出/幂等合并/outbox 处置） | 先 a 后 b 分两步走；R3 单列后续迭代 |

## 6. 建议下一步

1. 逐项确认第 5 节决策点（可一次回复，如「D-1a D-2a D-3a D-4a D-5a」）。
2. 采纳后记 ADR-047，按第 4 节级联清单执行（D1 变更管理 Step 2 向下级联：BA→AA→FL→D2→API）。
3. 代码深度（D-5b/c）另行排期，迁移以「先移文件不改行为」为第一刀。
