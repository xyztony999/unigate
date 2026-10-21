# UniGate admin 页面规格

| 属性 | 值 |
|---|---|
| 版本 | v1.0 |
| 日期 | 2026-10-21 |
| 状态 | ADR-044确认七项交互，D4归档；设计规格，运行验证另行执行 |
| App / 模块 | UniGate Web / unigate |
| 输入 | API v0.3 / OpenAPI0.3.0；ADR-042/043/044 |
| 类型 | 业务流程+来源集成+只读报表；页面使用本栈组件语义，不套用SCM引擎 |

全局规则、完整状态机及枚举索引见../../README.md；字段类型由page-contracts.json与OpenAPI精确绑定。表中系统字段不作为普通输入显示，所有按钮仍由服务端鉴权/校验。

## A01 账号与活动授权

路由`/admin/accounts`；角色：ADMIN；布局：management_list；FL：FL-A-001。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：账号表与授权抽屉；角色/活动/字段/动作分开，保留最后管理员和负责人。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| AccountList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| AccountList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| AccountList.query | 搜索 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| AccountCreate.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| AccountCreate.username | 登录名 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 80} |
| AccountCreate.displayName | 显示名称 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| AccountCreate.temporaryPassword | 初始密码 | string | password | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 12, "maxLength": 256} |
| AccountCreate.roles | 账号角色 | array<string> | multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minItems": 1, "maxItems": 4, "uniqueItems": true} |
| AccountUpdate.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| AccountUpdate.accountId | accountId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| AccountUpdate.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| AccountUpdate.displayName | 显示名称 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| AccountUpdate.enabled | 启用 | boolean | switch | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| AccountUpdate.roles | 账号角色 | array<string> | multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minItems": 1, "maxItems": 4, "uniqueItems": true} |
| GrantList.eventId | eventId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| GrantList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| GrantSave.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantSave.eventId | eventId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
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
| GrantRevoke.eventId | eventId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantRevoke.grantId | grantId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| GrantRevoke.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| GrantEventOptions.query | 搜索 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| GrantEventOptions.cursor | cursor | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | username | 登录名 | 服务端允许且字段存在 | left |
| 2 | displayName | 显示名称 | 服务端允许且字段存在 | 否 |
| 3 | roles | 账号角色 | 服务端允许且字段存在 | 否 |
| 4 | enabled | 启用 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| AccountList.items[].accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountList.items[].username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountList.items[].displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountList.items[].enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountList.items[].roles | 账号角色 | array<RoleValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountCreate.accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountCreate.username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountCreate.displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountCreate.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountCreate.roles | 账号角色 | array<RoleValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountCreate.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountUpdate.accountId | accountId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountUpdate.username | 登录名 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountUpdate.displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountUpdate.enabled | 启用 | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountUpdate.roles | 账号角色 | array<RoleValue> | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| AccountUpdate.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
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
| GrantEventOptions.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantEventOptions.items[].title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantEventOptions.items[].startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantEventOptions.nextCursor | nextCursor | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| AccountList | POST /api/v1/identity/account/list | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅账号元数据，不联带活动个人名单；username/accountId稳定排序。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| AccountCreate | POST /api/v1/identity/account/create | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；初始密码至少12字符且首次改密，禁止PARTICIPANT长期账号；username唯一，慢哈希存储。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| AccountUpdate | POST /api/v1/identity/account/update | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；不能停用/降权最后一个管理员；权限版本更新，新请求重新校验。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantList | POST /api/v1/identity/grant/list | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限自己的活动；系统管理员可管理授权但不能借接口读报名信息。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantSave | POST /api/v1/identity/grant/save | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人只授CHECKER/READER且不超自身范围；更新须grantId+version；READER必须recipient/purpose/retentionNote，真实启用条件未完成不激活；查看与EXPORT分开。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantRevoke | POST /api/v1/identity/grant/revoke | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人限本活动，不能撤掉最后一个负责人；在途写入授权线性化点见正文。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantEventOptions | POST /api/v1/identity/grant/event-options | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅管理员授权/连接配置所需活动ID、标题、开始时间，按startsAt,eventId稳定降序及游标每次最多100条；不返回名单、占额或到场数，不授活动业务读取权限。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## A02 平台连接

路由`/admin/connections`；角色：ADMIN；布局：management_list；FL：FL-A-002。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：凭据只输入不回显，掩码展示；授权在新窗口，原页刷新状态，配置保存不是接通。

### 查询、表单与系统绑定

