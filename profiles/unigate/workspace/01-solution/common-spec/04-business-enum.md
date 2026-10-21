# UniGate 业务枚举

版本v0.1；状态D3合同候选；2026-10-21。定义与openapi-unigate.json同源。请求code、展示code+label；不与历史离线状态混用。

## EventStatus

| code | label |
|---|---|
| DRAFT | 草稿 |
| READY | 已发布准备 |
| CHECKIN_OPEN | 检票开放 |
| CHECKIN_PAUSED | 检票暂停 |
| ENDED | 已结束 |
| ARCHIVED | 清理归档 |

## PartyStatus

| code | label |
|---|---|
| PENDING | 待审核 |
| APPROVED | 已批准 |
| REJECTED | 已拒绝 |
| CANCELLED | 已取消 |

## ApprovalMode

| code | label |
|---|---|
| IMMEDIATE | 立即发码 |
| REVIEW | 人工审核 |

## AdmissionProgress

| code | label |
|---|---|
| NONE | 未入场 |
| PARTIAL | 部分入场 |
| COMPLETE | 全部入场 |

## CredentialStatus

| code | label |
|---|---|
| ACTIVE | 有效 |
| REVOKED | 已撤销 |
| EXPIRED | 已过期 |

## SourceType

| code | label |
|---|---|
| BUILTIN | 内置报名 |
| FILE | 文件导入 |
| FEISHU | 飞书 |
| TENCENT | 腾讯文档 |
| WALK_IN | 现场补录 |

## SourceEligibility

| code | label |
|---|---|
| ELIGIBLE | 已获入场资格 |
| REVIEW_REQUIRED | 待审核 |

## ConnectionStatus

| code | label |
|---|---|
| UNAUTHORIZED | 未授权 |
| AUTHORIZED | 已授权 |
| EXPIRED | 过期 |
| REVOKED | 已撤销 |
| UNAVAILABLE | 不可用 |

## BatchStatus

| code | label |
|---|---|
| UPLOADED | 已接收 |
| PREVIEWED | 已预览 |
| COMMITTED | 已提交 |
| FAILED | 失败 |
| EXPIRED | 过期 |

## RowStatus

| code | label |
|---|---|
| VALID | 可提交 |
| ERROR | 行错误 |
| DUPLICATE_SUSPECT | 疑似重复 |
| ALREADY_LINKED | 同源已关联 |
| EXCLUDED | 已排除 |
| COMMITTED | 已提交 |

## DeviceMode

| code | label |
|---|---|
| STAFF | 工作人员 |
| KIOSK | 自助 |

## DeviceStatus

| code | label |
|---|---|
| UNCLAIMED | 待领取 |
| ACTIVE | 有效 |
| REVOKED | 撤销 |
| EXPIRED | 过期 |

## OperationStatus

| code | label |
|---|---|
| IN_PROGRESS | 执行中 |
| SUCCEEDED | 成功 |
| REJECTED | 确定拒绝 |

## ReviewKind

| code | label |
|---|---|
| INITIAL | 首次资格 |
| INCREASE | 增加名额 |

## ReviewResult

| code | label |
|---|---|
| PENDING | 待决定 |
| APPROVED | 通过 |
| REJECTED | 拒绝 |
| CANCELLED | 取消 |

## PartyField

| code | label |
|---|---|
| name | 姓名 |
| phone | 手机号 |
| email | 邮箱 |
| companionCount | 随行人数 |

## EventAction

| code | label |
|---|---|
| PUBLISH | 发布 |
| OPEN | 开启检票 |
| PAUSE | 暂停 |
| RESUME | 恢复 |
| END | 结束 |

## ExportFormat

| code | label |
|---|---|
| CSV | CSV |
| XLSX | Excel XLSX |

## RecoveryChannel

| code | label |
|---|---|
| EMAIL | 电子邮件 |
| SMS | 短信 |

## CountMode

| code | label |
|---|---|
| COMPANIONS | 随行人数 |
| TOTAL | 含本人人数 |

## DuplicateResolution

| code | label |
|---|---|
| LINK_EXISTING | 关联既有组 |
| KEEP_SEPARATE | 分别保留 |
| EXCLUDE | 排除 |

## Decision

| code | label |
|---|---|
| APPROVE | 通过 |
| REJECT | 拒绝 |

## CleanupScope

| code | label |
|---|---|
| PRIVATE_ACCESS | 私密入口 |
| LIVE_ROWS | 在线明细 |
| TEMP_FILES | 导入暂存 |
| EXPORT_FILES | 导出文件 |
| TASK_PAYLOADS | 任务副本 |
| BACKUPS | 备份覆盖 |

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v0.1 | 2026-10-21 | 在线D3初始候选，未实现运行枚举。 |

| v0.2 | 2026-10-21 | PageSpec契约补齐清理范围枚举，按ADR-043项目规范检查。 |
