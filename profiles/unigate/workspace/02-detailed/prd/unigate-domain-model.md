# UniGate 领域模型

日期：2026-11-11；版本：v2.0（v1.0 于 2026-10-20 由 ADR-041 确认；v2.0 按 ADR-047 以已交付实现为证据源重划上下文与聚合）；状态：ADR-047 采用，文档深度级联完成；Gate 1 待按新基线重跑，代码包结构重构（D-5b）待排期。

## 本域信息

| 项目 | 内容 |
|---|---|
| 本域名称 | UniGate 活动报名与入场 |
| 领域分层 | 核心：Enrollment（报名与资格）、Gate（现场检票）；支撑：EventAdmin（活动规则）、Intake（名单接入）、Delivery（通知与交付）、Lifecycle（留存清理）；通用：Identity（身份访问）、Connection（外部连接）；平台内核（横切）：幂等/审计/任务队列/挑战/发件箱与统计读模型 |
| 已确认聚合根 | Event、RegistrationParty、ImportBatch、SourceConnection、Device、ExportJob、CleanupRun、Account（8 个；v1.0 的 DeviceSession 更名 Device 并由 Identity 迁至 Gate，新增 Account） |

AA 四层是应用组织，以下上下文为建模边界，不要求按上下文拆部署或引入微服务。v2.0 边界以 Gate 2 已交付实现的表属主与事务边界为证据（取证见 ../../05-reports/domain-redivision-proposal.md）：核销/更正/现场增额在同一事务更新 Party 余额与 Admission 事实（Gate↔Enrollment），容量分配同事务更新 ug_event.allocated_count（Enrollment→EventAdmin），这两条为上下文间唯一允许的共写白名单；其余原多服务共写（ug_job/ug_challenge/ug_session/ug_event_grant 等）收敛为经属主上下文服务方法，属 D-5b 代码重构范围。

## 业务域与上下文

| 上下文 | 分层 | 聚合根 | 核心职责 | 关联上下文 |
|---|---|---|---|---|
| EventAdmin 活动与规则 | 支撑 | Event | 活动窗口、阶段状态机、容量与策略快照 | Enrollment、Gate、Lifecycle |
| Enrollment 报名与资格 | 核心 | RegistrationParty | 报名组生命周期、资格决定、总占额、组余额、凭证与审核 | EventAdmin、Intake、Delivery、Gate、Identity |
| Gate 现场检票 | 核心 | Device、Admission（追加事实+更正） | 终端授权与模式、组码识别、分批核销、增额补录与纠错 | Enrollment、Identity、平台内核 |
| Intake 名单接入 | 支撑 | ImportBatch | 文件/平台来源、映射、预览、同源幂等和人工核对 | Enrollment、Connection、平台内核 |
| Delivery 通知与交付 | 支撑 | ExportJob（通知尝试/交付登记为过程） | 按资格交付凭证通知、验证找回、受控导出与交付登记 | Enrollment、Identity、Lifecycle、平台内核 |
| Lifecycle 留存与清理 | 支撑 | CleanupRun | 到期隔离、实际清理、匿名汇总快照、私密访问失效及删除依据 | EventAdmin、Enrollment、Delivery、平台内核 |
| Identity 身份与访问 | 通用 | Account（Session/EventGrant 实体） | 账号、口令、会话与活动/字段/动作授权 | 所有获准调用方 |
| Connection 外部连接 | 通用 | SourceConnection（ConnectionEvent 实体） | 平台 OAuth 秘密、获权资源与连接×活动授权 | Intake、Delivery、平台内核 |
| 平台内核（横切底座，非业务上下文） | 平台 | 无业务聚合；Operation/Job/Challenge/Outbox 为过程记录 | 操作幂等与结果重放、审计持久化、任务队列与租约、短期挑战、发件箱、统计读模型 | 所有上下文 |

v1.0 的 ActivityAdmission（Event+RegistrationParty 合一）拆为 EventAdmin/Enrollment/Gate，与实现的 EventService/PartyService/GateService 三个独立生命周期一致；Reporting 解散——统计查询归各上下文读侧并由平台内核承载统一读模型，归档匿名快照归 Lifecycle；Identity/Connection 拆分与 IdentityService/IntegrationService 连接段+ProviderAdapter 的实际归属一致。

## 领域对象分类

