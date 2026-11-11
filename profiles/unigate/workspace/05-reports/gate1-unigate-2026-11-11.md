# UniGate Gate 1 验证报告（ADR-047 重划分后重跑）

日期：2026-11-11（Asia/Shanghai）；版本：v1.0；模块：unigate；执行：ZCode，ai-project-engine:design/validation。

**判定：PASS。ADR-047 领域重划分的文档级联在新基线上通过设计门控；既有 Gate 2 代码交付结论不受影响（代码未动）。**

本报告取代 [gate1-unigate-2026-10-21.md](gate1-unigate-2026-10-21.md) 成为当前有效 Gate 1 结论；旧报告为时点记录保留不改写。本次为**重划分重跑**，不是全量重设计：PRD、PageSpec、页面契约、枚举池、DDL、85 操作的路径/DTO 均未变更，变更范围为 BA/AA/FL 模块归属、领域模型上下文与聚合、API 文档 AA 编号追溯。

## 依据与检查范围

激活链 `unigate → springboot-vue-web`。依据为引擎 engine-rules 的 Gate1 十项、design 技能八项、validation 技能清单及本栈规约；新增依据 ADR-047（领域重划分，见 [domain-redivision-proposal.md](domain-redivision-proposal.md)）。

输入（新基线）：BA/AA v0.5（4 阶段/6 关键业务/17 活动；6 业务+3 平台模块，AA 叶 23）、FL v1.1（43 项重挂）、领域模型 v2.0（8 上下文+平台内核、8 聚合根）、API v0.4/OpenAPI 0.3.0（85 操作，AA 编号重挂）、25 表数据设计 v0.4、29 组枚举、33 页 PageSpec v1.0（未变）。ADR-043 项目检查依据与原版资产缺失记录继续有效。

## 机械检查（全部重跑，含新基线快照）

- D3 文档检查：**37 通过、0 失败**；[D3-project-check.json](D3-project-check.json)（本次重新生成）。
- D4 确认版检查：**26 通过、0 失败**，2,253 处字段绑定；[D4-project-check.json](D4-project-check.json)（重新生成，API/页面指纹已更新为新 SHA256）。
- 独立跨阶段检查：**19 通过、0 失败**；[Gate1-cross-stage-check.json](Gate1-cross-stage-check.json)（重新生成，含 19 份产物新 SHA256 及 UTC 时点）。
- 重划分级联校验（本次新增，见 CHANGELOG 2026-11-11）：AA 追溯 17/17 命中 BA L4；FL 43 条目覆盖 23/23 子模块且视图与表格一致；API md 143 处、openapi 200 处 AA 编号重挂后全集有效且无旧编号残留；JSON 合法、行尾保留。

复现命令（在项目根执行）：

```powershell
python -X utf8 tools/design/check_contract.py --write-report
python -X utf8 tools/design/check_pages.py --write-report
python -X utf8 tools/design/check_gate1.py --write-report
```

### 检查工具随基线的同步（如实披露）

tools/design/check_gate1.py 两处断言按 ADR-047 更新，其余断言未动：

1. `ba-aa-map`：BA L3 预期 8→6（evidence 注明 per ADR-047）。
2. `domain-api-roots`：聚合根清单 7→8（DeviceSession→Device、新增 Account）。

此外跨阶段报告 basis 追加 ADR-047。三个工具的其余 79 项断言（契约、页面、跨阶段）逐字未改且全部通过，说明重划分只改变了模块归属语义，未破坏任何结构契约。

## 引擎机制十项（ER-G1）

| 编号 | 检查 | 结果及证据 |
|---|---|---|
| ER-G1-01 | 架构与模块设计一致 | PASS；6 项 BA 关键业务/17 项活动映射 AA，23 个 AA 叶覆盖 43 项 FL 及 API/PageSpec，无新增模块（新基线） |
| ER-G1-02 | PRD 覆盖业务场景与角色 | PASS；PRD 未变更（ADR-041 确认版），16 项 AC、五角色场景保持 |
| ER-G1-03 | API 完整 | PASS；85 操作逐一列方法/路径/身份/字段/错误，仅 §4.2/§5.2 的 AA 编号重挂，D3 37 项重跑通过 |
| ER-G1-04 | 页面类型覆盖 | PASS；33 页未变更，列表/表单/详情/审核/导入/扫描台/kiosk 保持 |
| ER-G1-05 | API 与页面字段对齐 | PASS；2,253 处绑定重跑解析通过，字段路径未受重划分影响 |
| ER-G1-06 | 状态机完整 | PASS；PRD4/API2 状态机与逐动作守卫未变更 |
| ER-G1-07 | 枚举统一 | PASS；29 组池未变更，code/label 与 OAS 一致 |
| ER-G1-08 | 路径合规 | PASS；85 条 /api/v1 路径零变更 |
| ER-G1-09 | changelog | PASS；BA/AA/FL、PRD/模型、API/数据均含 2026-11-11 重划分条目及历史记录 |
| ER-G1-10 | 状态索引 | PASS；INDEX 注记已对齐新基线（D0—D4 complete、Gate1 本报告后 pass；Gate2 维持 pass，代码未动） |

