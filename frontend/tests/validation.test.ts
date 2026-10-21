import { describe,it,expect } from 'vitest'
import { validate,toZonedInput,fromZonedInput } from '../src/lib/validation'
import { contract } from '../src/lib/contract'
describe('表单合同边界',()=>{
 it('未选择账号、过短临时密码及空角色不能通过',()=>{const errors=validate({username:'a',displayName:'A',temporaryPassword:'short',roles:[]},contract.schemas.AccountCreateRequest);expect(errors.length).toBeGreaterThan(1)})
 it('校方授权必须填写接收方/用途/留存约定',()=>{const definition=contract.schemas.GrantSaveRequest;const errors=validate({operationId:'o',eventId:'e',accountId:'a',role:'READER',allowedFields:[],allowedActions:[],expiresAt:'2026-10-23T00:00:00Z'},definition);expect(errors.join('')).toContain('接收方');expect(errors.join('')).toContain('用途');expect(errors.join('')).toContain('留存约定')})
 it('只接受既定活动动作枚举和有效IANA时区',()=>{expect(validate({operationId:'o',eventId:'e',version:1,action:'OPEN_CHECKIN'},contract.schemas.EventTransitionRequest).join('')).toContain('有效');expect(validate({timeZone:'wrong/zone'},{type:'object',properties:{timeZone:{type:'string'}}}).join('')).toContain('时区')})
 it('时间按活动时区呈现并往返为同一时刻',()=>{const value='2026-10-21T01:30:00.000Z';expect(toZonedInput(value,'Asia/Shanghai')).toBe('2026-10-21T09:30');expect(fromZonedInput('2026-10-21T09:30','Asia/Shanghai')).toBe(value);expect(toZonedInput(value,'Australia/Melbourne')).toBe('2026-10-21T12:30')})
})
