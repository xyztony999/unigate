# UniGate organizer 页面规格

| 属性 | 值 |
|---|---|
| 版本 | v1.0 |
| 日期 | 2026-10-21 |
| 状态 | ADR-044确认七项交互，D4归档；设计规格，运行验证另行执行 |
| App / 模块 | UniGate Web / unigate |
| 输入 | API v0.3 / OpenAPI0.3.0；ADR-042/043/044 |
| 类型 | 业务流程+来源集成+只读报表；页面使用本栈组件语义，不套用SCM引擎 |

全局规则、完整状态机及枚举索引见../../README.md；字段类型由page-contracts.json与OpenAPI精确绑定。表中系统字段不作为普通输入显示，所有按钮仍由服务端鉴权/校验。

## O01 活动列表与基本信息

路由`/manage/events`；角色：ORGANIZER；布局：management_list；FL：FL-O-008。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：列表进入活动；新增/编辑同基本信息表单，保存不自动发布。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| EventList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| EventList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| EventList.status | 状态 | EventStatusCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| EventList.query | 搜索 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| EventCreate.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EventCreate.title | 活动名称 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 200} |
| EventCreate.venue | 活动地点 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 300} |
| EventCreate.startsAt | 开始时间 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| EventCreate.endsAt | 结束时间 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| EventCreate.timeZone | 活动时区 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 80} |
| EventCreate.registrationStartsAt | 报名开始 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| EventCreate.registrationEndsAt | 报名截止 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| EventCreate.capacity | 人数容量 | integer | integer_stepper | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 1000000} |
| EventDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EventUpdate.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EventUpdate.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EventUpdate.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| EventUpdate.title | 活动名称 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 200} |
| EventUpdate.venue | 活动地点 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 300} |
| EventUpdate.startsAt | 开始时间 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| EventUpdate.endsAt | 结束时间 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| EventUpdate.timeZone | 活动时区 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 80} |
| EventUpdate.registrationStartsAt | 报名开始 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| EventUpdate.registrationEndsAt | 报名截止 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| EventUpdate.capacity | 人数容量 | integer | integer_stepper | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 1000000} |
| EventOptions.cursor | cursor | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | title | 活动名称 | 服务端允许且字段存在 | left |
| 2 | startsAt | 开始时间 | 服务端允许且字段存在 | 否 |
| 3 | venue | 活动地点 | 服务端允许且字段存在 | 否 |
| 4 | capacity | 人数容量 | 服务端允许且字段存在 | 否 |
| 5 | allocatedCount | 已分配名额 | 服务端允许且字段存在 | 否 |
| 6 | status | 状态 | 服务端允许且字段存在 | 否 |

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
| EventCreate.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventCreate.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventUpdate.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventOptions.nextCursor | nextCursor | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| EventList | POST /api/v1/event/event/list | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；服务端限定活动范围，按startsAt,eventId稳定降序；管理员须另有业务角色。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| EventCreate | POST /api/v1/event/event/create | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；标题/时间/时区/人数必填；结束不早于开始，创建者获本活动负责人授权，500是目标而非固定容量。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| EventDetail | GET /api/v1/event/event/detail | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅授权活动元数据，无报名明细或能力凭据。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| EventUpdate | POST /api/v1/event/event/update | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅DRAFT/READY，容量不能小于已占额；不静默改已批准组数据，版本不匹配409。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| EventOptions | GET /api/v1/event/event/options | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只列授权活动，startsAt,eventId稳定排序，每次最多100条。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O02 报名规则

路由`/manage/events/:eventId/settings`；角色：ORGANIZER；布局：settings_form；FL：FL-O-009。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：四字段启用及必填、审核/短信/自助/留存分组；现有报名后的受限改动按409处理。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| EventDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PolicyGet.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PolicyUpdate.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PolicyUpdate.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PolicyUpdate.policy.fields.name.enabled | 启用 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.fields.name.required | required | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.fields.phone.enabled | 启用 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.fields.phone.required | required | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.fields.email.enabled | 启用 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.fields.email.required | required | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.fields.companionCount.enabled | 启用 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.fields.companionCount.required | required | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.approvalMode | 审核方式 | ApprovalModeCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.phoneVerificationEnabled | 验证手机号 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.selfEditEnabled | 允许自助修改 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.selfCancelEnabled | 允许自助取消 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| PolicyUpdate.policy.selfServiceEndsAt | 自助截止 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| PolicyUpdate.policy.retentionDays | 明细保留天数 | integer | integer_stepper | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 3650, "default": 30} |
| PolicyUpdate.policy.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| EventDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.name.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.name.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.phone.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.phone.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.email.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.email.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.companionCount.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.companionCount.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.approvalMode | 审核方式 | ApprovalModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.phoneVerificationEnabled | 验证手机号 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfEditEnabled | 允许自助修改 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfCancelEnabled | 允许自助取消 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfServiceEndsAt | 自助截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.retentionDays | 明细保留天数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.fields.name.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.fields.name.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.fields.phone.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.fields.phone.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.fields.email.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.fields.email.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.fields.companionCount.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.fields.companionCount.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.approvalMode | 审核方式 | ApprovalModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.phoneVerificationEnabled | 验证手机号 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.selfEditEnabled | 允许自助修改 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.selfCancelEnabled | 允许自助取消 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.selfServiceEndsAt | 自助截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.retentionDays | 明细保留天数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyUpdate.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| EventDetail | GET /api/v1/event/event/detail | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅授权活动元数据，无报名明细或能力凭据。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PolicyGet | GET /api/v1/event/policy/detail | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；fields和开关依赖完整返回给负责人，不向普通检票员返回配置秘密。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PolicyUpdate | POST /api/v1/event/policy/update | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅DRAFT/READY，policy.version乐观锁；required依赖enabled，短信启用需手机号required；自助截止不晚于报名截止/现场开始。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O03 报名组列表

路由`/manage/events/:eventId/parties`；角色：ORGANIZER；布局：management_list；FL：FL-O-002。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：按状态/进度/来源/时间筛选，申请数、批准数、已入场分列，不把整组等同到场。

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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| PartyList | POST /api/v1/enrollment/party/list | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权，只返回/检索获准字段；READER只投影获准字段；createdAt,partyId稳定降序，不返回票码/私密能力。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyDetail | GET /api/v1/enrollment/party/detail | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权且本组属该活动；跨活动或未获字段不返回；字段按权限省略，不先发全量再隐藏。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O04 报名组与凭证

