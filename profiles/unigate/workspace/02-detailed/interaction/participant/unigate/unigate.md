# UniGate participant 页面规格

| 属性 | 值 |
|---|---|
| 版本 | v1.0 |
| 日期 | 2026-10-21 |
| 状态 | ADR-044确认七项交互，D4归档；设计规格，运行验证另行执行 |
| App / 模块 | UniGate Web / unigate |
| 输入 | API v0.3 / OpenAPI0.3.0；ADR-042/043/044 |
| 类型 | 业务流程+来源集成+只读报表；页面使用本栈组件语义，不套用SCM引擎 |

全局规则、完整状态机及枚举索引见../../README.md；字段类型由page-contracts.json与OpenAPI精确绑定。表中系统字段不作为普通输入显示，所有按钮仍由服务端鉴权/校验。

## P01 活动报名

路由`/e/:eventId/register`；角色：PARTICIPANT；布局：mobile_form；FL：FL-P-003。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：微信内置浏览器/普通浏览器直接打开，动态四字段和告知，缺省随行0；一次提交，待审不承诺名额。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| PublicEvent.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsStart.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsStart.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsStart.phone | 手机号 | string | tel_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 40} |
| SmsVerify.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsVerify.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsVerify.challengeId | challengeId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsVerify.code | 验证码 | string | otp_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 12} |
| SubmitRegistration.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SubmitRegistration.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SubmitRegistration.fields.name | 姓名 | string | text_input | 按活动策略 | input；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 | {"minLength": 1, "maxLength": 100} |
| SubmitRegistration.fields.phone | 手机号 | string | tel_input | 按活动策略 | input；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 | {"minLength": 1, "maxLength": 40} |
| SubmitRegistration.fields.email | 邮箱 | string | email_input | 按活动策略 | input；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 | {"format": "email", "minLength": 1, "maxLength": 254} |
| SubmitRegistration.fields.companionCount | 随行人数 | integer | integer_stepper | 按活动策略 | input；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 | {"format": "int32", "minimum": 0, "maximum": 999999} |
| SubmitRegistration.phoneVerificationProof | phoneVerificationProof | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |
| SubmitRegistration.acceptedNoticeVersion | acceptedNoticeVersion | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 80} |
| RegistrationResult.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| RegistrationResult.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| PublicEvent.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.fields.name.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.fields.name.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.fields.phone.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.fields.phone.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.fields.email.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.fields.email.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.fields.companionCount.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.fields.companionCount.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.approvalMode | 审核方式 | ApprovalModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.phoneVerificationEnabled | 验证手机号 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.registrationAvailable | registrationAvailable | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.notice | 报名告知 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.noticeVersion | noticeVersion | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PublicEvent.retentionDays | 明细保留天数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.clientContextId | clientContextId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsStart.challengeId | challengeId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsStart.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsStart.retryAfterSeconds | retryAfterSeconds | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsVerify.verificationProof | verificationProof | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsVerify.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| SubmitRegistration.party.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| SubmitRegistration.party.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| SubmitRegistration.party.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| SubmitRegistration.party.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.party.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.access.queryToken | queryToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.access.changeToken | changeToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.access.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SubmitRegistration.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| RegistrationResult.party.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| RegistrationResult.party.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| RegistrationResult.party.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| RegistrationResult.party.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.access.queryToken | queryToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.access.changeToken | changeToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.access.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| PublicEvent | GET /api/v1/public/event/detail | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只展示已发布活动和告知/字段，不返回名单；未发布404，不泄漏其存在性。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SessionBootstrap | GET /api/v1/identity/session/bootstrap | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；同源GET，设置匿名客户端HttpOnly cookie及CSRF，未建立业务权限。 复用仍有效匿名cookie；返回clientContextId，标记变化时旧待确认报名不得自动重发。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SmsStart | POST /api/v1/public/verification/send | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；活动启用才可用；挑战5分钟/验证5次/发送间隔60秒；IP+电话+客户端限流；受理不等于已送达。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SmsVerify | POST /api/v1/public/verification/verify | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；证明绑定活动/电话/客户端，10分钟一次性；不验证全部随行身份。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SubmitRegistration | POST /api/v1/public/registration/submit | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；匿名客户端cookie+CSRF，READY/报名时间内；IMMEDIATE整组占额，REVIEW不占额不发有效票；同ID恢复仅原客户端可取私密能力。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| RegistrationResult | POST /api/v1/public/registration/result | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；原ANON cookie、CSRF及X-Client-Context-ID同时匹配，按eventId/SubmitRegistration/operationId定位原主体结果；不接收姓名/手机号，不重新报名。仅已提交原结果24小时内且明细未到期可取加密私密响应，返回replayed=true；无记录404仍先核对/原ID重试，执行中409，私密响应过期410转邮件/负责人。身份换新不能凭操作ID领取他人入口。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## P02 我的报名与入场码

