# UniGate 页面与交互规格总览

| 属性 | 值 |
|---|---|
| 版本 | v1.0 |
| 日期 | 2026-10-21 |
| 状态 | ADR-044已确认七项交互，D4归档；运行验证另行执行 |
| App / 模块 | UniGate Web / unigate |
| 输入 | API v0.3、OpenAPI0.3.0、ADR-042/043/044 |
| 检查依据 | 用户已采用的项目机械＋人工检查，不冒称原版FX/CH/GL全通过 |

## 1. 页面组织与组件语义

五业务角色共用会话与授权入口；shared不是新增角色。33个页面/工作区按六份角色/共享规格归档。管理端侧栏+活动工作区；移动端单列；工作人员扫描台和kiosk分别组织。此处使用Vue通用组件语义（列表、抽屉、表单、状态卡、扫码区），不要求ScmCardPage或特定UI库，不生成Vue源码。

### 路由与页面索引

| ID | 页面 | 路由 | 角色 | 布局 | 详情 |
|---|---|---|---|---|---|
| S01 | 登录 | /login | ADMIN,ORGANIZER,CHECKER,READER | mobile_form | [shared](shared/unigate/unigate.md) |
| S02 | 修改本人密码 | /account/password | ADMIN,ORGANIZER,CHECKER,READER | dialog | [shared](shared/unigate/unigate.md) |
| S03 | 我的有效授权 | /account/access | ORGANIZER,CHECKER,READER | responsive_list | [shared](shared/unigate/unigate.md) |
| S04 | 领取检票终端 | /terminal/activate | ANONYMOUS | mobile_form | [shared](shared/unigate/unigate.md) |
| A01 | 账号与活动授权 | /admin/accounts | ADMIN | management_list | [admin](admin/unigate/unigate.md) |
| A02 | 平台连接 | /admin/connections | ADMIN | management_list | [admin](admin/unigate/unigate.md) |
| A03 | 连接与任务状态 | /admin/jobs | ADMIN | management_list | [admin](admin/unigate/unigate.md) |
| O01 | 活动列表与基本信息 | /manage/events | ORGANIZER | management_list | [organizer](organizer/unigate/unigate.md) |
| O02 | 报名规则 | /manage/events/:eventId/settings | ORGANIZER | settings_form | [organizer](organizer/unigate/unigate.md) |
| O03 | 报名组列表 | /manage/events/:eventId/parties | ORGANIZER | management_list | [organizer](organizer/unigate/unigate.md) |
| O04 | 报名组与凭证 | /manage/events/:eventId/parties/:partyId | ORGANIZER | detail_cards | [organizer](organizer/unigate/unigate.md) |
| O05 | 首次及增员审核 | /manage/events/:eventId/reviews | ORGANIZER | review_workspace | [organizer](organizer/unigate/unigate.md) |
| O06 | 平台来源与字段映射 | /manage/events/:eventId/sources | ORGANIZER | settings_form | [organizer](organizer/unigate/unigate.md) |
| O07 | 名单导入工作区 | /manage/events/:eventId/imports | ORGANIZER | import_steps | [organizer](organizer/unigate/unigate.md) |
| O08 | 检票控制与终端 | /manage/events/:eventId/gate | ORGANIZER | control_workspace | [organizer](organizer/unigate/unigate.md) |
| O09 | 工作人员检票 | /manage/events/:eventId/scan | ORGANIZER | scanner | [organizer](organizer/unigate/unigate.md) |
| O10 | 现场补录与增员 | /manage/events/:eventId/onsite | ORGANIZER | onsite_form | [organizer](organizer/unigate/unigate.md) |
| O11 | 原核销及误检更正 | /manage/events/:eventId/records | ORGANIZER | management_list | [organizer](organizer/unigate/unigate.md) |
| O12 | 活动统计 | /manage/events/:eventId/statistics | ORGANIZER | statistics | [organizer](organizer/unigate/unigate.md) |
| O13 | 受控导出与交付登记 | /manage/events/:eventId/exports | ORGANIZER | delivery_workspace | [organizer](organizer/unigate/unigate.md) |
| O14 | 留存与清理 | /manage/events/:eventId/lifecycle | ORGANIZER | detail_cards | [organizer](organizer/unigate/unigate.md) |
| O15 | 任务与通知 | /manage/events/:eventId/jobs | ORGANIZER | management_list | [organizer](organizer/unigate/unigate.md) |
| O16 | 活动授权 | /manage/events/:eventId/access | ORGANIZER | management_list | [organizer](organizer/unigate/unigate.md) |
| O17 | 审计与原操作结果 | /manage/events/:eventId/audit | ORGANIZER | management_list | [organizer](organizer/unigate/unigate.md) |
| C01 | 工作人员检票 | /check/events/:eventId/scan | CHECKER | scanner | [checker](checker/unigate/unigate.md) |
| C02 | 辅助找票 | /check/events/:eventId/lookup | CHECKER | mobile_form | [checker](checker/unigate/unigate.md) |
| C03 | 本人及本端记录 | /check/events/:eventId/records | CHECKER | responsive_list | [checker](checker/unigate/unigate.md) |
| R01 | 获权活动统计 | /read/events/:eventId/statistics | READER | statistics | [reader](reader/unigate/unigate.md) |
| R02 | 获准字段明细 | /read/events/:eventId/parties | READER | responsive_list | [reader](reader/unigate/unigate.md) |
| P01 | 活动报名 | /e/:eventId/register | PARTICIPANT | mobile_form | [participant](participant/unigate/unigate.md) |
| P02 | 我的报名与入场码 | /me/party | PARTICIPANT | ticket_status | [participant](participant/unigate/unigate.md) |
| P03 | 找回本组入口 | /e/:eventId/recover | PARTICIPANT | mobile_form | [participant](participant/unigate/unigate.md) |
| P04 | 自助检票 | /terminal/kiosk | PARTICIPANT | kiosk | [participant](participant/unigate/unigate.md) |