路由`/manage/events/:eventId/parties/:partyId`；角色：ORGANIZER；布局：detail_cards；FL：FL-O-012, FL-O-013。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：资料、名额、票证动作和获权审计分区；撤票、取消、补发分别解释；补发先核对，邮件受理不等于送达。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| PartyDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyDetail.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyChange.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyChange.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyChange.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyChange.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| PartyChange.companionCount | 随行人数 | integer | integer_stepper | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 999999} |
| PartyChange.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| PartyCancel.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyCancel.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyCancel.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyCancel.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| PartyCancel.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| CredentialReissue.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CredentialReissue.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CredentialReissue.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CredentialReissue.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| CredentialReissue.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| CredentialReissue.rotateAccess | 同时更换私密入口 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| CredentialRevoke.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CredentialRevoke.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CredentialRevoke.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CredentialRevoke.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| CredentialRevoke.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| EmailResend.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EmailResend.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EmailResend.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EmailResend.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| AuditList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| AuditList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| AuditList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| AuditList.action | 操作类型 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| AuditList.from | 起始时间 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| AuditList.to | 截止时间 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
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
| PartyChange.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyChange.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyChange.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyChange.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyChange.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyChange.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyCancel.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyCancel.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyCancel.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| PartyCancel.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PartyCancel.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| CredentialReissue.party.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| CredentialReissue.party.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| CredentialReissue.party.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| CredentialReissue.party.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.party.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.credential.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.credential.status | 状态 | CredentialStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.credential.credentialVersion | credentialVersion | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.credential.ticketToken | ticketToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.credential.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.access.queryToken | queryToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.access.changeToken | changeToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.access.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialReissue.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| CredentialRevoke.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| CredentialRevoke.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| CredentialRevoke.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| CredentialRevoke.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CredentialRevoke.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.jobId | jobId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.kind | 类型 | JobKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.attempts | 尝试次数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.resultRef | resultRef | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EmailResend.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| PartyDetail | GET /api/v1/enrollment/party/detail | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权且本组属该活动；跨活动或未获字段不返回；字段按权限省略，不先发全量再隐藏。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyChange | POST /api/v1/enrollment/party/change | STAFF / ORGANIZER | READY且hasAdmissionHistory=false；其余规则由服务端复核；仅READY且无核销，增员差额仍遵审核规则；现场加人必须专用动作，不绕过冻结。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyCancel | POST /api/v1/enrollment/party/cancel | STAFF / ORGANIZER | READY且hasAdmissionHistory=false；明确取消并释放占额；仅READY且未核销，释放占额和撤票同事务，保留决定历史。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| CredentialReissue | POST /api/v1/enrollment/credential/reissue | STAFF / ORGANIZER | 负责人核对、有效资格且未过期，说明旧票立即失效；有效资格且未到期限，旧票同事务失效，不加名额；rotateAccess才旋转私密能力，普通检票员禁止。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| CredentialRevoke | POST /api/v1/enrollment/credential/revoke | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；撤票不等于取消资格或释放占额；保留核销，不能借撤票绕过现场人数冻结。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| EmailResend | POST /api/v1/delivery/notification/resend | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；须已提供可用邮箱且查询期/资格允许；私密链接密文投递，失败不撤销资格；无自动海外文件投递。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| AuditList | POST /api/v1/platform/audit/list | STAFF / ORGANIZER,CHECKER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；检票员限本人/获权端，日志不含全量联系方式/票码/秘密；occurredAt,auditId稳定降序。 检票员范围固定为本人记录，或经当前设备cookie验证的本端记录；客户端传deviceId不能扩大范围。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O05 首次及增员审核

路由`/manage/events/:eventId/reviews`；角色：ORGANIZER；布局：review_workspace；FL：FL-O-001。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：待办选择后看对应组资料/批准数/待增额，明确通过或拒绝并填理由；列表不能一键批量审批。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| ReviewList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ReviewList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| ReviewList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| ReviewList.kind | 类型 | ReviewKindCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ReviewList.result | 处理结果 | ReviewResultCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ReviewDecide.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ReviewDecide.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ReviewDecide.reviewId | reviewId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ReviewDecide.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| ReviewDecide.decision | 审核决定 | DecisionCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ReviewDecide.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| PartyDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyDetail.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | kind | 类型 | 服务端允许且字段存在 | left |
| 2 | requestedSize | 申请总人数 | 服务端允许且字段存在 | 否 |
| 3 | approvedSize | 批准名额 | 服务端允许且字段存在 | 否 |
| 4 | requestedIncrease | 申请增加名额 | 服务端允许且字段存在 | 否 |
| 5 | result | 处理结果 | 服务端允许且字段存在 | 否 |
| 6 | createdAt | 创建时间 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| ReviewList.items[].reviewId | reviewId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].kind | 类型 | ReviewKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].requestedIncrease | 申请增加名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].result | 处理结果 | ReviewResultValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].reason | 原因 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.items[].createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.reviewId | reviewId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.kind | 类型 | ReviewKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.requestedIncrease | 申请增加名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.result | 处理结果 | ReviewResultValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.reason | 原因 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ReviewDecide.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| ReviewList | POST /api/v1/enrollment/review/list | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；按createdAt,reviewId稳定降序，活动范围严格校验。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ReviewDecide | POST /api/v1/enrollment/review/decide | STAFF / ORGANIZER | 活动READY、当前申请PENDING且有理由；仅READY待审；整组/差额原子占额，不足不改决定/不发码；拒绝原因必填；现场普通审批冻结。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyDetail | GET /api/v1/enrollment/party/detail | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权且本组属该活动；跨活动或未获字段不返回；字段按权限省略，不先发全量再隐藏。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O06 平台来源与字段映射

