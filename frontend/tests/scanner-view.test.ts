import { beforeEach,afterEach,describe,it,expect,vi } from 'vitest'
import { mount,flushPromises } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { operations } from '../src/lib/contract'
import { createMemoryHistory,createRouter } from 'vue-router'
import ScannerView from '../src/views/ScannerView.vue'
import { identity,getPending } from '../src/lib/api'
const code=(value:string)=>({code:value,label:value})
let uncertain=false
beforeEach(()=>{sessionStorage.clear();vi.useFakeTimers();uncertain=false;identity.deviceCsrf='device-csrf';vi.stubGlobal('fetch',vi.fn(async(input:string)=>{
 let data:any
 if(input.includes('/device/detail'))data={device:{deviceId:'d1',eventId:'e1',name:'东门自助检票',status:code('ACTIVE')},csrfToken:'device-csrf',eventStatus:code('CHECKIN_OPEN')}
 else if(input.includes('/ticket/resolve'))data={scanContextToken:'scan-context-secret',eventId:'e1',approvedSize:3,netAdmitted:1,remainingCount:2,partyVersion:1}
 else if(input===operations.GateCommit.path){if(uncertain)throw new TypeError('network');data={status:code('SUCCEEDED'),committedCount:1,remainingCount:1,replayed:false,showAsNewAdmission:true}}
 else throw new Error('unexpected request '+input)
 return new Response(JSON.stringify({code:'OK',data,requestId:'r'}),{status:200,headers:{'Content-Type':'application/json'}})
 }))})
afterEach(()=>{vi.useRealTimers();vi.unstubAllGlobals()})
async function kiosk(){const router=createRouter({history:createMemoryHistory(),routes:[{path:'/terminal/kiosk',name:'P04',component:ScannerView}]});await router.push('/terminal/kiosk');await router.isReady();const wrapper=mount(defineComponent({template:'<RouterView />'}),{global:{plugins:[router]}});await flushPromises();return wrapper}
async function scan(wrapper:ReturnType<typeof mount>){await wrapper.find('#scan-input').setValue('ticket-secret');await wrapper.find('form').trigger('submit');await flushPromises()}
describe('kiosk公开屏幕交互',()=>{
 it('左扫码右确认、默认一人；新成功5秒清屏，不含名单搜索',async()=>{
  const w=await kiosk();expect(w.find('.scanner-page').classes()).toContain('kiosk');expect(w.find('.scan-columns').exists()).toBe(true)
  expect(w.text()).not.toContain('辅助找票');await scan(w);expect(w.find('#count').element).toHaveProperty('value','1')
  await w.find('.confirm-button').trigger('click');await flushPromises();expect(w.text()).toContain('允许 1 人入场')
  await vi.advanceTimersByTimeAsync(5000);await flushPromises();expect(w.text()).toContain('请出示您的入场码');expect(w.text()).not.toContain('允许 1 人入场');w.unmount()
 })
 it('请求已发出后断网，40秒后仍保留原操作且不自动扫下一组',async()=>{
  uncertain=true;const w=await kiosk();await scan(w);await w.find('.confirm-button').trigger('click');await flushPromises()
  const id=getPending()?.operationId;expect(id).toBeTruthy();expect(w.text()).toContain('结果待确认')
  await vi.advanceTimersByTimeAsync(40000);await flushPromises();expect(getPending()?.operationId).toBe(id);expect(w.text()).toContain('结果待确认');expect(w.find('.confirm-button').exists()).toBe(false);w.unmount()
 })
})
