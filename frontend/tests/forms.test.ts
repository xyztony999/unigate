import { describe,it,expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DynamicFields from '../src/components/DynamicFields.vue'
import ResultDisplay from '../src/components/ResultDisplay.vue'
import { contract,schema } from '../src/lib/contract'
describe('业务表单与名单呈现',()=>{
 it('字段策略分组包含开关和必填选项，不暴露操作ID与版本输入',()=>{
  const wrapper=mount(DynamicFields,{props:{modelValue:{policy:{fields:{name:{enabled:true,required:true}}}},definition:contract.schemas.PolicyUpdateRequest,locked:['eventId']}})
  expect(wrapper.findAll('input[type=checkbox]').length).toBeGreaterThan(4)
  expect(wrapper.text()).toContain('姓名');expect(wrapper.text()).not.toContain('operationId');expect(wrapper.find('textarea').exists()).toBe(false)
 })
 it('动态映射选择官方列code，展示label且能更改',async()=>{
  const wrapper=mount(DynamicFields,{props:{modelValue:{countMode:'COMPANIONS'},definition:contract.schemas.FieldMapping,options:{nameColumn:[{value:'col-a',label:'家长姓名'}]}}})
  const select=wrapper.findAll('select')[0];expect(select.text()).toContain('家长姓名');await select.setValue('col-a')
  expect(wrapper.emitted('update:modelValue')?.[0][0]).toMatchObject({nameColumn:'col-a'})
 })
 it('响应只显示实际返回字段，秘密和内部ID不出现在详情中',()=>{
  const wrapper=mount(ResultDisplay,{props:{value:{partyId:'private-id',name:'测试家长',csrfToken:'csrf-secret',ticketToken:'ticket-secret',phoneVerificationProof:'proof-secret'}}})
  expect(wrapper.text()).toContain('测试家长');expect(wrapper.text()).not.toMatch(/csrf-secret|ticket-secret|proof-secret|private-id/)
 })
 it('所有确认页面路由唯一，85操作对应原合同且请求引用可解析',()=>{
  expect(contract.pages).toHaveLength(33);expect(new Set(contract.pages.map(p=>p.route)).size).toBe(33)
  expect(contract.operations).toHaveLength(85)
  for(const op of contract.operations){expect(contract.schemas[op.request]).toBeDefined();for(const field of Object.values(contract.schemas[op.request].properties||{}))expect(schema(field as any)).toBeDefined()}
 })
 it('账号创建使用工作人员角色多选、密码控件和整数容量控件，不包含参与者账号',async()=>{
  const account=mount(DynamicFields,{props:{modelValue:{roles:[]},definition:contract.schemas.AccountCreateRequest}})
  expect(account.find('input[type=password]').attributes('minlength')).toBe('12');expect(account.text()).not.toContain('参与者')
  const role=account.findAll('label.check-choice').find(x=>x.text()==='检票员')!;await role.find('input').setValue(true)
  expect(account.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({roles:['CHECKER']})
  const display=account.findAll('label.field').find(x=>x.text().startsWith('姓名'))!;await display.find('input').setValue('工作人员')
  expect(account.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({displayName:'工作人员'})
  const event=mount(DynamicFields,{props:{modelValue:{capacity:100},definition:contract.schemas.EventCreateRequest}});await event.find('input[type=number]').setValue('120')
  expect(event.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({capacity:120})
 })
 it('规则开关产生布尔值；活动时间按活动时区而非浏览器时区提交',async()=>{
  const policy=mount(DynamicFields,{props:{modelValue:{selfEditEnabled:false},definition:contract.schemas.EventPolicyInput}})
  const toggle=policy.findAll('label.toggle-field').find(x=>x.text()==='允许自助修改')!;await toggle.find('input').setValue(true)
  expect(policy.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({selfEditEnabled:true})
  const event=mount(DynamicFields,{props:{modelValue:{timeZone:'Asia/Shanghai'},definition:contract.schemas.EventCreateRequest}});await event.find('input[type=datetime-local]').setValue('2026-11-01T10:00')
  expect(event.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({startsAt:'2026-11-01T02:00:00.000Z'})
  expect(event.text()).toContain('时区：Asia/Shanghai')
 })
 it('CSV/XLSX上传传递真实File，长原因仍使用可读文本框',async()=>{
  const upload=mount(DynamicFields,{props:{modelValue:{},definition:contract.schemas.BatchUploadRequest}});const file=new File(['name\n测试家长'],'roster.csv',{type:'text/csv'});const input=upload.find('input[type=file]')
  expect(input.attributes('accept')).toBe('.csv,.xlsx');Object.defineProperty(input.element,'files',{value:[file]});await input.trigger('change')
  expect(upload.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({file})
  const grant=mount(DynamicFields,{props:{modelValue:{},definition:contract.schemas.GrantSaveRequest}});await grant.find('textarea').setValue('活动结束后按约定删除')
  expect(grant.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({retentionNote:'活动结束后按约定删除'})
 })
 it('列表按后端实际字段展示，授权活动使用标题；选择和详情是独立动作',async()=>{
  const row={rowId:'row1',eventId:'event1',status:{code:'VALID',label:'可提交'},fields:{name:'测试家长'}}
  const wrapper=mount(ResultDisplay,{props:{value:{items:[row],total:1},columns:[{path:'items[].eventId',label:'活动',visible:true},{path:'items[].status',label:'状态',visible:true}],selectable:true,selected:[],references:{event1:'家长见面会'}}})
  expect(wrapper.text()).toContain('家长见面会');expect(wrapper.text()).not.toContain('event1');expect(wrapper.text()).toContain('可提交')
  await wrapper.find('input[type=checkbox]').setValue(true);expect(wrapper.emitted('toggle')?.[0]).toEqual([row,true]);await wrapper.find('button').trigger('click');expect(wrapper.emitted('select')?.[0][0]).toEqual(row)
  const empty=mount(ResultDisplay,{props:{value:{items:[],total:0}}});expect(empty.text()).toContain('暂无符合条件的记录')
 })
})