路由`/manage/events/:eventId/sources`；角色：ORGANIZER；布局：settings_form；FL：FL-O-011。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：只选管理员授予的连接/资源；读取列元数据后映射，明确有效资格或待审；拉取仅形成预览。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| SourceList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SourceSave.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SourceSave.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SourceSave.sourceId | sourceId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SourceSave.version | version | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| SourceSave.connectionId | connectionId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SourceSave.resourceId | 来源资源 | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 300} |
| SourceSave.mapping.nameColumn | 姓名列 | string | source_column_select | 可省略 | input；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 | {"minLength": 1, "maxLength": 200} |
| SourceSave.mapping.phoneColumn | 手机号列 | string | source_column_select | 可省略 | input；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 | {"minLength": 1, "maxLength": 200} |
| SourceSave.mapping.emailColumn | 邮箱列 | string | source_column_select | 可省略 | input；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 | {"minLength": 1, "maxLength": 200} |
| SourceSave.mapping.countColumn | 人数列 | string | source_column_select | 可省略 | input；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 | {"minLength": 1, "maxLength": 200} |
| SourceSave.mapping.countMode | 人数含义 | CountModeCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| SourceSave.eligibility | 来源资格 | SourceEligibilityCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| SourcePull.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SourcePull.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SourcePull.sourceId | sourceId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SourcePull.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| ResourceList.connectionId | connectionId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ResourceList.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ResourceList.cursor | cursor | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |
| ResourceFields.connectionId | connectionId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ResourceFields.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ResourceFields.resourceId | 来源资源 | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 300} |
| JobDetail.jobId | jobId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| JobDetail.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PolicyGet.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | provider | 提供方 | 服务端允许且字段存在 | left |
| 2 | resourceId | 来源资源 | 服务端允许且字段存在 | 否 |
| 3 | eligibility | 来源资格 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| SourceList.items[].sourceId | sourceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.items[].connectionId | connectionId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.items[].resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.items[].provider | 提供方 | ProviderValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.items[].mapping.nameColumn | 姓名列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| SourceList.items[].mapping.phoneColumn | 手机号列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| SourceList.items[].mapping.emailColumn | 邮箱列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| SourceList.items[].mapping.countColumn | 人数列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| SourceList.items[].mapping.countMode | 人数含义 | CountModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.items[].eligibility | 来源资格 | SourceEligibilityValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.availableConnections[].connectionId | connectionId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.availableConnections[].displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.availableConnections[].provider | 提供方 | ProviderValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.availableConnections[].accountKind | 账号类型 | AccountKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceList.availableConnections[].status | 状态 | ConnectionStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceSave.sourceId | sourceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceSave.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceSave.connectionId | connectionId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceSave.resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceSave.provider | 提供方 | ProviderValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceSave.mapping.nameColumn | 姓名列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| SourceSave.mapping.phoneColumn | 手机号列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| SourceSave.mapping.emailColumn | 邮箱列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| SourceSave.mapping.countColumn | 人数列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| SourceSave.mapping.countMode | 人数含义 | CountModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceSave.eligibility | 来源资格 | SourceEligibilityValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourceSave.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.jobId | jobId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.kind | 类型 | JobKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.attempts | 尝试次数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.resultRef | resultRef | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SourcePull.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.items[].resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.items[].title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.items[].kind | 类型 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.items[].readable | readable | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.nextCursor | nextCursor | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceFields.resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceFields.availableColumns[].code | 代码或验证码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceFields.availableColumns[].label | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.jobId | jobId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.kind | 类型 | JobKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.attempts | 尝试次数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.resultRef | resultRef | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.name.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.name.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.phone.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.phone.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.email.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.email.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.companionCount.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.companionCount.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.approvalMode | 审核方式 | ApprovalModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.phoneVerificationEnabled | 验证手机号 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfEditEnabled | 允许自助修改 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfCancelEnabled | 允许自助取消 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfServiceEndsAt | 自助截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.retentionDays | 明细保留天数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| SourceList | GET /api/v1/intake/source/list | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只列活动配置，不返回外部凭据。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SourceSave | POST /api/v1/intake/source/save | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；更新须sourceId+version；负责人明确来源资格，保存不等于同步成功；不能用外部字段自行授资格。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SourcePull | POST /api/v1/intake/source/pull | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅暂存预览不直接占额；腾讯结果表生成等副作用须配置获准并记录资源ID防重，真实权限不支持明确失败。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ResourceList | POST /api/v1/integration/resource/list | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人必须给eventId并同时有该活动及连接allowedEventIds授权，仅资源元数据，不含名单内容/秘密；实际文档类型和权限另验。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ResourceFields | POST /api/v1/integration/resource/fields | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅选中获准资源的列标识和标签，不读答卷值；负责人必需eventId且同时获该活动/连接范围。最多200列；不识别/无权限或提供方不可用明确失败，不猜造列名。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| JobDetail | GET /api/v1/platform/job/detail | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；敏感错误脱敏；202受理与真实完成区分。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PolicyGet | GET /api/v1/event/policy/detail | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；fields和开关依赖完整返回给负责人，不向普通检票员返回配置秘密。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O07 名单导入工作区

