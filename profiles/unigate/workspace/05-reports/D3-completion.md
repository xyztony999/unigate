# UniGate D3 项目检查与归档

日期2026-10-21；v1.0；结论：按用户采用的项目检查依据，D3完成，可进入D4。不是Gate1报告，也不声明插件原版CH/GL完整通过。

## 输入及授权依据

ADR-041确认FL/PRD/领域模型，D1-D2-completion.md记录输入检查；ADR-042采用API行为与字段合同，允许核对中修正既定功能缺口；ADR-043明确采用项目机械＋人工检查方案，缺失原版资产保留记录且不再阻塞D3/D4。未取消状态、权限、字段、并发及隐私要求，未授权业务代码。

## 机械检查

实际运行`python -X utf8 tools/design/check_contract.py --write-report`：37项通过、0项失败，API0.3.0共85个操作；机器证据[D3-project-check.json](D3-project-check.json)。包括本地schema引用、请求闭合/幂等/CSRF/匿名连续性、角色/AA/FL、字段/枚举、导入集合、示例JSON、25表设计、页面新增元数据以及固定GATE幂等作用域。

该工具不连接HTTP/DB/提供方，不是完整OpenAPI/JSON Schema验证器。原版可追溯来源164条及缺项见D3-check-source.json，`originalEngineD3Passed=false`保留，不能与“本项目D3通过”混淆。

## 人工核对与修正

| 项目 | 证据 | 结果 |
|---|---|---|
| 目标、范围、角色 | API1、PRD及ADR-039—043 | 在线首版、五角色、单主办方多活动保持 |
| 聚合/关系与数据 | 模型及数据设计1—3 | Event/Party容量同库原子，过程数据不变新业务对象 |
| 状态与守卫 | API2.1—2.2、EventTransition/ReviewDecide | 待审不占额；开放/暂停冻结公众变更；增额/纠错负责人限定 |
| 容量、组余额与纠错 | 数据设计4.1—4.4 | 不超容量/批准额，更正追加且保留ever_admitted |
| 核销操作身份 | API2.3、GateCommit/GateOperation | 固定GATE+活动+动作+操作ID；设备/组/原版本/人数入摘要，换组/换端不能命中新作用域重复扣数 |
| 匿名未知结果 | API3.2.1、RegistrationResult | 原cookie+CSRF+clientContextId读取原响应，无需持久化表单PII；换上下文不自动重发 |
| 本人授权/管理员边界 | MyGrantList、GrantEventOptions | 本人范围专用读取；管理员活动选择只给标识/标题/时间，不隐含名单或到场数 |
| 字段最小化与检索 | PartyView、GateCandidate、API3.2 | 校方服务器投影，检索也限获权字段；kiosk无名单搜索 |
| 导入输入与事务 | ResourceFields/BatchView、BatchCommit | 明确可选列及200列保护，预览指纹和同源防重、跨源核对，选中行整批原子 |
| 票与私密能力 | CredentialReissue/RecoveryVerify | 三种能力分离，补发不重置净核销，私密响应只限获权恢复 |
| 现场阶段可见 | DeviceSessionView.eventStatus | 浏览器有真实窗口状态输入，提交仍以服务端事务为准 |
| 校方查询时点 | PartyPage.asOf、Statistics.asOf | 不用本地刷新时钟冒充服务端统计快照 |
| 通知/外部任务 | API3.3—3.4 | 受理与完成分离，失败不撤资格；真实平台权限未验证不伪报接通 |
| 导出与交付 | API3.4、Export/Delivery接口 | EXPORT独立、列/时效重验、无秘密导出；登记不执行外发 |
| 实际清理/备份 | CleanupView.scopes、数据设计5 | 六范围结果，备份未覆盖不声称全部完成；物理失败入口仍失效 |
| 追溯与变更 | API6及D3三个文档/枚举/选择池 | v0.3为既定映射/授权/查询/清理功能的输入补齐，不新增业务角色或离线范围 |

上述为实际设计对照，不是运行测试。Java/Spring Boot+Vue是已选技术，JDK/具体框架补丁/PostgreSQL等仍为候选；实际应用/文档权限、相机/kiosk、并发容量、备份恢复和海外真实启用配置尚未验证。

## 归档与后续（D3归档时点）

D3产物位于02-detailed/api/，枚举及选择池位于common-spec；D4只从采用且经项目检查的API派生。当前D4草案待七项集中确认，确认并检查后独立执行Gate1，不能因本报告直接生成业务代码或标Gate1通过。

最新阶段注记（2026-10-21）：ADR-044已确认D4，归档见[D4完成报告](D4-completion.md)；独立门控[Gate1报告](gate1-unigate-2026-10-21.md)判定PASS。上述“待确认/未执行”描述D3归档时点。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-10-21 | 按ADR-043记录37项实际机械检查及16项人工核对，D3按项目依据完成归档。 |
