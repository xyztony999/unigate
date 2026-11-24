# Gate 1 验证报告 — 设计→代码门控（增量：ADR-048 技术栈迁移）

**模块**：unigate
**检查时间**：2026-11-24T08:32Z（UTC）／2026-11-24（Asia/Shanghai）
**API文档版本**：api-unigate.md v0.3.1（接口契约保持 OpenAPI 0.3.0）
**基线**：gate1-unigate-2026-10-21.md（PASS）
**执行**：Gate 1 门控子代理（多代理模式，只读；仅写本报告），ai-project-engine:validation

## 总体判定：WARN

**设计侧 CRITICAL 项全部通过，无阻断缺陷。** 契约字节级冻结、冻结范围零改动、ADR-048 四处表述与 OpenAPI 实录一致。3 项非阻塞建议（W-01~W-03，见下）：1 项 INDEX note 长度微超约定，2 项项目机械检查工具因栈迁移/引擎缓存路径过期而失效（工具侧问题，非设计缺陷）。按判定规则（非阻塞建议 → WARN）总体记 WARN；主会话确认建议项后可进入 Go 重写实现阶段。

## 检查范围与口径（增量模式）

- 编号源：engine-rules.md G1-01~G1-11（产物质量）+ design SKILL.md S1~S8（阶段完成度）。validation SKILL.md 的通用 CH1-CH5 / BE-01~34 / FE-001~141 清单为异栈（Java+特定 Vue 框架）资产，**不适用本项目，未套用**（沿用 ADR-043 项目依据口径）。
- 增量范围：ADR-048 后端 Java/Spring Boot → Go（go-vue3：gin/slog/pgx，单二进制 cmd/unigate + internal/，替换 backend/）；设计改动 7 文件（architecture-choices/index.md、design-decisions/index.md、00-architecture/CHANGELOG.md、api-unigate.md、INDEX.md、profiles/unigate/CLAUDE.md、profile.yaml）。
- 冻结沿用：前端 PageSpec/PRD/业务架构/领域模型/数据模型本次未改，其产物质量沿用 2026-10-21 Gate 1 PASS 结论（增量 GR-01 口径），不重复全量扫描；机械工具复跑仅作冻结完整性证据。
- 激活链核实：.claude/settings.json ACTIVE_PROFILE=unigate / ACTIVE_STACKS=go-vue3 / BACKEND_ROOT=backend；profile.yaml depends=[go-vue3]；.profile-mounts.json 挂载 go-vue3 rules/agents/stack-specs；profiles/go-vue3/ 含 backend-rules.md、frontend-rules.md、backend-02-api-conventions.md，构建门禁 `go build ./... && go vet ./...`。ER-03 绑定一致。

## 一、契约冻结硬校验（CRITICAL）

| 检查 | 结果 | 证据 |
|---|---|---|
| openapi-unigate.json sha256 | PASS | 实测 `cc185ea09d0f90ad1843e5716e7c5843566767ccce6e367dc4f26e667de6612b`，与要求值逐字一致 |
| git diff 该文件 | PASS | 空（工作区与 HEAD 完全一致） |
| 操作/路径数 | PASS | 85 路径 / 85 操作（POST 67、GET 18） |
| info.version | PASS | 0.3.0 |

## 二、冻结范围校验（CRITICAL）

`git diff --stat` 对下列路径全部为空（实测）：

- 00-architecture/business-arch.md、application-arch.md、function-list.md（AA 保持 v0.4）
- 02-detailed/api/data-model-unigate.md（25 表 DDL 不变）
- 02-detailed/interaction/（33 页 PageSpec）、02-detailed/prd/（PRD+领域模型）
- 01-solution/（含枚举池 03-common-enums.md / 04-business-enum.md）

workspace 内实际改动仅 3 文件：00-architecture/CHANGELOG.md（+4 行）、02-detailed/api/api-unigate.md（+7/-2 行）、INDEX.md（note 追加）。**冻结范围零改动，PASS。**

## 三、G1-01~G1-11 逐项结果