## 2. 查询、表单与字段规则

精确请求/响应字段、类型、必填、控件、可见、枚举和动作绑定见各角色文档及page-contracts.json，每一项均引用OpenAPI schema。请求中operationId/版本/路由ID/证明/能力为系统绑定，不要求参与者手工填写。默认分页1/20、上限100，保持后端稳定排序；检票辅助找票固定最多10条。只对契约已有参数筛选，不增加客户端任意排序、全名单下载或全文检索。

活动的姓名/手机号/邮箱/随行字段按enabled显示、按required校验；缺随行0，界面明确“本人+随行=申请总人数”。关闭字段省略，空值不私自转null。手机号保留国家区号，不强制11位；邮箱不当作唯一键。可选短信仅在活动启用时显示，验证码及证明仅内存；负责人现场表单的来宾验证使用独立ANON子流程和原客户端上下文，不以工作人员会话绕过验证。

映射列来自BatchView.availableColumns或ResourceFieldsView.availableColumns，选code提交，label展示；200列上限、未知列不变新表单字段。来源资格/人数COMPANIONS或TOTAL明确选择。必填字段未映射会预览行错，不默认猜表头。同源/疑似重复先核对，选中集合只含可提交行；最终容量或版本冲突整批回滚。

时间控件按活动IANA时区显示并标注时区，转换带偏移RFC3339；显示服务器asOf与本地刷新时间时区分命名。列表枚举只读code+label，状态颜色附文字；数组字典为多选。长文本独占一行；小操作抽屉/弹窗，大配置分组页面。导入为单路由四步（上传、映射、核对、整批提交），不添加generate-id或虚构分步保存接口。

## 3. 列与详情区块

默认列依各页清单顺序，首个业务参考列左固定、行操作右侧；移动端转卡片。后台人员字段不等于参与者名单字段。可选基础字段仅在后端已返回且获权时展示，不先下载完整字段再CSS隐藏。搜索也只能匹配获准字段和报名参考号，校方不能通过未获手机号搜索推断存在。

