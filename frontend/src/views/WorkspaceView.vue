<script setup lang="ts">
import { computed,onMounted,onBeforeUnmount,ref,watch } from 'vue'
import { useRoute,useRouter,onBeforeRouteLeave } from 'vue-router'
import QRCode from 'qrcode'
import { validate } from '../lib/validation'
import DynamicFields from '../components/DynamicFields.vue'
import ResultDisplay from '../components/ResultDisplay.vue'
import { contract,operations,schema,defaults,clean,label,display,enumCode,type Obj } from '../lib/contract'
import { call,identity,newId,errorText,ApiError,getPending,savePending,clearPending } from '../lib/api'
import { askConfirm } from '../lib/confirm'
const route=useRoute(),router=useRouter(),page=computed(()=>route.meta.page as Obj)
const results=ref<Record<string,any>>({}),selected=ref<Obj>({}),context=ref<Obj>({}),form=ref<Obj>({}),active=ref(''),busy=ref(false),error=ref(''),notice=ref(''),modal=ref(false),unknown=ref(!!getPending()),pairingQr=ref(''),pairingLink=ref(''),queryQr=ref(''),currentPage=ref(1),selectedRows=ref<string[]>([]),viewOp=ref(''),eventStatus=ref(''),policy=ref<Obj>(),grants=ref<Obj[]>([])
const roles=computed(()=>identity.staff?.roles.map(enumCode)||[])
const eventId=computed(()=>String(route.params.eventId||context.value.eventId||''))
const pageOps=computed(()=>page.value.operations.filter((n:string)=>!['SessionMe','Logout','SessionBootstrap','EventOptions','MyGrantList'].includes(n)||n===page.value.primary))
const dataOps=computed(()=>pageOps.value.filter((n:string)=>!operations[n].mutation))
const actions=computed(()=>pageOps.value.filter((n:string)=>operations[n].mutation&&operations[n].roles.some((r:string)=>roles.value.includes(r))))
const primary=computed(()=>page.value.primary)
const pendingEvent=computed(()=>unknown.value?getPending()?.eventId:undefined)
const signupQr=ref(''),signupLink=computed(()=>location.origin+'/e/'+eventId.value+'/register')
const successNavigation=ref(false)
const continuationBatch=ref('')
const uuidReference=(value:unknown):value is string=>typeof value==='string'&&/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value)
const importContinuation=computed(()=>continuationBatch.value&&eventId.value?'/manage/events/'+encodeURIComponent(eventId.value)+'/imports?batchId='+encodeURIComponent(continuationBatch.value):'')
const references=computed(()=>Object.fromEntries(Object.values(results.value).flatMap(value=>(value?.items||[]).filter((row:Obj)=>row.eventId&&row.title).map((row:Obj)=>[row.eventId,row.title]))))
const queryPayloads=ref<Record<string,Obj>>({})
const current=computed(()=>results.value[viewOp.value||primary.value])
const columns=computed(()=>page.value.columns.filter((c:Obj)=>c.operation===(viewOp.value||primary.value)))
const activeSchema=computed(()=>contract.schemas[operations[active.value]?.request]||{})
const locked=computed(()=>Object.keys(schema(activeSchema.value).properties||{}).filter(k=>['eventId','partyId','reviewId','recordId','accountId','grantId','deviceId','batchId','sourceId','connectionId','resourceId','rowId','cleanupId','exportId','jobId','partyVersion'].includes(k)&&form.value[k]!==undefined))
const options=computed(()=>{
 const all:Record<string,Obj[]>={allowedActions:[{value:'VIEW_DETAIL',label:'查看明细'},{value:'VIEW_STATISTICS',label:'查看统计'},{value:'CHECKIN',label:'核销'}],allowedEventIds:[],selectedRowIds:selectedRows.value.map(x=>({value:x,label:'已选择的可提交行'}))}
 if(roles.value.includes('ADMIN'))all.allowedActions.push({value:'EXPORT',label:'受控导出'})
 const exports=['partyReference','name','phone','email','companionCount','requestedSize','approvedSize','netAdmitted','remainingCount','status','progress','sourceType','createdAt']
 const own=grants.value.filter(g=>g.eventId===eventId.value&&enumCode(g.role)==='ORGANIZER')
 const allowed=new Set(own.flatMap(g=>g.allowedFields?.map(enumCode)||[]))
 all.columns=exports.filter(c=>!['name','phone','email','companionCount'].includes(c)||allowed.has(c)||allowed.has(c.toUpperCase())).map(c=>({value:c,label:label(c)}))
 for(const [op,result] of Object.entries(results.value)) {
  const rows=result?.items||[]
  for(const [field,titleKey] of Object.entries({eventId:'title',accountId:'displayName',connectionId:'displayName',sourceId:'resourceId',resourceId:'title',batchId:'sourceLabel',jobId:'kind',partyId:'partyReference'})) {
   const rowsWith=rows.filter((r:Obj)=>r[field]);if(rowsWith.length)all[field]=rowsWith.map((r:Obj)=>({value:r[field],label:display(r[titleKey]||r.name||r[field])}))
  }
  if(op==='GrantEventOptions'||op==='EventOptions') {const items=result.items||result.options||[];if(items.length)all.allowedEventIds=all.eventId=items.map((e:Obj)=>({value:e.eventId||e.code,label:e.title||e.label}))}
  if(result?.availableConnections)all.connectionId=result.availableConnections.map((c:Obj)=>({value:c.connectionId,label:c.displayName}))
  if(result?.availableColumns)for(const col of ['nameColumn','phoneColumn','emailColumn','countColumn'])all[col]=result.availableColumns.map((c:Obj)=>({value:c.code,label:c.label}))
 }
 if(page.value.id==='O16'&&results.value.GrantList?.items)all.accountId=results.value.GrantList.items.map((g:Obj)=>({value:g.accountId,label:g.accountDisplayName}))
 if(all.partyId){all.existingPartyId=all.partyId;all.existingPendingPartyId=all.partyId}
 if(form.value.role==='READER')all.allowedActions=all.allowedActions.filter(x=>['VIEW_DETAIL','VIEW_STATISTICS'].includes(x.value))
 if(form.value.role==='CHECKER')all.allowedActions=all.allowedActions.filter(x=>x.value!=='EXPORT')
 return all
})
function plain(v:any):any{if(v&&typeof v==='object'&&!Array.isArray(v)&&'code'in v)return v.code;if(Array.isArray(v))return v.map(plain);if(v&&typeof v==='object')return Object.fromEntries(Object.entries(v).map(([k,x])=>[k,plain(x)]));return v}
function payloadFor(op:string){const def=contract.schemas[operations[op].request];const base:Obj={...defaults(def),...context.value,...plain(selected.value),eventId:eventId.value||undefined,partyId:route.params.partyId||selected.value.partyId||context.value.partyId};const properties=def.properties||{}
 if(op==='PolicyUpdate'&&results.value.PolicyGet)base.policy=plain(results.value.PolicyGet)
 if(op==='GateCorrect'&&context.value.partyVersion!==undefined)base.partyVersion=context.value.partyVersion
 if(op==='BatchCommit'){base.selectedRowIds=[...selectedRows.value];base.previewVersion=context.value.previewVersion}
 if(op==='OperationResult'&&getPending()){const p=getPending()!;base.operationId=p.operationId;base.action=p.action;base.eventId=p.eventId}
 if(/Create$/.test(op)||op==='DeviceRegister'||op==='BatchUpload'){for(const key of Object.keys(base))if(!['eventId','page','pageSize'].includes(key))delete base[key]}
 if(op==='EventCreate')base.timeZone='Asia/Shanghai'
 if(op==='ConnectionSave'&&!selected.value.connectionId){delete base.version;delete base.connectionId}
 if(op==='GrantSave'&&!selected.value.grantId){delete base.version;delete base.grantId}
 if(properties.operationId&&!base.operationId)base.operationId=newId()
 return Object.fromEntries(Object.entries(base).filter(([k,v])=>k in properties&&v!==undefined))
}
function can(op:string){
 if(unknown.value||busy.value)return false
 if(page.value.id==='O07'&&op!=='BatchUpload'&&!context.value.batchId)return false
 const targets:Record<string,string>={AccountUpdate:'accountId',GrantRevoke:'grantId',ConnectionRevoke:'connectionId',DeviceRevoke:'deviceId',CredentialReissue:'partyId',CredentialRevoke:'partyId',EmailResend:'partyId',GateIncrease:'partyId',SourcePull:'sourceId',DuplicateResolve:'rowId',JobRetry:'jobId',ExportDownload:'exportId',DeliveryRecord:'exportId'}
 if(targets[op]&&!selected.value[targets[op]]&&!context.value[targets[op]]&&!route.params[targets[op]])return false
 if(['PartyChange','PartyCancel'].includes(op)&&(!selected.value.partyId||selected.value.hasAdmissionHistory||eventStatus.value!=='READY'))return false
 if(op==='ReviewDecide'&&(!selected.value.reviewId||enumCode(selected.value.result)!=='PENDING'||eventStatus.value!=='READY'))return false
 if(['EventUpdate','PolicyUpdate','SourceSave','BatchCommit'].includes(op)&&eventStatus.value&&!['DRAFT','READY'].includes(eventStatus.value))return false
 if(['GateWalkIn','GateIncrease'].includes(op)&&eventStatus.value!=='CHECKIN_OPEN')return false
 if(op==='GateCorrect'&&(!selected.value.recordId||!['CHECKIN_OPEN','CHECKIN_PAUSED'].includes(eventStatus.value)))return false
 if(op==='BatchCommit'&&!selectedRows.value.length)return false
 if(op==='ExportCreate'&&!grants.value.some(g=>g.eventId===eventId.value&&g.allowedActions.includes('EXPORT')))return false
 if(op==='ExportDownload'&&enumCode(selected.value.status)!=='SUCCEEDED')return false
 if(op==='CleanupRetry'&&enumCode((selected.value.cleanupId?selected.value:results.value.CleanupDetail)?.status)!=='FAILED')return false
 if(op==='JobRetry'&&enumCode(selected.value.status)!=='FAILED')return false
 return true
}
async function begin(op:string){if(!can(op)&&operations[op].mutation)return;error.value='';active.value=op;form.value={...payloadFor(op),...(queryPayloads.value[op]||{})};modal.value=true
 if(op==='EventTransition'){const valid:Record<string,string[]>={DRAFT:['PUBLISH'],READY:['OPEN'],CHECKIN_OPEN:['PAUSE','END'],CHECKIN_PAUSED:['RESUME','END']};const allowed=valid[eventStatus.value]||[];form.value.action=allowed[0]}
 if(op==='GateWalkIn'&&policy.value){form.value.fields={};for(const [k,v]of Object.entries(policy.value.fields||{}))if((v as Obj).enabled&&k==='companionCount')form.value.fields[k]=0}
}
function defForDisplay(){const base=structuredClone(activeSchema.value)
 if(active.value==='GrantSave'&&form.value.role==='READER')base.required=[...base.required,'recipient','purpose','retentionNote']
 if(active.value==='GrantSave'&&!roles.value.includes('ADMIN'))base.properties.role={type:'string',enum:['CHECKER','READER']}
 if(active.value==='EventTransition'){const actions:Record<string,string[]>={DRAFT:['PUBLISH'],READY:['OPEN'],CHECKIN_OPEN:['PAUSE','END'],CHECKIN_PAUSED:['RESUME','END']};base.properties.action={type:'string',enum:actions[eventStatus.value]||[]}}
 if(active.value==='GateWalkIn'&&policy.value){base.properties.fields={...contract.schemas.PartyInput,properties:Object.fromEntries(Object.entries(contract.schemas.PartyInput.properties).filter(([k])=>policy.value!.fields[k]?.enabled)),required:Object.keys(policy.value.fields).filter(k=>policy.value!.fields[k]?.enabled&&policy.value!.fields[k]?.required)}}
 return base
}
async function handleResult(op:string,value:any){results.value[op]=value
 if(value instanceof Blob){const href=URL.createObjectURL(value);const a=document.createElement('a');a.href=href;a.download='UniGate-报名名单.'+(enumCode(selected.value.format)==='XLSX'?'xlsx':'csv');a.click();setTimeout(()=>URL.revokeObjectURL(href),1000);return}
 if(value?.eventId)context.value.eventId=value.eventId
 for(const key of ['batchId','previewVersion','sourceId','connectionId','resourceId','exportId','jobId','cleanupId','version'])if(value?.[key]!==undefined)context.value[key]=value[key]
 if(value?.status&&value?.capacity!==undefined)eventStatus.value=enumCode(value.status)
 if(op==='PolicyGet')policy.value=value
 if(op==='JobDetail'||op==='SourcePull')continuationBatch.value=enumCode(value?.kind)==='PULL'&&enumCode(value?.status)==='SUCCEEDED'&&value?.eventId===eventId.value&&uuidReference(value?.resultRef)?value.resultRef:''
 if(op===primary.value&&value&&!value.items&&!Array.isArray(value)&&typeof value==='object')selected.value=value
 if(value?.availableColumns)context.value.availableColumns=value.availableColumns
 if(value?.authorizationUrl){const url=new URL(value.authorizationUrl);if(url.protocol==='https:')window.location.assign(url.href)}
 if(value?.pairingCode&&value?.device){pairingLink.value=location.origin+'/terminal/activate#'+new URLSearchParams({deviceId:value.device.deviceId,pairingCode:value.pairingCode});pairingQr.value=await QRCode.toDataURL(pairingLink.value,{width:300})}
 if(op==='CredentialReissue'&&value?.access){const access=value.access;queryQr.value=await QRCode.toDataURL(location.origin+'/me/party#'+new URLSearchParams({queryToken:access.queryToken,...(access.changeToken?{changeToken:access.changeToken}:{})}),{width:300})}
 if(value?.ticketToken)queryQr.value=await QRCode.toDataURL(value.ticketToken,{width:300})
}
async function execute(op:string,payload:Obj){const recovering=unknown.value;busy.value=true;error.value='';notice.value='';try{const result=await call(op,clean(payload));await handleResult(op,result);viewOp.value=op;modal.value=false;if(!operations[op].mutation)queryPayloads.value[op]=payload
 if(operations[op].mutation){clearPending();unknown.value=false;notice.value=operations[op].status===202?'请求已受理，可在任务状态中查看进度。':'操作已完成。';if(op==='PasswordChange'){identity.staff=null;identity.staffCsrf='';await navigateAfterSuccess('/login?passwordChanged=1')};if(op==='EventCreate')await navigateAfterSuccess('/manage/events/'+result.eventId+'/settings');if(op==='BatchUpload'&&uuidReference(result.batchId))await navigateAfterSuccess('/manage/events/'+encodeURIComponent(eventId.value)+'/imports?batchId='+encodeURIComponent(result.batchId))}
 }catch(e){error.value=errorText(e);if(operations[op].mutation){if(e instanceof ApiError&&!e.uncertain&&!recovering)clearPending();else unknown.value=true}}finally{busy.value=false}}
