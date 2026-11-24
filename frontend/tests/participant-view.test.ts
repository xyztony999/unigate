import { beforeEach,afterEach,describe,it,expect,vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount,flushPromises } from '@vue/test-utils'
import { createMemoryHistory,createRouter } from 'vue-router'
import ParticipantView from '../src/views/ParticipantView.vue'
import { identity,getPending } from '../src/lib/api'
import { operations } from '../src/lib/contract'
vi.mock('../src/lib/confirm',()=>({askConfirm:vi.fn(async()=>true)}))
const code=(value:string)=>({code:value,label:value})
const access={queryToken:'private-query-token',changeToken:'private-change-token',expiresAt:'2026-12-01T00:00:00Z'}
const baseParty={partyId:'party',partyReference:'U-123',eventId:'event',fields:{name:'测试家长',companionCount:2},status:code('PENDING'),approvedSize:0,pendingIncrease:0,netAdmitted:0,remainingCount:0,progress:code('NONE'),canChange:false,canCancel:false,version:1,expiresAt:'2026-12-01T00:00:00Z'}
const publicEvent={eventId:'event',title:'家长见面会',startsAt:'2026-11-01T02:00:00Z',endsAt:'2026-11-01T04:00:00Z',timeZone:'Asia/Shanghai',fields:{name:{enabled:true,required:true},phone:{enabled:false,required:false},email:{enabled:false,required:false},companionCount:{enabled:true,required:false}},approvalMode:code('REVIEW'),phoneVerificationEnabled:false,registrationAvailable:true,notice:'仅用于本次活动报名与入场。',noticeVersion:'1',retentionDays:30}
let event:any,party:any,requests:{op:string;body:any}[],submitUnknown:boolean
const ok=(data:any,status=200)=>new Response(JSON.stringify({code:'OK',data,requestId:'r'}),{status,headers:{'Content-Type':'application/json'}})
beforeEach(()=>{
 sessionStorage.clear();history.replaceState(null,'','/');requests=[];event=structuredClone(publicEvent);party=structuredClone(baseParty);submitUnknown=false
 Object.assign(identity,{staff:null,anonCsrf:'anon-csrf',staffCsrf:'',partyCsrf:'',deviceCsrf:'',clientContextId:'original-context'})
 vi.stubGlobal('fetch',vi.fn(async(url:string,init:RequestInit)=>{
  const op=(Object.values(operations) as any[]).find(o=>url.split('?')[0]===o.path)?.name
  const body=init?.body?JSON.parse(String(init.body)):{};requests.push({op,body})
  switch(op){
   case'SessionBootstrap':return ok({csrfToken:'anon-csrf',clientContextId:'original-context',expiresAt:'2026-12-01T00:00:00Z'})
   case'PublicEvent':return ok(event)
   case'SubmitRegistration':if(submitUnknown)throw new TypeError('network');return ok({party,access,operationId:body.operationId,replayed:false})
   case'RegistrationResult':return ok({party,access,operationId:body.operationId,replayed:true})
   case'PartyExchange':return ok({csrfToken:'party-csrf',clientContextId:'original-context',expiresAt:'2026-12-01T00:00:00Z'})
   case'MyParty':return ok(party)
   case'MyCredential':return new Response(JSON.stringify({code:'TICKET_UNAVAILABLE',message:'尚无有效入场码'}),{status:422})
   case'SmsStart':case'RecoveryStart':return ok({challengeId:'challenge',expiresAt:'2026-12-01T00:00:00Z'},202)
   case'SmsVerify':return ok({verificationProof:'verification-proof-secret',expiresAt:'2026-12-01T00:00:00Z'})
   case'RecoveryVerify':return ok(access)
   case'MyPartyChange':party={...party,fields:{companionCount:body.companionCount},version:2};return ok(party)
   case'MyPartyCancel':party={...party,status:code('CANCELLED'),canChange:false,canCancel:false,version:3};return ok(party)
   case'DeviceClaim':return ok({device:{deviceId:body.deviceId,eventId:'event',name:'东门',mode:code('KIOSK')},csrfToken:'device-csrf',eventStatus:code('CHECKIN_OPEN')})
   case'Logout':return ok({})
   default:throw new Error('Unexpected operation '+op)
  }
 }))
})
afterEach(()=>{vi.unstubAllGlobals();history.replaceState(null,'','/')})
async function participant(path:string){const blank=defineComponent({template:'<p>已领取终端</p>'});const router=createRouter({history:createMemoryHistory(),routes:[{path:'/e/:eventId/register',name:'P01',component:ParticipantView},{path:'/e/:eventId/recover',name:'P03',component:ParticipantView},{path:'/me/party',name:'P02',component:ParticipantView},{path:'/terminal/activate',name:'S04',component:ParticipantView},{path:'/terminal/kiosk',name:'P04',component:blank}]});await router.push(path);await router.isReady();const wrapper=mount(defineComponent({template:'<RouterView :key="$route.fullPath" />'}),{global:{plugins:[router]}});await flushPromises();return{wrapper,router}}
async function fill(wrapper:ReturnType<typeof mount>,label:string,value:string){const field=wrapper.findAll('label.field').find(f=>f.text().startsWith(label));expect(field).toBeDefined();await field!.find('input').setValue(value)}
async function button(wrapper:ReturnType<typeof mount>,text:string){const target=wrapper.findAll('button').find(b=>b.text()===text);expect(target).toBeDefined();await target!.trigger('click');await flushPromises()}
async function prepareRegistration(wrapper:ReturnType<typeof mount>){await fill(wrapper,'姓名','测试家长');await fill(wrapper,'随行人数','2');await wrapper.find('input[type=checkbox]').setValue(true)}
describe('参与者真实组件与接口边界',()=>{
 it('关闭的联系方式不显示也不提交；审核报名无有效码，保存私密入口与入场码分开',async()=>{
  const{wrapper,router}=await participant('/e/event/register');expect(wrapper.text()).not.toContain('手机号');expect(wrapper.find('input[type=email]').exists()).toBe(false)
  await prepareRegistration(wrapper);await wrapper.find('form').trigger('submit');await flushPromises()
  const request=requests.find(r=>r.op==='SubmitRegistration')!;expect(request.body.fields).toEqual({name:'测试家长',companionCount:2});expect(request.body.acceptedNoticeVersion).toBe('1');expect(router.currentRoute.value.path).toBe('/me/party');expect(wrapper.text()).toContain('报名正在审核');expect(wrapper.find('.ticket-qr').exists()).toBe(false);expect(wrapper.text()).toContain('复制私密查询入口');expect(getPending()).toBeNull();wrapper.unmount()
 })
 it('报名超时不重复提交，原上下文原ID查询并恢复报名入口',async()=>{
  submitUnknown=true;const{wrapper,router}=await participant('/e/event/register');await prepareRegistration(wrapper);await wrapper.find('form').trigger('submit');await flushPromises();const pending=getPending()!
  expect(wrapper.text()).toContain('结果待确认');expect(sessionStorage.getItem('unigate.pending.v1')).not.toMatch(/测试家长|private-query|companionCount/);await router.push('/e/event/recover');expect(router.currentRoute.value.path).toBe('/e/event/register')
  await button(wrapper,'查询原操作结果');expect(requests.filter(r=>r.op==='SubmitRegistration')).toHaveLength(1);expect(requests.find(r=>r.op==='RegistrationResult')?.body.operationId).toBe(pending.operationId);expect(router.currentRoute.value.path).toBe('/me/party');expect(getPending()).toBeNull();wrapper.unmount()
 })
 it('启用短信时必须验证，证明仅内存绑定到提交',async()=>{
  event.fields.phone={enabled:true,required:true};event.phoneVerificationEnabled=true;const{wrapper}=await participant('/e/event/register');await prepareRegistration(wrapper);await fill(wrapper,'手机号','+8613800000000')
  expect(wrapper.find('button.full').attributes('disabled')).toBeDefined();await button(wrapper,'获取手机验证码');expect(wrapper.text()).toContain('验证申请已受理');await fill(wrapper,'验证码','123456');await button(wrapper,'验证');expect(wrapper.text()).toContain('手机号验证完成');await wrapper.find('form').trigger('submit');await flushPromises();expect(requests.find(r=>r.op==='SubmitRegistration')?.body.phoneVerificationProof).toBe('verification-proof-secret');expect(sessionStorage.getItem('unigate.pending.v1')).toBeNull();wrapper.unmount()
 })
 it('找回统一受理后验证，再交换私密能力；不显示原令牌',async()=>{
  const{wrapper,router}=await participant('/e/event/recover');await fill(wrapper,'报名参考号','U-123');await fill(wrapper,'报名邮箱','parent@example.test');await wrapper.find('form').trigger('submit');await flushPromises();expect(wrapper.text()).toContain('如信息匹配');await fill(wrapper,'验证码','123456');await wrapper.find('form').trigger('submit');await flushPromises();expect(router.currentRoute.value.path).toBe('/me/party');expect(requests.find(r=>r.op==='RecoveryStart')?.body).toMatchObject({eventId:'event',partyReference:'U-123',channel:'EMAIL',contact:'parent@example.test'});expect(wrapper.text()).not.toContain('private-query-token');wrapper.unmount()
 })
 it('自助修改及取消按服务端标志显示，沿原版本提交后刷新状态',async()=>{
  party.canChange=true;party.canCancel=true;const{wrapper}=await participant('/me/party');await fill(wrapper,'随行人数','1');await button(wrapper,'更新随行人数');expect(requests.find(r=>r.op==='MyPartyChange')?.body).toMatchObject({version:1,companionCount:1});await button(wrapper,'取消报名');expect(requests.find(r=>r.op==='MyPartyCancel')?.body.version).toBe(2);expect(wrapper.text()).toContain('CANCELLED');expect(wrapper.text()).not.toContain('更新随行人数');wrapper.unmount()
 })
 it('fragment领取信息读入即清URL；kiosk激活清工作人员会话',async()=>{
  history.replaceState(null,'','/#deviceId=terminal-id&pairingCode=private-pairing-code');identity.staff={accountId:'staff',username:'operator',displayName:'工作人员',roles:[],csrfToken:'staff-csrf',expiresAt:'2026-12-01T00:00:00Z',mustChangePassword:false};identity.staffCsrf='staff-csrf';const{wrapper,router}=await participant('/terminal/activate');expect(location.hash).toBe('');await wrapper.find('form').trigger('submit');await flushPromises();expect(requests.find(r=>r.op==='DeviceClaim')?.body).toMatchObject({deviceId:'terminal-id',pairingCode:'private-pairing-code'});expect(requests.some(r=>r.op==='Logout')).toBe(true);expect(identity.staff).toBeNull();expect(router.currentRoute.value.path).toBe('/terminal/kiosk');wrapper.unmount()
 })
})
