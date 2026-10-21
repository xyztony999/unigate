import { beforeEach,afterEach,describe,it,expect,vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount,flushPromises } from '@vue/test-utils'
import { createMemoryHistory,createRouter } from 'vue-router'
import LoginView from '../src/views/LoginView.vue'
import { identity } from '../src/lib/api'
beforeEach(()=>{sessionStorage.clear();Object.assign(identity,{staff:null,anonCsrf:'anon-csrf',clientContextId:'context'})})
afterEach(()=>vi.unstubAllGlobals())
async function login(){const blank=defineComponent({template:'<p>修改密码</p>'});const router=createRouter({history:createMemoryHistory(),routes:[{path:'/login',component:LoginView},{path:'/account/password',component:blank},{path:'/admin/accounts',component:blank},{path:'/account/access',component:blank},{path:'/manage/events',component:blank}]});await router.push('/login');await router.isReady();const wrapper=mount(defineComponent({template:'<RouterView />'}),{global:{plugins:[router]}});await wrapper.find('input[autocomplete=username]').setValue('operator');await wrapper.find('input[type=password]').setValue('private-password');return{wrapper,router}}
describe('工作人员登录页面',()=>{
 it('初始密码账号只能进入修改密码，登录请求不把密码放URL',async()=>{
  const fetch=vi.fn(async()=>new Response(JSON.stringify({code:'OK',data:{accountId:'a',displayName:'工作人员',username:'operator',roles:[{code:'ADMIN',label:'管理员'}],csrfToken:'staff-csrf',mustChangePassword:true}}),{status:200}));vi.stubGlobal('fetch',fetch);const{wrapper,router}=await login();await wrapper.find('form').trigger('submit');await flushPromises();expect(router.currentRoute.value.path).toBe('/account/password');const [url,init]=fetch.mock.calls[0] as unknown as[string,RequestInit];expect(url).not.toContain('private-password');expect(JSON.parse(String(init.body))).toMatchObject({username:'operator',password:'private-password'});expect(sessionStorage.length).toBe(0);wrapper.unmount()
 })
 it('认证拒绝停留登录页并显示服务端可读消息',async()=>{
  vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({code:'AUTH_FAILED',message:'账号或密码错误'}),{status:401})));const{wrapper,router}=await login();await wrapper.find('form').trigger('submit');await flushPromises();expect(wrapper.text()).toContain('账号或密码错误');expect(router.currentRoute.value.path).toBe('/login');expect(identity.staff).toBeNull();wrapper.unmount()
 })
})
