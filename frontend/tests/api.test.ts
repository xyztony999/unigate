import { beforeEach,describe,it,expect,vi } from 'vitest'
import { call,identity,ApiError } from '../src/lib/api'
beforeEach(()=>{Object.assign(identity,{anonCsrf:'anon-csrf',staffCsrf:'staff-csrf',deviceCsrf:'device-csrf',partyCsrf:'',clientContextId:'original-context'});vi.unstubAllGlobals()})
function respond(data:any,status=200){return new Response(JSON.stringify({code:'OK',data,requestId:'request'}),{status,headers:{'Content-Type':'application/json'}})}
describe('Cookie/CSRF与不确定响应',()=>{
 it('匿名提交携带原上下文及CSRF，不保存或放URL中的个人字段',async()=>{
  const fetch=vi.fn(async()=>respond({party:{},access:{}}));vi.stubGlobal('fetch',fetch)
  await call('SubmitRegistration',{operationId:'op',eventId:'event',fields:{email:'person@example.test'},acceptedNoticeVersion:'1'},'original-context')
  const [url,request]=fetch.mock.calls[0] as unknown as [string,RequestInit]
  expect(url).not.toContain('person');expect(request.credentials).toBe('include');expect(request.cache).toBe('no-store')
  expect(request.headers).toMatchObject({'X-CSRF-Token':'anon-csrf','X-Client-Context-ID':'original-context'})
 })
 it('新匿名上下文不能自动重放旧报名',async()=>{
  const fetch=vi.fn();vi.stubGlobal('fetch',fetch)
  await expect(call('SubmitRegistration',{},'expired-context')).rejects.toMatchObject({code:'CLIENT_CONTEXT_CHANGED',uncertain:true})
  expect(fetch).not.toHaveBeenCalled()
 })
 it('HTTP 5xx和处理中不是确定失败',async()=>{
  vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({code:'SERVER_ERROR',message:'稍后核对'}),{status:503})))
  await expect(call('GateCommit',{})).rejects.toMatchObject({uncertain:true})
  vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({code:'OPERATION_IN_PROGRESS',message:'处理中'}),{status:409})))
  await expect(call('GateCommit',{})).rejects.toMatchObject({uncertain:true})
 })
 it('容量冲突是确定拒绝，设备核销使用设备CSRF',async()=>{
  const fetch=vi.fn(async()=>new Response(JSON.stringify({code:'CAPACITY_EXCEEDED',message:'名额不足'}),{status:409}));vi.stubGlobal('fetch',fetch)
  await expect(call('GateCommit',{})).rejects.toMatchObject({uncertain:false,code:'CAPACITY_EXCEEDED'})
  expect((fetch.mock.calls[0] as unknown as [string,RequestInit])[1].headers).toMatchObject({'X-CSRF-Token':'device-csrf'})
 })
 it('参与者刷新后使用bootstrap CSRF，仍通过HttpOnly组cookie鉴权',async()=>{
  const fetch=vi.fn(async()=>respond({}));vi.stubGlobal('fetch',fetch)
  await call('MyPartyCancel',{operationId:'op',version:1})
  expect((fetch.mock.calls[0] as unknown as [string,RequestInit])[1].headers).toMatchObject({'X-CSRF-Token':'anon-csrf'})
 })
})
