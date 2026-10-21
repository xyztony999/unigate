# UniGate checker 页面规格

| 属性 | 值 |
|---|---|
| 版本 | v1.0 |
| 日期 | 2026-10-21 |
| 状态 | ADR-044确认七项交互，D4归档；设计规格，运行验证另行执行 |
| App / 模块 | UniGate Web / unigate |
| 输入 | API v0.3 / OpenAPI0.3.0；ADR-042/043/044 |
| 类型 | 业务流程+来源集成+只读报表；页面使用本栈组件语义，不套用SCM引擎 |

全局规则、完整状态机及枚举索引见../../README.md；字段类型由page-contracts.json与OpenAPI精确绑定。表中系统字段不作为普通输入显示，所有按钮仍由服务端鉴权/校验。

## C01 工作人员检票

路由`/check/events/:eventId/scan`；角色：CHECKER；布局：scanner；FL：FL-C-001, FL-C-002, FL-C-004, FL-C-005, FL-C-007。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：相机或扫码枪；STAFF账号与设备均获权；未知结果查原ID，无纠错、增额或补发。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| EventList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| EventList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| EventList.status | 状态 | EventStatusCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| EventList.query | 搜索 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| MyGrantList.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| MyGrantList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| MyGrantList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
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
| EventList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].grantId | grantId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].role | 活动角色 | RoleValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].allowedFields | 可查看字段 | array<PartyFieldValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].allowedActions | 获准动作 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].recipient | 接收方 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].purpose | 用途 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].retentionNote | 副本处置约定 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].accountDisplayName | 工作人员名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.items[].username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| MyGrantList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.roles | 账号角色 | array<RoleValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.mustChangePassword | mustChangePassword | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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
| EventList | POST /api/v1/event/event/list | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；服务端限定活动范围，按startsAt,eventId稳定降序；管理员须另有业务角色。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| MyGrantList | POST /api/v1/identity/grant/mine | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；accountId固定从会话取，禁止请求他人账号；eventId仅收窄本人范围。只返回有效活动/字段/动作/期限，展示授权不扩权；按expiresAt,grantId稳定升序。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SessionMe | GET /api/v1/identity/session/me | STAFF / ADMIN,ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只返回账号自身元数据，角色不是全活动数据授权。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DeviceDetail | GET /api/v1/gate/device/detail | DEVICE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅受限设备cookie身份，绑定活动/模式；STAFF实际核销还需工作人员账号。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| TicketResolve | POST /api/v1/gate/ticket/resolve | GATE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；STAFF须获权账号+设备，KIOSK仅本次组票最小结果；签发5分钟组/设备/票版本上下文，不核销。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GateCommit | POST /api/v1/gate/checkin/commit | GATE / ORGANIZER,CHECKER,PARTICIPANT | 活动CHECKIN_OPEN、CHECKIN获权、当前上下文/设备有效、1<=count<=remainingCount、无待确认操作；事件锁→组锁，校验窗口/资格/票/端/账号/余额，保存结果与事实同事务；原成功重放不增加人数或再次绿灯。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GateOperation | POST /api/v1/gate/operation/result | GATE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限活动/检票员限获准操作；kiosk必须原组上下文，查无记录仍同ID重试，过期未知转负责人。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：相机权限请求/扫码枪输入；确认人数弹层；明确结果后完成/下一组；未知时锁定与联系工作人员。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

检票完整状态见全局SCANNER_STATES；默认本次1、范围1..remainingCount、显式确认；remaining=0禁提交。新核销成功才有绿色/可选提示音，原结果查询不再次放行。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## C02 辅助找票

