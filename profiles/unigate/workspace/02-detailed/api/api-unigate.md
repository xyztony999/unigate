# UniGate 在线版 API 设计

版本 v0.4；日期 2026-11-11；状态：ADR-047 领域重划分——85 操作 AA 编号全部重挂 AA v0.5 新编号，路径/DTO/枚举/DDL 零变更；此前 ADR-042采用合同、ADR-043采用项目检查，D3已按项目机械/人工检查完成归档。设计文档，不是已运行服务。

## 1. 模块设计概述

### 1.1 背景与边界

覆盖单主办方多活动的报名、来源名单、组票、手机/平板浏览器工作人员及 kiosk 检票、累计统计和受控交付。首版在线；没有本地服务器、离线核销或蓝牙同步。2—5 台同时检票、单场含随行者 500 人是设计目标，尚无性能实测。外部名单可独立运行检票，不依赖内置报名。

### 1.2 管理模式与依赖

内部报名与外部来源并存（MIXED），表示来源组合；外部接口不能绕开名额/审核/冻结，不接受任意外部指令直接更新有效人数或删除核销。单体模块部署候选；同库强事务，持久任务/outbox 处理第三方调用。无需先引入消息队列、微服务或工作流引擎。

Java/Spring Boot 后端和 Vue 已确认。详细设计候选为 JDK 21、Spring Boot 4.1.1、Vue 3、PostgreSQL 18；具体补丁、驱动和构建版本在实现前锁定，并非用户既有技术选择。选择关系库是为同事务处理总容量、组余额及操作结果。参考：[Spring Boot 官方要求](https://docs.spring.io/spring-boot/system-requirements.html)、[PostgreSQL 支持周期](https://www.postgresql.org/support/versioning/)。契约固定 [OpenAPI 3.1.1](https://spec.openapis.org/oas/v3.1.1.html)，不声称它是最新版本。

### 1.3 产物和追溯

`openapi-unigate.json` 为字段级合同，本文为跨字段/授权/事务语义，`data-model-unigate.md` 为表/约束/SQL 设计。每个操作保留 AA 子模块 ID（ADR-047 重划分后编号指向 AA v0.5，新旧映射见 ../05-reports/domain-redivision-proposal.md §3.2）；第5节对应全部43项 FL 功能。API 结构变更须同步三份文档和枚举池，禁止靠前端隐藏字段替代后端授权。

### 1.4 对象与服务边界

Event 负责阶段、容量及策略（EventAdmin）；RegistrationParty 负责资格、批准名额、待增员和票证（Enrollment）；Device（原 DeviceSession，ADR-047）与入场事实/更正管活动内受限终端和核销（Gate）；ImportBatch 管预览和来源关联（Intake）；SourceConnection 管平台秘密与获权资源（Connection）；ExportJob、CleanupRun 管受控文件和实际清理（Delivery/Lifecycle）。Account/授权为通用聚合与实体（Identity），统计为平台内核承载的派生读模型。关联及表见数据设计第1—3节。

## 2. 跨聚合根设计

### 2.1 不变量与状态

`0 <= allocatedPeople <= capacity`；占额等于所有有效组 approvedSize 之和，待审核不占额。`0 <= netAdmitted <= approvedSize`，`remainingCount = approvedSize - netAdmitted`。核销不释放活动占额，纠错不退报名名额。随行缺省0，总申请数=1+随行；无需主报名者每批到场，禁止按手机号/邮箱静默合并。

活动：DRAFT→发布→READY→开启→CHECKIN_OPEN；开放可暂停为 CHECKIN_PAUSED、再恢复；开放/暂停可结束为 ENDED；到期清理转 ARCHIVED。DRAFT、READY 不可核销；OPEN/PAUSED 冻结公众报名、普通审核、普通人数变更和取消，暂停仍保持冻结；负责人专用补录/增额仅 OPEN；纠错 OPEN/PAUSED；ENDED 不默认重开或特权纠错。打开窗口由负责人显式操作，时间到了不自动绕过状态检查。提前 OPEN 也同时冻结公众操作。

组：PENDING→批准→APPROVED，或拒绝→REJECTED；允许窗口可 CANCELLED。待增员是附加申请，不把原 APPROVED 变回 PENDING。每组最多一个未决增员；再次提交以组版本原子替换未决申请、旧申请记 CANCELLED，不累加两笔；选择低于原批准数则释放差额并取消待增员。通过时只占新增差额；容量不足保留待审和原名额。已发生核销（包括事后纠错）后不允许自助变更。终态报名不被普通修改复活。

票：ACTIVE/REVOKED/EXPIRED 与资格分开；撤票不退占额，补发旧票立即失效且不重置核销。补发返回新票；仅 rotateAccess=true 返回新查询/变更能力。全部核销后可展示状态，但剩余0不能再次放行。

### 2.2 原子编排、锁与冲突

语义写入先认领幂等键，再锁 Event，随后按 partyId 排序锁组，再锁关联票/审核/批次；所有容量/阶段写操作遵同一顺序。一个逻辑数据库裁决者不限制只有一个 Web 进程。事务内重新检查窗口、资格、当前角色/设备授权、票版本、组版本及人数；资格/占额/事实/幂等结果/审计/outbox 同事务。外部 HTTP、上传解析、导出生成和邮件发送在事务外。单事务超时设计5秒，失败整笔回滚，不能部分返回成功；锁等待/可重试冲突沿用原 operationId。

同一组两端提交不同人数：先提交者生效，后者因版本或余额冲突409；显示新余额后用户确认才生成新操作。结束/暂停与核销用 Event 锁排序，提交前通过守卫的事务可完成，排在暂停后者拒绝。授权变更与敏感写入还锁同一授权/设备行；撤销提交之后开始授权校验的请求必拒绝，已在线性化点通过者可能完成；界面不能承诺撤销抹去在途事实。授权锁只在统一 Event→Party→Grant/Device 顺序取得；先取得的幂等记录不得成为反向锁入口。

导入：解析/映射/疑似重复核对生成预览，不占额。仅选中 VALID 行及明确 LINK_EXISTING 的已关联行可提交；错误/未决疑似行必须排除。提交同时校验预览/策略/来源版本、源记录指纹、选中集合及总新增名额；LINK_EXISTING 只补来源，不改既有字段/名额。全批提交或回滚，容量不足整批不生效。提交失败后重新预览、确认新集合为新动作；原网络未知先查原动作，不能先换批重发。

固定GATE作用域防止换组/换端命中新作用域后重复建操作；原ID不同目标或设备返回载荷冲突，不能新核销。普通业务仍以原账号/匿名上下文为作用域，回包未知不得换上下文自动重发。

### 2.3 幂等与网络未知

每次语义写入客户端生成 UUID operationId；双击、自动重试及未知结果恢复复用同 ID。唯一键为规范化 actorScope + eventScope + action + operationId；匿名 actorScope 来源为服务端 HttpOnly 客户端 cookie，不能自报账号。核销在STAFF/kiosk均以固定GATE为actorScope，eventId+GateCommit+operationId在该活动全局唯一；稳定deviceId、组ID、原版本和人数纳入载荷摘要，实际账号另记审计并重验权限；同一次操作换组/换端不得作为新写入恢复。全局动作 eventScope=GLOBAL，不用可为空的唯一列。

载荷摘要采用规范化 JSON/文件内容摘要，包含稳定目标、人数、原版本及业务内容；排除 CSRF、短期扫描上下文等可更新的传输证明，恢复时仍校验其绑定关系。换人数/目标/内容沿用旧 ID 返回400 IDEMPOTENCY_PAYLOAD_MISMATCH。已提交重复请求重验当前身份/范围后返回原结果、replayed=true、showAsNewAdmission=false；不得再加人数、再响成功提示或再次发送邮件。查询 GateOperation 同样不触发绿色放行。

操作键和最小结果在活动个人明细到期前保留，避免短期清除后旧请求重新扣额；报名等含私密能力的可重放响应加密且最多24小时，过后只给原操作元数据，转验证恢复而不再次创建组。全局账号/连接配置键保留至对应主体终止，含秘密响应同样24小时内清理。到期归档先禁所有写入/失效能力再清键，不能先清键再放行旧请求。确定拒绝可记 REJECTED；修正业务内容必须明确新动作。事务未提交则可无记录，404 不代表需要新动作。

IN_PROGRESS 或锁等待返回409、Retry-After；客户端继续查询/原 ID 重试。超时、500、503、断网均为客户端 UNKNOWN，不能推断没扣数。扫码识别不核销。原结果查明前禁重复确认；断网只暂停该端，不自动改全场窗口；重启后保留最小 operationId/活动/组引用和原请求人数至核对完成或有效期，不存整份名单，不作离线成功队列。

## 3. 应用与基础设施规范

### 3.1 HTTP、字段、分页与错误

路由 `/api/v1/{domain}/{resource}/{action}`；查询 GET 或标记 safe query 的 POST，写 POST；POST 查询用于避免票秘密/找票搜索值进入URL或承载复合筛选，仍做 CSRF。OAuth callback 为一次 state 保护的唯一 GET 变更例外。JSON camelCase，表 snake_case；ID 为 UUID 字符串，不是授权凭证；UTC 存储，RFC3339 带偏移，活动时区 IANA。请求未知字段拒绝；未填非必需字段省略，不把空字符串/null 当缺省。仅 rate 允许 null（分母0）。服务端字段依赖/限额不能靠 JSON Schema 之外的浏览器校验替代。

统一响应 `{code,message,data,requestId}`，成功 code=OK，message 可空，Empty data={}；错误 data 为 ErrorData，不反射敏感原输入。200 表业务已提交或查询完成；202 表任务/挑战持久受理，核销不返回异步成功。文件下载成功二进制、attachment，失败仍 JSON。所有含身份/明细/能力响应 Cache-Control:no-store；静态 Vue 资源可正常缓存，不使离线核销成立。

分页 page缺省1/pageSize缺省20、最大100，total与items同授权过滤及读快照；固定次排序ID，索引表各项注明排序。组合筛选 AND，模糊查询长度2—100（账号精确前缀除外），不支持任意SQL或客户端字段表达式。列表期间数据变化可导致页间变化，不承诺跨请求固定快照；导出独立快照。游标签名含调用人/范围/最后排序值，15分钟，不接收任意 provider URL。

所有枚举在 common-spec 三文件登记。请求传 code；展示返回 `{code,label}`。字段名列表如 export.columns、授权动作名是契约标识，不是自由标签。错误码及 retryMode 是机器协议码，其展示由页面按错误码映射，响应不把内部异常堆栈作为标签。

### 3.2 认证、授权与秘密

STAFF：工作人员 Cookie 会话，闲置30分钟/绝对12小时；首次登录先改密码，除 me/logout/password-change 外拒绝业务操作。账号密码慢哈希，候选 Spring Security Argon2id；创建初始管理员为部署期单次命令/秘密挂载设计，不提供未鉴权建管理员接口。用户名规范化唯一；口令12—256字符，禁止日志、URL和回显，改密后退出旧会话重新登录。

角色只代表动作上限，还须事件授权、字段、独立 EXPORT 和有效期。管理员管理授权/连接元数据，不自动读全活动明细；只有额外 ORGANIZER 角色及相应活动授权才读。负责人只授本活动 CHECKER/READER，ORGANIZER 由管理员授且不能撤掉最后负责人；账号对应角色必须有效。allowedActions 白名单 VIEW_DETAIL/VIEW_STATISTICS/CHECKIN/EXPORT，角色不能通过字符串自行扩权；READER 禁 CHECKIN/EXPORT，CHECKER 禁 EXPORT；负责人有配置的导出权。活动创建者获得本活动负责人授权；EXPORT默认不授，需管理员单独授权，查看与导出字段交集分别验证。READER须登记recipient/purpose/retentionNote，真实启用配置未完成不激活。未授权跨活动资源统一404，动作明确禁止403。

DEVICE：一次5分钟领取码换受限设备 Cookie，12小时且不晚于活动结束，失效需重新授权。STAFF 模式实际识别/核销须工作人员账号 AND 设备，二者都限定该活动；KIOSK 只需设备但仅扫码当前组、不能名单搜索。OAS security 中 GATE 的 device-only 分支仅用于 KIOSK，绝不能放宽 STAFF。kiosk 空闲30秒/完成后清当前组画面、内存票码与上下文；若核销未知先锁屏提示工作人员核对，不能清掉未确认 operationId。普通全屏与 Android 系统 kiosk 锁定区分，后者部署实测。

PARTY/CHANGE：票、私密查询、私密变更三种能力分开；至少256位随机值，只保存 HMAC/摘要，需重复领票的票值密文保存。网页私密链接用 fragment，首屏读入内存即清URL，Referrer-Policy:no-referrer，不加第三方分析脚本。请求体交换 HttpOnly/Secure Cookie，查询最长12小时且不晚于明细到期；变更另验能力+开关+截止+无任何核销。票码只能识别入场，不能查询全字段或自助变更。URL fragment不是被截屏/复制后安全的证明，泄露需负责人撤销/旋转。

ANON：bootstrap 设置随机匿名客户端 Cookie 和CSRF，再提交；不能靠任何人自报 operationId 获取报名响应。全POST须合法Origin、CSRF与当前会话匹配；匿名同样适用。Cookie Secure/HttpOnly/SameSite，CORS 默认同源，不接受通配带凭据。OAuth回调 state 绑定发起管理员、提供方、连接、固定redirect且一次5分钟，code/state查询参数必须网关和应用双重脱敏。

找票用报名参考号+已留联系方式验证码，统一202，5分钟/5次限制；无已留联系方式由负责人线下核对后办理补发。不存在仅凭手机号尾号补票接口。短信默认关闭；启用时phone必收必填，验证证明绑定活动/号码/匿名客户端、10分钟一次性；同 ID 已提交重放不能因为证明已消费而再扣额或误报失败。负责人现场验证依策略，不能任意跳过。


#### 3.2.1 匿名与终端上下文连续性补充

SessionBootstrap复用仍有效的匿名cookie，返回公开clientContextId。所有ANON认证的语义POST（登录、报名、能力交换、验证码/恢复、设备领取）额外必需请求头X-Client-Context-ID，值来自发起该动作时的bootstrap；与当前HttpOnly cookie不匹配返回401 SESSION_EXPIRED，必须在认领幂等键前拒绝。该标记不具备认证能力，不能取代cookie/CSRF。有效上下文内CSRF轮换可沿原operationId恢复。

页面待确认条目保存原clientContextId、原设备/账号上下文与非秘密业务参数；匿名身份过期、Cookie清除或设备重新领取后，先停止旧动作自动重发，使用恢复入口/负责人核对。不得把新bootstrap的标记套到旧operationId上创建另一组。无联系方式时只能由负责人核对，不承诺仅靠本地ID找回。工作人员重新登录不同账号或换终端同样不自动重新提交待确认核销，原设备原组操作由获权查询核对后再继续。

MyGrantList为校方/工作人员提供本人有效活动、字段和期限，不复用管理员全局授权列表。检票员查询记录/审计时服务器强制本人或当前经cookie验证的本端范围，不能相信客户端任意deviceId。

检索也只匹配调用方获准的字段及报名参考号，不得以未获准手机号/邮箱作为查询侧信道；只读人员无基础字段授权时仅可按参考号检索。PartyView.hasAdmissionHistory来自ever_admitted，更正为0不重开普通变更；MyParty.canChange还须随行字段启用。

### 3.3 字段、时间和来源约束

每个基础字段 enabled/required，required⇒enabled；禁用字段输入拒绝。姓名最长100，电话最长40并保留国家区号；无明确区号不武断补+86，不作为唯一身份；邮箱最长254且格式校验，不整体强制小写本地段。companionCount 为0—999999整数，缺省0；超过活动容量在占额时拒绝。手机号短信适配不支持的格式明确422。修改只支持已确认的随行数变化，其他联系字段更正由后续范围变更另定，不暗加编辑全部资料入口。

时间需 registrationStartsAt <= registrationEndsAt <= startsAt <= endsAt；自助截止缺省 min(registrationEndsAt,startsAt)，不得更晚。检查服务端时间，READY且报名时间段内才收公众新报名；负责人普通审批/取消在READY可办，开始后冻结。策略修改需policy.version；已有报名后不得删改已用字段启用/必填、审核模式、短信开关或缩短已告知留存，返回409；可调整尚未开始的窗口及自助开关但不突破告知和截止。留存变更需新的告知与迁移方案，首版不通过普通update悄改已报名数据。

文件仅CSV UTF-8/BOM、无宏XLSX，最多200列，超出明确拒绝；10MiB/解压50MiB/10000数据行，解析限时30秒；多工作表明确选首个非空表，其余忽略并给预览说明。禁止取外部公式、链接和执行公式，缺表头/非法编码报422；不接受XLS/宏文件。映射 countMode=COMPANIONS 缺省0；TOTAL 最小1再减1，空总数为行错，不能当缺随行；无人数列时仍传COMPANIONS且countColumn省略，默认1人。公式单元格不求值，标行错；CSV导出以=,+,-,@等开头的文本按安全文本输出防表格公式执行，电话号码作为文本保留+号。

平台同源键=(eventId,connectionId,resourceId,sourceRecordId)，文件键=(文件内容SHA256,工作表标识,数据行号)作为来源身份；改文件产生新来源并走疑似重复检查。同源内容未变关联无新增组；已有源记录变更不得静默覆盖有效名额，产生SOURCE_CONFLICT供人工核对。跨源同电话/邮箱仅提示，明确KEEP_SEPARATE/LINK_EXISTING/EXCLUDE 后重新预览；LINK不拷贝既有核销、不更新其联系字段。未选行保留到暂存到期；COMMITTED 批次不能再增选，后续建立新批并同源防重。

SourcePull先产预览批次，API拉取不是直接注册；provider适配须验证飞书实际问卷载体、腾讯实际表/问卷权限，个人/组织账号只影响凭据。首版API范围保留，实际能力未证实返回PROVIDER_UNAVAILABLE，不假造成功或爬取代替。管理员为连接配置显式可用活动列表（ConnectionSave.allowedEventIds），负责人仅从该范围选资源；资源选择不泄露凭据。公网端点固定allowlist，任务不得接收任意URL。腾讯生成中间结果表之类副作用按已核实平台合同单独幂等，不能在通用GET偷偷创建。

SourceList.availableConnections仅列该活动被管理员分配的FEISHU/TENCENT连接薄元数据，供首次配置来源时选连接；不开放管理员ConnectionList或密钥/其他活动范围。SourceSave/ResourceFields必须再次验证该活动/连接授权及支持的平台，不能任意传EMAIL/SMS连接充当名单来源。

### 3.4 任务、统计、导出与清理

使用同库持久任务/outbox和租约抢占，事务提交后投递；无强制MQ。邮件/拉取/导出自动最多5次（1、2、4、8分钟退避，尊重Retry-After），超过FAILED；人工retry原业务键不变，新attempt，回调去重。SMTP超时可能已发，不能承诺外部恰好一次；业务资格/核销始终恰好一次数据库效果。短信验证码发送限流仍生效，不能用任务重试轰炸。

统计同一读快照：applicationGroups含所有非清理报名，pending/approved分状态；allocatedPeople有效批准额，grossAdmissions原核销、correctedPeople有效更正总数、netAdmissions=两者差、unusedPlaces=allocated-net；walkInGroups/onsiteAddedPlaces独立来源事实；admissionRate=net/allocated，分母0为null。不展示当前在场人数，不按剩余空位推算到场。

导出列白名单：partyReference、name、phone、email、companionCount、requestedSize、approvedSize、pendingIncrease、netAdmitted、remainingCount、status、progress、sourceType、createdAt、updatedAt。四基础字段须启用且获字段权，禁止票码/私密能力/平台秘密。生成及下载时分别重新鉴权，结果最多24小时且不晚于明细到期，下载无公链；读者仅获查看、不能绕过负责人导出。DeliveryRecord仅登记真实收件方/用途/列/副本期限，不执行发送，不等于批准真实跨境启用。

默认活动 endsAt+30天（可配置1—3650天技术范围，非法律结论）。即使负责人忘点END，到期仍按 endsAt触发并关闭写入。到期先撤所有入口/任务/文件下载、冻结活动，再实际清理Party/预览/结果/邮件目标/票秘密/导出文件及可关联日志，汇总只保无组ID/联系方式、无小分组可回查的活动总计；物理失败标FAILED并继续不可访问，重试不提前任意删除。暂存最多24h。备份加密且候选最长35天轮转，待到期备份不可对外恢复服务；独立删除依据与密钥删除机制须覆盖全部在存备份，恢复隔离环境先应用删除账本/密钥撤销再开放服务。备份过期未实测前清理状态不能宣称全副本已完成，已下载外部副本另约定。

## 4. 聚合根与接口详细设计

### 4.1 聚合/实体/服务及过程 I/O

按 ADR-047 上下文边界组织（实现为平铺单包，包结构重构属 D-5b）：EventAdmin/EventService：create/update/policy/transition，读写Event；Enrollment/PartyService：submit/review/change/cancel/increase/walkIn/凭证issue/revoke/reissue，原子协调Event+Party+Decision；Gate/GateService：resolve/commit/result/correct/walkIn/increase；Intake/IntakeService：preview/resolve/commit；Identity/IdentityService：account/grant/session；Connection/ConnectionService：allowlist资源与凭据；Delivery/DeliveryService：邮件与导出；Lifecycle/LifecycleService：到期隔离/清理/恢复依据；平台内核：幂等Operation/审计/任务队列/挑战与统计读模型（原独立ReportingService并入）。外部I/O通过FeishuAdapter/TencentAdapter/MailAdapter/SmsAdapter/ArtifactStore，数据库Repository不绕过应用服务变更余额。

子流程 I/O：批准=(event,party,delta,version,actor,operationId)→Decision+新占额或冲突；核销=(event,scanContext,count,version,device,actor,operationId)→GateResult；纠错=(originalRecord,count,partyVersion,reason,operationId)→CorrectionView；导入=(batch,previewVersion,rowSet,operationId)→BatchCommitView；补发=(party,version,rotateAccess,reason,operationId)→ReissueView；清理=(event,dueAt,managedScopes)→CleanupView。涉及的持久字段、DDL和SQL清单见 data-model-unigate.md；内部事件见第5节。

### 4.2 逐接口契约

下列为完整操作清单。请求表列出全部根字段，嵌套DTO在4.3；响应统一外壳和精确 data DTO 均可在 OpenAPI 找到。每个写操作遵2.2/2.3事务和幂等，查询不执行业务写入（会话bootstrap/显式OAuth回调例外）。全部操作适用3.2授权；错误表在4.4。不存在“未列字段可自由透传”。

#### 4.2.1 SessionBootstrap — 初始化CSRF/匿名客户端

`GET /api/v1/identity/session/bootstrap`；角色：ANONYMOUS；认证：ANON；AA：2.4.1.1。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| — | 空对象 | — | 不接受额外字段 |

成功：HTTP 200，SessionBootstrapResponse，data=CsrfView。

约束/结果：同源GET，设置匿名客户端HttpOnly cookie及CSRF，未建立业务权限。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.2 Login — 工作人员登录

`POST /api/v1/identity/session/login`；角色：ANONYMOUS；认证：ANON；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| username | string | 是 | minLength=1; maxLength=80 |
| password | string | 是 | minLength=1; maxLength=256 |

成功：HTTP 200，LoginResponse，data=SessionView。

约束/结果：合法Origin+CSRF；验证后轮换会话，5次失败退避；初始密码只可先改密，不记录密码。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.3 Logout — 注销工作人员会话

`POST /api/v1/identity/session/logout`；角色：ADMIN, ORGANIZER, CHECKER, READER；认证：STAFF；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，LogoutResponse，data=Empty。

约束/结果：幂等注销，清理工作人员cookie；不自动撤销独立终端会话，公开kiosk不保存管理员登录。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.4 SessionMe — 当前工作人员身份

`GET /api/v1/identity/session/me`；角色：ADMIN, ORGANIZER, CHECKER, READER；认证：STAFF；AA：2.4.1.1。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| — | 空对象 | — | 不接受额外字段 |

成功：HTTP 200，SessionMeResponse，data=SessionView。

约束/结果：只返回账号自身元数据，角色不是全活动数据授权。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.5 AccountList — 账号列表

`POST /api/v1/identity/account/list`；角色：ADMIN；认证：STAFF；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| query | string | 否（省略） | minLength=1; maxLength=100 |

成功：HTTP 200，AccountListResponse，data=AccountPage。

约束/结果：仅账号元数据，不联带活动个人名单；username/accountId稳定排序。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.6 AccountCreate — 创建工作人员账号

`POST /api/v1/identity/account/create`；角色：ADMIN；认证：STAFF；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| username | string | 是 | minLength=1; maxLength=80 |
| displayName | string | 是 | minLength=1; maxLength=100 |
| temporaryPassword | string | 是 | minLength=12; maxLength=256 |
| roles | array<string> | 是 | minItems=1; maxItems=4; uniqueItems=True |

成功：HTTP 200，AccountCreateResponse，data=AccountView。

约束/结果：初始密码至少12字符且首次改密，禁止PARTICIPANT长期账号；username唯一，慢哈希存储。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.7 AccountUpdate — 更新账号或停用

`POST /api/v1/identity/account/update`；角色：ADMIN；认证：STAFF；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| accountId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| displayName | string | 是 | minLength=1; maxLength=100 |
| enabled | boolean | 是 | 见同名DTO与通用约束 |
| roles | array<string> | 是 | minItems=1; maxItems=4; uniqueItems=True |

成功：HTTP 200，AccountUpdateResponse，data=AccountView。

约束/结果：不能停用/降权最后一个管理员；权限版本更新，新请求重新校验。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.8 PasswordChange — 修改本人密码

`POST /api/v1/identity/account/password-change`；角色：ADMIN, ORGANIZER, CHECKER, READER；认证：STAFF；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| oldPassword | string | 是 | minLength=1; maxLength=256 |
| newPassword | string | 是 | minLength=12; maxLength=256 |

成功：HTTP 200，PasswordChangeResponse，data=Empty。

约束/结果：新密码至少12字符，验证旧密码后轮换会话并使旧会话失效。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.9 GrantList — 活动授权列表

`POST /api/v1/identity/grant/list`；角色：ADMIN, ORGANIZER；认证：STAFF；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |

成功：HTTP 200，GrantListResponse，data=GrantPage。

约束/结果：负责人限自己的活动；系统管理员可管理授权但不能借接口读报名信息。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.10 GrantSave — 配置活动授权

`POST /api/v1/identity/grant/save`；角色：ADMIN, ORGANIZER；认证：STAFF；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| accountId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| role | string | 是 | ORGANIZER / CHECKER / READER |
| allowedFields | array<PartyFieldCode> | 是 | maxItems=1000 |
| allowedActions | array<string> | 是 | maxItems=1000 |
| recipient | string | 否（省略） | minLength=1; maxLength=200 |
| purpose | string | 否（省略） | minLength=1; maxLength=500 |
| retentionNote | string | 否（省略） | minLength=1; maxLength=1000 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| grantId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 否（省略） | format=int32; minimum=0; maximum=2147483647 |

成功：HTTP 200，GrantSaveResponse，data=GrantView。

约束/结果：负责人只授CHECKER/READER且不超自身范围；更新须grantId+version；READER必须recipient/purpose/retentionNote，真实启用条件未完成不激活；查看与EXPORT分开。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.11 GrantRevoke — 撤销活动授权

`POST /api/v1/identity/grant/revoke`；角色：ADMIN, ORGANIZER；认证：STAFF；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| grantId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

成功：HTTP 200，GrantRevokeResponse，data=Empty。

约束/结果：负责人限本活动，不能撤掉最后一个负责人；在途写入授权线性化点见正文。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.12 EventList — 授权活动列表

`POST /api/v1/event/event/list`；角色：ORGANIZER, CHECKER, READER；认证：STAFF；AA：2.2.1.1, 2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| status | EventStatusCode | 否（省略） | 见同名DTO与通用约束 |
| query | string | 否（省略） | minLength=1; maxLength=100 |

成功：HTTP 200，EventListResponse，data=EventPage。

约束/结果：服务端限定活动范围，按startsAt,eventId稳定降序；管理员须另有业务角色。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.13 EventDetail — 活动详情

`GET /api/v1/event/event/detail`；角色：ORGANIZER, CHECKER, READER；认证：STAFF；AA：2.2.1.1。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，EventDetailResponse，data=EventView。

约束/结果：仅授权活动元数据，无报名明细或能力凭据。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.14 EventCreate — 创建活动草稿

`POST /api/v1/event/event/create`；角色：ORGANIZER；认证：STAFF；AA：2.2.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| title | string | 是 | minLength=1; maxLength=200 |
| venue | string | 否（省略） | minLength=1; maxLength=300 |
| startsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| endsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| timeZone | string | 是 | minLength=1; maxLength=80 |
| registrationStartsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| registrationEndsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| capacity | integer | 是 | format=int32; minimum=1; maximum=1000000 |

成功：HTTP 200，EventCreateResponse，data=EventView。

约束/结果：标题/时间/时区/人数必填；结束不早于开始，创建者获本活动负责人授权，500是目标而非固定容量。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.15 EventUpdate — 更新活动信息

`POST /api/v1/event/event/update`；角色：ORGANIZER；认证：STAFF；AA：2.2.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| title | string | 是 | minLength=1; maxLength=200 |
| venue | string | 否（省略） | minLength=1; maxLength=300 |
| startsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| endsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| timeZone | string | 是 | minLength=1; maxLength=80 |
| registrationStartsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| registrationEndsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| capacity | integer | 是 | format=int32; minimum=1; maximum=1000000 |

成功：HTTP 200，EventUpdateResponse，data=EventView。

约束/结果：仅DRAFT/READY，容量不能小于已占额；不静默改已批准组数据，版本不匹配409。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.16 EventTransition — 发布与检票窗口流转

`POST /api/v1/event/event/transition`；角色：ORGANIZER；认证：STAFF；AA：2.1.2.1, 2.1.2.2, 2.2.3.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| action | EventActionCode | 是 | 见同名DTO与通用约束 |
| reason | string | 否（省略） | minLength=1; maxLength=500 |

成功：HTTP 200，EventTransitionResponse，data=EventView。

约束/结果：DRAFT发布READY；READY开启OPEN；OPEN暂停PAUSED/PAUSED恢复OPEN；OPEN或PAUSED结束ENDED。OPEN原子冻结公众变更；结束不自动重开，归档由到期任务执行。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.17 PolicyGet — 读取活动策略

`GET /api/v1/event/policy/detail`；角色：ORGANIZER；认证：STAFF；AA：2.2.1.2。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，PolicyGetResponse，data=EventPolicy。

约束/结果：fields和开关依赖完整返回给负责人，不向普通检票员返回配置秘密。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.18 PolicyUpdate — 更新基础字段与策略

`POST /api/v1/event/policy/update`；角色：ORGANIZER；认证：STAFF；AA：2.2.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| policy | EventPolicyInput | 是 | 见同名DTO与通用约束 |

成功：HTTP 200，PolicyUpdateResponse，data=EventPolicy。

约束/结果：仅DRAFT/READY，policy.version乐观锁；required依赖enabled，短信启用需手机号required；自助截止不晚于报名截止/现场开始。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.19 EventOptions — 活动选择器

`GET /api/v1/event/event/options`；角色：ORGANIZER, CHECKER, READER；认证：STAFF；AA：2.2.1.1。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| cursor | string | 否（省略） | minLength=1; maxLength=1000 |

成功：HTTP 200，EventOptionsResponse，data=OptionsView。

约束/结果：只列授权活动，startsAt,eventId稳定排序，每次最多100条。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.20 PublicEvent — 公开活动报名信息

`GET /api/v1/public/event/detail`；角色：ANONYMOUS；认证：ANON；AA：2.2.2.1。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，PublicEventResponse，data=EventPublicView。

约束/结果：只展示已发布活动和告知/字段，不返回名单；未发布404，不泄漏其存在性。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.21 SubmitRegistration — 提交内置报名

`POST /api/v1/public/registration/submit`；角色：PARTICIPANT；认证：ANON；AA：2.2.2.1, 2.1.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| fields | PartyInput | 是 | 见同名DTO与通用约束 |
| phoneVerificationProof | string | 否（省略） | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| acceptedNoticeVersion | string | 是 | minLength=1; maxLength=80 |

成功：HTTP 200，SubmitRegistrationResponse，data=SubmissionView。

约束/结果：匿名客户端cookie+CSRF，READY/报名时间内；IMMEDIATE整组占额，REVIEW不占额不发有效票；同ID恢复仅原客户端可取私密能力。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.22 PartyExchange — 交换本组私密能力

`POST /api/v1/public/party/exchange`；角色：PARTICIPANT；认证：ANON；AA：2.4.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| queryToken | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| changeToken | string | 否（省略） | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |

成功：HTTP 200，PartyExchangeResponse，data=CsrfView。

约束/结果：能力用请求体换HttpOnly cookie；变更能力独立可缺；页面读取URL fragment后清除，不写查询串或日志。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.23 MyParty — 查看本组状态

`GET /api/v1/public/party/detail`；角色：PARTICIPANT；认证：PARTY；AA：2.1.1.3。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| — | 空对象 | — | 不接受额外字段 |

成功：HTTP 200，MyPartyResponse，data=PartyStateView。

约束/结果：服务端仅从本组cookie定位，禁止传任意partyId查询别组；敏感响应no-store。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.24 MyCredential — 领取本组有效票

`GET /api/v1/public/credential/detail`；角色：PARTICIPANT；认证：PARTY；AA：2.1.1.3。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| — | 空对象 | — | 不接受额外字段 |

成功：HTTP 200，MyCredentialResponse，data=CredentialView。

约束/结果：仅APPROVED且票ACTIVE，待审/拒绝422 TICKET_UNAVAILABLE；票码不含联系方式。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.25 MyPartyChange — 自助改随行人数

`POST /api/v1/public/party/change`；角色：PARTICIPANT；认证：CHANGE；AA：2.1.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| companionCount | integer | 是 | format=int32; minimum=0; maximum=999999 |

成功：HTTP 200，MyPartyChangeResponse，data=PartyStateView。

约束/结果：开关/截止/READY/无核销守卫；审核模式只新增差额待审，原批准数保留，减少同步释放差额。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.26 MyPartyCancel — 自助取消报名

`POST /api/v1/public/party/cancel`；角色：PARTICIPANT；认证：CHANGE；AA：2.1.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

成功：HTTP 200，MyPartyCancelResponse，data=PartyStateView。

约束/结果：仅允许窗口且未核销；释放已批准数、撤票及取消待增员同事务，不删除历史。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.27 MyOperation — 本组原操作结果

`POST /api/v1/public/operation/result`；角色：PARTICIPANT；认证：PARTY；AA：2.4.3.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| action | string | 是 | minLength=1; maxLength=100 |

成功：HTTP 200，MyOperationResponse，data=OperationView。

约束/结果：只返回当前组获准操作元数据；action必须是本组允许操作的固定operation名称，无票码/其他组信息；原提交尚未拿到cookie则同ID重试submit，不能另起一笔。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.28 SmsStart — 发送手机验证挑战

`POST /api/v1/public/verification/send`；角色：PARTICIPANT；认证：ANON；AA：2.2.2.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| phone | string | 是 | minLength=1; maxLength=40 |

成功：HTTP 202，SmsStartResponse，data=ChallengeView。

约束/结果：活动启用才可用；挑战5分钟/验证5次/发送间隔60秒；IP+电话+客户端限流；受理不等于已送达。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.29 SmsVerify — 验证报名手机

`POST /api/v1/public/verification/verify`；角色：PARTICIPANT；认证：ANON；AA：2.2.2.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| challengeId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| code | string | 是 | minLength=1; maxLength=12 |

成功：HTTP 200，SmsVerifyResponse，data=VerificationView。

约束/结果：证明绑定活动/电话/客户端，10分钟一次性；不验证全部随行身份。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.30 RecoveryStart — 申请恢复本组入口

`POST /api/v1/public/recovery/start`；角色：PARTICIPANT；认证：ANON；AA：2.1.1.4。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyReference | string | 是 | minLength=1; maxLength=50 |
| channel | RecoveryChannelCode | 是 | 见同名DTO与通用约束 |
| contact | string | 是 | minLength=1; maxLength=254 |

成功：HTTP 202，RecoveryStartResponse，data=ChallengeView。

约束/结果：统一202不泄漏是否匹配；仅匹配记录并有服务配置才投递；无参考号/联系方式转负责人，不用电话当唯一身份。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.31 RecoveryVerify — 验证恢复私密能力

`POST /api/v1/public/recovery/verify`；角色：PARTICIPANT；认证：ANON；AA：2.1.1.4。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| challengeId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| code | string | 是 | minLength=1; maxLength=12 |

成功：HTTP 200，RecoveryVerifyResponse，data=PartyAccessBundle。

约束/结果：挑战5分钟最多5次；成功旋转查询/变更能力，不自动增加名额或核销，旧能力失效。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.32 PartyList — 报名组列表

`POST /api/v1/enrollment/party/list`；角色：ORGANIZER, READER；认证：STAFF；AA：2.1.1.2, 2.2.3.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| status | PartyStatusCode | 否（省略） | 见同名DTO与通用约束 |
| progress | AdmissionProgressCode | 否（省略） | 见同名DTO与通用约束 |
| sourceType | SourceTypeCode | 否（省略） | 见同名DTO与通用约束 |
| query | string | 否（省略） | minLength=1; maxLength=100 |
| createdFrom | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| createdTo | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| hasPendingIncrease | boolean | 否（省略） | 见同名DTO与通用约束 |

成功：HTTP 200，PartyListResponse，data=PartyPage。

约束/结果：READER只投影获准字段；createdAt,partyId稳定降序，不返回票码/私密能力。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.33 PartyDetail — 报名组详情

`GET /api/v1/enrollment/party/detail`；角色：ORGANIZER, READER；认证：STAFF；AA：2.1.1.2, 2.2.3.2。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，PartyDetailResponse，data=PartyView。

约束/结果：跨活动或未获字段不返回；字段按权限省略，不先发全量再隐藏。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.34 PartyChange — 准备期负责人变更

`POST /api/v1/enrollment/party/change`；角色：ORGANIZER；认证：STAFF；AA：2.1.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| companionCount | integer | 是 | format=int32; minimum=0; maximum=999999 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 200，PartyChangeResponse，data=PartyView。

约束/结果：仅READY且无核销，增员差额仍遵审核规则；现场加人必须专用动作，不绕过冻结。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.35 PartyCancel — 准备期负责人取消

`POST /api/v1/enrollment/party/cancel`；角色：ORGANIZER；认证：STAFF；AA：2.1.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 200，PartyCancelResponse，data=PartyView。

约束/结果：仅READY且未核销，释放占额和撤票同事务，保留决定历史。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.36 ReviewList — 首次/增员审核列表

`POST /api/v1/enrollment/review/list`；角色：ORGANIZER；认证：STAFF；AA：2.1.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| kind | ReviewKindCode | 否（省略） | 见同名DTO与通用约束 |
| result | ReviewResultCode | 否（省略） | 见同名DTO与通用约束 |

成功：HTTP 200，ReviewListResponse，data=ReviewPage。

约束/结果：按createdAt,reviewId稳定降序，活动范围严格校验。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.37 ReviewDecide — 审核通过或拒绝

`POST /api/v1/enrollment/review/decide`；角色：ORGANIZER；认证：STAFF；AA：2.1.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| reviewId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| decision | DecisionCode | 是 | 见同名DTO与通用约束 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 200，ReviewDecideResponse，data=ReviewView。

约束/结果：仅READY待审；整组/差额原子占额，不足不改决定/不发码；拒绝原因必填；现场普通审批冻结。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.38 CredentialReissue — 负责人核对后补发

`POST /api/v1/enrollment/credential/reissue`；角色：ORGANIZER；认证：STAFF；AA：2.1.1.3, 2.1.1.4。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| reason | string | 是 | minLength=1; maxLength=500 |
| rotateAccess | boolean | 是 | 见同名DTO与通用约束 |

成功：HTTP 200，CredentialReissueResponse，data=ReissueView。

约束/结果：有效资格且未到期限，旧票同事务失效，不加名额；rotateAccess才旋转私密能力，普通检票员禁止。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.39 CredentialRevoke — 撤销有效票

`POST /api/v1/enrollment/credential/revoke`；角色：ORGANIZER；认证：STAFF；AA：2.1.1.3。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 200，CredentialRevokeResponse，data=PartyView。

约束/结果：撤票不等于取消资格或释放占额；保留核销，不能借撤票绕过现场人数冻结。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.40 DeviceRegister — 授权浏览器终端

`POST /api/v1/gate/device/register`；角色：ORGANIZER；认证：STAFF；AA：2.4.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| name | string | 是 | minLength=1; maxLength=100 |
| mode | DeviceModeCode | 是 | 见同名DTO与通用约束 |

成功：HTTP 200，DeviceRegisterResponse，data=DeviceRegistration。

约束/结果：生成一次领取码5分钟，绑定活动和模式；负责人在私人端设置，不把管理员会话留在公开终端。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.41 DeviceClaim — 领取受限终端会话

`POST /api/v1/gate/device/claim`；角色：ANONYMOUS；认证：ANON；AA：2.4.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| deviceId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| pairingCode | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |

成功：HTTP 200，DeviceClaimResponse，data=DeviceSessionView。

约束/结果：合法Origin/CSRF，领取码一次有效最多5次错误；只发设备cookie，不授管理/工作人员账号权限。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.42 DeviceDetail — 当前终端就绪

`GET /api/v1/gate/device/detail`；角色：ORGANIZER, CHECKER, PARTICIPANT；认证：DEVICE；AA：2.4.1.2。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| — | 空对象 | — | 不接受额外字段 |

成功：HTTP 200，DeviceDetailResponse，data=DeviceSessionView。

约束/结果：仅受限设备cookie身份，绑定活动/模式；STAFF实际核销还需工作人员账号。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.43 DeviceList — 本活动终端列表

`POST /api/v1/gate/device/list`；角色：ORGANIZER；认证：STAFF；AA：2.4.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |

成功：HTTP 200，DeviceListResponse，data=DevicePage。

约束/结果：负责人查看模式/状态/最后在线，不能读取设备秘密或他活动信息。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.44 DeviceRevoke — 撤销终端

`POST /api/v1/gate/device/revoke`；角色：ORGANIZER；认证：STAFF；AA：2.4.1.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| deviceId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 200，DeviceRevokeResponse，data=Empty。

约束/结果：新请求即验证设备状态；在途操作以授权检查线性化点为准，保留核销事实。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.45 TicketResolve — 识别票并确认剩余名额

`POST /api/v1/gate/ticket/resolve`；角色：ORGANIZER, CHECKER, PARTICIPANT；认证：GATE；AA：2.1.2.3。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| ticketToken | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |

成功：HTTP 200，TicketResolveResponse，data=GatePreview。

约束/结果：STAFF须获权账号+设备，KIOSK仅本次组票最小结果；签发5分钟组/设备/票版本上下文，不核销。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.46 GateCommit — 核销本次人数

`POST /api/v1/gate/checkin/commit`；角色：ORGANIZER, CHECKER, PARTICIPANT；认证：GATE；AA：2.1.2.3, 2.4.3.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| scanContextToken | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| expectedPartyVersion | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| count | integer | 是 | format=int32; minimum=1; maximum=2147483647 |

成功：HTTP 200，GateCommitResponse，data=GateResult。

约束/结果：事件锁→组锁，校验窗口/资格/票/端/账号/余额，保存结果与事实同事务；原成功重放不增加人数或再次绿灯。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.47 GateLookup — 工作人员辅助找票

`POST /api/v1/gate/party/lookup`；角色：ORGANIZER, CHECKER；认证：GATE；AA：2.1.1.4。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| query | string | 是 | minLength=2; maxLength=100 |

成功：HTTP 200，GateLookupResponse，data=GateCandidatePage。

约束/结果：只允许STAFF+获权账号，最多10条固定第一页，仅姓名掩码/电话末四位（仅已收且获权时），无完整邮箱、票码或私密能力；KIOSK硬拒绝，尾号不用于补发，转负责人核对。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.48 GateOperation — 查询原核销结果

`POST /api/v1/gate/operation/result`；角色：ORGANIZER, CHECKER, PARTICIPANT；认证：GATE；AA：2.4.3.2, 2.1.2.5。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| scanContextToken | string | 否（省略） | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |

成功：HTTP 200，GateOperationResponse，data=GateResult。

约束/结果：负责人限活动/检票员限获准操作；kiosk必须原组上下文，查无记录仍同ID重试，过期未知转负责人。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.49 GateRecords — 核销记录

`POST /api/v1/gate/record/list`；角色：ORGANIZER, CHECKER；认证：STAFF；AA：2.4.3.3。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| deviceId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，GateRecordsResponse，data=AdmissionPage。

约束/结果：检票员只本人/获权端，过滤由服务端收缩，按committedAt,recordId稳定降序。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.50 GateCorrect — 负责人更正误检

`POST /api/v1/gate/record/correct`；角色：ORGANIZER；认证：STAFF；AA：2.1.2.5。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| recordId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyVersion | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| correctedCount | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 200，GateCorrectResponse，data=CorrectionView。

约束/结果：仅OPEN/PAUSED，累计更正不超原人数、组净数不负；追加记录不删除；ENDED后不默认开放特权更正。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.51 GateWalkIn — 现场补录或处理既有待审

`POST /api/v1/gate/party/walk-in`；角色：ORGANIZER；认证：STAFF；AA：2.1.2.4。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| fields | PartyInput | 是 | 见同名DTO与通用约束 |
| reason | string | 是 | minLength=1; maxLength=500 |
| existingPendingPartyId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyVersion | integer | 否（省略） | format=int32; minimum=0; maximum=2147483647 |
| phoneVerificationProof | string | 否（省略） | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |

成功：HTTP 200，GateWalkInResponse，data=PartyView。

约束/结果：仅CHECKIN_OPEN，整组占额；关联待审组须partyVersion并保留来源/原申请，不能新建重复组；不自动核销，短信依活动规则。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.52 GateIncrease — 现场负责人增加名额

`POST /api/v1/gate/party/increase`；角色：ORGANIZER；认证：STAFF；AA：2.1.2.4。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| additionalCount | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 200，GateIncreaseResponse，data=PartyView。

约束/结果：仅CHECKIN_OPEN有效组，原子占差额且审计，不自动核销，不借普通审批改现场人数。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.53 ConnectionList — 平台连接元数据

`POST /api/v1/integration/connection/list`；角色：ADMIN；认证：STAFF；AA：2.4.2.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| provider | string | 否（省略） | FEISHU / TENCENT / EMAIL / SMS |

成功：HTTP 200，ConnectionListResponse，data=ConnectionPage。

约束/结果：只有脱敏授权状态和资源引用，不联带报名个人明细。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.54 ConnectionSave — 建立或更新连接

`POST /api/v1/integration/connection/save`；角色：ADMIN；认证：STAFF；AA：2.4.2.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| connectionId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 否（省略） | format=int32; minimum=0; maximum=2147483647 |
| provider | string | 是 | FEISHU / TENCENT / EMAIL / SMS |
| accountKind | AccountKindCode | 是 | 见同名DTO与通用约束 |
| displayName | string | 是 | minLength=1; maxLength=100 |
| allowedEventIds | array<string> | 是 | maxItems=1000 |
| clientId | string | 否（省略） | minLength=1; maxLength=200 |
| clientSecret | string | 否（省略） | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |

成功：HTTP 200，ConnectionSaveResponse，data=ConnectionView。

约束/结果：更新须ID+version，秘密仅密文写不回显；提供方URL固定allowlist，不接受任意抓取地址。allowedEventIds由管理员明确配置，负责人无权自行扩展。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.55 ConnectionAuthorize — 发起提供方授权

`POST /api/v1/integration/connection/authorize`；角色：ADMIN；认证：STAFF；AA：2.4.2.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| connectionId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

成功：HTTP 200，ConnectionAuthorizeResponse，data=AuthorizationView。

约束/结果：state一次性5分钟绑定管理员及redirect allowlist；真实账号/应用能力未验证则明确能力不可用，不虚报接通。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.56 ConnectionCallback — OAuth协议回调

`GET /api/v1/integration/connection/callback`；角色：PROVIDER_CALLBACK；认证：ANON；AA：2.4.2.1。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| state | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| code | string | 是 | minLength=1; maxLength=2048 |

成功：HTTP 200，ConnectionCallbackResponse，data=ConnectionView。

约束/结果：唯一GET变更例外；一次性state、管理员授权会话和5分钟有效期；code/state日志脱敏，不能当免鉴权通用写接口。

重试/事务：OAuth state一次消费，重入不再次交换提供方code。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.57 ConnectionRevoke — 撤销平台授权

`POST /api/v1/integration/connection/revoke`；角色：ADMIN；认证：STAFF；AA：2.4.2.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| connectionId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

成功：HTTP 200，ConnectionRevokeResponse，data=ConnectionView。

约束/结果：清理受管凭据并阻止新拉取，历史业务资格不因断连接自动取消。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.58 ResourceList — 获准资源选择器

`POST /api/v1/integration/resource/list`；角色：ADMIN, ORGANIZER；认证：STAFF；AA：2.2.2.2, 2.4.3.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| connectionId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| cursor | string | 否（省略） | minLength=1; maxLength=1000 |

成功：HTTP 200，ResourceListResponse，data=ResourcePage。

约束/结果：负责人必须给eventId并同时有该活动及连接allowedEventIds授权，仅资源元数据，不含名单内容/秘密；实际文档类型和权限另验。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.59 SourceSave — 配置来源与映射

`POST /api/v1/intake/source/save`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| sourceId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 否（省略） | format=int32; minimum=0; maximum=2147483647 |
| connectionId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| resourceId | string | 是 | minLength=1; maxLength=300 |
| mapping | FieldMapping | 是 | 见同名DTO与通用约束 |
| eligibility | SourceEligibilityCode | 是 | 见同名DTO与通用约束 |

成功：HTTP 200，SourceSaveResponse，data=SourceView。

约束/结果：更新须sourceId+version；负责人明确来源资格，保存不等于同步成功；不能用外部字段自行授资格。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.60 SourceList — 活动来源列表

`GET /api/v1/intake/source/list`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，SourceListResponse，data=SourceListView。

约束/结果：只列活动配置，不返回外部凭据。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.61 SourcePull — 拉取形成预览批次

`POST /api/v1/intake/source/pull`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2, 2.4.3.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| sourceId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

成功：HTTP 202，SourcePullResponse，data=JobView。

约束/结果：仅暂存预览不直接占额；腾讯结果表生成等副作用须配置获准并记录资源ID防重，真实权限不支持明确失败。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.62 BatchUpload — 上传名单

`POST /api/v1/intake/batch/upload`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2。

请求位置：multipart/form-data（文件与其余字段分part）。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| file | string | 是 | format=binary |
| sourceLabel | string | 是 | minLength=1; maxLength=100 |

成功：HTTP 200，BatchUploadResponse，data=BatchView。

约束/结果：multipart CSV UTF-8/BOM或XLSX，10MiB、解压后50MiB、最多10000行；不执行宏/公式，受管暂存24h且不超过活动明细期限。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.63 BatchDetail — 批次状态

`GET /api/v1/intake/batch/detail`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| batchId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，BatchDetailResponse，data=BatchView。

约束/结果：校验活动/负责人和暂存有效期，过期410且不可提交。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.64 BatchPreview — 映射校验预览

`POST /api/v1/intake/batch/preview`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| batchId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| mapping | FieldMapping | 是 | 见同名DTO与通用约束 |
| eligibility | SourceEligibilityCode | 是 | 见同名DTO与通用约束 |

成功：HTTP 200，BatchPreviewResponse，data=BatchView。

约束/结果：不占额，生成policy/source指纹和previewVersion；错误/未核对重复行不能提交；TOTAL显式减本人，缺随行0。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.65 BatchRows — 预览行与错误

`POST /api/v1/intake/batch/rows`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| batchId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| status | RowStatusCode | 否（省略） | 见同名DTO与通用约束 |

成功：HTTP 200，BatchRowsResponse，data=ImportRowPage。

约束/结果：稳定lineNumber,rowId；字段只受管预览，不以未知列扩展问卷。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.66 DuplicateResolve — 人工核对疑似重复

`POST /api/v1/intake/row/resolve`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| batchId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| rowId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| previewVersion | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| resolution | DuplicateResolutionCode | 是 | 见同名DTO与通用约束 |
| existingPartyId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 200，DuplicateResolveResponse，data=BatchView。

约束/结果：LINK_EXISTING须明确既有组并保留来源，不复制核销或加占额；重建预览版本，不按电话静默合并。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.67 BatchCommit — 选中行整批提交

`POST /api/v1/intake/batch/commit`；角色：ORGANIZER；认证：STAFF；AA：2.2.2.2, 2.1.1.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| batchId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| previewVersion | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| selectedRowIds | array<string> | 是 | minItems=1; maxItems=10000; uniqueItems=True |

成功：HTTP 200，BatchCommitResponse，data=BatchCommitView。

约束/结果：仅READY且选中1到10000行；版本/指纹/来源/字段/容量整体校验，事件锁→排序组锁，全部提交或回滚，无部分有效票。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.68 JobList — 任务列表

`POST /api/v1/platform/job/list`；角色：ADMIN, ORGANIZER；认证：STAFF；AA：2.4.3.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| kind | JobKindCode | 否（省略） | 见同名DTO与通用约束 |

成功：HTTP 200，JobListResponse，data=JobPage。

约束/结果：管理员仅脱敏元数据，负责人限活动；无任意代码/任务执行权限。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.69 JobDetail — 任务详情

`GET /api/v1/platform/job/detail`；角色：ADMIN, ORGANIZER；认证：STAFF；AA：2.4.3.1。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| jobId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，JobDetailResponse，data=JobView。

约束/结果：敏感错误脱敏；202受理与真实完成区分。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.70 JobRetry — 重试失败任务

`POST /api/v1/platform/job/retry`；角色：ADMIN, ORGANIZER；认证：STAFF；AA：2.4.3.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| jobId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

成功：HTTP 202，JobRetryResponse，data=JobView。

约束/结果：原范围/现场冻结/副作用幂等继续校验；管理员不能借重试审批或读取名单。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.71 EmailResend — 重试领票邮件

`POST /api/v1/delivery/notification/resend`；角色：ORGANIZER；认证：STAFF；AA：2.1.1.4。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| reason | string | 是 | minLength=1; maxLength=500 |

成功：HTTP 202，EmailResendResponse，data=JobView。

约束/结果：须已提供可用邮箱且查询期/资格允许；私密链接密文投递，失败不撤销资格；无自动海外文件投递。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.72 Statistics — 活动统计

`GET /api/v1/report/attendance/summary`；角色：ORGANIZER, READER；认证：STAFF；AA：2.2.3.2。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，StatisticsResponse，data=StatisticsView。

约束/结果：同一数据库一致读快照返回asOf，含原核销/更正/净数；零分母rate=null，无当前在场人数。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.73 ExportCreate — 生成受控导出

`POST /api/v1/report/export/create`；角色：ORGANIZER；认证：STAFF；AA：2.2.3.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| format | ExportFormatCode | 是 | 见同名DTO与通用约束 |
| columns | array<string> | 是 | minItems=1; maxItems=40; uniqueItems=True |
| purpose | string | 是 | minLength=1; maxLength=500 |
| filter | PartyFilter | 否（省略） | 见同名DTO与通用约束 |

成功：HTTP 202，ExportCreateResponse，data=ExportView。

约束/结果：需独立EXPORT授权，列白名单且获权；禁止票码/私密能力/秘密；有效期min(24h,明细到期)，生成/下载重验权限。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.74 ExportDetail — 导出状态

`GET /api/v1/report/export/detail`；角色：ORGANIZER；认证：STAFF；AA：2.2.3.2。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| exportId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，ExportDetailResponse，data=ExportView。

约束/结果：查询不等于文件已生成，过期或失败明确呈现。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.75 ExportDownload — 下载未过期文件

`GET /api/v1/report/export/download`；角色：ORGANIZER；认证：STAFF；AA：2.2.3.2。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| exportId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，二进制文件（CSV/XLSX），错误为ErrorEnvelope。

约束/结果：成功为二进制CSV/XLSX，不是JSON；重验列权限/范围/到期，下载审计，无长期公开链接。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.76 DeliveryList — 明细交付记录

`POST /api/v1/delivery/record/list`；角色：ORGANIZER；认证：STAFF；AA：2.2.4.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |

成功：HTTP 200，DeliveryListResponse，data=DeliveryPage。

约束/结果：限活动，按createdAt,deliveryId降序；不是校方自动导出入口。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.77 DeliveryRecord — 登记明细交付范围

`POST /api/v1/delivery/record/create`；角色：ORGANIZER；认证：STAFF；AA：2.2.4.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| exportId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| recipient | string | 是 | minLength=1; maxLength=200 |
| purpose | string | 是 | minLength=1; maxLength=500 |
| columns | array<string> | 是 | maxItems=1000 |
| retentionNote | string | 是 | minLength=1; maxLength=1000 |

成功：HTTP 200，DeliveryRecordResponse，data=DeliveryView。

约束/结果：登记获准接收方/用途/范围及副本责任，不自动发海外邮件或作合规批准；真实启用配置先落实。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.78 CleanupDetail — 到期及实际清理状态

`GET /api/v1/lifecycle/cleanup/detail`；角色：ORGANIZER；认证：STAFF；AA：2.2.4.1。

请求位置：URL query。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

成功：HTTP 200，CleanupDetailResponse，data=CleanupView。

约束/结果：展示受管清理结果与失败范围，不以逻辑删除假装实际清理完成。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.79 CleanupRetry — 重试到期失败清理

`POST /api/v1/lifecycle/cleanup/retry`；角色：ORGANIZER；认证：STAFF；AA：2.2.4.1, 2.4.3.1。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| cleanupId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

成功：HTTP 202，CleanupRetryResponse，data=JobView。

约束/结果：仅到期且失败，不能提前任意批量删除；根据到期依据实际处置/失效能力并登记恢复防复活结果。

重试/事务：语义写入，按2.2—2.3持久化原结果；只有202任务提交成功不代表任务完成。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.80 AuditList — 获准审计记录

`POST /api/v1/platform/audit/list`；角色：ORGANIZER, CHECKER；认证：STAFF；AA：2.4.3.3。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| page | integer | 否（省略） | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否（省略） | format=int32; minimum=1; maximum=100; default=20 |
| action | string | 否（省略） | minLength=1; maxLength=100 |
| from | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| to | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

成功：HTTP 200，AuditListResponse，data=AuditPage。

约束/结果：检票员限本人/获权端，日志不含全量联系方式/票码/秘密；occurredAt,auditId稳定降序。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.81 OperationResult — 普通业务原操作元数据

`POST /api/v1/platform/operation/result`；角色：ORGANIZER, CHECKER；认证：STAFF；AA：2.4.3.2。

请求位置：application/json body。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| action | string | 是 | minLength=1; maxLength=100 |

成功：HTTP 200，OperationResultResponse，data=OperationView。

约束/结果：负责人限活动和原动作权限，检票员限获准操作；action是固定operation名称并校验原动作权限，不从通用查询取票或私密能力，恢复具体原请求仍用同ID。

重试/事务：安全读取；仍须当前授权；不改变名额或核销。

异常：请求/字段错误400或422；身份/范围401/403/404；版本/阶段/名额/预览冲突409；能力或文件过期410；限流429；内部或依赖未知500/503。适用条件按本接口前述守卫，不适用者不人为制造，详4.4。

#### 4.2.82 MyGrantList — 查看本人有效活动授权

`POST /api/v1/identity/grant/mine`；角色：ORGANIZER/CHECKER/READER；认证STAFF；AA2.4.1.1。POST安全查询，CSRF和同源校验，用于授权范围筛选；不修改权限。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 否，省略 | format=uuid |
| page | integer | 否，省略 | format=int32; minimum=1; maximum=2147483647; default=1 |
| pageSize | integer | 否，省略 | format=int32; minimum=1; maximum=100; default=20 |

成功HTTP200，MyGrantListResponse，data=GrantPage，字段见4.3 GrantView/GrantPage。accountId只来自当前会话，不接受请求他人账号；按expiresAt,grantId稳定升序，默认page=1/pageSize=20、上限100。仅返回本人仍有效授权，eventId可选收窄，角色不因此获得新增字段或动作。401登录过期、403无相应角色、404不可访问活动、400/422请求错误、429限流、500/503读取失败；查询可重试，不产生业务写入。

#### 4.2.83 ResourceFields — 读取获准来源字段元数据

`POST /api/v1/integration/resource/fields`；角色ADMIN/ORGANIZER；STAFF；AA2.2.2.2/2.4.3.1；安全查询，用于避免资源引用进入URL，须Origin/CSRF。

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| connectionId | string | 是 | format=uuid |
| eventId | string | 否，省略 | format=uuid |
| resourceId | string | 是 | maxLength=300 |

负责人条件必填eventId，同时校验活动及connection.allowedEventIds、提供方实际资源权限；返回HTTP200 ResourceFieldsResponse，data=ResourceFieldsView，仅列code/label、不含答卷值，最多200列。401/403/404为身份/范围拒绝，422为不支持资源类型或超列数，429限流，503提供方不可用；不修改名单、资格或名额，可安全重试。连接失效或列变更后旧预览不能提交，沿批次指纹重新预览。

#### 4.2.84 GrantEventOptions — 管理员授权目标活动选择

`POST /api/v1/identity/grant/event-options`；ADMIN/STAFF；AA2.4.1.1；安全查询，CSRF/Origin。只用于账号授权与连接可用活动选择。

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| query | string | 否，省略 | maxLength=100 |
| cursor | string | 否，省略 | maxLength=1000 |

HTTP200 GrantEventOptionsResponse，data=GrantEventOptionsView。每次最多100条，startsAt,eventId稳定降序，游标绑定管理员且15分钟失效。只返回活动标识/标题/开始时间，不返回占额、名单或核销数，不授活动明细权。查询title非个人信息，POST复合筛选；401/403身份角色、400/422参数、429限流、500/503读取失败；无业务写入，可安全重试。

#### 4.2.85 RegistrationResult — 原匿名客户端恢复报名结果

`POST /api/v1/public/registration/result`；PARTICIPANT/ANON；AA2.2.2.1/2.4.3.2；安全读取，必须原匿名cookie、CSRF、X-Client-Context-ID及Origin，敏感请求/响应no-store，不写日志。

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| eventId | string | 是 | format=uuid |
| operationId | string | 是 | format=uuid |

成功HTTP200 RegistrationResultResponse，data=SubmissionView且replayed=true；只在原提交客户端、原提交成功且加密私密结果24小时内/个人明细未到期时返回。报名窗口关闭不阻止读取已提交原结果，不能触发新占额或重发通知。无记录404/执行中409不是新动作许可；身份不同401/403，私密结果过期410转已留邮件或负责人，429/500/503保持待确认。刷新后保留的仅为eventId/operationId/clientContextId即可查原结果，无需把原表单PII持久化到浏览器。

### 4.3 共享 DTO 字段字典

所有对象默认 additionalProperties=false；省略不代表允许null。枚举Value固定code+label，Code候选值见枚举池。分页DTO均items/total/page/pageSize。下表也包含请求内嵌对象，防止只写根字段却缺少内部字段。

#### Empty

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| — | 空对象 | — | 不接受额外字段 |

#### FieldRule

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| enabled | boolean | 是 | 见同名DTO与通用约束 |
| required | boolean | 是 | 见同名DTO与通用约束 |

#### FieldRules

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| name | FieldRule | 是 | 见同名DTO与通用约束 |
| phone | FieldRule | 是 | 见同名DTO与通用约束 |
| email | FieldRule | 是 | 见同名DTO与通用约束 |
| companionCount | FieldRule | 是 | 见同名DTO与通用约束 |

#### PartyInput

遵循活动enabled/required；未收字段拒绝，随行缺省0；手机号保留区号、邮箱格式校验，联系方式非唯一键。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| name | string | 否（省略） | minLength=1; maxLength=100 |
| phone | string | 否（省略） | minLength=1; maxLength=40 |
| email | string | 否（省略） | format=email; minLength=1; maxLength=254 |
| companionCount | integer | 否（省略） | format=int32; minimum=0; maximum=999999 |

#### FieldError

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| field | string | 是 | minLength=1; maxLength=100 |
| code | string | 是 | minLength=1; maxLength=80 |
| message | string | 是 | minLength=1; maxLength=200 |

#### ErrorData

不含秘密/他人记录；UNKNOWN为客户端状态，不当作确定失败。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| fieldErrors | array<FieldError> | 否（省略） | maxItems=1000 |
| retryMode | RetryModeCode | 否（省略） | 见同名DTO与通用约束 |
| retryAfterSeconds | integer | 否（省略） | format=int32; minimum=0; maximum=2147483647 |

#### ErrorEnvelope

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| code | string | 是 | minLength=1; maxLength=80 |
| message | string | 是 | minLength=0; maxLength=300 |
| data | ErrorData | 是 | 见同名DTO与通用约束 |
| requestId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

#### CsrfView

bootstrap复用有效匿名身份；身份过期重新建立时clientContextId变化。页面保留原上下文标记以阻止跨上下文重放待确认操作。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| csrfToken | string | 是 | 当前有效授权范围 |
| expiresAt | string | 是 | format=date-time |
| clientContextId | string | 是 | format=uuid |

#### SessionView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| accountId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| username | string | 是 | minLength=1; maxLength=80 |
| displayName | string | 是 | minLength=1; maxLength=100 |
| roles | array<RoleValue> | 是 | maxItems=1000 |
| csrfToken | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| mustChangePassword | boolean | 是 | 见同名DTO与通用约束 |

#### AccountView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| accountId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| username | string | 是 | minLength=1; maxLength=80 |
| displayName | string | 是 | minLength=1; maxLength=100 |
| enabled | boolean | 是 | 见同名DTO与通用约束 |
| roles | array<RoleValue> | 是 | maxItems=1000 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### GrantView

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| grantId | string | 是 | format=uuid |
| accountId | string | 是 | format=uuid |
| eventId | string | 是 | format=uuid |
| role | RoleValue | 是 | 见通用规则 |
| allowedFields | array<PartyFieldValue> | 是 | maxItems=1000 |
| allowedActions | array<string> | 是 | maxItems=1000 |
| recipient | string | 否，省略 | maxLength=200 |
| purpose | string | 否，省略 | maxLength=500 |
| retentionNote | string | 否，省略 | maxLength=1000 |
| expiresAt | string | 是 | format=date-time |
| version | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| accountDisplayName | string | 是 | maxLength=100 |
| username | string | 是 | maxLength=80 |

#### EventView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| title | string | 是 | minLength=1; maxLength=200 |
| venue | string | 否（省略） | minLength=1; maxLength=300 |
| startsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| endsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| timeZone | string | 是 | minLength=1; maxLength=80 |
| registrationStartsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| registrationEndsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| capacity | integer | 是 | format=int32; minimum=1; maximum=1000000 |
| allocatedCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| status | EventStatusValue | 是 | 见同名DTO与通用约束 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### EventPolicy

默认IMMEDIATE/短信及自助开关关闭/30天。required依赖enabled；短信开启须手机号enabled且required；自助截止不晚于报名截止及检票开始。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| fields | FieldRules | 是 | 见同名DTO与通用约束 |
| approvalMode | ApprovalModeValue | 是 | 见同名DTO与通用约束 |
| phoneVerificationEnabled | boolean | 是 | 见同名DTO与通用约束 |
| selfEditEnabled | boolean | 是 | 见同名DTO与通用约束 |
| selfCancelEnabled | boolean | 是 | 见同名DTO与通用约束 |
| selfServiceEndsAt | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| retentionDays | integer | 是 | format=int32; minimum=1; maximum=3650; default=30 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### EventPublicView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| title | string | 是 | minLength=1; maxLength=200 |
| venue | string | 否（省略） | minLength=1; maxLength=300 |
| startsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| endsAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| timeZone | string | 是 | minLength=1; maxLength=80 |
| fields | FieldRules | 是 | 见同名DTO与通用约束 |
| approvalMode | ApprovalModeValue | 是 | 见同名DTO与通用约束 |
| phoneVerificationEnabled | boolean | 是 | 见同名DTO与通用约束 |
| registrationAvailable | boolean | 是 | 见同名DTO与通用约束 |
| notice | string | 是 | minLength=1; maxLength=4000 |
| noticeVersion | string | 是 | minLength=1; maxLength=80 |
| retentionDays | integer | 是 | format=int32; minimum=1; maximum=2147483647 |

#### PartyView

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| partyId | string | 是 | format=uuid |
| partyReference | string | 是 | maxLength=50 |
| eventId | string | 是 | format=uuid |
| fields | PartyInput | 是 | 见通用规则 |
| requestedSize | integer | 是 | format=int32；minimum=1；maximum=2147483647 |
| approvedSize | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| pendingIncrease | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| netAdmitted | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| remainingCount | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| status | PartyStatusValue | 是 | 见通用规则 |
| progress | AdmissionProgressValue | 是 | 见通用规则 |
| sourceType | SourceTypeValue | 是 | 见通用规则 |
| version | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| createdAt | string | 是 | format=date-time |
| updatedAt | string | 是 | format=date-time |
| hasAdmissionHistory | boolean | 是 | 见通用规则 |

#### PartyAccessBundle

仅原提交客户端、验证恢复或负责人核对获取；私密fragment由页面读取后清除，查询及变更能力分开。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| queryToken | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| changeToken | string | 否（省略） | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### SubmissionView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| party | PartyView | 是 | 见同名DTO与通用约束 |
| access | PartyAccessBundle | 是 | 见同名DTO与通用约束 |
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| replayed | boolean | 是 | 见同名DTO与通用约束 |

#### ReissueView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| party | PartyView | 是 | 见同名DTO与通用约束 |
| credential | CredentialView | 是 | 见同名DTO与通用约束 |
| access | PartyAccessBundle | 否（省略） | 见同名DTO与通用约束 |
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| replayed | boolean | 是 | 见同名DTO与通用约束 |

#### PartyStateView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyReference | string | 是 | minLength=1; maxLength=50 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| fields | PartyInput | 是 | 见同名DTO与通用约束 |
| status | PartyStatusValue | 是 | 见同名DTO与通用约束 |
| approvedSize | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| pendingIncrease | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| netAdmitted | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| remainingCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| progress | AdmissionProgressValue | 是 | 见同名DTO与通用约束 |
| canChange | boolean | 是 | 见同名DTO与通用约束 |
| canCancel | boolean | 是 | 见同名DTO与通用约束 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### CredentialView

只有ACTIVE且资格有效时返回ticketToken；待审核无有效票；全部no-store。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| status | CredentialStatusValue | 是 | 见同名DTO与通用约束 |
| credentialVersion | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| ticketToken | string | 否（省略） | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### ReviewView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| reviewId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| kind | ReviewKindValue | 是 | 见同名DTO与通用约束 |
| requestedSize | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| approvedSize | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| requestedIncrease | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| result | ReviewResultValue | 是 | 见同名DTO与通用约束 |
| reason | string | 否（省略） | minLength=1; maxLength=500 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| createdAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### GatePreview

绑定活动/终端/组/票版本，5分钟；不含完整PII且不核销。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| scanContextToken | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| approvedSize | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| netAdmitted | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| remainingCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| partyVersion | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| progress | AdmissionProgressValue | 是 | 见同名DTO与通用约束 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### GateCandidate

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyReference | string | 是 | minLength=1; maxLength=50 |
| displayNameMasked | string | 否（省略） | minLength=1; maxLength=100 |
| phoneSuffix | string | 否（省略） | minLength=1; maxLength=4 |
| approvedSize | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| netAdmitted | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| remainingCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| progress | AdmissionProgressValue | 是 | 见同名DTO与通用约束 |

#### GateCandidatePage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<GateCandidate> | 是 | maxItems=10 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | const=1 |
| pageSize | integer | 是 | const=10 |

#### GateResult

补查/重放replayed=true且showAsNewAdmission=false；原时间人数可追溯，不再提示新增放行。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| recordId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| status | OperationStatusValue | 是 | 见同名DTO与通用约束 |
| requestedCount | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| committedCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| remainingCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| committedAt | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| replayed | boolean | 是 | 见同名DTO与通用约束 |
| showAsNewAdmission | boolean | 是 | 见同名DTO与通用约束 |

#### AdmissionView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| recordId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| deviceId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| actorId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| admittedCount | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| correctedCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| committedAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### CorrectionView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| correctionId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| originalRecordId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| partyId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| correctedCount | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| reason | string | 是 | minLength=1; maxLength=500 |
| createdAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |

#### OperationView

仅原主体或获权活动范围；查无记录仍同ID重试，不表示可以另发一笔。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| status | OperationStatusValue | 是 | 见同名DTO与通用约束 |
| action | string | 是 | minLength=1; maxLength=100 |
| resourceId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| committedAt | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| resultCode | string | 是 | minLength=1; maxLength=80 |
| replayed | boolean | 是 | 见同名DTO与通用约束 |

#### DeviceView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| deviceId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| name | string | 是 | minLength=1; maxLength=100 |
| mode | DeviceModeValue | 是 | 见同名DTO与通用约束 |
| status | DeviceStatusValue | 是 | 见同名DTO与通用约束 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| lastSeenAt | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### DeviceRegistration

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| device | DeviceView | 是 | 见同名DTO与通用约束 |
| pairingCode | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| pairingExpiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### DeviceSessionView

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| device | DeviceView | 是 | 见通用规则 |
| csrfToken | string | 是 | maxLength=4096 |
| expiresAt | string | 是 | format=date-time |
| eventStatus | EventStatusValue | 是 | 见通用规则 |

#### ConnectionView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| connectionId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| provider | ProviderValue | 是 | 见同名DTO与通用约束 |
| accountKind | AccountKindValue | 是 | 见同名DTO与通用约束 |
| displayName | string | 是 | minLength=1; maxLength=100 |
| status | ConnectionStatusValue | 是 | 见同名DTO与通用约束 |
| allowedEventIds | array<string> | 是 | maxItems=1000 |
| credentialMask | string | 否（省略） | minLength=1; maxLength=50 |
| expiresAt | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### AuthorizationView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| authorizationUrl | string | 是 | format=uri; minLength=1; maxLength=4096 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### ResourceView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| resourceId | string | 是 | minLength=1; maxLength=300 |
| title | string | 是 | minLength=1; maxLength=300 |
| kind | string | 是 | minLength=1; maxLength=60 |
| readable | boolean | 是 | 见同名DTO与通用约束 |

#### ResourcePage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<ResourceView> | 是 | maxItems=1000 |
| nextCursor | string | 否（省略） | minLength=1; maxLength=1000 |

#### FieldMapping

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| nameColumn | string | 否（省略） | minLength=1; maxLength=200 |
| phoneColumn | string | 否（省略） | minLength=1; maxLength=200 |
| emailColumn | string | 否（省略） | minLength=1; maxLength=200 |
| countColumn | string | 否（省略） | minLength=1; maxLength=200 |
| countMode | CountModeCode | 是 | 见同名DTO与通用约束 |

#### SourceView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| sourceId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| connectionId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| resourceId | string | 是 | minLength=1; maxLength=300 |
| provider | ProviderValue | 是 | 见同名DTO与通用约束 |
| mapping | FieldMappingView | 是 | 见同名DTO与通用约束 |
| eligibility | SourceEligibilityValue | 是 | 见同名DTO与通用约束 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### SourceListView

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| items | array<SourceView> | 是 | maxItems=1000 |
| availableConnections | array<SourceConnectionOption> | 是 | maxItems=1000 |

#### BatchView

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| batchId | string | 是 | format=uuid |
| eventId | string | 是 | format=uuid |
| sourceId | string | 否，省略 | format=uuid |
| fileDigest | string | 否，省略 | maxLength=64 |
| status | BatchStatusValue | 是 | 见通用规则 |
| eligibility | SourceEligibilityValue | 否，省略 | 见通用规则 |
| mapping | FieldMappingView | 否，省略 | 见通用规则 |
| previewVersion | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| validRows | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| errorRows | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| suspectRows | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| policyVersion | integer | 否，省略 | format=int32；minimum=0；maximum=2147483647 |
| expiresAt | string | 是 | format=date-time |
| version | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| availableColumns | array<SourceColumnOption> | 是 | maxItems=200 |

#### ImportRowView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| rowId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| sourceRecordId | string | 否（省略） | minLength=1; maxLength=300 |
| lineNumber | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| fields | PartyInput | 是 | 见同名DTO与通用约束 |
| requestedSize | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| status | RowStatusValue | 是 | 见同名DTO与通用约束 |
| errors | array<FieldError> | 是 | maxItems=1000 |
| linkedPartyId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |

#### BatchCommitView

选中行整体事务，容量/字段/来源冲突全部回滚，无部分有效票。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| batchId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| operationId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| createdCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| linkedCount | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| allocatedPeople | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| replayed | boolean | 是 | 见同名DTO与通用约束 |

#### JobView

202仅持久受理，不是核销/投递成功；管理员只看脱敏任务元数据。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| jobId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| kind | JobKindValue | 是 | 见同名DTO与通用约束 |
| status | JobStatusValue | 是 | 见同名DTO与通用约束 |
| attempts | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| createdAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| completedAt | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| resultRef | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| errorCode | string | 否（省略） | minLength=1; maxLength=80 |
| expiresAt | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### ExportView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| exportId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| status | JobStatusValue | 是 | 见同名DTO与通用约束 |
| format | ExportFormatValue | 是 | 见同名DTO与通用约束 |
| columns | array<string> | 是 | maxItems=1000 |
| rowCount | integer | 否（省略） | format=int32; minimum=0; maximum=2147483647 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### StatisticsView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| asOf | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| applicationGroups | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| pendingGroups | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| approvedGroups | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| allocatedPeople | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| grossAdmissions | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| correctedPeople | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| netAdmissions | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| unusedPlaces | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| walkInGroups | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| onsiteAddedPlaces | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| admissionRate | ['number', 'null'] | 是 | minimum=0; maximum=1 |

#### AuditView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| auditId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| action | string | 是 | minLength=1; maxLength=100 |
| actorId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| deviceId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| resourceId | string | 否（省略） | format=uuid; 不透明字符串标识，不是访问能力。 |
| reason | string | 否（省略） | minLength=1; maxLength=500 |
| occurredAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### DeliveryView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| deliveryId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| eventId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| exportId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| recipient | string | 是 | minLength=1; maxLength=200 |
| purpose | string | 是 | minLength=1; maxLength=500 |
| columns | array<string> | 是 | maxItems=1000 |
| retentionNote | string | 是 | minLength=1; maxLength=1000 |
| createdAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### CleanupView

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| cleanupId | string | 是 | format=uuid |
| eventId | string | 是 | format=uuid |
| dueAt | string | 是 | format=date-time |
| status | JobStatusValue | 是 | 见通用规则 |
| processedCount | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| failedCount | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| completedAt | string | 否，省略 | format=date-time |
| version | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| scopes | array<CleanupScopeResult> | 是 | minItems=6；maxItems=6 |

#### ChallengeView

恢复流程统一受理响应，不泄漏参考号/联系方式是否匹配。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| challengeId | string | 是 | format=uuid; 不透明字符串标识，不是访问能力。 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| retryAfterSeconds | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### VerificationView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| verificationProof | string | 是 | minLength=16; maxLength=4096; 高熵秘密，仅 TLS 受控请求/响应，禁止写普通日志、URL查询或导出。 |
| expiresAt | string | 是 | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### OptionsView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<EventView> | 是 | maxItems=1000 |
| nextCursor | string | 否（省略） | minLength=1; maxLength=1000 |

#### PartyFilter

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| status | PartyStatusCode | 否（省略） | 见同名DTO与通用约束 |
| progress | AdmissionProgressCode | 否（省略） | 见同名DTO与通用约束 |
| sourceType | SourceTypeCode | 否（省略） | 见同名DTO与通用约束 |
| createdFrom | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| createdTo | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |

#### EventPolicyInput

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| fields | FieldRules | 是 | 见同名DTO与通用约束 |
| approvalMode | ApprovalModeCode | 是 | 见同名DTO与通用约束 |
| phoneVerificationEnabled | boolean | 是 | 见同名DTO与通用约束 |
| selfEditEnabled | boolean | 是 | 见同名DTO与通用约束 |
| selfCancelEnabled | boolean | 是 | 见同名DTO与通用约束 |
| selfServiceEndsAt | string | 否（省略） | format=date-time; 带时区偏移的 RFC3339，服务端 UTC。 |
| retentionDays | integer | 是 | format=int32; minimum=1; maximum=3650; default=30 |
| version | integer | 是 | format=int32; minimum=0; maximum=2147483647 |

#### FieldMappingView

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| nameColumn | string | 否（省略） | minLength=1; maxLength=200 |
| phoneColumn | string | 否（省略） | minLength=1; maxLength=200 |
| emailColumn | string | 否（省略） | minLength=1; maxLength=200 |
| countColumn | string | 否（省略） | minLength=1; maxLength=200 |
| countMode | CountModeValue | 是 | 见同名DTO与通用约束 |

#### AccountPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<AccountView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### GrantPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<GrantView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### EventPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<EventView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### PartyPage

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| items | array<PartyView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32；minimum=0；maximum=2147483647 |
| page | integer | 是 | format=int32；minimum=1；maximum=2147483647 |
| pageSize | integer | 是 | format=int32；minimum=1；maximum=100 |
| asOf | string | 是 | format=date-time |

#### ReviewPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<ReviewView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### ImportRowPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<ImportRowView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### AdmissionPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<AdmissionView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### AuditPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<AuditView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### JobPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<JobView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### DeliveryPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<DeliveryView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### ConnectionPage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<ConnectionView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### DevicePage

遵第2—3节及引用接口的约束。

| 字段 | 类型/引用 | 必填 | 边界/说明 |
|---|---|---|---|
| items | array<DeviceView> | 是 | maxItems=100 |
| total | integer | 是 | format=int32; minimum=0; maximum=2147483647 |
| page | integer | 是 | format=int32; minimum=1; maximum=2147483647 |
| pageSize | integer | 是 | format=int32; minimum=1; maximum=100 |

#### SourceColumnOption

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| code | string | 是 | maxLength=200 |
| label | string | 是 | maxLength=200 |

#### ResourceFieldsView

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| resourceId | string | 是 | maxLength=300 |
| availableColumns | array<SourceColumnOption> | 是 | maxItems=200 |

#### CleanupScopeResult

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| scope | CleanupScopeValue | 是 | 见通用规则 |
| status | JobStatusValue | 是 | 见通用规则 |
| processedCount | integer | 是 | minimum=0；maximum=2147483647 |
| failedCount | integer | 是 | minimum=0；maximum=2147483647 |
| errorCode | string | 否，省略 | maxLength=80 |

#### GrantEventOption

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| eventId | string | 是 | format=uuid |
| title | string | 是 | maxLength=200 |
| startsAt | string | 是 | format=date-time |

#### GrantEventOptionsView

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| items | array<GrantEventOption> | 是 | maxItems=100 |
| nextCursor | string | 否，省略 | maxLength=1000 |

#### SourceConnectionOption

| 字段 | 类型/引用 | 必填 | 边界 |
|---|---|---|---|
| connectionId | string | 是 | format=uuid |
| displayName | string | 是 | maxLength=100 |
| provider | ProviderValue | 是 | 见通用规则 |
| accountKind | AccountKindValue | 是 | 见通用规则 |
| status | ConnectionStatusValue | 是 | 见通用规则 |

### 4.4 统一错误与恢复

| HTTP | code | 客户端处理 |
|---|---|---|
| 400 | INVALID_REQUEST / IDEMPOTENCY_PAYLOAD_MISMATCH | 纠正请求；同ID载荷不一致先核对原结果。 |
| 401 | UNAUTHENTICATED / SESSION_EXPIRED | 重新建立授权会话；原动作先核对，不能另起核销。 |
| 403 | FORBIDDEN / CSRF_INVALID | 停止该动作，申请相应权限或更新CSRF。 |
| 404 | RESOURCE_NOT_FOUND / OPERATION_NOT_FOUND | 不泄漏跨范围存在性；原操作无记录仍用原ID重试。 |
| 409 | VERSION_CONFLICT / CAPACITY_EXCEEDED / PARTY_REMAINING_EXCEEDED / EVENT_STATE_CONFLICT / PUBLIC_MUTATIONS_FROZEN / PREVIEW_STALE / SOURCE_CONFLICT / OPERATION_IN_PROGRESS | 人数/版本先刷新并明确确认新动作；IN_PROGRESS沿原ID查询，不新建。 |
| 410 | TOKEN_EXPIRED / ARTIFACT_EXPIRED | 停止使用旧能力/文件；恢复入口或联系负责人，不重复入场。 |
| 422 | FIELD_VALIDATION_FAILED / TICKET_UNAVAILABLE / CREDENTIAL_INVALID / VERIFICATION_REQUIRED / DUPLICATE_REVIEW_REQUIRED | 按字段/资格处理，不视为已放行。 |
| 429 | RATE_LIMITED | 遵Retry-After，不循环轰炸。 |
| 500 | INTERNAL_ERROR | 结果未知，先按原ID查询；记录requestId排查。 |
| 503 | SERVICE_UNAVAILABLE / PROVIDER_UNAVAILABLE | 结果未知，暂停该端并原ID恢复；后台任务看最终状态。 |

### 4.5 核销成功、冲突和未知结果示例

以下为合同示例，ID/上下文是假数据，未向真实服务发请求。已识别当前组且剩3人，本次核销2人：

```json
{
  "operationId": "ed6c48bc-50cb-4420-b805-e22d6f19c1ec",
  "eventId": "43b045c1-37df-4ab9-a5d3-ddb1fb20f94f",
  "scanContextToken": "EXAMPLE_SCAN_CONTEXT_TOKEN_0001",
  "expectedPartyVersion": 7,
  "count": 2
}
```

HTTP200首次提交成功：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "operationId": "ed6c48bc-50cb-4420-b805-e22d6f19c1ec",
    "recordId": "cb08da51-d8ca-48ae-8b3f-e86957dbbd40",
    "status": {"code": "SUCCEEDED", "label": "成功"},
    "requestedCount": 2,
    "committedCount": 2,
    "remainingCount": 1,
    "committedAt": "2026-10-21T09:00:00Z",
    "replayed": false,
    "showAsNewAdmission": true
  },
  "requestId": "18ca37f2-827c-4570-be42-3b2b7311758c"
}
```

同请求重放/查询时仍返回原人数与时间，但replayed=true、showAsNewAdmission=false，不再次显示新放行。若另一设备已先改变组版本且本操作确定未提交，HTTP409：

```json
{
  "code": "VERSION_CONFLICT",
  "message": "组状态已变化，请刷新并确认本次人数",
  "data": {
    "operationId": "ed6c48bc-50cb-4420-b805-e22d6f19c1ec",
    "retryMode": "NEW_AFTER_CONFIRM"
  },
  "requestId": "18ca37f2-827c-4570-be42-3b2b7311758c"
}
```

断网/HTTP超时则没有可信成功或确定拒绝响应，客户端显示待确认、禁再次核销，恢复后以原ID查询/重试。不得生成一个假“失败”结果并自动另起一笔。导入CAPACITY_EXCEEDED同属409，但整批回滚，不能从失败响应假设有部分票可用。

## 5. 模块清单汇总

### 5.1 服务、仓储、SQL与事件

应用服务见4.1，仓储/SQL清单及DDL逐表定义于data-model-unigate.md（设计SQL不执行）。读写统一数据库事务，不用HTTP串联占额和审批。对外无通用表CRUD、无外部任意改删票接口、无MQ或工作流回调契约。

内部持久事件 PartyApproved、PartyChanged、CredentialReissued、AdmissionCommitted、AdmissionCorrected、ImportCommitted、EventEnded、PersonalDataDue，仅在对应事务提交后由worker消费；eventId/opaque resourceId/operationId/createdAt/schemaVersion为最小载荷，不放票码、完整联系方式和密钥。投递目标需读取当前受权/未过期数据；幂等键=事件ID+消费者用途。通知私密链接使用受管密文秘密引用，清理时同时作废事件目标，不能在失败队列长期保留PII。统计首版同库快照，不依赖消费事件决定是否放行。

### 5.2 FL—AA—API 追溯

每行FL至少一个同AA操作；UI同页动作可复用多个接口，43项功能不等于43个接口。下列追溯检查覆盖功能入口，不把角色写在接口元数据里当成已验证运行授权。

| FL | AA | 关联操作 |
|---|---|---|
| FL-A-001 | 2.4.1.1 | SessionBootstrap, Login, Logout, SessionMe, AccountList, AccountCreate, AccountUpdate, PasswordChange, GrantList, GrantSave, GrantRevoke, EventList, PartyExchange |
| FL-A-002 | 2.4.2.1 | ConnectionList, ConnectionSave, ConnectionAuthorize, ConnectionCallback, ConnectionRevoke |
| FL-A-003 | 2.4.3.1 | ResourceList, SourcePull, JobList, JobDetail, JobRetry, CleanupRetry |
| FL-O-001 | 2.1.1.1 | SubmitRegistration, ReviewList, ReviewDecide, BatchCommit |
| FL-O-002 | 2.1.1.2 | MyPartyChange, MyPartyCancel, PartyList, PartyDetail, PartyChange, PartyCancel |
| FL-O-003 | 2.1.2.1 | EventTransition |
| FL-O-004 | 2.1.2.2 | EventTransition |
| FL-O-005 | 2.1.2.3 | TicketResolve, GateCommit |
| FL-O-006 | 2.1.2.4 | GateWalkIn, GateIncrease |
| FL-O-007 | 2.1.2.5 | GateOperation, GateCorrect |
| FL-O-008 | 2.2.1.1 | EventList, EventDetail, EventCreate, EventUpdate, EventOptions |
| FL-O-009 | 2.2.1.2 | PolicyGet, PolicyUpdate |
| FL-O-010 | 2.2.2.2 | ResourceList, SourceSave, SourceList, SourcePull, BatchUpload, BatchDetail, BatchPreview, BatchRows, DuplicateResolve, BatchCommit |
| FL-O-011 | 2.2.2.2 | ResourceList, SourceSave, SourceList, SourcePull, BatchUpload, BatchDetail, BatchPreview, BatchRows, DuplicateResolve, BatchCommit |
| FL-O-012 | 2.1.1.3 | MyParty, MyCredential, CredentialReissue, CredentialRevoke |
| FL-O-013 | 2.1.1.4 | RecoveryStart, RecoveryVerify, CredentialReissue, GateLookup, EmailResend |
| FL-O-014 | 2.2.3.1 | EventTransition |
| FL-O-015 | 2.2.3.2 | PartyList, PartyDetail, Statistics, ExportCreate, ExportDetail, ExportDownload |
| FL-O-016 | 2.2.3.2 | PartyList, PartyDetail, Statistics, ExportCreate, ExportDetail, ExportDownload |
| FL-O-017 | 2.2.4.1 | CleanupDetail, CleanupRetry |
| FL-O-018 | 2.2.4.2 | DeliveryList, DeliveryRecord |
| FL-O-019 | 2.4.1.1 | SessionBootstrap, Login, Logout, SessionMe, AccountList, AccountCreate, AccountUpdate, PasswordChange, GrantList, GrantSave, GrantRevoke, EventList, PartyExchange |
| FL-O-020 | 2.4.1.2 | DeviceRegister, DeviceClaim, DeviceDetail, DeviceList, DeviceRevoke |
| FL-O-021 | 2.4.3.1 | ResourceList, SourcePull, JobList, JobDetail, JobRetry, CleanupRetry |
| FL-O-022 | 2.4.3.3 | GateRecords, AuditList |
| FL-O-023 | 2.4.3.2 | MyOperation, GateCommit, GateOperation, OperationResult |
| FL-C-001 | 2.1.2.3 | TicketResolve, GateCommit |
| FL-C-002 | 2.1.2.5 | GateOperation, GateCorrect |
| FL-C-003 | 2.1.1.4 | RecoveryStart, RecoveryVerify, CredentialReissue, GateLookup, EmailResend |
| FL-C-004 | 2.4.1.1 | SessionBootstrap, Login, Logout, SessionMe, AccountList, AccountCreate, AccountUpdate, PasswordChange, GrantList, GrantSave, GrantRevoke, EventList, PartyExchange |
| FL-C-005 | 2.4.1.2 | DeviceRegister, DeviceClaim, DeviceDetail, DeviceList, DeviceRevoke |
| FL-C-006 | 2.4.3.3 | GateRecords, AuditList |
| FL-C-007 | 2.4.3.2 | MyOperation, GateCommit, GateOperation, OperationResult |
| FL-R-001 | 2.2.3.2 | PartyList, PartyDetail, Statistics, ExportCreate, ExportDetail, ExportDownload |
| FL-R-002 | 2.4.1.1 | SessionBootstrap, Login, Logout, MyGrantList, SessionMe, AccountList, AccountCreate, AccountUpdate, PasswordChange, GrantList, GrantSave, GrantRevoke, EventList, PartyExchange |
| FL-P-001 | 2.1.1.2 | MyPartyChange, MyPartyCancel, PartyList, PartyDetail, PartyChange, PartyCancel |
| FL-P-002 | 2.1.2.3 | TicketResolve, GateCommit |
| FL-P-003 | 2.2.2.1 | PublicEvent, SubmitRegistration, SmsStart, SmsVerify |
| FL-P-004 | 2.1.1.3 | MyParty, MyCredential, CredentialReissue, CredentialRevoke |
| FL-P-005 | 2.1.1.4 | RecoveryStart, RecoveryVerify, CredentialReissue, GateLookup, EmailResend |
| FL-P-006 | 2.4.1.1 | SessionBootstrap, Login, Logout, SessionMe, AccountList, AccountCreate, AccountUpdate, PasswordChange, GrantList, GrantSave, GrantRevoke, EventList, PartyExchange |
| FL-P-007 | 2.4.1.2 | DeviceRegister, DeviceClaim, DeviceDetail, DeviceList, DeviceRevoke |
| FL-P-008 | 2.4.3.2 | MyOperation, GateCommit, GateOperation, OperationResult |

### 5.3 设计验证边界

JSON引用、接口/路由唯一性、请求字段/幂等键、AA/FL追溯和枚举检查见05-reports/D3-contract-check.md。尚无HTTP服务、迁移执行、并发测试、实际账户授权或手机/平板测试。插件声明的prd-to-api-design/template.md、check.sh和checklist.md在安装包及技能路径搜索未找到；本地补充检查不能冒称其CH/GL全部通过。D3按ADR-043项目检查依据执行，原版缺失记录保留且不阻塞；D4由ADR-044集中确认并归档，独立Gate1结论见[门控报告](../../05-reports/gate1-unigate-2026-10-21.md)。

## 6. 更新记录

| 版本 | 日期 | 类型 | 变更与影响 |
|---|---|---|---|
| v0.1 | 2026-10-21 | 初始设计 | 基于ADR-041新增在线API、字段/错误、幂等/权限/事务、数据设计及追溯；无既有接口破坏迁移，待合同审阅及完整阶段检查。 |
| v0.2 | 2026-10-21 | 一致性修正 | ADR-042采用接口；补齐MyGrantList、匿名上下文连续性请求头及检票员记录范围，修复既定功能缺口，不增加角色业务权限。 |
| v0.3 | 2026-10-21 | 交互契约补齐 | ADR-043采用项目检查；补齐来源字段/授权活动元数据、报名原结果读取、清理分范围、终端阶段、核销历史和固定GATE幂等作用域，修复既定功能的展示/重试缺口，不增加名单权限。 |
| v0.3注记 | 2026-10-21 | 阶段引用 | ADR-044确认D4；验证边界改为指向独立Gate1报告，接口及字段保持0.3.0。 |
| v0.4 | 2026-11-11 | 领域重划分 | ADR-047：85操作AA编号重挂AA v0.5（§4.2/§5.2及openapi x-aa-submodules），§1.4/§4.1对齐8上下文+平台内核（DeviceSession更名Device）；路径、DTO、枚举、DDL零变更，OpenAPI合同版本仍为0.3.0。 |
