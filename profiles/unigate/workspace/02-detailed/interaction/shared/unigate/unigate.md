# UniGate shared 页面规格

| 属性 | 值 |
|---|---|
| 版本 | v1.0 |
| 日期 | 2026-10-21 |
| 状态 | ADR-044确认七项交互，D4归档；设计规格，运行验证另行执行 |
| App / 模块 | UniGate Web / unigate |
| 输入 | API v0.3 / OpenAPI0.3.0；ADR-042/043/044 |
| 类型 | 业务流程+来源集成+只读报表；页面使用本栈组件语义，不套用SCM引擎 |

全局规则、完整状态机及枚举索引见../../README.md；字段类型由page-contracts.json与OpenAPI精确绑定。表中系统字段不作为普通输入显示，所有按钮仍由服务端鉴权/校验。

## S01 登录

路由`/login`；角色：ADMIN, ORGANIZER, CHECKER, READER；布局：mobile_form；FL：共享会话/终端流程。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：登录后按角色进入活动/管理，首次改密优先；此页业务身份ANON，工作人员角色只表示登录后的去向。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| Login.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| Login.username | 登录名 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 80} |
| Login.password | 密码 | string | password | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 256} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| SessionBootstrap.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.clientContextId | clientContextId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Login.accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Login.username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Login.displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Login.roles | 账号角色 | array<RoleValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Login.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Login.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| Login.mustChangePassword | mustChangePassword | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.roles | 账号角色 | array<RoleValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.mustChangePassword | mustChangePassword | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| SessionBootstrap | GET /api/v1/identity/session/bootstrap | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；同源GET，设置匿名客户端HttpOnly cookie及CSRF，未建立业务权限。 复用仍有效匿名cookie；返回clientContextId，标记变化时旧待确认报名不得自动重发。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| Login | POST /api/v1/identity/session/login | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；合法Origin+CSRF；验证后轮换会话，5次失败退避；初始密码只可先改密，不记录密码。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SessionMe | GET /api/v1/identity/session/me | STAFF / ADMIN,ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只返回账号自身元数据，角色不是全活动数据授权。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## S02 修改本人密码

路由`/account/password`；角色：ADMIN, ORGANIZER, CHECKER, READER；布局：dialog；FL：共享会话/终端流程。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：旧密码/新密码双输入，成功失效旧会话后重新登录，不显示密码。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| PasswordChange.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| PasswordChange.oldPassword | 原密码 | string | password | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 256} |
| PasswordChange.newPassword | 新密码 | string | password | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 12, "maxLength": 256} |
| Logout.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |

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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| PasswordChange | POST /api/v1/identity/account/password-change | STAFF / ADMIN,ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；新密码至少12字符，验证旧密码后轮换会话并使旧会话失效。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| Logout | POST /api/v1/identity/session/logout | STAFF / ADMIN,ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；幂等注销，清理工作人员cookie；不自动撤销独立终端会话，公开kiosk不保存管理员登录。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SessionMe | GET /api/v1/identity/session/me | STAFF / ADMIN,ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只返回账号自身元数据，角色不是全活动数据授权。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## S03 我的有效授权

路由`/account/access`；角色：ORGANIZER, CHECKER, READER；布局：responsive_list；FL：FL-R-002。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：只显示本人活动、字段、动作和期限；校方无导出按钮。 活动标题从EventOptions按eventId关联显示，不把原始ID作为活动名称。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| MyGrantList.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| MyGrantList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| MyGrantList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| EventOptions.cursor | cursor | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |
| Logout.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | eventId | 活动 | 服务端允许且字段存在 | left |
| 2 | role | 活动角色 | 服务端允许且字段存在 | 否 |
| 3 | allowedFields | 可查看字段 | 服务端允许且字段存在 | 否 |
| 4 | allowedActions | 获准动作 | 服务端允许且字段存在 | 否 |
| 5 | expiresAt | 有效期限 | 服务端允许且字段存在 | 否 |

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
| SessionMe.accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.roles | 账号角色 | array<RoleValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionMe.mustChangePassword | mustChangePassword | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| MyGrantList | POST /api/v1/identity/grant/mine | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；accountId固定从会话取，禁止请求他人账号；eventId仅收窄本人范围。只返回有效活动/字段/动作/期限，展示授权不扩权；按expiresAt,grantId稳定升序。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| EventOptions | GET /api/v1/event/event/options | STAFF / ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只列授权活动，startsAt,eventId稳定排序，每次最多100条。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| SessionMe | GET /api/v1/identity/session/me | STAFF / ADMIN,ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只返回账号自身元数据，角色不是全活动数据授权。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| Logout | POST /api/v1/identity/session/logout | STAFF / ADMIN,ORGANIZER,CHECKER,READER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；幂等注销，清理工作人员cookie；不自动撤销独立终端会话，公开kiosk不保存管理员登录。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## S04 领取检票终端

路由`/terminal/activate`；角色：ANONYMOUS；布局：mobile_form；FL：共享会话/终端流程。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：负责人私人端生成领取码，在目标端一次领取；只形成设备会话，不把负责人账号留在公共端。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| DeviceClaim.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeviceClaim.deviceId | deviceId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| DeviceClaim.pairingCode | pairingCode | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |

### 列表列与详情输出

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| SessionBootstrap.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| SessionBootstrap.clientContextId | clientContextId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.device.deviceId | deviceId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.device.eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.device.name | 姓名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.device.mode | 终端模式 | DeviceModeValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.device.status | 状态 | DeviceStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.device.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.device.lastSeenAt | 最近在线 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.device.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.csrfToken | csrfToken | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| DeviceClaim.eventStatus | 活动检票状态 | EventStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| SessionBootstrap | GET /api/v1/identity/session/bootstrap | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；同源GET，设置匿名客户端HttpOnly cookie及CSRF，未建立业务权限。 复用仍有效匿名cookie；返回clientContextId，标记变化时旧待确认报名不得自动重发。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DeviceClaim | POST /api/v1/gate/device/claim | ANON / ANONYMOUS | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；合法Origin/CSRF，领取码一次有效最多5次错误；只发设备cookie，不授管理/工作人员账号权限。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| DeviceDetail | GET /api/v1/gate/device/detail | DEVICE / ORGANIZER,CHECKER,PARTICIPANT | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅受限设备cookie身份，绑定活动/模式；STAFF实际核销还需工作人员账号。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

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