async function navigateAfterSuccess(path:string){successNavigation.value=true;try{await router.push(path)}finally{successNavigation.value=false}}
async function submit(){if(busy.value||unknown.value)return;const op=active.value;const spec=operations[op];const payload=clean(form.value)
 const invalid=validate(payload,defForDisplay());if(invalid.length){error.value=invalid.join('\n');return}
 if(spec.mutation){if(!(await askConfirm(confirmText(op),{title:operations[op].summary,confirmText:'确认提交'})))return;savePending({operationId:payload.operationId,action:op,eventId:payload.eventId})}
 await execute(op,payload)
}
function confirmText(op:string){const texts:Record<string,string>={EventTransition:form.value.action==='OPEN'?'开始检票后将冻结公众报名、普通审核和取消。确认开始？':form.value.action==='END'?'结束后不能自动重新开放检票。确认结束？':'确认更改活动状态？',BatchCommit:`确认整批提交选中的 ${selectedRows.value.length} 行？任一冲突将整批退回。`,PartyCancel:'确认取消本组报名并使入场码失效？',GateWalkIn:'确认分配现场报名名额？此操作不代表已入场。',GateIncrease:'确认增加本组名额？此操作不代表已入场。',GateCorrect:'确认更正原核销人数？将追加更正记录。',CredentialRevoke:'确认撤销入场码？此操作不释放名额。',CredentialReissue:'确认补发？旧入场码将失效。',ConnectionRevoke:'确认撤销平台连接？',GrantRevoke:'确认撤销此活动授权？'};return texts[op]||'确认提交本次操作？'}
async function read(op:string,extra:Obj={}){const def=contract.schemas[operations[op].request];const payload=Object.fromEntries(Object.entries({...payloadFor(op),...(queryPayloads.value[op]||{}),...extra}).filter(([key])=>Object.hasOwn(def.properties||{},key)));const missing=(def.required||[]).filter((k:string)=>payload[k]===undefined||payload[k]==='');if(missing.length){await begin(op);return}await execute(op,payload)}
async function refresh(){viewOp.value=primary.value;if(page.value.id==='O07'){if(!context.value.batchId)return;await read('BatchDetail');if(['PREVIEWED','COMMITTED'].includes(enumCode(results.value.BatchDetail?.status)))await read('BatchRows',{page:currentPage.value});return}const fields=contract.schemas[operations[primary.value].request].properties||{};await read(primary.value,Object.hasOwn(fields,'page')?{page:currentPage.value}:{})}
async function select(row:Obj){selected.value=row;notice.value='已选中记录，可查看详情或执行适用操作。';context.value={eventId:eventId.value,...Object.fromEntries(Object.entries(context.value).filter(([k])=>['batchId','previewVersion'].includes(k))),...Object.fromEntries(Object.entries(row).filter(([k])=>/Id$/.test(k)||k==='version'||k==='previewVersion'))}
 if(row.eventId&&page.value.id==='O01'){await router.push('/manage/events/'+row.eventId+'/settings');return}
 if(row.partyId&&['O03','R02'].includes(page.value.id)){if(page.value.id==='O03'){await router.push('/manage/events/'+eventId.value+'/parties/'+row.partyId);return}await read('PartyDetail',{eventId:eventId.value,partyId:row.partyId});return}
 if(row.partyId&&page.value.operations.includes('PartyDetail')){try{const party=await call('PartyDetail',{eventId:eventId.value,partyId:row.partyId});results.value.PartyDetail=party;context.value.partyVersion=party.version;if(!row.recordId&&!row.reviewId)selected.value=party}catch(e){error.value=errorText(e)}}
}
function toggle(row:Obj,checked:boolean){if(!row.rowId)return;if(enumCode(row.status)!=='VALID'){error.value='仅可选择通过校验且已完成重复核对的行。';return}selectedRows.value=checked?[...new Set([...selectedRows.value,row.rowId])]:selectedRows.value.filter(x=>x!==row.rowId)}
async function resolveUnknown(){const p=getPending();if(!p?.eventId)return;busy.value=true;try{const r=await call('OperationResult',{eventId:p.eventId,operationId:p.operationId,action:p.action});if(['SUCCEEDED','REJECTED'].includes(enumCode(r.status))){unknown.value=false;clearPending();modal.value=false;notice.value='已查到原操作结果，请刷新并核对记录。';await handleResult('OperationResult',r)}else notice.value='原操作仍在处理中，请稍后再查。'}catch(e){error.value=errorText(e)}finally{busy.value=false}}
async function retryOriginal(){if(!unknown.value||!active.value||!form.value.operationId)return;await execute(active.value,form.value)}
function paging(delta:number){currentPage.value=Math.max(1,currentPage.value+delta);void refresh()}
async function close(){if(unknown.value||busy.value)return;if(operations[active.value]?.mutation&&!(await askConfirm('放弃尚未提交的内容？',{title:'放弃编辑',confirmText:'放弃',danger:true})))return;form.value={};modal.value=false}
async function makeSignupQr(){signupQr.value=await QRCode.toDataURL(signupLink.value,{width:360,margin:3})}
async function copyPairing(){try{await navigator.clipboard.writeText(pairingLink.value);notice.value='领取链接已复制。'}catch{error.value='暂时无法复制，请在目标终端扫描二维码。'}}
onMounted(async()=>{let restoredBatch=false;busy.value=true;try{if(eventId.value&&roles.value.includes('ORGANIZER')){const detail=await call('EventDetail',{eventId:eventId.value});eventStatus.value=enumCode(detail.status);results.value.EventDetail=detail;context.value.eventId=eventId.value}
 if(page.value.id==='O10')policy.value=await call('PolicyGet',{eventId:eventId.value})
 if(page.value.operations.includes('MyGrantList')||page.value.id==='O13'){grants.value=(await call('MyGrantList',{})).items||[];results.value.MyGrantList={items:grants.value}}
 if(page.value.id==='S03')results.value.EventOptions=await call('EventOptions',{})
 if(page.value.operations.includes('GrantEventOptions'))results.value.GrantEventOptions=await call('GrantEventOptions',{})
 if(page.value.id==='A01')results.value.AccountList=await call('AccountList',{page:1,pageSize:100})
 if(page.value.id==='O07'&&route.query.batchId!==undefined){restoredBatch=true;if(!uuidReference(route.query.batchId))error.value='名单导入链接无效，请重新上传或从来源拉取任务继续。';else{const batch=await call('BatchDetail',{eventId:eventId.value,batchId:route.query.batchId});if(batch.eventId!==eventId.value)throw new ApiError('RESOURCE_NOT_FOUND','未找到本活动的名单批次。',404);await handleResult('BatchDetail',batch);viewOp.value='BatchDetail'}}
 }catch(e){error.value=errorText(e)}finally{busy.value=false}if(!unknown.value&&!restoredBatch)await refresh()})
