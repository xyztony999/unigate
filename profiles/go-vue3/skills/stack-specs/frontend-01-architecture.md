# frontend-01 — 前端架构（Vue3 + Naive UI）

> **层级**：L2 技术栈 profile skills。配套规则：`rules/frontend-rules.md` §2§3。

## 1. 目录结构

```
{FRONTEND_ROOT}/
├─ src/
│  ├─ api/                    # 接口层（唯一请求出口）：{module}.ts
│  ├─ types/                  # 类型定义：api.ts（包络/分页）+ {module}.ts（业务类型）
│  ├─ stores/                 # Pinia（跨页面共享状态才建）
│  ├─ views/{module}/         # 页面：index.vue + 局部组件
│  ├─ components/             # 跨页面复用组件（App 前缀：AppDataTable/AppTerminal…）
│  ├─ composables/            # use 前缀组合式函数（useMessage 之外的业务复用）
│  ├─ router/index.ts         # 路由表（懒加载）
│  ├─ App.vue
│  └─ main.ts                 # n-config-provider + 各 Provider 挂载
└─ vite.config.ts             # proxy 指向后端；naive-ui 按需自动导入
```

## 2. 请求与类型（与后端包络对齐）

```typescript
// src/types/api.ts — 与 backend-02-api-conventions 对齐
export interface Envelope<T> { code: number; msg: string; data: T }
export interface PageResult<T> { records: T[]; total: number }
export interface PageQuery { pageNum: number; pageSize: number }
```

- `src/api/http.ts`：统一实例（拦截器：包络 `code !== 0` → message + reject；未认证 → 跳登录）
- `src/api/{module}.ts`：按模块导出类型化函数，**禁止** views 直接 fetch/axios
- 枚举映射表集中在 `src/types/{module}.ts`：`export const SITE_STATUS: Record<string, { label: string; tag: TagColor }>`

## 3. 状态管理边界

- 页面内状态：`ref/reactive`，**不进** Pinia
- 跨页面/全局（登录态、主题、布局折叠）：Pinia store（`defineStore` setup 风格）
- 服务端数据不进 Pinia（跟随请求生命周期）；确需缓存的用 store 存"最后快照 + 失效动作"

## 4. 路由

```typescript
{ path: '/sites', name: 'sites', component: () => import('@/views/sites/index.vue'), meta: { title: '网站管理' } }
```

- 全部懒加载；`meta.title` 供面包屑；需要权限的路由挂 `meta.roles`，路由守卫统一校验
- 布局路由嵌套：`/` → Layout（sider+header）→ 业务页面

## 5. Naive UI 接入

- 按需自动导入（unplugin-auto-import + unplugin-vue-components NaiveUiResolver）；**禁止**全量 `import naive from 'naive-ui'; app.use(naive)` 后随处散用（保留 message/dialog 等 Provider 用法）
- 反馈组件（message/dialog/notification/loadingbar）必须在 `main.ts` 挂对应 Provider，页面内 `useMessage()` 等取用
- 主题：`n-config-provider` + `darkTheme` 切换；面板类应用默认跟随系统 `prefers-color-scheme`
- 语言：`n-config-provider :locale` 集中配置（zhCN/enUS 可切换）
