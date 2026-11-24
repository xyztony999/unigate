import { beforeEach,afterEach,describe,it,expect,vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount,flushPromises } from '@vue/test-utils'
import { createMemoryHistory,createRouter } from 'vue-router'
import WorkspaceView from '../src/views/WorkspaceView.vue'
import { contract,operations } from '../src/lib/contract'
import { identity,getPending } from '../src/lib/api'
vi.mock('../src/lib/confirm',()=>({askConfirm:vi.fn(async()=>true)}))
let finish:(response:Response)=>void=()=>{}
let mutationOp=''
const ok=(data:any)=>new Response(JSON.stringify({code:'OK',data,requestId:'r'}),{status:200,headers:{'Content-Type':'application/json'}})
beforeEach(()=>{
 sessionStorage.clear();identity.staff={accountId:'staff',username:'operator',displayName:'工作人员',roles:[{code:'ORGANIZER',label:'负责人'}],csrfToken:'csrf',expiresAt:'2026-12-01T00:00:00Z',mustChangePassword:false};identity.staffCsrf='csrf'
 vi.stubGlobal('fetch',vi.fn(async(url:string)=>{
  if(url.startsWith(operations.EventDetail.path))return ok({eventId:'event',title:'家长见面会',status:{code:'READY',label:'已发布'},version:1})
  if(url.startsWith(operations.PolicyGet.path))return ok({fields:{name:{enabled:true,required:true}},approvalMode:{code:'REVIEW',label:'审核'},version:1,retentionDays:30})
  if(url===operations.EventList.path)return ok({items:[],total:0,page:1,pageSize:20})
  if(url===operations[mutationOp].path)return new Promise<Response>(resolve=>{finish=resolve})
  throw new Error('unexpected request '+url)
 }))
})
afterEach(()=>{vi.unstubAllGlobals()})
async function workspace(pageId:string,path?:string){const page=contract.pages.find(p=>p.id===pageId)!;const blank=defineComponent({template:'<p>目标页面</p>'});const router=createRouter({history:createMemoryHistory(),routes:[{path:page.route,name:page.id,component:WorkspaceView,meta:{page}},{path:'/login',component:blank},{path:'/manage/events/:eventId/settings',component:blank},{path:'/other',component:blank}]});await router.push(path||page.route);await router.isReady();const wrapper=mount(defineComponent({template:'<RouterView />'}),{global:{plugins:[router],stubs:{teleport:true}}});await flushPromises();return{wrapper,router}}
describe('已知成功后的工作区导航',()=>{
 it('改密响应前阻止离开；成功后清会话并跳到重新登录',async()=>{
  mutationOp='PasswordChange';const{wrapper,router}=await workspace('S02');const fields=wrapper.findAll('input[type=password]');expect(fields).toHaveLength(2)
  await fields[0].setValue('old-password-long');await wrapper.findAll('input[type=password]')[1].setValue('new-password-long');await wrapper.find('form').trigger('submit');await flushPromises()
  expect(getPending()?.action).toBe('PasswordChange');await router.push('/other');expect(router.currentRoute.value.path).toBe('/account/password')
  finish(ok({}));await flushPromises();expect(router.currentRoute.value.path).toBe('/login');expect(router.currentRoute.value.query.passwordChanged).toBe('1');expect(identity.staff).toBeNull();expect(getPending()).toBeNull();wrapper.unmount()
 })
 it('创建活动响应前不能离开；成功后跳转新活动规则页',async()=>{
  mutationOp='EventCreate';const{wrapper,router}=await workspace('O01');const create=wrapper.findAll('button').find(b=>b.text()==='创建活动草稿')!;await create.trigger('click');await flushPromises()
  const fill=async(label:string,value:string)=>{const field=wrapper.findAll('label.field').find(f=>f.text().startsWith(label));expect(field).toBeDefined();await field!.find('input').setValue(value)}
  await fill('活动名称','家长见面会');await fill('开始时间','2026-11-01T10:00');await fill('结束时间','2026-11-01T12:00');await fill('报名开始','2026-10-22T10:00');await fill('报名截止','2026-10-31T12:00');await fill('人数容量','100');await wrapper.find('form').trigger('submit');await flushPromises()
  expect(getPending()?.action).toBe('EventCreate');await router.push('/other');expect(router.currentRoute.value.path).toBe('/manage/events')
  finish(ok({eventId:'created-event',title:'家长见面会',status:{code:'DRAFT',label:'草稿'},version:0}));await flushPromises();expect(router.currentRoute.value.path).toBe('/manage/events/created-event/settings');expect(getPending()).toBeNull();wrapper.unmount()
 })
 it('报名规则初始加载和刷新只发送正式字段，不给非列表接口附加分页',async()=>{
  mutationOp='PolicyUpdate';const{wrapper}=await workspace('O02','/manage/events/event/settings');expect(wrapper.findAll('.alert')).toHaveLength(0)
  const readUrls=()=>((fetch as any).mock.calls as [string,RequestInit][]).map(([url])=>url).filter(url=>url.startsWith(operations.PolicyGet.path)||url.startsWith(operations.EventDetail.path))
  expect(readUrls().length).toBeGreaterThan(1)
  for(const url of readUrls())expect([...new URL(url,'http://localhost').searchParams.keys()]).toEqual(['eventId'])
  await wrapper.findAll('button').find(b=>b.text()==='刷新')!.trigger('click');await flushPromises()
  for(const url of readUrls())expect([...new URL(url,'http://localhost').searchParams.keys()]).toEqual(['eventId'])
  wrapper.unmount()
 })
 it('名单导入无批次时从上传开始，不请求批次UUID或未定义分页参数',async()=>{
  mutationOp='BatchUpload';const{wrapper}=await workspace('O07','/manage/events/event/imports');expect(wrapper.find('[role=dialog]').exists()).toBe(false)
  expect(wrapper.text()).toContain('上传名单后');expect(wrapper.findAll('.alert')).toHaveLength(0)
  expect(((fetch as any).mock.calls as [string,RequestInit][]).some(([url])=>url.startsWith(operations.BatchRows.path))).toBe(false)
  await wrapper.findAll('button').find(b=>b.text()==='上传名单')!.trigger('click');await flushPromises();expect(wrapper.find('input[type=file]').exists()).toBe(true)
  expect(wrapper.findAll('label.field').some(f=>f.text().includes('批次'))).toBe(false);wrapper.unmount()
 })
 it('未知响应继续阻止离开，不能套用成功导航例外',async()=>{
  mutationOp='PasswordChange';const{wrapper,router}=await workspace('S02');const fields=wrapper.findAll('input[type=password]');await fields[0].setValue('old-password-long');await wrapper.findAll('input[type=password]')[1].setValue('new-password-long');await wrapper.find('form').trigger('submit');await flushPromises()
  finish(new Response(JSON.stringify({code:'SERVER_ERROR',message:'结果待确认'}),{status:503}));await flushPromises();await router.push('/other');expect(router.currentRoute.value.path).toBe('/account/password');expect(getPending()?.action).toBe('PasswordChange');expect(wrapper.findAll('button').some(b=>b.text()==='查询原操作结果')).toBe(false)
  const anchor=wrapper.find('a[target=_blank]');expect(anchor.attributes('href')).toBe('/login');expect(anchor.attributes('rel')).toContain('noopener')
  const originalId=getPending()!.operationId;await wrapper.findAll('button').find(b=>b.text()==='查看原操作')!.trigger('click');await flushPromises();await wrapper.findAll('button').find(b=>b.text()==='重试原操作')!.trigger('click');await flushPromises()
  finish(new Response(JSON.stringify({code:'AUTH_REQUIRED',message:'请重新登录'}),{status:401}));await flushPromises();expect(getPending()?.operationId).toBe(originalId);expect(wrapper.text()).toContain('结果待确认');wrapper.unmount()
 })
})
