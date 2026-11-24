# Vue3 前端规则（frontend-rules）

> **层级**：L2 技术栈 profile。激活含本栈的业务 profile 时由挂载机制生效。
> **分级**：🔴 CRITICAL = Gate 2 FAIL 必须修复；🟡 WARNING = 需说明理由或修复。每条附可 grep 检查方法。
> **组件库**：Naive UI（`naive-ui`）。📎 文中代码示例为通用参考，可按项目调整。

---

## 1. 编码规约

| 技术 | 版本 | 说明 |
|------|------|------|
| Vue | 3.4+ | 组合式 API，`<script setup lang="ts">` |
| TypeScript | 5.0+ | 全量类型约束，vue-tsc 门禁 |
| Naive UI | 2.x | 唯一组件库，按需自动导入（unplugin-auto-import + naive-ui resolver） |
| Pinia | 最新 | 状态管理（仅跨页面共享状态；页面内状态用 `ref/reactive`） |
| Vue Router | 4.x | 路由 |
| Vite | 5.x | 构建 |

### 1.1 Vue/TS 语法

- 🔴 **强制** `<script setup>` + 组合式 API；**禁止** Options API
  → `grep -rLn "script setup" --include="*.vue" src/`（命中即人工核对）
- 🔴 **禁止** `any`（确需时 `unknown` + 类型收窄；豁免须注释理由）
  → `grep -rn ": any\|as any" --include="*.ts" --include="*.vue" src/ | grep -v "// eslint-disable"`
- 🔴 template 内**禁止**复杂表达式（抽为 `computed` 或方法）
- 🟡 组件名多词（PascalCase）；props 用 `defineProps<T>()` 类型声明式；emit 事件名 camelCase
- 🟡 所有 API 响应/表单数据必须有 TS 类型（`src/types/` 集中定义），与后端 errcode 包络对齐

### 1.2 组件与导入

- 🔴 UI 组件统一 Naive UI（`n-*`）；**禁止**混入其他组件库 → `grep -rnE "from '(antd|element-ui|element-plus|ant-design-vue)'" src/`
- 🟡 **禁止**复制组件库实现到项目内；二次封装放 `src/components/` 并以 `App` 前缀命名（如 `AppDataTable`）
- 🟡 图标统一 `@vicons/*` 系列，与 Naive UI 的 `n-icon` 配合

---

## 2. 目录结构

```
{FRONTEND_ROOT}/
├─ src/
│  ├─ api/                    # 接口层：{module}.ts（唯一 fetch/axios 出口）
│  ├─ types/                  # API 类型定义（与后端包络/DTO 对齐）
│  ├─ stores/                 # Pinia store（跨页面共享状态才建）
│  ├─ views/{module}/         # 页面：index.vue + 局部组件
│  ├─ components/             # 跨页面复用组件（App 前缀）
│  ├─ composables/            # 组合式函数（use 前缀）
│  ├─ router/                 # 路由表
│  └─ App.vue / main.ts
└─ vite.config.ts
```

- 🔴 **禁止**在 views/components 里直接 `fetch`/`axios`——一切请求走 `src/api/` → `grep -rnE "(axios|fetch)\(" --include="*.vue" src/views src/components`
- 🟡 **禁止**跨模块 import 其他模块 views 内部文件；公共逻辑下沉 composables/components
- 🟡 路由懒加载：页面组件 `() => import('@/views/...')`

---

## 3. API 层约定

- `src/api/{module}.ts`：按模块导出函数，入参/返回值均为 TS 类型；错误在统一拦截器处理（包络 `code !== 0` → message + reject）
- 🔴 分页/列表响应字段与后端契约一致（`records` + `pageNum/pageSize`），前端不得私自改名映射两套字段
- 🟡 请求函数命名动词结尾：`listXxx / getXxx / createXxx / updateXxx / deleteXxx`
- 🟡 WebSocket 封装为 composable（`useXxxSocket`）：断线重连、心跳、会话过期登出，组件不裸连

---

## 4. UI/UX 规约（Naive UI）

### 4.1 布局

- 布局壳用 `n-layout` + `n-layout-sider`（侧边菜单）+ `n-layout-header`；内容区留白 16-24px
- 🟡 支持暗色模式：色值用主题变量（`useThemeVars()`/`n-config-provider` theme），**禁止**大面积硬编码十六进制背景色
  → `grep -rnE "#[0-9a-fA-F]{6}" --include="*.vue" src/ | grep -vE "transparent|border"`（命中人工核对）
- 🟡 卡片容器 `n-card`；页面结构：查询区卡片 + 内容卡片，卡片间 gap 12-16px

### 4.2 列表页（n-data-table）

