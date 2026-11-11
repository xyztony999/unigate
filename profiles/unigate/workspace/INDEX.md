---
pipeline:
  - id: discovery
    title: 需求澄清
    path: 05-reports/D0-unigate.md
    status: complete
    next: ADR-046 pipeline本地实现与Gate2已完成，运行见根README及Gate2报告；真实外部账号/安卓设备另行启用验收
  - id: architecture
    title: 架构设计 D1
    path: 00-architecture
    status: complete
    depends_on: [discovery]
    note: ADR-047重划分后基线为BA/AA v0.5与FL v1.1（2026-11-11，6业务+3平台模块、43项重挂）；原v0.4/v1.0确认历史见CHANGELOG，Gate 1待按新基线重跑
  - id: prd
    title: 产品需求 D2
    path: 02-detailed/prd
    status: complete
    depends_on: [architecture]
    note: PRD仍为ADR-041确认版；领域模型已升级v2.0（8上下文+平台内核、8聚合根，ADR-047）
  - id: api
    title: API 设计 D3
    path: 02-detailed/api
    status: complete
    depends_on: [prd]
    note: API v0.3按ADR-043项目依据完成；85操作、37项机械检查及16项人工核对通过，原版缺失记录保留
  - id: interaction
    title: 交互设计 D4
    path: 02-detailed/interaction
    status: complete
    depends_on: [api]
    note: ADR-044确认七项；ADR-045确认客户版式及横屏kiosk，均v1.0归档，design结束
reference:
  - title: 领域架构重划分提案（待评审）
    path: 05-reports/domain-redivision-proposal.md
  - title: 检票台与kiosk版式补充
    path: 02-detailed/interaction/layouts/README.md
  - title: 需求基线
    path: ../../../docs/00-intake.md
  - title: 已确认决策
    path: ../knowledge/design-decisions/index.md
modules:
  - name: unigate
    domain: core
    stages:
      requirement: {status: complete, note: ADR-041 已确认 FL/PRD/领域模型，D1/D2 检查通过}
      api: {status: complete, note: ADR-042采用合同/ADR-043采用项目检查；v0.3完成37项机械及16项人工核对，见D3-completion.md}
      interaction: {status: complete, note: ADR-044采用七项；33页v1.0，复跑及人工核对见D4-completion.md}
      backend: {status: complete, note: Java21/SpringBoot3.5.16及25表迁移完成，clean verify 47项无跳过通过；Service87.59%/Controller78.48%}
      frontend: {status: complete, note: Vue 33页及85操作契约对齐，42测试/类型检查/构建通过；真实浏览器8项含18管理页与CSV全流程通过}
      validation: {status: complete, note: 16项跨栈检查及最终真实浏览器8组通过，Gate2人工复核已归档}
    gates:
      gate1: {status: pass, note: 2026-11-11按ADR-047新基线重跑PASS，见05-reports/gate1-unigate-2026-11-11.md（取代2026-10-21版）；机械37/26/19重跑通过}
      gate2: {status: pass, note: 2026-10-21本地代码交付通过，见05-reports/gate2-unigate-2026-10-21.md；代码未动结论维持，D-5b重构后按新基线复查}
# ADR-046 pipeline本地代码交付完成；Gate2不表示公网部署或真实外部账号/设备已验收。
---
