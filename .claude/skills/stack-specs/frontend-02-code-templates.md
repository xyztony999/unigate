# frontend-02 — 代码模板（Naive UI）

> **层级**：L2 技术栈 profile skills。📎 模板为通用参考，按 PageSpec 实际字段替换。

## 1. api 层模板（src/api/{module}.ts）

```typescript
import { http } from './http'
import type { Envelope, PageQuery, PageResult } from '@/types/api'
import type { Site, SiteInput } from '@/types/site'

export const listSites = (q: PageQuery & { keyword?: string }) =>
  http.get<Envelope<PageResult<Site>>>('/api/v1/sites', { params: q })
export const getSite = (id: string) => http.get<Envelope<Site>>(`/api/v1/sites/${id}`)
export const createSite = (data: SiteInput) => http.post<Envelope<Site>>('/api/v1/sites', data)
export const deleteSite = (id: string) => http.delete<Envelope<void>>(`/api/v1/sites/${id}`)
export const reloadSite = (id: string) => http.post<Envelope<void>>(`/api/v1/sites/${id}/reload`)
```

## 2. 列表页模板（n-data-table，remote 分页）

```vue
<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NDropdown, NTag, type DataTableColumns } from 'naive-ui'
import { listSites, deleteSite } from '@/api/site'
import { SITE_STATUS } from '@/types/site'
import type { Site } from '@/types/site'

const records = ref<Site[]>([])
const total = ref(0)
const loading = ref(false)
const query = ref({ pageNum: 1, pageSize: 20, keyword: '' })

const columns: DataTableColumns<Site> = [
  { title: '名称', key: 'name', width: 200, ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 100,
    render: (row) => { const s = SITE_STATUS[row.status]; return h(NTag, { type: s.tag }, { default: () => s.label }) } },
  { title: '更新时间', key: 'updatedAt', width: 170 },
  { title: '操作', key: 'actions', width: 180, fixed: 'right',
    render: (row) => h('div', { class: 'flex gap-2' }, [
      h(NButton, { size: 'small', onClick: () => onEdit(row) }, { default: () => '编辑' }),
      h(NDropdown, { options: rowActions, onSelect: (k) => onRowAction(k, row) },
        { default: () => h(NButton, { size: 'small' }, { default: () => '更多' }) }),
    ]) },
]

async function load() { loading.value = true; try { const { data } = await listSites(query.value); records.value = data.data.records; total.value = data.data.total } finally { loading.value = false } }
async function onDelete(row: Site) { await deleteSite(row.id); load() }
function onEdit(row: Site) { /* 打开表单弹窗 */ }
function onRowAction(key: string, row: Site) { if (key === 'delete') dialog.warning({ title: '确认删除', content: `删除 ${row.name}？`, positiveText: '删除', onPositiveClick: () => onDelete(row) }) }

onMounted(load)
</script>

<template>
  <div class="flex flex-col gap-4 p-4">
    <n-card> <!-- 查询区：n-form 内联 + 查询/重置按钮 --> </n-card>
    <n-card>
      <n-data-table :columns="columns" :data="records" :loading="loading" remote
        :pagination="{ page: query.pageNum, pageSize: query.pageSize, itemCount: total, onChange: (p) => { query.pageNum = p; load() } }"
        :scroll-x="900" />
    </n-card>
  </div>
</template>
```

**要点**：columns 类型化；状态列映射表 + NTag；危险操作进 NDropdown + dialog.warning 确认；remote 分页受控。

## 3. 表单弹窗模板（n-modal + n-form）

```vue
<script setup lang="ts">
import { ref } from 'vue'
import type { FormInst, FormRules } from 'naive-ui'
import { createSite } from '@/api/site'
import type { SiteInput } from '@/types/site'

const visible = defineModel<boolean>('visible', { default: false })
const saving = ref(false)
const formRef = ref<FormInst | null>(null)
const form = ref<SiteInput>({ name: '', root: '' })

const rules: FormRules = {
  name: { required: true, message: '请输入域名', trigger: 'blur' },
  root: { required: true, message: '请输入根目录', trigger: 'blur' },
}

async function onSubmit() {
  await formRef.value?.validate()               // 🔴 提交前必须 validate
  saving.value = true
  try { await createSite(form.value); visible.value = false }
  finally { saving.value = false }
}
</script>

<template>
  <n-modal v-model:show="visible" preset="card" title="新建站点" style="width: 640px"
    :mask-closable="false">
    <n-form ref="formRef" :model="form" :rules="rules" label-placement="top">
      <n-grid :cols="2" :x-gap="16">
        <n-form-item-gi :span="1" label="域名" path="name"><n-input v-model:value="form.name" /></n-form-item-gi>
        <n-form-item-gi :span="1" label="根目录" path="root"><n-input v-model:value="form.root" /></n-form-item-gi>
      </n-grid>
    </n-form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <n-button @click="visible = false">取消</n-button>
        <n-button type="primary" :loading="saving" @click="onSubmit">确定</n-button>
      </div>
    </template>
  </n-modal>
</template>
```

## 4. 详情抽屉模板（n-drawer + n-descriptions）

- `n-drawer` + `n-descriptions` 分组展示；空值统一显示 `-`；字典字段用 NTag；底部关闭按钮

## 5. 图表卡片模板（echarts 封装）

```vue
<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { darkThemeKey } from '@/composables/useTheme'

const el = ref<HTMLElement | null>(null)
let chart: echarts.ECharts | null = null
let timer: ReturnType<typeof setInterval> | null = null

function render() { chart?.setOption(buildOption()) }          // buildOption 按数据构造

onMounted(() => {
  chart = echarts.init(el.value!, darkThemeKey.value ? 'dark' : undefined)
  render()
  timer = setInterval(refresh, 5000)                            // 轮询
  window.addEventListener('resize', render)
})
onUnmounted(() => {                                             // 🔴 全部清理
  if (timer) clearInterval(timer)
  window.removeEventListener('resize', render)
  chart?.dispose()
})
</script>
<template><div ref="el" class="h-64" /></template>
```

## 6. 路由注册模板

```typescript
// src/router/index.ts 追加
{ path: '/sites', name: 'sites', component: () => import('@/views/sites/index.vue'), meta: { title: '网站管理' } }
```