路由`/manage/events/:eventId/imports`；角色：ORGANIZER；布局：import_steps；FL：FL-O-010。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：单路由上传/映射/核对/提交四步；只前三步不占额，选择有效行整批确认；失败不部分成功。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| BatchUpload.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchUpload.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchUpload.file | 名单文件 | string | file_upload | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "binary"} |
| BatchUpload.sourceLabel | 名单来源说明 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| BatchDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchDetail.batchId | batchId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchPreview.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchPreview.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchPreview.batchId | batchId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchPreview.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| BatchPreview.mapping.nameColumn | 姓名列 | string | source_column_select | 可省略 | input；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 | {"minLength": 1, "maxLength": 200} |
| BatchPreview.mapping.phoneColumn | 手机号列 | string | source_column_select | 可省略 | input；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 | {"minLength": 1, "maxLength": 200} |
| BatchPreview.mapping.emailColumn | 邮箱列 | string | source_column_select | 可省略 | input；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 | {"minLength": 1, "maxLength": 200} |
| BatchPreview.mapping.countColumn | 人数列 | string | source_column_select | 可省略 | input；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 | {"minLength": 1, "maxLength": 200} |
| BatchPreview.mapping.countMode | 人数含义 | CountModeCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| BatchPreview.eligibility | 来源资格 | SourceEligibilityCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| BatchRows.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchRows.batchId | batchId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchRows.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| BatchRows.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| BatchRows.status | 状态 | RowStatusCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| DuplicateResolve.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DuplicateResolve.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DuplicateResolve.batchId | batchId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DuplicateResolve.rowId | rowId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DuplicateResolve.previewVersion | 预览版本 | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| DuplicateResolve.resolution | 重复处理方式 | DuplicateResolutionCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| DuplicateResolve.existingPartyId | existingPartyId | string | authorized_select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DuplicateResolve.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| BatchCommit.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchCommit.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchCommit.batchId | batchId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| BatchCommit.previewVersion | 预览版本 | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| BatchCommit.selectedRowIds | 选中行 | array<string> | valid_row_selection | 是 | input；仅选VALID及人工核对可关联行，至少1行 | {"minItems": 1, "maxItems": 10000, "uniqueItems": true} |
| JobDetail.jobId | jobId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| JobDetail.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PolicyGet.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
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

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | lineNumber | 原始行号 | 服务端允许且字段存在 | left |
| 2 | fields.name | 姓名 | 服务端允许且字段存在 | 否 |
| 3 | fields.phone | 手机号 | 服务端允许且字段存在 | 否 |
| 4 | fields.email | 邮箱 | 服务端允许且字段存在 | 否 |
| 5 | requestedSize | 申请总人数 | 服务端允许且字段存在 | 否 |
| 6 | status | 状态 | 服务端允许且字段存在 | 否 |
| 7 | errors | errors | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| BatchUpload.batchId | batchId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.sourceId | sourceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.fileDigest | fileDigest | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.status | 状态 | BatchStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.eligibility | 来源资格 | SourceEligibilityValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.mapping.nameColumn | 姓名列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchUpload.mapping.phoneColumn | 手机号列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchUpload.mapping.emailColumn | 邮箱列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchUpload.mapping.countColumn | 人数列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchUpload.mapping.countMode | 人数含义 | CountModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.previewVersion | 预览版本 | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.validRows | 有效行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.errorRows | 错误行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.suspectRows | 待核对行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.policyVersion | policyVersion | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.availableColumns[].code | 代码或验证码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchUpload.availableColumns[].label | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.batchId | batchId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.sourceId | sourceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.fileDigest | fileDigest | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.status | 状态 | BatchStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.eligibility | 来源资格 | SourceEligibilityValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.mapping.nameColumn | 姓名列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchDetail.mapping.phoneColumn | 手机号列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchDetail.mapping.emailColumn | 邮箱列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchDetail.mapping.countColumn | 人数列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchDetail.mapping.countMode | 人数含义 | CountModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.previewVersion | 预览版本 | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.validRows | 有效行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.errorRows | 错误行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.suspectRows | 待核对行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.policyVersion | policyVersion | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.availableColumns[].code | 代码或验证码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchDetail.availableColumns[].label | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.batchId | batchId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.sourceId | sourceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.fileDigest | fileDigest | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.status | 状态 | BatchStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.eligibility | 来源资格 | SourceEligibilityValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.mapping.nameColumn | 姓名列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchPreview.mapping.phoneColumn | 手机号列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchPreview.mapping.emailColumn | 邮箱列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchPreview.mapping.countColumn | 人数列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| BatchPreview.mapping.countMode | 人数含义 | CountModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.previewVersion | 预览版本 | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.validRows | 有效行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.errorRows | 错误行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.suspectRows | 待核对行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.policyVersion | policyVersion | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.availableColumns[].code | 代码或验证码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchPreview.availableColumns[].label | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].rowId | rowId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].sourceRecordId | sourceRecordId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].lineNumber | 原始行号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| BatchRows.items[].fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| BatchRows.items[].fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| BatchRows.items[].fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| BatchRows.items[].requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].status | 状态 | RowStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].errors[].field | field | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].errors[].code | 代码或验证码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].errors[].message | message | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.items[].linkedPartyId | 关联报名组 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchRows.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.batchId | batchId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.sourceId | sourceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.fileDigest | fileDigest | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.status | 状态 | BatchStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.eligibility | 来源资格 | SourceEligibilityValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.mapping.nameColumn | 姓名列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| DuplicateResolve.mapping.phoneColumn | 手机号列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| DuplicateResolve.mapping.emailColumn | 邮箱列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| DuplicateResolve.mapping.countColumn | 人数列 | string | readonly_value | authorized_readonly；当前活动对应基础字段已启用；未映射必填项会预览行错；列必须来自当前资源元数据 |
| DuplicateResolve.mapping.countMode | 人数含义 | CountModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.previewVersion | 预览版本 | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.validRows | 有效行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.errorRows | 错误行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.suspectRows | 待核对行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.policyVersion | policyVersion | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.availableColumns[].code | 代码或验证码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DuplicateResolve.availableColumns[].label | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchCommit.batchId | batchId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchCommit.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchCommit.createdCount | createdCount | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchCommit.linkedCount | linkedCount | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchCommit.allocatedPeople | 有效名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| BatchCommit.replayed | replayed | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.jobId | jobId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.kind | 类型 | JobKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.attempts | 尝试次数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.resultRef | resultRef | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.name.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.name.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.phone.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.phone.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.email.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.email.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.companionCount.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.fields.companionCount.required | required | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.approvalMode | 审核方式 | ApprovalModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.phoneVerificationEnabled | 验证手机号 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfEditEnabled | 允许自助修改 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfCancelEnabled | 允许自助取消 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.selfServiceEndsAt | 自助截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.retentionDays | 明细保留天数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| PolicyGet.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| BatchUpload | POST /api/v1/intake/batch/upload | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；multipart CSV UTF-8/BOM或XLSX，10MiB、解压后50MiB、最多10000行；不执行宏/公式，受管暂存24h且不超过活动明细期限。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| BatchDetail | GET /api/v1/intake/batch/detail | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；校验活动/负责人和暂存有效期，过期410且不可提交。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| BatchPreview | POST /api/v1/intake/batch/preview | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；不占额，生成policy/source指纹和previewVersion；错误/未核对重复行不能提交；TOTAL显式减本人，缺随行0。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| BatchRows | POST /api/v1/intake/batch/rows | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；稳定lineNumber,rowId；字段只受管预览，不以未知列扩展问卷。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DuplicateResolve | POST /api/v1/intake/row/resolve | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；LINK_EXISTING须明确既有组并保留来源，不复制核销或加占额；重建预览版本，不按电话静默合并。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| BatchCommit | POST /api/v1/intake/batch/commit | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅READY且选中1到10000行；版本/指纹/来源/字段/容量整体校验，事件锁→排序组锁，全部提交或回滚，无部分有效票。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| JobDetail | GET /api/v1/platform/job/detail | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；敏感错误脱敏；202受理与真实完成区分。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PolicyGet | GET /api/v1/event/policy/detail | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；fields和开关依赖完整返回给负责人，不向普通检票员返回配置秘密。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyList | POST /api/v1/enrollment/party/list | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权，只返回/检索获准字段；READER只投影获准字段；createdAt,partyId稳定降序，不返回票码/私密能力。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyDetail | GET /api/v1/enrollment/party/detail | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权且本组属该活动；跨活动或未获字段不返回；字段按权限省略，不先发全量再隐藏。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O08 检票控制与终端

