# UniGate 架构变更记录

## 2026-10-20

- business-arch.md v0.1：初始 BA 提案，4 个 L2、8 个 L3、17 个 L4。
- business-arch.md v0.2：用户本批 B1 确认层级及范围；同条公网优先补充优先于旧的始终本地裁决，交接描述同步更新。
- application-arch.md v0.1：生成候选 AA，8 个业务模块映射全部 17 个 BA 活动，另有 3 个平台模块/6 个技术子模块；基础数据域本期无独立维护模块。DDD 归类、角色及调用待用户确认，未生成 FL 或宣布 D1 完成。
- 公网优先接管技术提案及当前选择分别见 ../05-reports/cloud-first-checkin-design.md、../05-reports/D1-failover-options.md。未将联网优先级调整推导为重新开放公众报名等现场变更。
- business-arch.md v0.3、application-arch.md v0.2：按用户明确选择的 ADR-039 裁剪为在线首版，现场准备与控制/暂停恢复替代云地接管，保留 4/8/17 业务映射。浏览器连公网服务，断网暂停，原恢复 B 不适用；AA 候选仍待确认，D0 业务澄清完成。
- application-arch.md v0.3：ADR-040 采用在线 AA，补齐已确认的检票员/本组原操作结果查询标注；权限受范围约束。
- function-list.md v0.1：展开五角色 43 项功能，覆盖全部 23 个 AA 子模块及勾选角色，菜单/功能待集中确认。
- 产品需求与领域模型预备稿位于 ../02-detailed/prd/，与 FL 同批审阅；不能据文件生成标记 D1/D2 完成，Gate 1 未执行。
- ADR-041：用户选择1确认五角色43项FL、PRD/领域模型及选中行整批原子导入。BA/AA v0.4统一模板表格，FL/PRD/模型v1.0归档；映射、权限、需求完备性与领域分离检查通过，见../05-reports/D1-D2-completion.md，D1/D2完成。

## 2026-10-21

- D3 v0.1新增API/OpenAPI/数据与事务设计，枚举及选择池同步；81个操作覆盖43项FL/23个AA子模块。21项项目补充静态检查通过；合同尚待审阅，插件原版检查资产缺失，D3保持partial。未进入D4或执行Gate1，无业务实现。
- ADR-042—044：采用API、项目验收依据及D4七项交互，API0.3.0/33页PageSpec v1.0归档；独立Gate1 PASS，见../05-reports/gate1-unigate-2026-10-21.md。本次仅更新阶段引用与版本来源标注，无模块、业务层级或权限扩大。