| API字段 | 展示名称 | 类型 | 控件 | 必填 | 可见/来源规则 | 精确边界 |
|---|---|---|---|---|---|---|
| ConnectionList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| ConnectionList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| ConnectionList.provider | 提供方 | string | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ConnectionSave.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ConnectionSave.connectionId | connectionId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ConnectionSave.version | version | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| ConnectionSave.provider | 提供方 | string | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ConnectionSave.accountKind | 账号类型 | AccountKindCode | select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |
| ConnectionSave.displayName | 显示名称 | string | text_input | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| ConnectionSave.allowedEventIds | 可用活动 | array<string> | authorized_multi_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"maxItems": 1000} |
| ConnectionSave.clientId | 应用标识 | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 200} |
| ConnectionSave.clientSecret | 应用密钥 | string | password | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 16, "maxLength": 4096} |
| ConnectionAuthorize.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ConnectionAuthorize.connectionId | connectionId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ConnectionAuthorize.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| ConnectionRevoke.operationId | operationId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ConnectionRevoke.connectionId | connectionId | string | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ConnectionRevoke.version | version | integer | system_binding | 是 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 0, "maximum": 2147483647} |
| ResourceList.connectionId | connectionId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ResourceList.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ResourceList.cursor | cursor | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |
| ResourceFields.connectionId | connectionId | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ResourceFields.eventId | eventId | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "uuid"} |
| ResourceFields.resourceId | 来源资源 | string | authorized_select | 是 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 300} |
| GrantEventOptions.query | 搜索 | string | text_input | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 100} |
| GrantEventOptions.cursor | cursor | string | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"minLength": 1, "maxLength": 1000} |

### 列表列与详情输出

| 顺序 | API data.items字段 | 名称 | 显示 | 固定 |
|---|---|---|---|---|
| 1 | displayName | 显示名称 | 服务端允许且字段存在 | left |
| 2 | provider | 提供方 | 服务端允许且字段存在 | 否 |
| 3 | accountKind | 账号类型 | 服务端允许且字段存在 | 否 |
| 4 | status | 状态 | 服务端允许且字段存在 | 否 |
| 5 | credentialMask | 凭据掩码 | 服务端允许且字段存在 | 否 |
| 6 | expiresAt | 有效期限 | 服务端允许且字段存在 | 否 |

详情分资料/状态与名额/关联结果等语义区块；子结果只从下表操作读取，不臆造审批历史或工作流接口。原始ID/版本/秘密只作隐藏绑定，票仅在获权领票页渲染QR，不直接显示令牌文本。

| API data字段 | 名称 | 类型 | 展示组件 | 可见范围 |
|---|---|---|---|---|
| ConnectionList.items[].connectionId | connectionId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].provider | 提供方 | ProviderValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].accountKind | 账号类型 | AccountKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].status | 状态 | ConnectionStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].allowedEventIds | 可用活动 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].credentialMask | 凭据掩码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.connectionId | connectionId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.provider | 提供方 | ProviderValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.accountKind | 账号类型 | AccountKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.status | 状态 | ConnectionStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.allowedEventIds | 可用活动 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.credentialMask | 凭据掩码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionSave.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionAuthorize.authorizationUrl | authorizationUrl | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionAuthorize.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.connectionId | connectionId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.provider | 提供方 | ProviderValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.accountKind | 账号类型 | AccountKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.status | 状态 | ConnectionStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.allowedEventIds | 可用活动 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.credentialMask | 凭据掩码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionRevoke.version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.items[].resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.items[].title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.items[].kind | 类型 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.items[].readable | readable | boolean | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceList.nextCursor | nextCursor | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceFields.resourceId | 来源资源 | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceFields.availableColumns[].code | 代码或验证码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ResourceFields.availableColumns[].label | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantEventOptions.items[].eventId | eventId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantEventOptions.items[].title | 活动名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantEventOptions.items[].startsAt | 开始时间 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| GrantEventOptions.nextCursor | nextCursor | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| ConnectionList | POST /api/v1/integration/connection/list | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只有脱敏授权状态和资源引用，不联带报名个人明细。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ConnectionSave | POST /api/v1/integration/connection/save | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；更新须ID+version，秘密仅密文写不回显；提供方URL固定allowlist，不接受任意抓取地址。allowedEventIds由管理员明确配置，负责人无权自行扩展。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ConnectionAuthorize | POST /api/v1/integration/connection/authorize | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；state一次性5分钟绑定管理员及redirect allowlist；真实账号/应用能力未验证则明确能力不可用，不虚报接通。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ConnectionRevoke | POST /api/v1/integration/connection/revoke | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；清理受管凭据并阻止新拉取，历史业务资格不因断连接自动取消。 明确确认后提交；请求期间禁重复；同ID恢复 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ResourceList | POST /api/v1/integration/resource/list | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；负责人必须给eventId并同时有该活动及连接allowedEventIds授权，仅资源元数据，不含名单内容/秘密；实际文档类型和权限另验。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ResourceFields | POST /api/v1/integration/resource/fields | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅选中获准资源的列标识和标签，不读答卷值；负责人必需eventId且同时获该活动/连接范围。最多200列；不识别/无权限或提供方不可用明确失败，不猜造列名。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| GrantEventOptions | POST /api/v1/identity/grant/event-options | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；仅管理员授权/连接配置所需活动ID、标题、开始时间，按startsAt,eventId稳定降序及游标每次最多100条；不返回名单、占额或到场数，不授活动业务读取权限。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