路由`/manage/events/:eventId/gate`；角色：ORGANIZER；布局：control_workspace；FL：FL-O-003, FL-O-004, FL-O-014, FL-O-020。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：活动窗口、在线就绪与设备授权分开；开始/暂停/恢复/结束需明确确认，断一端不自动暂停全场。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| EventDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EventTransition.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EventTransition.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| EventTransition.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| EventTransition.action | 操作类型 | EventActionCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| EventTransition.reason | 原因 | string | textarea | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| DeviceList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeviceList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| DeviceList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| DeviceRegister.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeviceRegister.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeviceRegister.name | 姓名 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| DeviceRegister.mode | 终端模式 | DeviceModeCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| DeviceRevoke.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeviceRevoke.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeviceRevoke.deviceId | deviceId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeviceRevoke.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| DeviceRevoke.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| GrantList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| GrantList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| GrantSave.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.accountId | accountId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.role | 活动角色 | string | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| GrantSave.allowedFields | 可查看字段 | array<PartyFieldCode> | multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"maxItems": 1000} |
| GrantSave.allowedActions | 获准动作 | array<string> | multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"maxItems": 1000} |
| GrantSave.recipient | 接收方 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 200} |
| GrantSave.purpose | 用途 | string | textarea | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| GrantSave.retentionNote | 副本处置约定 | string | textarea | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |
| GrantSave.expiresAt | 有效期限 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| GrantSave.grantId | grantId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.version | version | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| GrantRevoke.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantRevoke.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantRevoke.grantId | grantId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantRevoke.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | name | 姓名 | 服务端允许且字段存在 | left |
| 2 | mode | 终端模式 | 服务端允许且字段存在 | 否 |
| 3 | status | 状态 | 服务端允许且字段存在 | 否 |
| 4 | lastSeenAt | 最近在线 | 服务端允许且字段存在 | 否 |
| 5 | expiresAt | 有效期限 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| EventDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventTransition.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.items[].deviceId | deviceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.items[].name | 姓名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.items[].mode | 终端模式 | DeviceModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.items[].status | 状态 | DeviceStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.items[].expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.items[].lastSeenAt | 最近在线 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.device.deviceId | deviceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.device.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.device.name | 姓名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.device.mode | 终端模式 | DeviceModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.device.status | 状态 | DeviceStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.device.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.device.lastSeenAt | 最近在线 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.device.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.pairingCode | pairingCode | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceRegister.pairingExpiresAt | pairingExpiresAt | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].grantId | grantId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].role | 活动角色 | RoleValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].allowedFields | 可查看字段 | array<PartyFieldValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].allowedActions | 获准动作 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].recipient | 接收方 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].purpose | 用途 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].retentionNote | 副本处置约定 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].accountDisplayName | 工作人员名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.grantId | grantId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.role | 活动角色 | RoleValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.allowedFields | 可查看字段 | array<PartyFieldValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.allowedActions | 获准动作 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.recipient | 接收方 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.purpose | 用途 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.retentionNote | 副本处置约定 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.accountDisplayName | 工作人员名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| EventDetail | GET /api/v1/event/event/detail | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅授权活动元数据，无报名明细或能力凭据。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| EventTransition | POST /api/v1/event/event/transition | STAFF / ORGANIZER | 状态对应PUBLISH/OPEN/PAUSE/RESUME/END，操作前确认影响；DRAFT发布READY；READY开启OPEN；OPEN暂停PAUSED/PAUSED恢复OPEN；OPEN或PAUSED结束ENDED。OPEN原子冻结公众变更；结束不自动重开，归档由到期任务执行。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DeviceList | POST /api/v1/gate/device/list | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人查看模式/状态/最后在线，不能读取设备秘密或他活动信息。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DeviceRegister | POST /api/v1/gate/device/register | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；生成一次领取码5分钟，绑定活动和模式；负责人在私人端设置，不把管理员会话留在公开终端。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DeviceRevoke | POST /api/v1/gate/device/revoke | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；新请求即验证设备状态；在途操作以授权检查线性化点为准，保留核销事实。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantList | POST /api/v1/identity/grant/list | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限自己的活动；系统管理员可管理授权但不能借接口读报名信息。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantSave | POST /api/v1/identity/grant/save | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人只授CHECKER/READER且不超自身范围；更新须grantId+version；READER必须recipient/purpose/retentionNote，真实启用条件未完成不激活；查看与EXPORT分开。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantRevoke | POST /api/v1/identity/grant/revoke | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限本活动，不能撤掉最后一个负责人；在途写入授权线性化点见正文。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航；在负责人私人端展示5分钟一次性终端领取二维码。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O09 工作人员检票

路由`/manage/events/:eventId/scan`；角色：ORGANIZER；布局：scanner；FL：FL-O-005。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：手机/平板镜头或键盘扫码输入；扫码后显示剩余，人数默认1但须确认；提交前锁本组，未知不扫下一组。

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