| 对象 | 分类 | 上级关系 | 核心字段 | 关键枚举 | 说明 |
|---|---|---|---|---|---|
| Event | 聚合根 | 无 | 时间、容量、当前占额、版本、窗口 | 活动阶段 | 拥有容量规则与阶段控制，计数不由浏览器写 |
| EventPolicy | 值对象/版本快照 | Event 内 | 字段策略、审核、自助、留存 | 审核模式 | 策略变更受既有报名/预览版本约束 |
| RegistrationParty | 聚合根 | 引用 eventId | 基础字段、批准数、待增员、资格、净核销、版本 | 资格状态 | 同一组名额与票证一致，随行不是逐人实体 |
| SourceLink | 实体 | RegistrationParty 内 | 来源/资源/源记录 ID、批次 | 来源类型 | 来源 ID 与内部人员/组身份分开 |
| EligibilityDecision | 实体/决定记录 | RegistrationParty 内 | 申请差额、结果、原因、决定人 | 决定类型/结果 | 通过时触发占额，不是先落决定后异步抢名额 |
| AdmissionCredential | 实体 | RegistrationParty 内 | token 摘要、版本、到期 | 凭证状态 | 与本组查询/修改能力分开，补发替换旧票 |
| AdmissionRecord | 追加事实实体 | Gate 上下文（引用 partyId） | 人数、操作 ID、设备/人、时间 | 事实类型 | 成功持久化结果，与网络请求状态分开；v2.0 归 Gate，组余额仍同事务更新 |
| AdmissionCorrection | 实体 | 引用原 AdmissionRecord | 数量、理由、负责人 | 更正状态 | 不删除原事实，不建离场状态 |
| ImportBatch | 聚合根 | 引用 eventId | 映射、来源资格、预览版本、选中行 | 批次状态 | 过程中的整批确认边界，不是业务人员实体 |
| ImportRow | 过程数据 | ImportBatch 内 | 原始值、错误、候选关联 | 行结果 | 到期清理，不能长期复用为第二份名单 |
| Device | 通用聚合根（Gate） | 引用 eventId | 角色模式、授权/到期/撤销 | 终端模式/状态 | 原 DeviceSession；kiosk 能力受限，不共用全能管理凭据 |
| Account | 通用聚合根（Identity） | 无 | 用户名、口令哈希、角色、启停 | 角色 | v2.0 自实体升聚合根；工作人员身份不等于活动数据授权 |
| SourceConnection | 通用聚合根（Connection） | 引用授权主体 | provider、资源/凭据引用 | 授权状态 | 真实平台身份不是公共多租户建模 |
| EventGrant | 通用授权实体（Identity） | 引用账号及 Event | 活动、字段、动作、期限 | 角色 | 管理员账户身份不等于活动数据授权 |
| CheckInOperation/IdempotencyEntry | 技术过程记录（平台内核） | 引用活动/组及操作者 | ID、载荷摘要、执行及结果 | 执行状态 | 不把请求状态当报名资格或再次入场 |
| ExportJob/GeneratedArtifact | 支撑聚合/交付实体（Delivery） | 引用 eventId | 范围、字段、结果、文件到期 | 任务状态 | 不能在任务结果中泄漏票码和秘密 |
| NotificationAttempt | 过程数据（Delivery） | 引用业务事件/组 | 收件范围、模板、尝试、结果 | 投递状态 | 失败不回滚资格 |
| CleanupRun | 支撑聚合根（Lifecycle） | 引用 eventId | 到期目标、删除依据、结果 | 清理状态 | 区分逻辑不可见与实际处置 |
| AttendanceSnapshot | 派生读模型（平台内核） | 引用事实与更正 | 指标、口径、时点 | 无业务状态机 | 不是新主数据，不独立修改余额；归档匿名快照由 Lifecycle 物化 |

## 聚合一致性与写入边界

容量批准、组批准名额与资格决定需同一原子事务，在 Enrollment 内协调 Event 和 RegistrationParty；首版可在同一持久化边界实现，不将一次容量竞争拆为异步最终生效。

核销需原子检查资格、窗口、授权、票证和剩余数并记录操作结果。若事件阶段会与核销并发，必须使“关闭前/关闭后”有明确序列；不能先无锁查窗口再在结束后写入。更正、增额和普通核销共享组版本/约束，净数不越界。核销/更正/增额事务横跨 Gate 与 Enrollment（Admission 事实与 Party 余额），属上文共写白名单。

导入批次选中行的资格/占额提交调用同一核心服务，在同一事务边界可整体校验并创建关联；失败不产生部分有效票。外部 HTTP/文件解析及预览在该事务之外，避免持有长锁。

