# frontend-03 — UI/UX 规约（Naive UI / 面板类）

> **层级**：L2 技术栈 profile skills。配套规则：`rules/frontend-rules.md` §4。

## 1. 布局与主题

- 布局壳：`n-layout` + `n-layout-sider`（可折叠侧边菜单）+ `n-layout-header`；内容区 padding 16-24px，卡片间 gap 12-16px
- 页面结构惯例：查询区卡片 + 内容卡片（列表/详情）；控制台页为卡片网格（`n-grid` 响应式）
- 主题：`n-config-provider` + `darkTheme`；**禁止**大面积硬编码背景/文字色（用主题变量 `useThemeVars()`）；品牌色经 `theme-overrides` 集中配置
- 面板类应用暗色为一等公民：所有新页面必须在明暗两种主题下各验收一次

## 2. 表格列规范（n-data-table）

| 项 | 规范 |
|----|------|
| 列顺序 | 序号 → 业务主键/名称 → 分类 → 状态 → 时间 → 操作 |
| 固定列 | 主键列 `fixed: 'left'`（长列表）；操作列 `fixed: 'right'`；宽表 `:scroll-x` |
| 宽度 | 编码 120 / 名称 150-200 / 时间 170 / 状态 100 / 操作 ≤3 按钮 160，更多折叠 180 |
| 长文本 | `ellipsis: { tooltip: true }` |
| 状态列 | NTag 语义色（success/warning/error/info），文案来自映射表 |

## 3. 表单规范（n-form）

- `label-placement="top"` + `n-grid` 响应式（`cols="1 s:2 m:3"`）；textarea/upload 占满整行
- rules 类型化（`FormRules`）+ 提交前 `validate()`；服务端校验错误回填到对应字段（`setFieldValue` 或字段级错误展示）
- 弹窗表单 `:mask-closable="false"`；底部按钮：取消（default）+ 确定（primary，提交中 loading 且禁重复提交）
- 表单项数量多时用 `n-card` 分组 + 锚点（超过 3 组必须有分组导航）

## 4. 反馈与状态

- 消息：`useMessage()`（成功/失败即时反馈）；确认：`useDialog()` warning（危险操作必须确认，文案含操作对象名）
- loading 层级：按钮级 `:loading` → 区块级 `n-spin` → 页面级骨架 `n-skeleton`
- 空态 `n-empty`（带引导动作）；错误态可重试（不裸白屏）
- 危险操作分级：可逆→普通按钮；不可逆→ error 色 + dialog 确认 + 输入确认名（高危）

## 5. 面板类专项（终端/监控/编辑器/文件）

### 5.1 实时监控

- 轮询间隔常量集中管理（如 `POLL_INTERVAL = 5000`）；页面不可见时暂停（visibilitychange）
- 🔴 定时器/订阅/图表实例 onUnmounted 全部清理；WS 断连自动重连（指数退避，上限次数）
- 数据断流展示"最后更新时间"而非静默停滞；数值格式化集中（bytes 百分比速率）

### 5.2 Web 终端

- 封装 `AppTerminal`（xterm.js + FitAddon + WebLinksAddon）：连接、断连重连 UI、resize 同步、主题联动暗色
- 输入回显由 PTY 负责，前端不本地拼接命令回显；粘贴大文本提示确认
- 终端缓冲上限（scrollback）防止内存膨胀

### 5.3 代码/配置编辑器

- monaco 按需注册语言；只读/编辑模式 props 区分；保存双通道（Ctrl+S + 按钮）；未保存离开（路由/弹窗）需确认
- 大文件（>1MB）禁用语法高亮并提示只读

### 5.4 文件管理器

- 路径面包屑可点击逐级导航；危险目录（/ 等）删除操作强制输入确认
- 上传进度条（分块上传显示合并进度）；失败可重试
- 文件类型图标映射表集中；权限展示用等宽字体

### 5.5 操作日志/审计列表

- 时间列 RFC3339 解析后本地时区展示（`YYYY-MM-DD HH:mm:ss`）；操作人/动作/对象/结果四要素齐全
- 结果列 error 失败红色 NTag；详情抽屉展示请求参数 diff

## 6. 交互一致性

- 同类操作全站同色同位置（主操作 primary 右一；取消永远在左侧）
- 加载态、空态、错误态三态每个列表页必备
- 键盘：Esc 关闭弹窗（Naive 默认）；表单 Enter 提交（单字段搜索场景）
- 所有破坏性接口调用前必须有确认；所有写操作完成后刷新对应列表/详情数据