## O10 现场补录与增员

路由`/manage/events/:eventId/onsite`；角色：ORGANIZER；布局：onsite_form；FL：FL-O-006。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：负责人专用；已有待审组可关联，先查找防重复；占额成功后仍需独立扫码核销。短信启用时使用受限匿名来宾验证子流程。

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
| GateWalkIn.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateWalkIn.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateWalkIn.fields.name | 姓名 | string | text_input | 按活动策略 | input；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 | {"minLength": 1, "maxLength": 100} |
| GateWalkIn.fields.phone | 手机号 | string | tel_input | 按活动策略 | input；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 | {"minLength": 1, "maxLength": 40} |
| GateWalkIn.fields.email | 邮箱 | string | email_input | 按活动策略 | input；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 | {"format": "email", "minLength": 1, "maxLength": 254} |
| GateWalkIn.fields.companionCount | 随行人数 | integer | integer_stepper | 按活动策略 | input；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 | {"format": "int32", "minimum": 0, "maximum": 999999} |
| GateWalkIn.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| GateWalkIn.existingPendingPartyId | existingPendingPartyId | string | authorized_select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateWalkIn.partyVersion | partyVersion | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| GateWalkIn.phoneVerificationProof | phoneVerificationProof | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |
| GateIncrease.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateIncrease.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateIncrease.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateIncrease.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| GateIncrease.additionalCount | 增加名额 | integer | integer_stepper | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647} |
| GateIncrease.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| SmsStart.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsStart.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsStart.phone | 手机号 | string | tel_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 40} |
| SmsVerify.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsVerify.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsVerify.challengeId | challengeId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| SmsVerify.code | 验证码 | string | otp_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 12} |

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
| GateWalkIn.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| GateWalkIn.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| GateWalkIn.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| GateWalkIn.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| GateWalkIn.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateWalkIn.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.partyReference | 报名参考号 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.fields.name | 姓名 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| GateIncrease.fields.phone | 手机号 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| GateIncrease.fields.email | 邮箱 | string | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| GateIncrease.fields.companionCount | 随行人数 | integer | readonly_value | authorized_readonly；活动字段enabled且调用人获字段权限；required取活动策略，不因DTO可省略而取消校验 |
| GateIncrease.requestedSize | 申请总人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.approvedSize | 批准名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.pendingIncrease | 待审新增名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.netAdmitted | 累计净入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.remainingCount | 剩余可入场 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.status | 状态 | PartyStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.progress | 入场进度 | AdmissionProgressValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.sourceType | 来源 | SourceTypeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.updatedAt | 更新时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateIncrease.hasAdmissionHistory | hasAdmissionHistory | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.clientContextId | clientContextId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsStart.challengeId | challengeId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsStart.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsStart.retryAfterSeconds | retryAfterSeconds | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsVerify.verificationProof | verificationProof | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SmsVerify.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| PartyList | POST /api/v1/enrollment/party/list | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权，只返回/检索获准字段；READER只投影获准字段；createdAt,partyId稳定降序，不返回票码/私密能力。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyDetail | GET /api/v1/enrollment/party/detail | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权且本组属该活动；跨活动或未获字段不返回；字段按权限省略，不先发全量再隐藏。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GateWalkIn | POST /api/v1/gate/party/walk-in | STAFF / ORGANIZER | 仅负责人、CHECKIN_OPEN、字段/验证完整且有原因；仅CHECKIN_OPEN，整组占额；关联待审组须partyVersion并保留来源/原申请，不能新建重复组；不自动核销，短信依活动规则。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GateIncrease | POST /api/v1/gate/party/increase | STAFF / ORGANIZER | 仅负责人、CHECKIN_OPEN、有效组、增加人数>=1且有原因；仅CHECKIN_OPEN有效组，原子占差额且审计，不自动核销，不借普通审批改现场人数。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SessionBootstrap | GET /api/v1/identity/session/bootstrap | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；同源GET，设置匿名客户端HttpOnly cookie及CSRF，未建立业务权限。 复用仍有效匿名cookie；返回clientContextId，标记变化时旧待确认报名不得自动重发。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SmsStart | POST /api/v1/public/verification/send | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；活动启用才可用；挑战5分钟/验证5次/发送间隔60秒；IP+电话+客户端限流；受理不等于已送达。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SmsVerify | POST /api/v1/public/verification/verify | ANON / PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；证明绑定活动/电话/客户端，10分钟一次性；不验证全部随行身份。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O11 原核销及误检更正

路由`/manage/events/:eventId/records`；角色：ORGANIZER；布局：management_list；FL：FL-O-007。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：显示原记录及已更正人数，填写本次更正和理由；不删除事实、不作为离场或再次入场。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| GateRecords.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateRecords.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| GateRecords.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| GateRecords.deviceId | deviceId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateRecords.partyId | partyId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateCorrect.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateCorrect.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateCorrect.recordId | recordId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GateCorrect.partyVersion | partyVersion | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| GateCorrect.correctedCount | 更正人数 | integer | integer_stepper | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647} |
| GateCorrect.reason | 原因 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| PartyDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PartyDetail.partyId | partyId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
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
| GateCorrect.correctionId | correctionId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCorrect.originalRecordId | originalRecordId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCorrect.partyId | partyId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCorrect.correctedCount | 更正人数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCorrect.reason | 原因 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCorrect.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GateCorrect.operationId | operationId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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
| GateCorrect | POST /api/v1/gate/record/correct | STAFF / ORGANIZER | 仅负责人、CHECKIN_OPEN/CHECKIN_PAUSED、未更正余额足够且有理由；仅OPEN/PAUSED，累计更正不超原人数、组净数不负；追加记录不删除；ENDED后不默认开放特权更正。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyDetail | GET /api/v1/enrollment/party/detail | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权且本组属该活动；跨活动或未获字段不返回；字段按权限省略，不先发全量再隐藏。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GateOperation | POST /api/v1/gate/operation/result | GATE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限活动/检票员限获准操作；kiosk必须原组上下文，查无记录仍同ID重试，过期未知转负责人。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O12 活动统计