报名详情分基础信息、资格与名额、票证动作及可用关联结果；审核工作区仅展示当前选择的申请和其组详情，不编造完整审批历史；审计只有页面获准且调用AuditList时存在。只读页面不显示编辑/导出；kiosk只有当前组名额，不显示姓名/电话/邮箱。统计卡来自Statistics，报名列表时点来自PartyPage.asOf；无“当前在场人数”。

## 4. 动作、权限与操作前提

按钮是本栈静态动作集，按角色、活动授权、字段、数据状态与独立动作权显示/启用；服务器最终鉴权。未提供operList不引入参考页面引擎。管理员只读授权目标活动的id/title/startsAt元数据，不因此读取名单/占额；负责人对已由管理员分配到本活动的账号调整检票员/校方范围，不开放全局账号浏览。

EventTransition仅显示当前状态有效的动作：DRAFT发布、READY开始、OPEN暂停/结束、PAUSED恢复/结束；开始说明冻结公众报名变更，结束说明不自动重开。首次/增员审核仅READY，决定及理由必填，容量不足保持待审及原批准数。普通改人数/取消需READY且hasAdmissionHistory=false；补录/现场增额仅负责人OPEN，成功只分配名额、不自动核销。误检更正仅负责人OPEN/PAUSED、关联原记录及理由，不删除事实或开放自助取消。

导出需要独立EXPORT，表单列仅限获准白名单，生成及下载分别重验；202只显示“已受理”，轮询至成功才给下载。邮件202不显示已送达。交付登记不是自动发送；清理按六个scope展示状态，备份未完成不能说全副本清完，重试仅到期失败范围。

表单提交前校验、提交期间禁双击，业务操作ID及原版本固定。409按code而非仅HTTP判断：OPERATION_IN_PROGRESS仍为进行中，遵Retry-After沿原ID等待/查询；只有明确版本/容量等业务拒绝后刷新并由用户确认新动作；网络、超时、5xx未知不自动换ID。纯导航、图片生成/保存、复制私密入口、相机权限及清屏是本地展示动作，不新建业务接口。取消脏表单提示，未知操作离开后仍保留最小非秘密引用。

## 5. 检票台状态机及交互默认

工作人员与kiosk均“识别组码→显示剩余→确认本次数量（默认1）→服务器核销→结果”。整数1..remainingCount，剩余0禁提交；识别完成本身不放行。只有新成功且showAsNewAdmission=true才有新放行绿色及可选提示音；重放/查询显示原人数/时间，明确请核对，不重复绿灯。

| 状态 | 触发 | 界面/限制 | 下一步 |
|---|---|---|---|
| READY | 设备有效且活动开放 | 展示扫码区，不预存名单 | 扫码 |
| RESOLVING | 读取二维码 | 停止重复识别；TicketResolve不核销 | 等候识别结果 |
| CONFIRMING | 已取得当前组上下文 | 大字显示批准/已入/剩余，本次默认1且不大于剩余 | 确认本次入场 |
| SUBMITTING | 已生成operationId并提交 | 锁确认与下一组，不能另起一笔 | 等待 |
| SUCCESS | GateResult为新提交成功且showAsNewAdmission=true | 显示本次及剩余，文字/图标/声音可选，允许放行 | 完成/下一组 |
| KNOWN_REPLAY | 原操作查询或重放成功 | 显示原时间人数，明确已记录待核对，不再绿灯/重复响铃 | 核对或转负责人 |
| REJECTED | 已收到明确业务拒绝 | 显示无效/不足/冲突文字；无成功反馈 | 刷新并明确确认新动作或转负责人 |
| UNKNOWN | 超时、断网或5xx无确定结果 | 停本端该次操作；保留原ID/非秘密参数，不保存名单 | 恢复后同ID查询/重试 |
| BLOCKED | 活动暂停/结束、权限或设备失效 | 禁止提交；查询旧结果仍依当前有效权限 | 联系负责人 |