watch(()=>form.value.role,()=>{if(form.value.allowedActions)form.value.allowedActions=form.value.allowedActions.filter((x:string)=>options.value.allowedActions.some(o=>o.value===x))})
watch(()=>form.value.connectionId,async(id)=>{if(!id||!modal.value||!page.value.operations.includes('ResourceList'))return;try{results.value.ResourceList=await call('ResourceList',{connectionId:id,...(eventId.value?{eventId:eventId.value}:{})})}catch(e){error.value=errorText(e)}})
watch(()=>form.value.resourceId,async(id)=>{if(!id||!form.value.connectionId||!modal.value||!page.value.operations.includes('ResourceFields'))return;try{results.value.ResourceFields=await call('ResourceFields',{connectionId:form.value.connectionId,resourceId:id,...(eventId.value?{eventId:eventId.value}:{})})}catch(e){error.value=errorText(e)}})
onBeforeRouteLeave(()=>!unknown.value&&(!busy.value||successNavigation.value))
onBeforeUnmount(()=>{results.value={};selected.value={};form.value={};pairingQr.value='';pairingLink.value='';queryQr.value=''})
</script>
<template>
 <section>
  <p v-if="error" class="alert" role="alert">{{error}}</p><p v-if="notice" class="notice" role="status">{{notice}}</p>
  <div v-if="unknown" class="unknown-panel"><h2>结果待确认</h2><p>请先核对原操作，勿重新发起相同操作。</p><button v-if="pendingEvent" :disabled="busy" @click="resolveUnknown">查询原操作结果</button><p v-else>此操作需保留原填写内容，使用下方按钮重试同一次操作。若页面已刷新或会话失效，请在独立登录页使用当前有效密码登录，由管理员核对原记录；勿重复创建。</p><a v-if="!pendingEvent" class="button secondary" href="/login" target="_blank" rel="noopener noreferrer">打开独立登录页核对</a><button v-if="active&&form.operationId" class="secondary" :disabled="busy" @click="retryOriginal">重试原操作</button></div>
  <p v-if="page.id==='C02'" class="notice">查找到报名信息后，请参与者出示原入场码。无法出示时，请由活动负责人核对并补发。</p>
  <div v-if="eventId&&roles.includes('ORGANIZER')&&eventStatus!=='DRAFT'" class="toolbar"><button class="secondary" @click="signupQr?signupQr='':makeSignupQr()">活动报名二维码</button></div>
  <section v-if="signupQr" class="card pairing-card"><h2>扫码报名</h2><img :src="signupQr" alt="活动报名二维码"><a class="button secondary" :href="signupQr" download="UniGate-活动报名码.png">保存报名码</a><p><a :href="signupLink" target="_blank" rel="noopener">打开活动报名页</a></p></section>
  <div class="toolbar"><button v-if="dataOps.length>0||page.layout==='statistics'" class="secondary" :disabled="busy||unknown" @click="refresh">刷新</button><button v-for="op in actions.filter((name:string)=>!(page.id==='O07'&&!context.batchId&&name==='BatchUpload'))" :key="op" :disabled="!can(op)" @click="begin(op)">{{operations[op].summary}}</button><RouterLink v-if="page.id==='O08'" class="button" :to="'/manage/events/'+eventId+'/scan'">进入工作人员检票</RouterLink></div>
  <div v-if="importContinuation&&roles.includes('ORGANIZER')&&['O06','O15'].includes(page.id)" class="card"><h2>来源名单已准备好</h2><p>继续核对字段映射和记录，再提交入场资格。</p><RouterLink class="button" :to="importContinuation">继续名单导入</RouterLink></div>
  <div v-if="dataOps.length>1" class="tabs"><button v-for="op in dataOps" :key="op" class="tab" :class="{active:viewOp===op}" :disabled="busy||unknown||(page.id==='O07'&&!context.batchId)" @click="read(op)">{{operations[op].summary}}</button></div>
  <section v-if="page.id==='O07'&&!context.batchId" class="card"><h2>导入名单</h2><p>上传名单后，依次映射字段、核对行内容，再提交选中的有效记录。</p><p class="muted">支持 CSV 和 XLSX。飞书问卷和腾讯文档可在“平台来源与字段映射”中连接并拉取。</p><button :disabled="busy||unknown" @click="begin('BatchUpload')">上传名单</button><RouterLink class="button secondary" :to="'/manage/events/'+eventId+'/sources'">连接平台来源</RouterLink></section>
  <section v-else-if="dataOps.length>0||page.layout==='statistics'" class="card">
   <div class="card-heading"><h2>{{operations[viewOp||primary]?.summary||page.title}}</h2><button v-if="!operations[viewOp||primary]?.mutation" class="quiet" :disabled="busy||unknown" @click="begin(viewOp||primary)">筛选与查询</button></div>
   <p v-if="busy" role="status" class="muted">正在读取…</p>
   <template v-if="page.layout==='statistics'&&current"><div class="metric-grid"><div v-for="key in ['applicationGroups','approvedGroups','allocatedPeople','netAdmissions','unusedPlaces','pendingGroups']" :key="key"><strong>{{current[key]??'—'}}</strong><span>{{({applicationGroups:'申请组数',approvedGroups:'批准组数',allocatedPeople:'有效名额',netAdmissions:'累计入场',unusedPlaces:'未用名额',pendingGroups:'待审核组数'} as Record<string,string>)[key]}}</span></div></div><p class="muted">数据截至 {{display(current.asOf)}}</p></template>
   <ResultDisplay v-else :value="current" :columns="columns" :selectable="page.id==='O07'" :selected="selectedRows" :references="references" @select="select" @toggle="toggle" />
   <div v-if="current?.total!==undefined" class="pagination"><button class="secondary" :disabled="currentPage<=1||busy||unknown" @click="paging(-1)">上一页</button><span>第 {{currentPage}} 页</span><button class="secondary" :disabled="currentPage*20>=current.total||busy||unknown" @click="paging(1)">下一页</button></div>
  </section>
  <section v-if="Object.keys(selected).length" class="card"><div class="card-heading"><h2>当前选择</h2><button class="quiet" :disabled="busy||unknown" @click="selected={}">取消选择</button></div><ResultDisplay :value="selected"/><ResultDisplay v-if="results.PartyDetail&&selected.reviewId" :value="results.PartyDetail"/></section>
  <section v-if="pairingQr" class="card pairing-card"><h2>领取检票终端</h2><p>请在目标手机或平板扫码领取。领取信息仅一次有效，请妥善保管。</p><img :src="pairingQr" alt="检票终端领取二维码"><button class="secondary" @click="copyPairing">复制领取链接</button><button class="quiet" @click="pairingQr='';pairingLink=''">关闭领取信息</button></section>
  <section v-if="queryQr" class="card pairing-card"><h2>本组入口</h2><img :src="queryQr" alt="本组入口二维码"><button class="quiet" @click="queryQr=''">关闭</button></section>
  <Teleport to="body"><div v-if="modal" class="modal-backdrop" @click.self="close"><section class="modal" role="dialog" aria-modal="true" :aria-label="operations[active]?.summary"><div class="card-heading"><h2>{{operations[active]?.summary}}</h2><button class="quiet" :disabled="busy||unknown" @click="close" aria-label="关闭">关闭</button></div><form @submit.prevent="submit"><DynamicFields v-model="form" :definition="defForDisplay()" :page="page" :operation="active" :options="options" :locked="locked" :time-zone="results.EventDetail?.timeZone||'Asia/Shanghai'"/><p v-if="active==='BatchCommit'" class="notice">将整批提交 {{selectedRows.length}} 行；发生冲突时整批退回。</p><p v-if="error" class="alert" role="alert">{{error}}</p><p v-if="unknown" class="alert">结果待确认，请关闭弹层后核对原操作。</p><div class="form-actions"><button type="button" class="secondary" :disabled="busy" @click="unknown?modal=false:close()">{{unknown?'查看原操作':'取消'}}</button><button :disabled="busy||unknown">{{busy?'正在处理…':operations[active]?.mutation?'确认提交':'查询'}}</button></div></form></section></div></Teleport>
 </section>
</template>
