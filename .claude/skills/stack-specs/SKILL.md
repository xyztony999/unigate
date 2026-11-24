---
name: stack-specs
description: Go + Vue3 全栈编码规约与生成模板统一技能（后端 Go/gin + 前端 Vue3/Naive UI，随栈激活挂载）
---

# Go + Vue3 全栈 — 代码规格（stack-specs）

> **层级**：L2 技术栈 profile skills，随栈激活挂载。配套规则正文：本栈 `rules/backend-rules.md` 与 `rules/frontend-rules.md`（Gate 2 等价规则集，激活后分别位于 `.claude/rules/`）。
> **性质**：强制规约，所有使用本栈的代码必须遵循。文中 `{service}`/`{module}` 为占位符（业务 profile 替换）；📎 标记处为通用示例。

**使用方式**：生成/审查代码时，根据当前任务类型引用对应专题文件。

## 文件清单

### 后端（Go / gin）

| 文件 | 主题 |
|------|------|
| [backend-01-architecture.md](backend-01-architecture.md) | 分层架构：cmd/internal 布局、依赖方向、错误处理、context 传播、优雅退出 |
| [backend-02-api-conventions.md](backend-02-api-conventions.md) | API 设计规约：包络结构、错误码、路径命名、分页约定、WS 端点（design skill D3 引用 {stack.api_spec}） |
| [backend-03-patterns.md](backend-03-patterns.md) | 常用模式：配置加载、DDL 与类型映射、外部命令安全调用、后台任务编排、测试模板 |
| [backend-04-validation-checklist.md](backend-04-validation-checklist.md) | 后端合规验证清单（CRITICAL 扫描项，与 rules §7 对应） |

### 前端（Vue3 / Naive UI）

| 文件 | 主题 |
|------|------|
| [frontend-01-architecture.md](frontend-01-architecture.md) | 前端架构：目录结构、路由、状态、api 层、类型对齐 |
| [frontend-02-code-templates.md](frontend-02-code-templates.md) | 代码模板：列表页/表单弹窗/详情抽屉/图表卡片（Naive UI） |
| [frontend-03-ui-ux-specs.md](frontend-03-ui-ux-specs.md) | UI/UX 规约：布局、表格列、表单、反馈、暗色主题、面板类专项（终端/图表/编辑器） |

## 快速定位

- 生成 store/service → [backend-01-architecture.md](backend-01-architecture.md)
- 设计/生成 REST 接口 → [backend-02-api-conventions.md](backend-02-api-conventions.md)
- 写 DDL / 调外部命令 → [backend-03-patterns.md](backend-03-patterns.md)
- 生成列表页 → [frontend-02-code-templates.md](frontend-02-code-templates.md) §2
- 生成表单弹窗 → [frontend-02-code-templates.md](frontend-02-code-templates.md) §3
- 图表/终端组件 → [frontend-03-ui-ux-specs.md](frontend-03-ui-ux-specs.md) §5
- 收尾合规检查 → backend-04 / rules §6§7