外部同步、邮件与统计更新可使用本地任务/持久化事件等候选方式，不预设必须上消息中间件。最终一致读模型应标时点；是否可放行仍以核心事务结果为准。

## 子流程 I/O 契约

| 子流程 | 触发/前提 | 输入 | 输出 | 写入位置 |
|---|---|---|---|---|
| 确认有效名额 | 即发、有效来源或批准 | 活动、组、请求总数/新增差额、版本、授权 | 有效资格与名额或明确冲突 | Event 占额、Party、Decision 同事务 |
| 核销 | 有效票、窗口开放、端获权 | 活动、票能力、人数、操作 ID/载荷 | 持久化成功/确定拒绝/原结果 | Party 余额、AdmissionRecord、操作结果 |
| 更正 | 负责人、原核销及理由 | 原事实、数量、版本、操作 ID | 关联更正与更新净数 | Correction 与 Party 同事务 |
| 导入提交 | 预览未过期且选中有效行 | 批次、行集合、映射/规则版本、来源资格 | 整体提交或可解释失败 | Batch/SourceLink/核心名额一致提交 |
| 补发凭证 | 负责人核对，资格有效 | Party、旧版本、理由 | 新凭证及旧票失效 | Party 凭证与审计，不加名额 |
| 结束检票 | 负责人及活动阶段 | 活动、版本、操作 ID | 停止普通写入及统计核对依据 | Event 窗口与审计 |
| 到期清理 | 活动期限到达 | 到期依据和受管范围 | 实际清理结果、入口失效 | 对应上下文及 CleanupRun |

## 关联领域对象

| 引用对象 | 所属上下文 | 关联字段 | 需获取信息 | 触发关系 | 映射方式 |
|---|---|---|---|---|---|
| EventPolicy | EventAdmin | eventId/policyVersion | 字段、容量与审核规则 | 预览/提交/核销 | 域内引用或版本快照 |
| RegistrationParty | Enrollment | partyId/eventId | 有效资格及票证/名额 | 领票、核销、统计 | 受限应用服务，不跨域直改表 |
| EventGrant/Device | Identity/Gate | accountId/deviceId/eventId | 当前动作/字段范围 | 每次敏感查询或写入 | 服务端鉴权结果，不信任前端声明 |
| SourceConnection | Connection | connectionId/resourceId | 获权资源及秘密引用 | 平台接入任务 | 薄适配，秘密不向业务浏览器展开 |
| GeneratedArtifact | Delivery | artifactId/eventId | 字段范围、到期、交付结果 | 下载与清理 | 限时受控访问 |

## 过程数据分类与清理

| 数据项 | 分类 | 处理 |
|---|---|---|
| 原文件、原答卷副本、预览行 | 导入暂存 | 最小期限保存并受活动明细生命周期清理，不保留无限原始名单 |
| 操作幂等、网络未知结果 | 技术执行状态 | 明确作用域和保存期；期限内结果可恢复，期限后不无条件重放旧动作 |
| 页面待确认 ID | 端上最小元数据 | 不含全量名单或秘密，登出/到期/结束清理，不能变离线成功队列 |
| 邮件投递尝试、导出结果 | 支撑过程 | 字段受权、到期清理，不能成为持久泄漏的明细副本 |
| 审计与更正 | 受控业务追溯 | 限范围留痕，个人明细到期后的去标识/删除与统计可反查性单独定义 |

## 更新记录

- v0.1，2026-10-20：从 PRD 预备稿提取上下文、聚合/过程分类、引用和原子子流程；待集中审阅及详细契约验证，未生成数据库或业务代码。

- v1.0，2026-10-20：ADR-041 确认 FL/需求/领域模型并保留整批提交，完成 D1/D2 检查后进入 API 合同设计；不代表 Gate 1 通过。

- v1.0注记，2026-10-21：确认聚合根标注及阶段引用对齐ADR-041—044；模型边界保持。

- v2.0，2026-11-11：ADR-047 领域重划分：7 上下文改为 8 上下文+平台内核（ActivityAdmission 拆为 EventAdmin/Enrollment/Gate；Reporting 解散为各上下文读侧+Lifecycle 归档快照；Identity/Connection 拆分；DeviceSession 更名 Device 迁至 Gate 并新增 Account 聚合根，共 8 个）。共写白名单显式化（核销事务 Gate↔Enrollment、容量分配 Enrollment→EventAdmin）；子流程 I/O 与过程数据分类不变。