路由`/me/party`；角色：PARTICIPANT；布局：ticket_status；FL：FL-P-001, FL-P-004, FL-P-006, FL-P-008。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：私密fragment读入后清除；有效票可保存截图，另提醒保存私密查询入口；待审/拒绝无有效票，修改按可用开关。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| PartyExchange.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyExchange.queryToken | queryToken | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |
| PartyExchange.changeToken | changeToken | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |
| MyPartyChange.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| MyPartyChange.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| MyPartyChange.companionCount | 随行人数 | integer | integer_stepper | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 999999} |
| MyPartyCancel.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| MyPartyCancel.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| MyOperation.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| MyOperation.action | 操作类型 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| RegistrationResult.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| RegistrationResult.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| SessionBootstrap.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.clientContextId | clientContextId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyExchange.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyExchange.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyExchange.clientContextId | clientContextId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyParty.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyParty.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyParty.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyParty.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.canChange | canChange | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.canCancel | canCancel | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyParty.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyCredential.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyCredential.status | 状态 | CredentialStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyCredential.credentialVersion | credentialVersion | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyCredential.ticketToken | ticketToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyCredential.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyPartyChange.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyPartyChange.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyPartyChange.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyPartyChange.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.canChange | canChange | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.canCancel | canCancel | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyChange.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyPartyCancel.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyPartyCancel.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyPartyCancel.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| MyPartyCancel.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.canChange | canChange | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.canCancel | canCancel | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyPartyCancel.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyOperation.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyOperation.status | 状态 | OperationStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyOperation.action | 操作类型 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyOperation.resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyOperation.committedAt | 核销时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyOperation.resultCode | resultCode | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyOperation.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| RegistrationResult.party.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| RegistrationResult.party.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| RegistrationResult.party.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| RegistrationResult.party.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.party.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.access.queryToken | queryToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.access.changeToken | changeToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.access.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RegistrationResult.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| SessionBootstrap | GET /api/v1/identity/session/bootstrap | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；同源GET，设置匿名客户端HttpOnly cookie及CSRF，未建立业务权限。 复用仍有效匿名cookie；返回clientContextId，标记变化时旧待确认报名不得自动重发。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyExchange | POST /api/v1/public/party/exchange | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；能力用请求体换HttpOnly cookie；变更能力独立可缺；页面读取URL fragment后清除，不写查询串或日志。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| MyParty | GET /api/v1/public/party/detail | PARTY / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；服务端仅从本组cookie定位，禁止传任意partyId查询别组；敏感响应no-store。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| MyCredential | GET /api/v1/public/credential/detail | PARTY / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅APPROVED且票ACTIVE，待审/拒绝422 TICKET_UNAVAILABLE；票码不含联系方式。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| MyPartyChange | POST /api/v1/public/party/change | CHANGE / PARTICIPANT | MyParty.canChange=true；独立CHANGE能力有效；开关/截止/READY/无核销守卫；审核模式只新增差额待审，原批准数保留，减少同步释放差额。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| MyPartyCancel | POST /api/v1/public/party/cancel | CHANGE / PARTICIPANT | MyParty.canCancel=true；独立CHANGE能力有效；仅允许窗口且未核销；释放已批准数、撤票及取消待增员同事务，不删除历史。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| MyOperation | POST /api/v1/public/operation/result | PARTY / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只返回当前组获准操作元数据；action必须是本组允许操作的固定operation名称，无票码/其他组信息；原提交尚未拿到cookie则同ID重试submit，不能另起一笔。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| RegistrationResult | POST /api/v1/public/registration/result | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；原ANON cookie、CSRF及X-Client-Context-ID同时匹配，按eventId/SubmitRegistration/operationId定位原主体结果；不接收姓名/手机号，不重新报名。仅已提交原结果24小时内且明细未到期可取加密私密响应，返回replayed=true；无记录404仍先核对/原ID重试，执行中409，私密响应过期410转邮件/负责人。身份换新不能凭操作ID领取他人入口。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：仅对有效票生成二维码图片并保存；仅原提交/验证恢复时复制私密查询入口；返回状态页。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## P03 找回本组入口