路由`/manage/events/:eventId/statistics`；角色：ORGANIZER；布局：statistics；FL：FL-O-015。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：总组/待审/批准额/净入场等指标分开，显示asOf；不显示当前在场人数。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| Statistics.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
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

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| Statistics | GET /api/v1/report/attendance/summary | STAFF / ORGANIZER,READER | 本活动VIEW_STATISTICS获权；同一数据库一致读快照返回asOf，含原核销/更正/净数；零分母rate=null，无当前在场人数。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| PartyList | POST /api/v1/enrollment/party/list | STAFF / ORGANIZER,READER | 本活动VIEW_DETAIL获权，只返回/检索获准字段；READER只投影获准字段；createdAt,partyId稳定降序，不返回票码/私密能力。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O13 受控导出与交付登记

路由`/manage/events/:eventId/exports`；角色：ORGANIZER；布局：delivery_workspace；FL：FL-O-016, FL-O-018。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：无EXPORT权隐藏生成及下载；选择获权列、目的，结果到期不可下载；登记交付不自动发送文件。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| MyGrantList.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| MyGrantList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| MyGrantList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| ExportCreate.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ExportCreate.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ExportCreate.format | 文件格式 | ExportFormatCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ExportCreate.columns | 导出字段 | array<string> | multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minItems": 1, "maxItems": 40, "uniqueItems": true} |
| ExportCreate.purpose | 用途 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| ExportCreate.filter.status | 状态 | PartyStatusCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ExportCreate.filter.progress | 入场进度 | AdmissionProgressCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ExportCreate.filter.sourceType | 来源 | SourceTypeCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ExportCreate.filter.createdFrom | 创建开始 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| ExportCreate.filter.createdTo | 创建截止 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| ExportDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ExportDetail.exportId | exportId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ExportDownload.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ExportDownload.exportId | exportId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeliveryList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeliveryList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| DeliveryList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| DeliveryRecord.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeliveryRecord.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeliveryRecord.exportId | exportId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeliveryRecord.recipient | 接收方 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 200} |
| DeliveryRecord.purpose | 用途 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| DeliveryRecord.columns | 导出字段 | array<string> | multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"maxItems": 1000} |
| DeliveryRecord.retentionNote | 副本处置约定 | string | textarea | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | recipient | 接收方 | 服务端允许且字段存在 | left |
| 2 | purpose | 用途 | 服务端允许且字段存在 | 否 |
| 3 | columns | 导出字段 | 服务端允许且字段存在 | 否 |
| 4 | retentionNote | 副本处置约定 | 服务端允许且字段存在 | 否 |
| 5 | createdAt | 创建时间 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
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
| ExportCreate.exportId | exportId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportCreate.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportCreate.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportCreate.format | 文件格式 | ExportFormatValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportCreate.columns | 导出字段 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportCreate.rowCount | 导出行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportCreate.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportCreate.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportDetail.exportId | exportId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportDetail.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportDetail.format | 文件格式 | ExportFormatValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportDetail.columns | 导出字段 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportDetail.rowCount | 导出行数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportDetail.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ExportDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].deliveryId | deliveryId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].exportId | exportId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].recipient | 接收方 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].purpose | 用途 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].columns | 导出字段 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].retentionNote | 副本处置约定 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.deliveryId | deliveryId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.exportId | exportId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.recipient | 接收方 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.purpose | 用途 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.columns | 导出字段 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.retentionNote | 副本处置约定 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeliveryRecord.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| MyGrantList | POST /api/v1/identity/grant/mine | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；accountId固定从会话取，禁止请求他人账号；eventId仅收窄本人范围。只返回有效活动/字段/动作/期限，展示授权不扩权；按expiresAt,grantId稳定升序。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ExportCreate | POST /api/v1/report/export/create | STAFF / ORGANIZER | 本活动EXPORT明确授权，列均获权；需独立EXPORT授权，列白名单且获权；禁止票码/私密能力/秘密；有效期min(24h,明细到期)，生成/下载重验权限。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ExportDetail | GET /api/v1/report/export/detail | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；查询不等于文件已生成，过期或失败明确呈现。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ExportDownload | GET /api/v1/report/export/download | STAFF / ORGANIZER | EXPORT仍有效，文件成功且未到期，下载重验；成功为二进制CSV/XLSX，不是JSON；重验列权限/范围/到期，下载审计，无长期公开链接。 安全读取；仅当前权限范围 | 二进制文件受控下载；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DeliveryList | POST /api/v1/delivery/record/list | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；限活动，按createdAt,deliveryId降序；不是校方自动导出入口。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DeliveryRecord | POST /api/v1/delivery/record/create | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；登记获准接收方/用途/范围及副本责任，不自动发海外邮件或作合规批准；真实启用配置先落实。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O14 留存与清理

路由`/manage/events/:eventId/lifecycle`；角色：ORGANIZER；布局：detail_cards；FL：FL-O-017。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：显示期限、六个范围进度、失败码和备份覆盖；只在到期失败可重试，不提供任意提前删除。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| CleanupDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CleanupRetry.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CleanupRetry.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CleanupRetry.cleanupId | cleanupId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| CleanupRetry.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| EventDetail.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| CleanupDetail.cleanupId | cleanupId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.dueAt | 明细到期 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.processedCount | 已处理条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.failedCount | 失败条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.scopes[].scope | 清理范围 | CleanupScopeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.scopes[].status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.scopes[].processedCount | 已处理条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.scopes[].failedCount | 失败条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupDetail.scopes[].errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.jobId | jobId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.kind | 类型 | JobKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.attempts | 尝试次数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.resultRef | resultRef | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| CleanupRetry.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.venue | 活动地点 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.endsAt | 结束时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.timeZone | 活动时区 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.registrationStartsAt | 报名开始 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.registrationEndsAt | 报名截止 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.capacity | 人数容量 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.allocatedCount | 已分配名额 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.status | 状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| EventDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| CleanupDetail | GET /api/v1/lifecycle/cleanup/detail | STAFF / ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；展示受管清理结果与失败范围，不以逻辑删除假装实际清理完成。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| CleanupRetry | POST /api/v1/lifecycle/cleanup/retry | STAFF / ORGANIZER | 已到期且清理失败，不提供提前删除；仅到期且失败，不能提前任意批量删除；根据到期依据实际处置/失效能力并登记恢复防复活结果。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| EventDetail | GET /api/v1/event/event/detail | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅授权活动元数据，无报名明细或能力凭据。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O15 任务与通知