| 编号 | 检查项 | 结果 | 证据/说明 |
|---|---|---|---|
| G1-01 | 架构文档与模块设计一致 | PASS | BA/AA/FL 经 grep 核实为纯业务描述、无 Java/Spring/JDK 措辞，业务内容零改动；architecture-choices「技术与集成」节已按 ADR-048 更新为 Go，唯一 Java/Spring 出现处为迁移叙述本身；历史 ADR-028 段落按豁免保留；fl-role-coverage（43 项）/aa-leaves（23 叶）机械检查通过 |
| G1-02 | PRD 覆盖业务场景与角色 | PASS（沿用） | PRD/领域模型冻结未改，沿用基线 PASS；本次无业务范围变更 |
| G1-03 | API 设计完整 | PASS | 契约 intact（见第一节）；api-unigate.md §1.2 技术候选已改 Go 1.22+/gin/slog/pgx 并新增「项目冻结契约覆盖栈通用规约（ADR-048）」专段；§3.2 慢哈希候选改 Argon2id（Go 实现）；git diff 仅 4 处 hunk（头行/§1.2/§3.2/section-6），85 操作语义节未动 |
| G1-04 | 交互设计覆盖页面类型 | PASS（沿用） | PageSpec 冻结；check_pages.py 复跑 26/26 通过、33 页、interactionConfirmed=true |
| G1-05 | API 字段与交互字段对齐 | PASS（沿用） | 2,253 处字段绑定复跑全过；契约与 PageSpec 均未变 |
| G1-06 | 状态机完整 | PASS（沿用） | PRD/API 状态机节冻结未改 |
| G1-07 | 枚举统一无冲突 | PASS | 枚举池文件未改；enum-values（29 组池 ↔ OAS Code/Value 对象）与 enum-pool-files 机械检查通过 |
| G1-08 | 接口路径符合规范 | PASS（沿用） | 85 路径字节级未变，沿用基线项目路径规约结论（/api/v1/... 及协议例外） |
| G1-09 | changelog 已填写 | PASS | api-unigate.md section-6 有 v0.3.1（2026-11-24）行，载明契约 sha256 改前后一致（本次实测属实）；00-architecture/CHANGELOG.md 有 2026-11-24 节 |
| G1-10 | INDEX 状态已更新 | PASS（附 W-01） | modules[unigate] 六 stage status 均保持 complete；requirement/api/backend note 及 pipeline architecture note 已追加 ADR-048 增量说明，文末有增量注记行；backend note 全长 126 字，若 ≤120 字约定按整条 note 计则微超 6 字（增量部分本身约 37 字），见 W-01 |
| G1-11 | 跨阶段模型一致性 | PASS | 契约 ↔ PageSpec 未变（沿用）；ADR-048 在 architecture-choices/design-decisions/api-unigate/CLAUDE.md 四处表述一致，且与 OpenAPI 实录逐项核对通过（见第四节） |

## 四、ADR-048 表述 ↔ OpenAPI 实录核对（G1-11 展开）

| 文档声称 | OpenAPI 实测 | 结果 |
|---|---|---|
| 真实 HTTP 状态码 200/202/400/401/403/404/409/410/422/429/500/503 | 恰为该 12 种（200×78、202×7，其余错误码各 85 处），无全 200 风格 | 一致 |
| 响应包络 {code,message,data,requestId} | 全部 934 个 JSON 响应 schema 顶层属性恰为 code/data/message/requestId；ErrorEnvelope.code 类型 string | 一致 |
| 成功 code="OK" | 200/202 的 code enum 均为 ["OK"]（唯一例外为 export/download 的 text/csv 与 xlsx 二进制响应，无 JSON 包络，属预期） | 一致 |
| 401 用 UNAUTHENTICATED/SESSION_EXPIRED | 401 响应 description 为 "UNAUTHENTICATED / SESSION_EXPIRED"；UNAUTHENTICATED 出现 86 次、SESSION_EXPIRED 95 次；**AUTHENTICATION_FAILED 在契约与四份文档中均 0 次** | 一致 |
| 字符串错误码示例 VERSION_CONFLICT/CSRF_INVALID/UNAUTHENTICATED | 三者均在契约中存在（各 86 次） | 一致 |
| 列表 items、分页 page/pageSize（响应含 items/total/page/pageSize） | 抽查 list 接口 data 属性恰为 items/page/pageSize/total | 一致 |
| go-vue3 通用规约为全 200 {code,msg,data}/records/pageNum/数字码 | go-vue3 backend-02-api-conventions.md 第 8/11/40 行属实；ADR「让位」描述准确 | 一致 |
| go-vue3 默认 sqlite/MySQL 让位于 pgx/PostgreSQL | backend-03-patterns.md 默认 SQLite/MySQL 属实；pgx 例外已记录 | 一致 |