本地展示动作（不调用新增业务接口）：返回/脏表单提示；有权详情导航。

### 页面状态与异常

初始检查会话/终端/权限；加载时保留布局并禁提交；空态展示无数据/无授权/无待办及有权下一步，不给越权新增入口。字段422就地说明，409按code区分：OPERATION_IN_PROGRESS沿原ID等待/查询，明确版本/余额拒绝才刷新并确认新动作；提交时禁双击，成功以后端结果为准。超时/断网/5xx无确定结果则unknown，保留原上下文/ID，绝不自动另起业务写入；401/403撤下相关动作、410进入恢复/联系负责人。无HTTP服务或设备测试结果。

返回：返回所属活动/角色入口；脏表单提示；未知核销先核对，不以离开页面清空事实。

触控/无障碍：触控目标至少48 CSS px、文字+图标表达状态、键盘可操作、错误关联字段、数字输入整数；相机拒绝有工作人员辅助或扫码枪路径。

## A03 连接与任务状态

路由`/admin/jobs`；角色：ADMIN；布局：management_list；FL：FL-A-003。

进入：同源HTTPS；按角色及事件授权进入，深链重新鉴权，不凭客户端路由授权限。

页面组织：只看脱敏元数据；无名单、任意任务脚本或业务审批权限。

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
| ConnectionList.page | 页号 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 2147483647, "default": 1} |
| ConnectionList.pageSize | 每页条数 | integer | system_binding | 可省略 | system；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {"format": "int32", "minimum": 1, "maximum": 100, "default": 20} |
| ConnectionList.provider | 提供方 | string | select | 可省略 | input；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null | {} |

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
| ConnectionList.items[].connectionId | connectionId | string | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].provider | 提供方 | ProviderValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].accountKind | 账号类型 | AccountKindValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].displayName | 显示名称 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].status | 状态 | ConnectionStatusValue | status_badge | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].allowedEventIds | 可用活动 | array<string> | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].credentialMask | 凭据掩码 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].expiresAt | 有效期限 | string | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.items[].version | version | integer | opaque_reference | memory_only；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.total | total | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.page | 页号 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |
| ConnectionList.pageSize | 每页条数 | integer | readonly_value | authorized_readonly；接口角色/作用域/状态守卫成立；省略值不转为空字符串或null |

### 动作、权限与返回路径

| 操作 | 完整请求 | 身份/角色上限 | 前置与确认 | 成功与失败 |
|---|---|---|---|---|
| JobList | POST /api/v1/platform/job/list | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；管理员仅脱敏元数据，负责人限活动；无任意代码/任务执行权限。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| JobDetail | GET /api/v1/platform/job/detail | STAFF / ADMIN,ORGANIZER | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；敏感错误脱敏；202受理与真实完成区分。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| JobRetry | POST /api/v1/platform/job/retry | STAFF / ADMIN,ORGANIZER | FAILED且仍获原范围/动作权限；原范围/现场冻结/副作用幂等继续校验；管理员不能借重试审批或读取名单。 明确确认后提交；请求期间禁重复；同ID恢复 | 202仅受理，轮询任务/挑战最终结果；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |
| ConnectionList | POST /api/v1/integration/connection/list | STAFF / ADMIN | 当前角色/活动/字段获权，加载完成；隐藏按钮不代替后端权限；只有脱敏授权状态和资源引用，不联带报名个人明细。 安全读取；仅当前权限范围 | 读取/提交真实服务器结果后更新；401/403停止权限动作；409按code区分：OPERATION_IN_PROGRESS沿原ID查询/等待，不新建；明确版本/容量拒绝才刷新并确认新动作；410恢复入口；网络未知沿原ID核对 |

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
