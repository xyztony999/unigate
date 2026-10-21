import source from '../generated/contract.json'
export type Obj = Record<string, any>
export const contract = source as unknown as { schemas: Record<string, Obj>; operations: Obj[]; enums: Record<string, Record<string,string>>; pages: Obj[]; sourceSha256:string }
export const operations = Object.fromEntries(contract.operations.map(o => [o.name, o]))
export function schema(s: Obj): Obj { return s?.$ref ? schema(contract.schemas[s.$ref.split('/').pop()!]) : s || {} }
export function label(key: string, page?: Obj, operation?: string): string {
  const binding = [...(page?.requestFields || []), ...(page?.responseFields || [])].find((b:Obj) => b.path === key && (!operation || b.operation === operation))
  if (binding && binding.label !== key && !/^[a-zA-Z][a-zA-Z0-9.]*$/.test(binding.label)) return binding.label
  const tail = key.replace(/\[\]/g,'').split('.').pop()!
  return labels[tail] || tail
}
export const labels: Record<string,string> = {
  temporaryPassword:'初始密码',additionalCount:'新增人数',allowedActions:'动作权限',allowedFields:'可见基础字段',allowedEventIds:'可用活动',contact:'报名联系方式',correctedCount:'更正人数',cleanupId:'清理任务',cursor:'下一页标记',existingPartyId:'关联已有报名组',existingPendingPartyId:'关联待审报名组',filter:'筛选条件',from:'开始时间',to:'截止时间',kind:'任务类型',policy:'报名规则',result:'处理结果',sourceLabel:'名单来源名称',partyVersion:'报名组版本',applicationGroups:'申请组数',pendingGroups:'待审核组数',approvedGroups:'批准组数',allocatedPeople:'有效名额',grossAdmissions:'原始核销人数',correctedPeople:'累计更正人数',netAdmissions:'累计入场',unusedPlaces:'未用名额',walkInGroups:'现场补录组数',onsiteAddedPlaces:'现场新增名额',admissionRate:'入场率',validRows:'可提交行',errorRows:'错误行',suspectRows:'待核对行',dueAt:'预计清理时间',processedCount:'已完成数量',failedCount:'未完成数量',completedAt:'完成时间',attempts:'尝试次数',pairingExpiresAt:'领取截止时间',lastSeenAt:'最后连接时间',resultCode:'处理结果',policyVersion:'规则版本',sourceVersion:'来源版本',boundEvents:'获准活动'
  ,
  name:'姓名',phone:'手机号',email:'邮箱',companionCount:'随行人数',fields:'报名信息',username:'登录账号',password:'密码',displayName:'姓名',oldPassword:'当前密码',newPassword:'新密码',title:'活动名称',venue:'地点',startsAt:'开始时间',endsAt:'结束时间',timeZone:'活动时区',registrationStartsAt:'报名开始',registrationEndsAt:'报名截止',capacity:'总名额',status:'状态',version:'资料版本',reason:'原因',query:'搜索',page:'页号',pageSize:'每页条数',role:'角色',roles:'角色',enabled:'启用',required:'必填',approvalMode:'审核方式',phoneVerificationEnabled:'验证手机号',selfEditEnabled:'允许自助修改',selfCancelEnabled:'允许自助取消',selfServiceEndsAt:'自助操作截止',retentionDays:'明细保留天数',notice:'报名告知',noticeVersion:'告知版本',action:'操作',decision:'审核决定',requestedSize:'申请人数',approvedSize:'已批准人数',netAdmitted:'累计入场',remainingCount:'剩余名额',allocatedCount:'已分配名额',pendingIncrease:'待审核增员',partyReference:'报名参考号',count:'本次人数',progress:'入场进度',createdAt:'创建时间',createdFrom:'创建开始',createdTo:'创建截止',sourceType:'来源',accountId:'工作人员',eventId:'活动',partyId:'报名组',grantId:'授权',deviceId:'终端',reviewId:'申请',recordId:'核销记录',batchId:'导入批次',connectionId:'平台连接',resourceId:'平台资源',sourceId:'来源配置',exportId:'导出任务',jobId:'后台任务',rowId:'导入行',recipient:'接收方',purpose:'用途',retentionNote:'留存约定',expiresAt:'有效期',actions:'动作权限',columns:'可见字段',fieldsAllowed:'可见字段',accountKind:'账号类型',provider:'服务平台',clientId:'应用标识',clientSecret:'应用密钥',redirectUri:'授权回调地址',credentials:'连接凭据',mapping:'字段映射',eligibility:'来源资格',nameColumn:'姓名列',phoneColumn:'手机号列',emailColumn:'邮箱列',countColumn:'人数列',countMode:'人数含义',externalIdColumn:'原始编号列',file:'名单文件',selectedRowIds:'选中行',resolution:'处理方式',targetPartyId:'关联报名组',scopes:'清理范围',scope:'范围',format:'文件格式',deliveryMethod:'交付方式',channel:'验证方式',code:'验证码',challengeId:'验证申请',pairingCode:'终端领取码',ticketToken:'入场码',rotateAccess:'同时重置私密入口',includeContacts:'包括联系方式',asOf:'数据截至',total:'共计',items:'记录',summary:'概览',message:'处理说明',errorCode:'错误类型',lastError:'上次错误',state:'处理状态',mode:'终端模式',cleanupAfter:'明细清理时间',retryable:'允许重试',registrationAvailable:'当前开放报名',canChange:'允许修改',canCancel:'允许取消',hasPendingIncrease:'有待审增员',phoneVerified:'已验证手机',updatedAt:'更新时间',committedAt:'记录时间',committedCount:'本次入场',requestedCount:'申请入场',replayed:'原操作结果',showAsNewAdmission:'本次新登记',at:'时间',actorName:'操作人',operation:'操作',operationId:'操作编号',targetType:'对象类型',targetId:'对象编号',bindingId:'资源绑定',active:'有效',disabled:'停用',expectedPartyVersion:'报名组版本',previewVersion:'预览版本',initialPassword:'初始密码',accountDisplayName:'姓名',accountUsername:'登录账号',fieldRules:'基础字段',submittedAt:'提交时间',approvedAt:'通过时间',reviewedAt:'审核时间',failureReason:'未完成原因',availableColumns:'可映射列',availableConnections:'可用连接',reference:'参考号'
}
export function display(v:any):string {
  if(v===null || v===undefined) return '—'
  if(typeof v === 'boolean') return v ? '是' : '否'
  if(typeof v === 'object' && 'code' in v && 'label' in v) return v.label
  if(Array.isArray(v)) return v.map(display).join('、') || '—'
  if(typeof v === 'object') return Object.entries(v).filter(([k])=>!secret(k)).map(([k,x])=>`${label(k)}：${display(x)}`).join('；') || '—'
  if(typeof v === 'string' && /^\d{4}-\d\d-\d\dT/.test(v)) return new Date(v).toLocaleString('zh-CN', {timeZone:'Asia/Shanghai',hour12:false}) + '（上海）'
  return String(v)
}
export const secret = (k:string) => /token|password|secret|pairingCode|proof|clientContextId|scanContext|queryUrl/i.test(k)
export const enumCode = (v:any):string => typeof v==='object' && v ? v.code : v || ''
export const readPath = (o:Obj,p:string):any => p.split('.').reduce((v,k)=>v?.[k],o)
export function defaults(s:Obj):Obj {
  s=schema(s); const out:Obj={}
  for(const [k,raw] of Object.entries(s.properties || {})) {
    const f=schema(raw as Obj)
    if(f.default!==undefined) out[k]=f.default
    else if(f.type==='object' && (s.required||[]).includes(k)) out[k]=defaults(f)
    else if(f.type==='array' && (s.required||[]).includes(k)) out[k]=[]
    else if(f.type==='boolean' && (s.required||[]).includes(k)) out[k]=false
  }
  return out
}
export function clean(v:any):any {
  if(Array.isArray(v)) return v.map(clean)
  if(v && typeof v==='object' && !(v instanceof File)) return Object.fromEntries(Object.entries(v).filter(([,x])=>x!=='' && x!==undefined).map(([k,x])=>[k,clean(x)]))
  return v
}
