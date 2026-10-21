# UniGate 页面准备清单

日期2026-10-21；v0.1（历史输入准备稿）；ADR-043后D3已按项目依据完成，正式D4草案见../02-detailed/interaction/README.md及D4-review.md，本页不再代表当前阶段状态。以下入口为当时准备记录，不是D4完成或Gate1证据。

## 工作人员公共入口

- 登录/首次改密：SessionBootstrap→Login→SessionMe→PasswordChange，过期返回登录并保留待确认原操作引用。
- 我的授权与活动：MyGrantList、EventList/EventOptions，展示本人有效活动、字段/动作和期限；管理员可兼任其他角色，但不靠管理员身份查看名单。

## 系统管理员

- 账号与角色：AccountList/Create/Update、GrantList/Save/Revoke；活动负责人只用自身范围授权动作，不开放全局账号浏览。
- 平台连接：ConnectionList/Save/Authorize/Revoke；回调ConnectionCallback，显示已配置/授权/不可用，不把保存当接通。
- 任务状态：JobList/Detail/Retry，脱敏元数据，不展示名单或任意执行能力。

## 活动负责人

- 活动列表/信息与规则：EventList/Detail/Create/Update、PolicyGet/Update。
- 来源与导入：SourceList/Save/Pull、ResourceList、BatchUpload/Detail/Preview/Rows、DuplicateResolve/BatchCommit；结果页接JobDetail。
- 报名与审核：PartyList/Detail/Change/Cancel、ReviewList/Decide；凭证CredentialReissue/Revoke和EmailResend。
- 检票控制与设备：EventTransition、DeviceRegister/List/Revoke；领取端DeviceClaim/Detail。
- 工作人员检票：TicketResolve→GateCommit/GateOperation，辅助找票GateLookup，异常转负责人。
- 现场异常：GateWalkIn/Increase、GateRecords/Correct，名额变化与核销分别确认。
- 统计、导出与交付：Statistics、PartyList/Detail、ExportCreate/Detail/Download、DeliveryList/Record。
- 留存、任务和审计：CleanupDetail/Retry、JobList/Detail/Retry、AuditList、OperationResult。

## 检票员

- 进入获权活动及就绪：EventList、MyGrantList、SessionMe、DeviceDetail；STAFF核销同时需要获权账号与设备。
- 扫码/本次数量/结果：TicketResolve、GateCommit、GateOperation；未知结果锁住本次动作，恢复查原ID。
- 辅助找票：GateLookup只显示最小候选，交负责人核对补发；无增额、纠错、导出或补发按钮。
- 本人/本端记录：GateRecords、AuditList、OperationResult，服务端收缩到本人或经cookie验证本端。

## 校方只读人员

- 我的授权：MyGrantList，活动、字段、动作和到期；没有全局授权列表入口。
- 活动统计/明细：Statistics、PartyList/Detail，按字段授权投影；无导出、修改、检票和平台凭据。

## 参与者与自助终端

- 活动报名：PublicEvent、SessionBootstrap、可选SmsStart/Verify、SubmitRegistration，先展示动态告知/字段。
- 我的报名与票：PartyExchange、MyParty/MyCredential、获准MyPartyChange/Cancel，页面私密入口与入场二维码分开。
- 找回入口：RecoveryStart/Verify；无已留联系方式转负责人核对，不凭尾号补发。
- 待确认操作：MyOperation或原SubmitRegistration同ID重试；匿名上下文改变则停止旧动作自动重发。
- kiosk：DeviceDetail、TicketResolve、GateCommit/GateOperation；单组完成清屏，未知时转工作人员，不提供名单搜索。

## 布局候选（待正式D4）

桌面管理以侧栏和工作区组织；移动报名以单列短表单/状态/票为主；工作人员检票突出当前组剩余数及本次人数；kiosk大触控区、最小个人信息和完成后清屏。不增加离线操作，不把普通网页全屏等同系统kiosk锁定。正式路由、字段、验证、按钮和七项确认在D4展开。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v0.1 | 2026-10-21 | 从82个既有API操作准备五角色入口，仅为后续D4输入。 |