路由`/check/events/:eventId/lookup`；角色：CHECKER；布局：mobile_form；FL：FL-C-003。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：最多10个最小候选，名字掩码/末四位按许可显示，交负责人核对补发；尾号不直接放行。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| GateLookup.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateLookup.query | 搜索 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 2, "maxLength": 100} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | partyReference | 报名参考号 | 服务端允许且字段存在 | left |
| 2 | displayNameMasked | displayNameMasked | 服务端允许且字段存在 | 否 |
| 3 | phoneSuffix | phoneSuffix | 服务端允许且字段存在 | 否 |
| 4 | approvedSize | 批准名额 | 服务端允许且字段存在 | 否 |
| 5 | remainingCount | 剩余可入场 | 服务端允许且字段存在 | 否 |
| 6 | progress | 入场进度 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| GateLookup.items[].partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.items[].partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.items[].displayNameMasked | displayNameMasked | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.items[].phoneSuffix | phoneSuffix | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.items[].approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.items[].netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.items[].remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.items[].progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateLookup.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| GateLookup | POST /api/v1/gate/party/lookup | GATE / ORGANIZER,CHECKER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只允许STAFF+获权账号，最多10条固定第一页，仅姓名掩码/电话末四位（仅已收且获权时），无完整邮箱、票码或私密能力；KIOSK硬拒绝，尾号不用于补发，转负责人核对。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## C03 本人及本端记录

路由`/check/events/:eventId/records`；角色：CHECKER；布局：responsive_list；FL：FL-C-006。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：服务器限制本人或经当前cookie验证本端；无原操作重新放行按钮。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| GateRecords.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateRecords.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| GateRecords.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| GateRecords.deviceId | deviceId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateRecords.partyId | partyId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| AuditList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| AuditList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| AuditList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| AuditList.action | 操作类型 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| AuditList.from | 起始时间 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| AuditList.to | 截止时间 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| OperationResult.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| OperationResult.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| OperationResult.action | 操作类型 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| GateOperation.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateOperation.scanContextToken | scanContextToken | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | admittedCount | 原核销人数 | 服务端允许且字段存在 | left |
| 2 | correctedCount | 更正人数 | 服务端允许且字段存在 | 否 |
| 3 | committedAt | 核销时间 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| GateRecords.items[].recordId | recordId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.items[].partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.items[].operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.items[].deviceId | deviceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.items[].actorId | actorId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.items[].admittedCount | 原核销人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.items[].correctedCount | 更正人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.items[].committedAt | 核销时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateRecords.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.items[].auditId | auditId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.items[].action | 操作类型 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.items[].actorId | actorId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.items[].deviceId | deviceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.items[].resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.items[].reason | 原因 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.items[].occurredAt | occurredAt | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AuditList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| OperationResult.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| OperationResult.status | 状态 | OperationStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| OperationResult.action | 操作类型 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| OperationResult.resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| OperationResult.committedAt | 核销时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| OperationResult.resultCode | resultCode | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| OperationResult.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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
| GateRecords | POST /api/v1/gate/record/list | STAFF / ORGANIZER,CHECKER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；检票员只本人/获权端，过滤由服务端收缩，按committedAt,recordId稳定降序。 检票员范围固定为本人记录，或经当前设备cookie验证的本端记录；客户端传deviceId不能扩大范围。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| AuditList | POST /api/v1/platform/audit/list | STAFF / ORGANIZER,CHECKER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；检票员限本人/获权端，日志不含全量联系方式/票码/秘密；occurredAt,auditId稳定降序。 检票员范围固定为本人记录，或经当前设备cookie验证的本端记录；客户端传deviceId不能扩大范围。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| OperationResult | POST /api/v1/platform/operation/result | STAFF / ORGANIZER,CHECKER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限活动和原动作权限，检票员限获准操作；action是固定operation名称并校验原动作权限，不从通用查询取票或私密能力，恢复具体原请求仍用同ID。 检票员范围固定为本人记录，或经当前设备cookie验证的本端记录；客户端传deviceId不能扩大范围。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GateOperation | POST /api/v1/gate/operation/result | GATE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限活动/检票员限获准操作；kiosk必须原组上下文，查无记录仍同ID重试，过期未知转负责人。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## 更新记录

| 版本 | 日期 | 类型 | 说明 |
|---|---|---|---|
| v0.1 | 2026-10-21 | 初始草案 | 从采用的API生成字段/动作/权限/状态绑定；待七项集中交互确认，不代表Gate1通过 |
| v1.0 | 2026-10-21 | 确认归档 | ADR-044选择1采用七项及默认流程；独立Gate1结果见报告，不授权业务代码 |
| v1.0注记 | 2026-10-21 | Gate1一致性 | 409按业务code细分：处理中保留原ID；明确冲突才允许确认新动作，对齐API4.4。 |
