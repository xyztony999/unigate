<script setup lang="ts">
import { ref } from 'vue'
import { useRouter,useRoute } from 'vue-router'
import { call,newId,errorText,identity } from '../lib/api'
import { enumCode } from '../lib/contract'
import ThemeToggle from '../components/ThemeToggle.vue'
const route=useRoute(),router=useRouter(),username=ref(''),password=ref(''),busy=ref(false),error=ref('')
async function submit(){busy.value=true;error.value='';try{const result=await call('Login',{operationId:newId(),username:username.value,password:password.value});password.value='';const roles=result.roles.map(enumCode); await router.push(result.mustChangePassword?'/account/password':roles.includes('ORGANIZER')?'/manage/events':roles.includes('ADMIN')?'/admin/accounts':'/account/access')}catch(e){error.value=errorText(e)}finally{busy.value=false}}
</script>
<template><main class="public-shell"><div class="public-brand">UniGate<ThemeToggle/></div><section class="center-card login-card"><h1>工作人员登录</h1><p class="muted">登录后管理活动、名单与现场检票。</p><p v-if="route.query.passwordChanged" class="notice">密码已更新，请使用新密码重新登录。</p><form @submit.prevent="submit"><label class="field"><span>登录账号</span><input v-model="username" required autocomplete="username" maxlength="80"></label><label class="field"><span>密码</span><input v-model="password" required type="password" autocomplete="current-password" maxlength="256"></label><p class="alert" role="alert" v-if="error">{{error}}</p><button class="full" :disabled="busy">{{busy?'正在登录…':'登录'}}</button></form></section></main></template>
