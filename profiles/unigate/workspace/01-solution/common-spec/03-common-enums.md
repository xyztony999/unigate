# UniGate 公共枚举

版本v0.1；状态D3合同候选；2026-10-21。定义与openapi-unigate.json同源。请求code、展示code+label；不与历史离线状态混用。

## Role

| code | label |
|---|---|
| ADMIN | 系统管理员 |
| ORGANIZER | 活动负责人 |
| CHECKER | 检票员 |
| READER | 校方只读人员 |
| PARTICIPANT | 参与者 |

## Provider

| code | label |
|---|---|
| FEISHU | 飞书 |
| TENCENT | 腾讯文档 |
| EMAIL | 电子邮件 |
| SMS | 短信 |

## AccountKind

| code | label |
|---|---|
| PERSONAL | 个人 |
| ORGANIZATION | 组织 |

## JobStatus

| code | label |
|---|---|
| PENDING | 等待 |
| RUNNING | 执行中 |
| SUCCEEDED | 成功 |
| FAILED | 失败 |
| EXPIRED | 过期 |

## JobKind

| code | label |
|---|---|
| PULL | 来源拉取 |
| EMAIL | 邮件 |
| EXPORT | 导出 |
| CLEANUP | 清理 |

## RetryMode

| code | label |
|---|---|
| NONE | 不重试 |
| SAME_OPERATION | 同操作重试 |
| NEW_AFTER_CONFIRM | 明确确认后新操作 |

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v0.1 | 2026-10-21 | 在线D3初始候选，未实现运行枚举。 |
