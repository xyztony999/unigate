import { reactive } from 'vue'
import { operations, type Obj } from './contract'
import type { OperationMap, SessionView } from '../generated/api-types'
export const identity = reactive<{staff:SessionView|null; anonCsrf:string; staffCsrf:string; partyCsrf:string; deviceCsrf:string; clientContextId:string}>({staff:null,anonCsrf:'',staffCsrf:'',partyCsrf:'',deviceCsrf:'',clientContextId:''})
export class ApiError extends Error {
  constructor(public code:string, message:string, public status:number=0, public uncertain=false, public requestId='') { super(message) }
}
export function acceptAuth(op:string, data:Obj) {
  if(op==='SessionBootstrap') { identity.anonCsrf=data.csrfToken; identity.clientContextId=data.clientContextId }
  if(op==='Login' || op==='SessionMe') { identity.staff=data as SessionView; identity.staffCsrf=data.csrfToken }
  if(op==='PartyExchange') identity.partyCsrf=data.csrfToken
  if(op==='DeviceClaim'||op==='DeviceDetail') identity.deviceCsrf=data.csrfToken
  if(op==='Logout') { identity.staff=null; identity.staffCsrf='' }
}
export async function call<K extends keyof OperationMap>(op:K, payload:OperationMap[K]['request'], originalContext?:string):Promise<OperationMap[K]['data']>
export async function call(op:string,payload?:Obj,originalContext?:string):Promise<any>
export async function call(op:string,payload:Obj={},originalContext?:string):Promise<any> {
  const spec=operations[op]; if(!spec) throw new Error('未知操作')
  if(spec.auth==='ANON' && op!=='SessionBootstrap' && !identity.anonCsrf) await bootstrap()
  if(originalContext && identity.clientContextId!==originalContext) throw new ApiError('CLIENT_CONTEXT_CHANGED','原页面已失效，请联系工作人员核对，勿再次提交。',409,true)
  const csrf=spec.auth==='ANON' ? identity.anonCsrf : spec.auth==='PARTY'||spec.auth==='CHANGE' ? (identity.partyCsrf||identity.anonCsrf) : spec.auth==='DEVICE'||spec.auth==='GATE' ? (identity.deviceCsrf||identity.staffCsrf) : identity.staffCsrf
  const headers:Record<string,string>={Accept:'application/json'}
  if(spec.method!=='get') headers['X-CSRF-Token']=csrf
  if(spec.auth==='ANON' && (spec.method!=='get'||op==='RegistrationResult')) headers['X-Client-Context-ID']=originalContext || identity.clientContextId
  let url=spec.path, body:BodyInit|undefined
  if(spec.method==='get') { const qs=new URLSearchParams(); Object.entries(payload).forEach(([k,v])=>{ if(v!==undefined&&v!==null&&v!=='') qs.set(k,String(v)) }); if(qs.size) url+='?'+qs }
  else if(spec.upload) { const data=new FormData(); Object.entries(payload).forEach(([k,v])=>data.append(k,v instanceof Blob?v:String(v))); body=data }
  else { headers['Content-Type']='application/json'; body=JSON.stringify(payload) }
  const controller=new AbortController(); const timer=setTimeout(()=>controller.abort(),20000)
  try {
    const response=await fetch(url,{method:spec.method.toUpperCase(),body,headers,credentials:'include',cache:'no-store',signal:controller.signal,redirect:'error'})
    if(op==='ExportDownload' && response.ok && !response.headers.get('content-type')?.includes('json')) return await response.blob()
    const result=await response.json().catch(()=>({code:'INVALID_RESPONSE',message:'服务暂时未能返回结果'}))
    if(!response.ok || result.code!=='OK') throw new ApiError(result.code||'REQUEST_FAILED',result.message||'请求未完成',response.status,(!!spec.mutation && (response.status>=500 || result.code==='OPERATION_IN_PROGRESS')),result.requestId)
    acceptAuth(op,result.data||{})
    return result.data
  } catch(e) {
    if(e instanceof ApiError) throw e
    throw new ApiError('NETWORK_UNAVAILABLE',spec.mutation?'结果待确认，请勿重复提交。':'暂时无法连接服务，请稍后重试。',0,!!spec.mutation)
  } finally { clearTimeout(timer) }
}
let boot:Promise<any>|undefined
export async function bootstrap() { return boot ||= call('SessionBootstrap',{}).finally(()=>{boot=undefined}) }
export async function initStaff() {
  await bootstrap()
  try { await call('SessionMe',{}) } catch(e) { if(!(e instanceof ApiError)||e.status!==401) throw e }
}
export interface PendingRef { operationId:string; action:string; eventId?:string; clientContextId?:string; deviceId?:string }
const pendingKey='unigate.pending.v1'
export function savePending(p:PendingRef) { sessionStorage.setItem(pendingKey,JSON.stringify(p)) }
export function getPending():PendingRef|null { try { return JSON.parse(sessionStorage.getItem(pendingKey)||'null') } catch { return null } }
export function clearPending() { sessionStorage.removeItem(pendingKey) }
let privateEntryHandoff: {link:string;expiresAt:number}|undefined
export function holdPrivateEntry(link:string) { privateEntryHandoff={link,expiresAt:Date.now()+60000} }
export function takePrivateEntry() { const entry=privateEntryHandoff;privateEntryHandoff=undefined;return entry&&entry.expiresAt>Date.now()?entry.link:'' }
export const newId=()=>crypto.randomUUID()
export function errorText(e:unknown) { return e instanceof Error ? e.message : '操作未完成，请稍后重试。' }