## 设计技能八项（DS-G1）

| 编号 | 检查 | 结果及证据 |
|---|---|---|
| DS-G1-01 | 三文档全局检查 | PASS；BA→AA→FL→API 重挂后逐级验证通过（本次级联校验），模板格式与树状视图一致 |
| DS-G1-02 | PRD 完备性 | PASS；PRD 未变更，原 16 项 AC 结论沿用 |
| DS-G1-03 | 领域模型分离 | PASS；v2.0 八个聚合根（Event、RegistrationParty、ImportBatch、SourceConnection、Device、ExportJob、CleanupRun、Account），实体/值对象/过程数据/读模型分类及子流程 I/O 保持，共写白名单显式化 |
| DS-G1-04 | API 全局检查 | PASS（ADR-043 项目依据）；37 项机械重跑通过，v0.4 变更仅为 AA 编号重挂与 §1.4/§4.1 上下文表述 |
| DS-G1-05 | API 变更摘要 | PASS；API 第 6 节新增 v0.4 条目，重挂范围与零变更声明可追溯 |
| DS-G1-06 | PageSpec 输出检查 | PASS；26 项机械重跑通过，PageSpec 不引用 AA 编号（以 API 操作为锚），无需改动 |
| DS-G1-07 | 跨阶段一致性 | PASS；19 项独立机械重跑通过；API 领域对象表述与 D2 v2.0 一致（Device 为域级更名，wire DTO DeviceSessionView 按 ADR-047 零变更保留，映射已在 API §1.4 声明） |
| DS-G1-08 | 枚举无冲突 | PASS；29 组无重复定义，未变更 |

## 重划分特有人工核对

| 核对点 | 结论 |
|---|---|
| 新旧编号映射正确性 | 11 条映射（如 2.2.3.1→2.1.1.3、2.1.3.x→2.1.2.3—5、2.4.3.1/2.4.3.3 互换）经 git diff 抽检与全集校验双向一致；两阶段替换防链式误替 |
| FL 归属迁移语义 | 凭证/领票/找票入核心域 2.1.1.3/2.1.1.4、检票类入 2.1.2.3—2.1.2.5、统计/留存移 2.2.3.x/2.2.4.x、任务/幂等/审计归平台运行 2.4.3.1/2/3——与 AA v0.5 模块定义逐一对应；条目功能点描述未改，无权限扩大 |
| 合并的 BA 依据 | 凭证=RegistrationParty 聚合内实体、现场会话=单一业务概念，合并不改变 L4 活动边界与角色覆盖（17 项 L4 全部保留） |
| 聚合根更名影响 | Device 为域级更名（原 DeviceSession），表 ug_device、wire DTO、路径不变；Account 自实体升聚合根，实现 IdentityService 已有对应 |
| 上下文与实现对应 | 8 上下文+平台内核与 20 文件实现的服务/表属主对应关系见提案 §3.3；包结构重构属 D-5b，不阻塞设计门控 |
| 历史记录完整性 | 05-reports 旧报告、旧 Gate1 JSON 为时点记录未改写；新检查 JSON 覆盖为新快照，CHANGELOG 记录取代关系 |

## 判定

设计阻断项 0，建议项 2（不阻塞）：

1. Gate 2 的 16 项跨栈及浏览器证据引用旧模块语义锚（SW-03 业务链路描述），建议在 D-5b 代码重构时随 Gate 2 复查一并刷新；代码未动，现有 Gate 2 结论维持。
2. 后端包结构仍为平铺单包（20 文件），与 v2.0 上下文边界的对齐属 D-5b 排期事项。

**判定 PASS**。设计基线（BA/AA/FL v0.5/v1.1、模型 v2.0、API v0.4）满足进入后续阶段的条件；下一步为 D-5b 代码包重构与 R1—R3 边界修复，完成后按新基线复查 Gate 2。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-11 | ADR-047 重划分后独立重跑 Gate 1；工具两处基线锚点同步（BA L3=6、8 聚合根）并如实披露；机械 37/26/19 全部重跑通过，结论 PASS。 |