kiosk成功显示5秒后清当前组，点“完成”立即清；未提交空闲30秒清屏。UNKNOWN不自动清未确认ID，不自动扫下一组，显示联系工作人员核对。窗口和设备状态从DeviceSessionView.eventStatus/device读取并定时检查，提交仍重新验证；一台掉线不自动暂停全场。普通全屏不能替代Android系统kiosk锁定。相机与扫码枪输入适配尚需设备实测。

## 6. 私密入口、恢复和浏览器保存

登录、设备、匿名、本组查询与变更cookie分开。首次改密前禁业务操作。bootstrap复用有效匿名上下文，全部ANON写及RegistrationResult带原X-Client-Context-ID；上下文变更时停止自动重发。手机号/邮箱/姓名不写本地存储或服务工作线程缓存，不整份缓存名单，敏感HTTP一律no-store。

活动报名二维码仅打开公开活动页，不强制微信身份。报名成功先展示资格/状态、提示保存私密查询入口；有效票另显示组二维码，参与者可显式保存入场码图片，票不含联系信息。查询fragment读入即清URL，禁止第三方分析脚本；只在原提交/验证恢复时复制私密查询链接，普通MyParty响应不泄漏新秘密。保存入场码和保存查询入口分别解释，普通书签不被承诺永久领票。

报名提交未知且刷新：仅保eventId/operationId/clientContextId等非秘密元数据；原上下文有效先调用RegistrationResult，不需持久化原表单PII。无记录仍不自动新建；原内容仍在内存可同ID重试，缺内容/响应过期转邮件或负责人核对。找回统一受理，不以尾号/手机号当唯一身份。

核销作用域为固定GATE+活动+动作+operationId；稳定deviceId/组ID/原版本/人数在摘要内，换组/换端复用原ID必须冲突，不能命中新作用域再扣一次。原端上下文有效才原ID重试；重启后凭原ID补查，kiosk需要重扫原组取得当前有效上下文，服务端核对组一致；换端/失效或无法确认交负责人查看事实，不自动重核销。

## 7. 触控、无障碍及异常

触控至少48 CSS px，数字步进器支持键盘，焦点清晰，字段错误关联输入。颜色/声音不是唯一信号，读屏可读状态；提示“已处理/待确认/失败”分别表达。管理弹窗可取消但不抹去服务器结果，页面离开/会话失效清敏感内存；保留最小待确认引用用于获权恢复。相机拒绝给工作人员辅助/扫码枪路径，kiosk不提供名单搜索。布局断点是设计候选，不虚报屏幕、浏览器或性能测试。

## 8. 字典与Action来源

所有Action路径/角色/字段定义来自OpenAPI0.3.0；页面JSON保留source schema和operationId。29组枚举统一引用03-common-enums/04-business-enum，动态列/账号/资源选择有明确服务，非自由标签。OAuth ConnectionCallback属于协议回调而非用户页面按钮，前端不获取/手动转发code；原连接页刷新授权状态。没有独立字典写接口或臆造工作流端点。

## 9. 检查与确认范围

机械检查覆盖路由唯一、所有API/FL对应、角色/上下文、请求响应字段与类型、列表列、字典、必填、未知状态与隐私默认；人工再按七项内容核对。草案检查留存于[D4-draft-check.md](../../05-reports/D4-draft-check.md)，确认记录见[D4-review.md](../../05-reports/D4-review.md)。用户选择1采用七项及默认流程（ADR-044），本版归档D4；复跑与人工核对见[D4-completion.md](../../05-reports/D4-completion.md)，独立Gate1另报告。本次无业务代码授权。

## 更新记录

| 版本 | 日期 | 类型 | 说明 |
|---|---|---|---|
| v0.1 | 2026-10-21 | 初始草案 | 33页路由/字段/动作/角色/状态及恢复；供七项集中确认，未运行Vue/后端/设备。 |
| v1.0 | 2026-10-21 | 确认归档 | ADR-044集中采用七项及默认流程，复跑项目检查后独立执行Gate1；无运行测试声明 |
| v1.0注记 | 2026-10-21 | Gate1一致性 | 细化409处理中与确定拒绝的不同恢复路径，避免新ID重复写；接口不变。 |
