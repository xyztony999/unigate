# UniGate reader 页面规格

| 属性 | 值 |
|---|---|
| 版本 | v1.0 |
| 日期 | 2026-10-21 |
| 状态 | ADR-044确认七项交互，D4归档；设计规格，运行验证另行执行 |
| App / 模块 | UniGate Web / unigate |
| 输入 | API v0.3 / OpenAPI0.3.0；ADR-042/043/044 |
| 类型 | 业务流程+来源集成+只读报表；页面使用本栈组件语义，不套用SCM引擎 |

全局规则、完整状态机及枚举索引见../../README.md；字段类型由page-contracts.json与OpenAPI精确绑定。表中系统字段不作为普通输入显示，所有按钮仍由服务端鉴权/校验。

## R01 获权活动统计

路由`/read/events/:eventId/statistics`；角色：READER；布局：statistics；FL：FL-R-001。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：只读活动指标和时点，过期移除活动；无核销/导出或管理入口。

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
| Statistics.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |

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
| Statistics.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.asOf | 统计时点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.applicationGroups | 申请组数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.pendingGroups | 待审组数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.approvedGroups | 已批准组数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.allocatedPeople | 有效名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.grossAdmissions | 原核销人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.correctedPeople | 更正人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.netAdmissions | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.unusedPlaces | 未使用名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.walkInGroups | 现场补录组数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.onsiteAddedPlaces | 现场增加名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Statistics.admissionRate | 累计核销率 | number/null | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| EventList | POST /api/v1/event/event/list | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；服务端限定活动范围，按startsAt,eventId稳定降序；管理员须另有业务角色。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| MyGrantList | POST /api/v1/identity/grant/mine | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；accountId固定从会话取，禁止请求他人账号；eventId仅收窄本人范围。只返回有效活动/字段/动作/期限，展示授权不扩权；按expiresAt,grantId稳定升序。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| Statistics | GET /api/v1/report/attendance/summary | STAFF / ORGANIZER,READER | 本活动VIEW_STATISTICS获权；同一数据库一致读快照返回asOf，含原核销/更正/净数；零分母rate=null，无当前在场人数。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## R02 获准字段明细

路由`/read/events/:eventId/parties`；角色：READER；布局：responsive_list；FL：FL-R-001。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：服务端只返回获准字段，未获字段不先下载；详情只读，不提供导出。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| PartyList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| PartyList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| PartyList.status | 状态 | PartyStatusCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PartyList.progress | 入场进度 | AdmissionProgressCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PartyList.sourceType | 来源 | SourceTypeCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PartyList.query | 搜索 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| PartyList.createdFrom | 创建开始 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| PartyList.createdTo | 创建截止 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| PartyList.hasPendingIncrease | 有待审增员 | boolean | switch | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PartyDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyDetail.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| MyGrantList.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| MyGrantList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| MyGrantList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | partyReference | 报名参考号 | 服务端允许且字段存在 | left |
| 2 | fields.name | 姓名 | 服务端允许且字段存在 | 否 |
| 3 | fields.phone | 手机号 | 服务端允许且字段存在 | 否 |
| 4 | fields.email | 邮箱 | 服务端允许且字段存在 | 否 |
| 5 | requestedSize | 申请总人数 | 服务端允许且字段存在 | 否 |
| 6 | approvedSize | 批准名额 | 服务端允许且字段存在 | 否 |
| 7 | netAdmitted | 累计净入场 | 服务端允许且字段存在 | 否 |
| 8 | remainingCount | 剩余可入场 | 服务端允许且字段存在 | 否 |
| 9 | status | 状态 | 服务端允许且字段存在 | 否 |
| 10 | progress | 入场进度 | 服务端允许且字段存在 | 否 |
| 11 | sourceType | 来源 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| PartyList.items[].partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyList.items[].fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyList.items[].fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyList.items[].fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyList.items[].requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.items[].hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyList.asOf | 统计时点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyDetail.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyDetail.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyDetail.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyDetail.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyDetail.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| PartyList | POST /api/v1/enrollment/party/list | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权，只返回/检索获准字段；READER只投影获准字段；createdAt,partyId稳定降序，不返回票码/私密能力。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyDetail | GET /api/v1/enrollment/party/detail | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权且本组属该活动；跨活动或未获字段不返回；字段按权限省略，不先发全量再隐藏。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| MyGrantList | POST /api/v1/identity/grant/mine | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；accountId固定从会话取，禁止请求他人账号；eventId仅收窄本人范围。只返回有效活动/字段/动作/期限，展示授权不扩权；按expiresAt,grantId稳定升序。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

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