路由`/manage/events/:eventId/jobs`；角色：ORGANIZER；布局：management_list；FL：FL-O-021。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：区分受理/执行/成功/失败；仅该活动，不能借重试增加资格。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| JobList.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| JobList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| JobList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| JobList.kind | 类型 | JobKindCode | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| JobDetail.jobId | jobId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| JobDetail.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| JobRetry.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| JobRetry.jobId | jobId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| JobRetry.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| JobRetry.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | kind | 类型 | 服务端允许且字段存在 | left |
| 2 | status | 状态 | 服务端允许且字段存在 | 否 |
| 3 | attempts | 尝试次数 | 服务端允许且字段存在 | 否 |
| 4 | createdAt | 创建时间 | 服务端允许且字段存在 | 否 |
| 5 | completedAt | 完成时间 | 服务端允许且字段存在 | 否 |
| 6 | errorCode | 错误码 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| JobList.items[].jobId | jobId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].kind | 类型 | JobKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].attempts | 尝试次数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].resultRef | resultRef | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.jobId | jobId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.kind | 类型 | JobKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.attempts | 尝试次数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.resultRef | resultRef | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobDetail.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.jobId | jobId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.kind | 类型 | JobKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.status | 状态 | JobStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.attempts | 尝试次数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.createdAt | 创建时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.completedAt | 完成时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.resultRef | resultRef | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.errorCode | 错误码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| JobRetry.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| JobList | POST /api/v1/platform/job/list | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；管理员仅脱敏元数据，负责人限活动；无任意代码/任务执行权限。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| JobDetail | GET /api/v1/platform/job/detail | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；敏感错误脱敏；202受理与真实完成区分。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| JobRetry | POST /api/v1/platform/job/retry | STAFF / ADMIN,ORGANIZER | FAILED且仍获原范围/动作权限；原范围/现场冻结/副作用幂等继续校验；管理员不能借重试审批或读取名单。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O16 活动授权

路由`/manage/events/:eventId/access`；角色：ORGANIZER；布局：management_list；FL：FL-O-019。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：仅授CHECKER/READER且不超自身范围，ORGANIZER/EXPORT需管理员明确授权。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| GrantList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| GrantList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| GrantSave.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.accountId | accountId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.role | 活动角色 | string | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| GrantSave.allowedFields | 可查看字段 | array<PartyFieldCode> | multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"maxItems": 1000} |
| GrantSave.allowedActions | 获准动作 | array<string> | multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"maxItems": 1000} |
| GrantSave.recipient | 接收方 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 200} |
| GrantSave.purpose | 用途 | string | textarea | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 500} |
| GrantSave.retentionNote | 副本处置约定 | string | textarea | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |
| GrantSave.expiresAt | 有效期限 | string | datetime_in_event_zone | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| GrantSave.grantId | grantId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.version | version | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| GrantRevoke.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantRevoke.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantRevoke.grantId | grantId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantRevoke.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | accountDisplayName | 工作人员名称 | 服务端允许且字段存在 | left |
| 2 | username | 登录名 | 服务端允许且字段存在 | 否 |
| 3 | role | 活动角色 | 服务端允许且字段存在 | 否 |
| 4 | allowedFields | 可查看字段 | 服务端允许且字段存在 | 否 |
| 5 | allowedActions | 获准动作 | 服务端允许且字段存在 | 否 |
| 6 | expiresAt | 有效期限 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| GrantList.items[].grantId | grantId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].role | 活动角色 | RoleValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].allowedFields | 可查看字段 | array<PartyFieldValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].allowedActions | 获准动作 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].recipient | 接收方 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].purpose | 用途 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].retentionNote | 副本处置约定 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].accountDisplayName | 工作人员名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.items[].username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.grantId | grantId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.role | 活动角色 | RoleValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.allowedFields | 可查看字段 | array<PartyFieldValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.allowedActions | 获准动作 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.recipient | 接收方 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.purpose | 用途 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.retentionNote | 副本处置约定 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.accountDisplayName | 工作人员名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantSave.username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| GrantList | POST /api/v1/identity/grant/list | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限自己的活动；系统管理员可管理授权但不能借接口读报名信息。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantSave | POST /api/v1/identity/grant/save | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人只授CHECKER/READER且不超自身范围；更新须grantId+version；READER必须recipient/purpose/retentionNote，真实启用条件未完成不激活；查看与EXPORT分开。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantRevoke | POST /api/v1/identity/grant/revoke | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限本活动，不能撤掉最后一个负责人；在途写入授权线性化点见正文。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## O17 审计与原操作结果

路由`/manage/events/:eventId/audit`；角色：ORGANIZER；布局：management_list；FL：FL-O-022, FL-O-023。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：无秘密/全量联系方式；原操作查询用于核对，不重新执行放行。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| AuditList.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| AuditList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| AuditList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| AuditList.action | 操作类型 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| AuditList.from | 起始时间 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| AuditList.to | 截止时间 | string | datetime_in_event_zone | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "date-time"} |
| OperationResult.eventId | eventId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| OperationResult.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| OperationResult.action | 操作类型 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | action | 操作类型 | 服务端允许且字段存在 | left |
| 2 | reason | 原因 | 服务端允许且字段存在 | 否 |
| 3 | occurredAt | occurredAt | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| AuditList | POST /api/v1/platform/audit/list | STAFF / ORGANIZER,CHECKER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；检票员限本人/获权端，日志不含全量联系方式/票码/秘密；occurredAt,auditId稳定降序。 检票员范围固定为本人记录，或经当前设备cookie验证的本端记录；客户端传deviceId不能扩大范围。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| OperationResult | POST /api/v1/platform/operation/result | STAFF / ORGANIZER,CHECKER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限活动和原动作权限，检票员限获准操作；action是固定operation名称并校验原动作权限，不从通用查询取票或私密能力，恢复具体原请求仍用同ID。 检票员范围固定为本人记录，或经当前设备cookie验证的本端记录；客户端传deviceId不能扩大范围。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

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