ADR-048 要素完整性：内存动机（2核2G/3Mbps，JVM 占用过大）✓、保留面板 PostgreSQL 18 及全部 PG 特性（FOR UPDATE/SKIP LOCKED/pg_advisory_xact_lock/jsonb/timestamptz/uuid/ON CONFLICT）✓、85 操作契约+33 页 PageSpec 冻结/前端零改动 ✓、替换 backend/（cmd/unigate + internal/，旧 Java 留 git 历史）✓、栈通用规约让位（openapi-unigate.json 为唯一权威）✓、pgx 例外 ✓、取代 ADR-028 后端选择且 ADR-046 证据作为历史保留 ✓。

## 五、S1~S8 阶段完成度预检

| # | 预检项 | 结果 | 说明 |
|---|---|---|---|
| S1 | 三文档全局检查 | PASS | BA/AA/FL 冻结、纯业务无栈措辞；architecture-choices 增量一致；43 FL/23 AA 叶映射机械复跑通过 |
| S2 | PRD 完备性检查 | PASS（沿用） | 冻结未改，沿用基线 |
| S3 | 领域模型分离 | PASS（沿用） | unigate-domain-model.md 未改，七聚合根分类沿用 |
| S4 | API 文档全局检查 | PASS（附 W-03） | 项目机械 36/37；唯一 FAIL（source-inventory）为工具硬编码的旧引擎缓存路径失效所致，非设计缺陷（见 W-03）；按 ADR-043 项目依据验收，不声称原版全条目通过 |
| S5 | API 变更摘要 | PASS | section-6 v0.3.1 行已写入，变更范围与实测 diff 一致 |
| S6 | PageSpec 输出校验 | PASS | 26/26 机械通过、2,253 字段绑定、33 页，冻结复跑 |
| S7 | 跨阶段一致性 | PASS（附 W-02） | 跨阶段机械 18/19；唯一 FAIL（active-chain）为工具硬编码 springboot-vue-web 所致，激活链已人工核实一致（见 W-02）；ADR-048 四处表述与 OAS 实录一致（第四节） |
| S8 | 枚举池无冲突 | PASS | 29 组池唯一、code/label 与 OAS 及池文件一致，机械复跑通过 |

## 六、机械检查证据（本次复跑，未写工具报告）

复跑命令（项目根，只读 stdout，未加 --write-report）：`python -I tools/design/check_contract.py`、`check_pages.py`、`check_gate1.py`。

| 工具 | 结果 | 说明 |
|---|---|---|
| check_contract.py | 36 通过 / 1 失败 | 失败项 source-inventory：SOURCE 硬编码 `C:\Users\zhang\.codex\plugins\cache\ai-project-engine-local\ai-project-engine\0.4.0\...\validation\SKILL.md`，该路径已不存在（实测），回收行数 0≠164。环境/工具过期，与本次设计增量无关 |
| check_pages.py | 26 通过 / 0 失败 | 33 页、85 操作、2,253 字段绑定、interactionConfirmed=true |
| check_gate1.py | 18 通过 / 1 失败 | 失败项 active-chain：断言硬编码 ACTIVE_STACKS=='springboot-vue-web'（第 44 行），迁移后合法值为 go-vue3。工具过期，非设计矛盾 |

工具只输出机械断言，最终判定由本报告结合人工核对作出（沿用基线口径）。未执行 HTTP/SQL/构建/真机调用。

## 七、不通过项（CRITICAL）

无。

## 八、建议项（WARN，非阻塞）

