# UniGate 选择服务池

版本v0.2；2026-10-21；状态D3按ADR-043项目依据完成，D4草案使用。静态code/label由枚举池提供，不另开写字典接口。远程选择均当前会话授权过滤。

| 选择用途 | 操作 | 输入 | 输出及范围 |
|---|---|---|---|
| 活动 | EventOptions | cursor可选 | items为EventView，nextCursor可选，最多100条；无授权活动不出现 |
| 本人授权范围 | MyGrantList | eventId可选、page/pageSize | GrantPage，只含当前账号有效授权；不接受他人accountId，不改变活动/字段权限 |
| 管理员授权目标活动 | GrantEventOptions | query/cursor可选 | GrantEventOptionsView，仅标识/标题/开始时间，不提供名单、占额或核销数 |
| 外部资源 | ResourceList | connectionId必填、eventId负责人必填、cursor可选 | ResourcePage；管理员或获显式活动连接范围负责人；不返回答卷/密钥 |
| 来源可选列 | ResourceFields / BatchDetail | 获准connectionId/resourceId（负责人eventId）或eventId/batchId | availableColumns为code/label元数据，最多200列，不返回样本答卷值；用于字段映射 |
| 账号授权对象 | AccountList | page/pageSize/query | 仅管理员管理授权时使用；负责人不通过它浏览全局账号，先由管理员提供本活动既有账号引用 |
| 来源 | SourceList | eventId | SourceListView，当前活动负责人 |
| 既有组关联 | PartyList | eventId/query/分页 | 负责人核对疑似重复，不将其开放给kiosk |

候选值必须用ID或code提交，标签不能作为唯一键。资源不再可读时提交明确失败，不静默改资源。接口完整路由、字段与分页见api-unigate.md。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v0.1 | 2026-10-21 | 首批选择服务登记，组织接入和实际资源权限未验证。 |
| v0.2 | 2026-10-21 | 按项目检查补齐授权目标活动、来源列元数据选择，D3归档；无新增名单访问权。 |
