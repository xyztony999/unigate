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
    note: ADR-047重划分基线BA/AA v0.5与FL v1.1（2026-11-11，6业务+3平台模块、43项重挂）；ADR-048后端Go重写wire契约与基线逐结构一致（仅域标签/描述差异），业务内容不变；历史见CHANGELOG
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
    note: API v0.3按ADR-043项目依据完成；85操作、37项机械检查及16项人工核对通过，原版缺失记录保留；ADR-047 Go迁移v0.3.1，契约冻结不变
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
      requirement: {status: complete, note: ADR-041 已确认 FL/PRD/领域模型，D1/D2 检查通过；ADR-047 Go迁移设计增量，业务/领域模型冻结不变}
      api: {status: complete, note: ADR-042采用合同/ADR-043采用项目检查；v0.3完成37项机械及16项人工核对，见D3-completion.md；ADR-047 Go迁移，85操作契约字节级冻结，仅技术候选节更新v0.3.1}
      interaction: {status: complete, note: ADR-044采用七项；33页v1.0，复跑及人工核对见D4-completion.md}
      backend: {status: complete, note: ADR-048 Go全量重写完成(gin/slog/pgx单二进制,内嵌25表迁移)，142测试/41集成全过,-race净,service覆盖69.9%；Java遗留已移除}
      frontend: {status: complete, note: Vue 33页及85操作契约对齐，42测试/类型检查/构建通过；真实浏览器8项含18管理页与CSV全流程通过}
      validation: {status: complete, note: Go重写142测试/41集成+-race全过,G-01~G-08合规,契约冻结;浏览器/真机属部署验收边界}
    gates:
      gate1: {status: pass, note: 2026-11-11按ADR-047重划分新基线重跑PASS，见gate1-unigate-2026-11-11.md（机械37/26/19）；ADR-048 Go重写wire契约与基线一致，11-24增量门控见gate1-unigate-2026-11-24.md}
      gate2: {status: pass, note: 2026-11-24 ADR-048 Go重写Gate2重判WARN放行(0 CRITICAL,覆盖补齐69.9%,5bug修复)，见gate2-unigate-2026-11-24.md；部署/真机验收边界单列}
# ADR-046 pipeline本地代码交付完成；Gate2不表示公网部署或真实外部账号/设备已验收。
# ADR-048 Go迁移（2026-11-24）：建立在ADR-047重划分基线，wire契约/PageSpec/DDL冻结，前端零改动；Go全量重写+部署适配完成，Java遗留移除，Gate2重判WARN放行。
---
