import { beforeEach, describe, expect,it,vi } from 'vitest'
import { createGate } from '../src/lib/gate'
import { ApiError,getPending,savePending } from '../src/lib/api'
const value=(code:string)=>({code,label:code})
const device={deviceId:'device-1',eventId:'event-1',name:'入口',status:value('ACTIVE')}
const preview={eventId:'event-1',scanContextToken:'private-scan-token',approvedSize:3,netAdmitted:1,remainingCount:2,partyVersion:7}
const success={operationId:'op',status:value('SUCCEEDED'),committedCount:1,remainingCount:1,replayed:false,showAsNewAdmission:true,committedAt:'2026-10-21T08:00:00Z'}
beforeEach(()=>sessionStorage.clear())
describe('组人数核销与原操作恢复',()=>{
 it('扫码只预览，默认一人；显式提交一次并防止双击',async()=>{
  let finish:(x:any)=>void=()=>{}
  const api=vi.fn(async(op:string,_payload:any={})=>op==='DeviceDetail'?{device,eventStatus:value('CHECKIN_OPEN')}:op==='TicketResolve'?preview:new Promise(r=>{finish=r}))
  const {gate,connect,resolve,commit}=createGate(api)
  await connect();await resolve('ticket')
  expect(gate.state).toBe('CONFIRMING');expect(gate.count).toBe(1)
  expect(api.mock.calls.map(c=>c[0])).toEqual(['DeviceDetail','TicketResolve'])
  const pending=commit();await commit()
  expect(api.mock.calls.filter(c=>c[0]==='GateCommit')).toHaveLength(1)
  expect(gate.state).toBe('SUBMITTING')
  finish(success);await pending;expect(gate.state).toBe('SUCCESS');expect(getPending()).toBeNull()
 })
 it('超时未知保留最小引用、锁下一组；查询原结果不再次绿灯',async()=>{
  const api=vi.fn(async(op:string,_payload:any={})=>{if(op==='DeviceDetail')return{device,eventStatus:value('CHECKIN_OPEN')};if(op==='TicketResolve')return preview;if(op==='GateCommit')throw new ApiError('NETWORK_UNAVAILABLE','结果待确认',0,true);return success})
  const {gate,connect,resolve,commit,reset,recover}=createGate(api)
  await connect();await resolve('secret-ticket');await commit()
  expect(gate.state).toBe('UNKNOWN')
  const stored=sessionStorage.getItem('unigate.pending.v1')!
  expect(stored).toContain('operationId');expect(stored).not.toMatch(/secret-ticket|private-scan-token|姓名|phone|count/)
  const id=getPending()!.operationId
  await reset();await resolve('secret-ticket')
  expect(gate.state).toBe('KNOWN_REPLAY')
  expect(api.mock.calls.find(c=>c[0]==='GateOperation')![1]).toMatchObject({operationId:id})
  expect(gate.message).not.toContain('允许')
 })
 it('OPERATION_IN_PROGRESS保持未知并且重试复用原操作ID和载荷',async()=>{
  let commits=0
  const api=vi.fn(async(op:string,_payload:any={})=>{if(op==='DeviceDetail')return{device,eventStatus:value('CHECKIN_OPEN')};if(op==='TicketResolve')return preview;if(op==='GateCommit'){commits++;if(commits===1)throw new ApiError('OPERATION_IN_PROGRESS','正在处理',409,true);return {...success,replayed:true,showAsNewAdmission:false}}})
  const {gate,connect,resolve,commit,retrySame}=createGate(api)
  await connect();await resolve('ticket');await commit();expect(gate.state).toBe('UNKNOWN')
  await retrySame();const calls=api.mock.calls.filter(c=>c[0]==='GateCommit')
  expect(calls[0][1]).toEqual(calls[1][1]);expect(gate.state).toBe('KNOWN_REPLAY')
 })
 it('已确认业务拒绝不显示成功，结束与暂停不能核销',async()=>{
  const api=vi.fn(async(op:string,_payload:any={})=>op==='DeviceDetail'?{device,eventStatus:value('CHECKIN_PAUSED')}:preview)
  const {gate,connect,resolve,commit}=createGate(api);await connect();await resolve('ticket');await commit()
  expect(gate.state).toBe('BLOCKED');expect(api).toHaveBeenCalledTimes(1)
 })
 it('剩余0或非法人数不能提交',async()=>{
  const api=vi.fn(async(op:string,_payload:any={})=>op==='DeviceDetail'?{device,eventStatus:value('CHECKIN_OPEN')}:preview)
  const {gate,connect,resolve,commit}=createGate(api);await connect();await resolve('ticket');gate.count=3;await commit();expect(api).toHaveBeenCalledTimes(2)
  gate.count=1.5;await commit();expect(api).toHaveBeenCalledTimes(2)
 })
 it('刷新后的未知操作仍需原终端；不能在新终端查询或重扣',async()=>{
  savePending({operationId:'original',action:'GateCommit',eventId:'event-1',deviceId:'old-device'})
  const api=vi.fn(async(op:string,_payload:any={})=>op==='DeviceDetail'?{device,eventStatus:value('CHECKIN_OPEN')}:preview)
  const {gate,connect,resolve}=createGate(api);await connect();await resolve('ticket')
  expect(gate.state).toBe('UNKNOWN');expect(gate.message).toContain('原终端');expect(api.mock.calls.map(c=>c[0])).not.toContain('GateCommit');expect(getPending()?.operationId).toBe('original')
 })
 it('查询处理中不清除待确认引用，查询确定拒绝才解锁',async()=>{
  let rejected=false
  const api=vi.fn(async(op:string,_payload:any={})=>{if(op==='DeviceDetail')return{device,eventStatus:value('CHECKIN_OPEN')};if(op==='TicketResolve')return preview;if(op==='GateCommit')throw new ApiError('NETWORK_UNAVAILABLE','未知',0,true);return{...success,status:value(rejected?'REJECTED':'IN_PROGRESS')}})
  const {gate,connect,resolve,commit,recover}=createGate(api);await connect();await resolve('ticket');await commit();await recover()
  expect(gate.state).toBe('UNKNOWN');expect(getPending()).not.toBeNull()
  rejected=true;await recover();expect(gate.state).toBe('REJECTED');expect(getPending()).toBeNull()
 })
})
