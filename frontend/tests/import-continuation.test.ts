import { beforeEach,afterEach,describe,it,expect,vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount,flushPromises } from '@vue/test-utils'
import { createMemoryHistory,createRouter } from 'vue-router'
import WorkspaceView from '../src/views/WorkspaceView.vue'
import { identity } from '../src/lib/api'
import { contract,operations } from '../src/lib/contract'
const eventId='11111111-1111-4111-8111-111111111111',jobId='22222222-2222-4222-8222-222222222222',batchId='33333333-3333-4333-8333-333333333333'
const value=(code:string)=>({code,label:code})
let job:any,requests:{op:string;url:string;body:any}[]
const ok=(data:any)=>new Response(JSON.stringify({code:'OK',data,requestId:'r'}),{status:200,headers:{'Content-Type':'application/json'}})
beforeEach(()=>{
 sessionStorage.clear();requests=[];job={jobId,eventId,kind:value('PULL'),status:value('SUCCEEDED'),resultRef:batchId,version:1}
 Object.assign(identity,{staff:{accountId:'a',roles:[value('ORGANIZER')],displayName:'负责人'},staffCsrf:'csrf'})
 vi.stubGlobal('fetch',vi.fn(async(url:string,init:RequestInit)=>{
  const op=(Object.values(operations) as any[]).find(o=>url.split('?')[0]===o.path)?.name;requests.push({op,url,body:init?.body?JSON.parse(String(init.body)):{}})
  switch(op){
   case'EventDetail':return ok({eventId,title:'家长见面会',status:value('READY'),version:1})
   case'JobList':return ok({items:[job],total:1,page:1,pageSize:20})
   case'JobDetail':return ok(job)
   case'BatchDetail':return ok({eventId,batchId,status:value('UPLOADED'),version:7,previewVersion:0,validRows:0,errorRows:0,suspectRows:0,availableColumns:[{code:'name-col',label:'家长姓名'}]})
   default:throw new Error('Unexpected operation '+op)
  }
 }))
})
afterEach(()=>vi.unstubAllGlobals())
async function workspace(path:string){const pages=['O15','O07'].map(id=>contract.pages.find(p=>p.id===id)!);const router=createRouter({history:createMemoryHistory(),routes:pages.map(page=>({path:page.route,component:WorkspaceView,meta:{page}}))});await router.push(path);await router.isReady();const wrapper=mount(defineComponent({template:'<RouterView :key="$route.fullPath" />'}),{global:{plugins:[router],stubs:{teleport:true}}});await flushPromises();return{wrapper,router}}
async function inspectJob(wrapper:ReturnType<typeof mount>){await wrapper.find('td button').trigger('click');await flushPromises();await wrapper.findAll('button.tab').find(b=>b.text()===operations.JobDetail.summary)!.trigger('click');await flushPromises()}
describe('平台拉取任务到名单导入的安全接续',()=>{
 it('成功PULL任务通过正式resultRef产生入口，O07先读取所属活动的批次版本及列',async()=>{
  const{wrapper,router}=await workspace('/manage/events/'+eventId+'/jobs');await inspectJob(wrapper)
  const link=wrapper.findAll('a').find(a=>a.text()==='继续名单导入')!;expect(link).toBeDefined();expect(link.attributes('href')).toBe('/manage/events/'+eventId+'/imports?batchId='+batchId)
  await link.trigger('click');await flushPromises();expect(router.currentRoute.value.query.batchId).toBe(batchId)
  const read=requests.find(r=>r.op==='BatchDetail')!;expect(new URL(read.url,'http://localhost').searchParams.get('eventId')).toBe(eventId);expect(new URL(read.url,'http://localhost').searchParams.get('batchId')).toBe(batchId);expect(read.url).not.toContain('page=')
  expect(requests.some(r=>r.op==='BatchRows')).toBe(false);expect(wrapper.findAll('[role=dialog]')).toHaveLength(0)
  await wrapper.findAll('button').find(b=>b.text()===operations.BatchPreview.summary)!.trigger('click');await flushPromises();expect(wrapper.findAll('option').some(o=>o.text()==='家长姓名')).toBe(true);expect(wrapper.findAll('label.field').some(f=>f.text().includes('批次'))).toBe(false)
  const deepLink=router.currentRoute.value.fullPath;wrapper.unmount();const refreshed=await workspace(deepLink);expect(requests.filter(r=>r.op==='BatchDetail')).toHaveLength(2);expect(refreshed.wrapper.text()).not.toContain('上传名单后');refreshed.wrapper.unmount()
 })
 it('非来源拉取任务的resultRef不能冒充导入批次',async()=>{
  job.kind=value('EXPORT');const{wrapper}=await workspace('/manage/events/'+eventId+'/jobs');await inspectJob(wrapper);expect(wrapper.findAll('a').some(a=>a.text()==='继续名单导入')).toBe(false);wrapper.unmount()
 })
 it('非法query不读取批次，仍只有一个上传起点按钮',async()=>{
  const{wrapper}=await workspace('/manage/events/'+eventId+'/imports?batchId=not-a-uuid');expect(wrapper.text()).toContain('名单导入链接无效');expect(requests.some(r=>r.op==='BatchDetail'||r.op==='BatchRows')).toBe(false);expect(wrapper.findAll('button').filter(b=>b.text()==='上传名单')).toHaveLength(1);wrapper.unmount()
 })
})
