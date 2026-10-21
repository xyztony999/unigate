import { reactive } from 'vue'
import { ApiError, call, clearPending, getPending, newId, savePending, errorText } from './api'
import { enumCode, type Obj } from './contract'
export type GateState='READY'|'RESOLVING'|'CONFIRMING'|'SUBMITTING'|'SUCCESS'|'KNOWN_REPLAY'|'REJECTED'|'UNKNOWN'|'BLOCKED'
export function createGate(api:(op:string,payload:Obj)=>Promise<any>=call) {
  const gate=reactive({state:'BLOCKED' as GateState,preview:null as Obj|null,result:null as Obj|null,device:null as Obj|null,count:1,message:'正在连接终端…',eventId:'',scanToken:'',pendingRequest:null as Obj|null})
  const pending=getPending()
  if(pending?.action==='GateCommit'){gate.state='UNKNOWN';gate.eventId=pending.eventId||'';gate.message='结果待确认，请工作人员核对。'}
  async function connect() {
    const wasUnknown=gate.state==='UNKNOWN'
    try {const data=await api('DeviceDetail',{});gate.device=data.device;gate.eventId=data.device.eventId
      if(wasUnknown){gate.state='UNKNOWN';gate.message='请重扫原组入场码，查询这次操作结果。';return}
      if(enumCode(data.eventStatus)!=='CHECKIN_OPEN'||enumCode(data.device.status)!=='ACTIVE'){gate.state='BLOCKED';gate.message=enumCode(data.eventStatus)==='CHECKIN_PAUSED'?'检票已暂停，请联系工作人员。':'本终端暂时无法检票，请联系工作人员。'}else{gate.state='READY';gate.message='请出示您的入场码'}
    }catch(e){gate.state=wasUnknown?'UNKNOWN':'BLOCKED';gate.message=errorText(e)}
  }
  async function resolve(ticketToken:string) {
    if(!['READY','UNKNOWN'].includes(gate.state))return
    const recovering=gate.state==='UNKNOWN'
    gate.state=recovering?'UNKNOWN':'RESOLVING';gate.scanToken=''
    try {const preview=await api('TicketResolve',{eventId:gate.eventId,ticketToken});gate.preview=preview
      if(recovering){await recover();return}
      gate.count=1
      if(preview.remainingCount<1){gate.state='REJECTED';gate.message='本组名额已全部使用';return}
      gate.state='CONFIRMING';gate.message='这次共有几位入场？'
    }catch(e){gate.state=recovering?'UNKNOWN':'REJECTED';gate.message=errorText(e)}
  }
  function applyResult(result:Obj,fromQuery=false) {
    gate.result=result
    if(enumCode(result.status)==='IN_PROGRESS'){gate.state='UNKNOWN';gate.message='正在确认这次操作，请稍候。';return}
    if(enumCode(result.status)==='REJECTED'){gate.state='REJECTED';gate.message='本次操作未通过，请联系工作人员核对。';clearPending();gate.pendingRequest=null;return}
    gate.state=!fromQuery&&result.showAsNewAdmission===true&&!result.replayed?'SUCCESS':'KNOWN_REPLAY'
    gate.message=gate.state==='SUCCESS'?`允许 ${result.committedCount} 人入场`:'这次操作已经记录，请工作人员核对。'
    clearPending();gate.pendingRequest=null
  }
  async function commit() {
    if(gate.state!=='CONFIRMING'||!gate.preview)return
    if(!Number.isInteger(gate.count)||gate.count<1||gate.count>gate.preview.remainingCount){gate.message='请选择有效的本次入场人数';return}
    const payload={operationId:newId(),eventId:gate.eventId,scanContextToken:gate.preview.scanContextToken,expectedPartyVersion:gate.preview.partyVersion,count:gate.count}
    gate.pendingRequest=payload
    savePending({operationId:payload.operationId,action:'GateCommit',eventId:gate.eventId,deviceId:gate.device?.deviceId})
    gate.state='SUBMITTING'
    try {applyResult(await api('GateCommit',payload))}catch(e){
      if(e instanceof ApiError && !e.uncertain){gate.state=e.status===401||e.status===403?'BLOCKED':'REJECTED';clearPending();gate.pendingRequest=null}else gate.state='UNKNOWN'
      gate.message=errorText(e)
    }
  }
  async function recover() {
    const p=getPending();if(!p||p.action!=='GateCommit')return
    if(p.deviceId!==gate.device?.deviceId){gate.message='请在原终端核对，或联系负责人查看核销记录。';return}
    try {const result=await api('GateOperation',{operationId:p.operationId,...(gate.preview?{scanContextToken:gate.preview.scanContextToken}:{})});applyResult(result,true)}catch(e){gate.state='UNKNOWN';gate.message=errorText(e)}
  }
  async function retrySame() {
    if(gate.state!=='UNKNOWN'||!gate.pendingRequest)return
    try {applyResult(await api('GateCommit',gate.pendingRequest),true)}catch(e){gate.message=errorText(e)}
  }
  async function reset(){if(['UNKNOWN','SUBMITTING'].includes(gate.state))return;gate.preview=null;gate.result=null;gate.count=1;await connect()}
  return {gate,connect,resolve,commit,recover,retrySame,reset}
}