路由`/e/:eventId/recover`；角色：PARTICIPANT；布局：mobile_form；FL：FL-P-005。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：参考号+已留联系方式验证码，统一受理；无联系方式由负责人核对，不允许只用尾号。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| RecoveryStart.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| RecoveryStart.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| RecoveryStart.partyReference | 报名参考号 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 50} |
| RecoveryStart.channel | 验证方式 | RecoveryChannelCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| RecoveryStart.contact | 已留联系方式 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 254} |
| RecoveryVerify.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| RecoveryVerify.challengeId | challengeId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| RecoveryVerify.code | 验证码 | string | otp_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 12} |
| PartyExchange.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyExchange.queryToken | queryToken | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |
| PartyExchange.changeToken | changeToken | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| SessionBootstrap.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.clientContextId | clientContextId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RecoveryStart.challengeId | challengeId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RecoveryStart.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RecoveryStart.retryAfterSeconds | retryAfterSeconds | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RecoveryVerify.queryToken | queryToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RecoveryVerify.changeToken | changeToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| RecoveryVerify.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyExchange.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyExchange.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyExchange.clientContextId | clientContextId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| SessionBootstrap | GET /api/v1/identity/session/bootstrap | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；同源GET，设置匿名客户端HttpOnly cookie及CSRF，未建立业务权限。 复用仍有效匿名cookie；返回clientContextId，标记变化时旧待确认报名不得自动重发。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| RecoveryStart | POST /api/v1/public/recovery/start | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；统一202不泄漏是否匹配；仅匹配记录并有服务配置才投递；无参考号/联系方式转负责人，不用电话当唯一身份。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| RecoveryVerify | POST /api/v1/public/recovery/verify | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；挑战5分钟最多5次；成功旋转查询/变更能力，不自动增加名额或核销，旧能力失效。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyExchange | POST /api/v1/public/party/exchange | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；能力用请求体换HttpOnly cookie；变更能力独立可缺；页面读取URL fragment后清除，不写查询串或日志。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## P04 自助检票

路由`/terminal/kiosk`；角色：PARTICIPANT；布局：kiosk；FL：FL-P-002, FL-P-007, FL-P-008。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：受限设备先获权，不保存管理员会话；扫码后确认本次数量，完成清屏，未知时锁定交工作人员。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| TicketResolve.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| TicketResolve.ticketToken | ticketToken | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |
| GateCommit.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateCommit.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateCommit.scanContextToken | scanContextToken | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |
| GateCommit.expectedPartyVersion | expectedPartyVersion | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| GateCommit.count | 本次入场人数 | integer | integer_stepper | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647} |
| GateOperation.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateOperation.scanContextToken | scanContextToken | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| DeviceDetail.device.deviceId | deviceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.device.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.device.name | 姓名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.device.mode | 终端模式 | DeviceModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.device.status | 状态 | DeviceStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.device.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.device.lastSeenAt | 最近在线 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.device.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceDetail.eventStatus | 活动检票状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| TicketResolve.scanContextToken | scanContextToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| TicketResolve.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| TicketResolve.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| TicketResolve.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| TicketResolve.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| TicketResolve.partyVersion | partyVersion | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| TicketResolve.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| TicketResolve.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.recordId | recordId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.status | 状态 | OperationStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.requestedCount | 请求人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.committedCount | 本次已核销 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.committedAt | 核销时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCommit.showAsNewAdmission | showAsNewAdmission | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.recordId | recordId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.status | 状态 | OperationStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.requestedCount | 请求人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.committedCount | 本次已核销 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.committedAt | 核销时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateOperation.showAsNewAdmission | showAsNewAdmission | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| DeviceDetail | GET /api/v1/gate/device/detail | DEVICE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅受限设备cookie身份，绑定活动/模式；STAFF实际核销还需工作人员账号。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| TicketResolve | POST /api/v1/gate/ticket/resolve | GATE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；STAFF须获权账号+设备，KIOSK仅本次组票最小结果；签发5分钟组/设备/票版本上下文，不核销。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GateCommit | POST /api/v1/gate/checkin/commit | GATE / ORGANIZER,CHECKER,PARTICIPANT | 活动CHECKIN_OPEN、CHECKIN获权、当前上下文/设备有效、1<=count<=remainingCount、无待确认操作；事件锁→组锁，校验窗口/资格/票/端/账号/余额，保存结果与事实同事务；原成功重放不增加人数或再次绿灯。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GateOperation | POST /api/v1/gate/operation/result | GATE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限活动/检票员限获准操作；kiosk必须原组上下文，查无记录仍同ID重试，过期未知转负责人。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：相机权限请求/扫码枪输入；确认人数弹层；明确结果后完成/下一组；未知时锁定与联系工作人员。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

检票完整状态见全局SCANNER_STATES；默认本次1、范围1..remainingCount、显式确认；remaining=0禁提交。新核销成功才有绿色/可选提示音，原结果查询不再次放行。

成功结果展示5秒或点“完成”立即清组，空闲30秒清未提交组；unknown不自动清未确认ID，显示“请联系工作人员核对”。公开端不能退出到管理会话；配置/解锁在负责人私人端重新认证办理。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## 更新记录

| 版本 | 日期 | 类型 | 说明 |
|---|---|---|---|
| v0.1 | 2026-10-21 | 初始草案 | 从采用的API生成字段/动作/权限/状态绑定；待七项集中交互确认，不代表Gate1通过 |
| v1.0 | 2026-10-21 | 确认归档 | ADR-044选择1采用七项及默认流程；独立Gate1结果见报告，不授权业务代码 |
| v1.0注记 | 2026-10-21 | Gate1一致性 | 409按业务code细分：处理中保留原ID；明确冲突才允许确认新动作，对齐API4.4。 |