- 列定义集中为 `columns: DataTableColumns<T>`（类型化）；render 函数保持纯函数
- 🔴 状态/枚举列用 `n-tag` 渲染并带语义色；绑定编码字段，展示文案由映射表（types 中）转换，**禁止**硬编码文案在 render 里 → 人工核对 render 函数
- 🟡 列顺序：序号 → 业务主键/名称 → 分类 → 状态 → 时间 → 操作；操作列 `fixed: 'right'`，长列表主键列 `fixed: 'left'`
- 🟡 操作列 ≤3 个内联按钮，超出折叠进 `n-dropdown`；危险操作（删除/重启）用 `error` 类型 + `n-modal` 确认（dialog.warning）
- 🟡 分页受控：`pagination` 用 `PaginationProps`，remote 模式（后端分页）必须 `remote: true`

### 4.3 表单（n-form）

- 🔴 表单校验用 `n-form` rules（类型化 `FormRules`），提交前 `formRef.validate()`；**禁止**只靠后端校验
- 🟡 字段布局 `label-placement="top"` + 响应式栅格（`n-grid` `cols="1 s:2 m:3"`）；长文本/上传占满整行
- 🟡 弹窗表单用 `n-modal`（`:mask-closable="false"`），底部：取消（default）+ 确定（primary，提交中 loading）

### 4.4 反馈

- 消息统一 `useMessage()`/`useDialog()`（在 `n-message-provider` 内）；**禁止** `alert/confirm` 原生弹窗
  → `grep -rnE "(^|\W)(alert|confirm)\(" --include="*.vue" --include="*.ts" src/`
- 🟡 loading：按钮 `:loading`；列表 `n-spin`；空态 `n-empty`；错误态可重试

### 4.5 面板类专项

- 图表：echarts 封装为组件（resize 监听 + 主题联动暗色）；数据定时轮询时组件卸载必须清定时器
  → `grep -rn "setInterval\|setTimeout" --include="*.vue" src/ | grep -v onUnmounted`（人工核对清理）
- 终端：xterm.js 封装 `AppTerminal`；resize 用 FitAddon；断连展示重连 UI
- 代码/配置编辑：monaco-editor 按需注册语言；保存走 `Ctrl+S` + 显式按钮双通道
- 实时数据（监控）：轮询间隔常量集中管理；页面不可见时暂停（`document.visibilityState`）

### 4.6 用户可见文案（去 AI 味）

> 机制源：engine-rules.md **ER-04**。范围 = `useMessage()` / `useDialog()` / 校验提示 / placeholder / `n-empty` 空状态 / 按钮文字，及后端透传展示的错误消息。代码注释不在范围。

三类禁令（出现即 Gate 2 不通过）：

| 禁令 | 反例 | 正例 |
|------|------|------|
| emoji/装饰符与感叹号堆叠 | `message.success('保存成功！🎉')` | `message.success('已保存')` |
| 空洞夸赞 / 同义反复 | `'操作成功，太棒了！'`；`'请确认您是否确定要删除'` | `'已保存'`；`'确认删除？'` |
| 系统腔 / 翻译腔 | `'请求失败，发生错误'`；`'数据加载完成'` | `'保存失败：网络超时，请重试'`；`'已加载 12 条'` |

自查：grep 文案字段中 emoji 与 `！`/`!` → 0 命中；逐条读——说清**结果/原因/下一步**了吗？

---

## 5. 构建与门禁

| 门禁 | 命令 | 说明 |
|------|------|------|
| 类型检查 | `vue-tsc --noEmit`（或 `npm run typecheck`） | Gate 2 必过 |
| 构建 | `vite build` | 产物输出 `{BACKEND_ROOT}/web/dist` 或复制过去（供 go:embed） |
| 单测 | `vitest run` | composables/工具函数必须有测试 |
| Lint | eslint + prettier | CI 必过 |

- 开发联调：Vite dev server proxy 到后端（`vite.config.ts` server.proxy），**禁止**前端硬编码后端地址
- 🔴 typecheck 不绿不得报告完成

---

## 6. Gate 2 扫描清单（CRITICAL 汇总）

| # | 检查 | grep 方法 | 判定 |
|---|------|----------|------|
| F-01 | 无 Options API | `grep -rLn "script setup" --include="*.vue" src/` | 0 命中 |
| F-02 | 无裸 any | `grep -rnE "(:\s*any\|as any)" --include="*.ts" --include="*.vue" src/` | 0 命中（豁免有注释） |
| F-03 | 请求只走 api 层 | `grep -rnE "(axios\|fetch)\(" --include="*.vue" src/views src/components` | 0 命中 |
| F-04 | 无混入其他组件库 | `grep -rnE "from '(antd\|element\|ant-design-vue)'" src/` | 0 命中 |
| F-05 | 无原生弹窗 | `grep -rnE "(^|\W)(alert\|confirm)\(" --include="*.vue" src/` | 0 命中 |
| F-06 | typecheck + build 全绿 | `vue-tsc --noEmit && vite build` | 全绿 |
| F-07 | 用户可见文案无 AI 味（无 emoji/感叹号堆叠，见 §4.6） | `grep -rnE "success\(|error\(|warning\(|placeholder[^,]*[！!]|🎉|✨|太棒|恭喜" --include="*.vue" src/ | grep -v 注明` + 逐条人工读 | 0 命中 |