| # | 问题描述 | 影响 | 修复建议 |
|---|---|---|---|
| W-01 | INDEX modules[unigate].backend note 全长 126 字，超「≤120字」约定 6 字（若该约定按整条 note 计；追加的增量部分本身约 37 字，合规） | 仅索引文案约定，不影响状态正确性 | 压缩历史覆盖率数字（如改引 gate2 报告）使整条 ≤120 字；通过与否由主会话按约定口径裁定 |
| W-02 | tools/design/check_gate1.py 第 44 行 active-chain 断言硬编码 springboot-vue-web，go-vue3 激活下必然 FAIL（18/19） | 后续 Gate 复跑会持续出现假失败 | 将断言改为 go-vue3（或从 profile.yaml depends 动态读取期望值）后复跑；属工具代码修改，不在本门控只读范围 |
| W-03 | tools/design/check_contract.py SOURCE 指向已不存在的旧 codex 引擎 0.4.0 缓存路径，source-inventory FAIL（36/37） | 原版 CH/GL 源清单指纹无法回收；后续复跑持续假失败 | 将 SOURCE 指向当前引擎（1.0.4）validation SKILL.md 并按新版式重新基线化行数，或按 ADR-043 口径将该项显式标注为环境缺失豁免 |

## 九、观察项（INFO，不构成门控结论）

1. **工作区存在设计冻结范围之外的未提交代码/配置改动**：backend Java 密码重置功能（IdentityService.java +28 行、PasswordResetTest.java 新增、backend/README.md）、deploy/、docs/deployment.md、.github/、前端少量 UI 微调（App.vue/LoginView.vue/ScannerView.vue/ResultDisplay.vue/style.css）、.claude 栈重挂载、根 CLAUDE.md/README.md。这些属代码/交付层（Gate 2 范畴），不违反 Gate 1 设计冻结；但提请注意：Go 重写时需决定 Java 密码重置功能的移植或废弃；前端代码微调与迁移无关，交付时应与「前端零改动」基线核对归并。
2. 历史版本措辞差异（INDEX/CLAUDE.md 历史节 SpringBoot3.5.16 vs 旧 §1.2 候选 Spring Boot 4.1.1）为基线前已存在的候选/实现差异，本次迁移后已无实际影响，不需处理。
3. INDEX gates.gate1 目前仍指向 2026-10-21 报告；按引擎规则 gates 节应记录最近一次结果，**由主会话在本报告确认后更新**（子代理只读纪律，不代改）。

## 十、需主会话裁决/知悉事项（非阻塞）

1. W-02/W-03 两处检查工具修复的时机：立即修复，或留待 Go 实现阶段启动前修复（建议后者启动前必须完成，否则 Gate 复跑带假失败）。
2. W-01 的 ≤120 字口径（整条 note vs 追加增量）由主会话裁定；如需整条合规，按建议压缩。
3. 未提交 Java 密码重置功能在 Go 重写中的去留（移植/废弃），属后续后端阶段决策，此处仅登记知悉。

## 总结

**判定：WARN**

- CRITICAL 失败：0 项（契约冻结、冻结范围、跨阶段一致性全部通过）
- G1-01~G1-11：11/11 PASS（其中 5 项为冻结沿用口径）
- S1~S8：8/8 PASS（S4/S7 附工具侧 WARN 注记）
- WARN：3 项（W-01 note 长度微超、W-02/W-03 检查工具过期）
- 契约 sha256 实测：`cc185ea09d0f90ad1843e5716e7c5843566767ccce6e367dc4f26e667de6612b`（与冻结要求一致）

设计侧满足进入 Go 重写实现阶段的条件；主会话确认 W-01~W-03 处理口径后继续。本报告不授权启动业务代码、部署或外发；Go 实现与 Gate 2 重跑属后续阶段。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-24 | ADR-048 技术栈迁移增量 Gate 1：契约冻结硬校验、冻结范围 git diff 证据、G1/S 逐项、ADR-048 四处表述与 OpenAPI 实录核对、机械复跑（36/37、26/26、18/19，两项失败均为工具过期）；判定 WARN，3 项非阻塞建议 |