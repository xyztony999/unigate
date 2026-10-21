<script setup lang="ts">
import { computed, onMounted, ref,watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { contract, enumCode, type Obj } from './lib/contract'
import { identity, initStaff, call, newId, errorText, getPending, bootstrap } from './lib/api'
const route=useRoute(),router=useRouter(),ready=ref(false),error=ref(''),events=ref<Obj[]>([])
const page=computed(()=>route.meta.page as Obj)
const publicPage=computed(()=>['P01','P02','P03','P04','S01','S04'].includes(page.value?.id))
const roles=computed(()=>identity.staff?.roles.map(enumCode)||[])
const nav=computed(()=>contract.pages.filter(p=>!['S01','S04','O04','O09','C02','C03'].includes(p.id)&&!p.id.startsWith('P')&&p.roles.some((r:string)=>roles.value.includes(r))))
const eventId=computed(()=>String(route.params.eventId||events.value[0]?.eventId||''))
function href(p:Obj) { return p.route.replace(':eventId',eventId.value).replace(':partyId','') }
async function load() { try { await router.isReady();if(route.name==='P04'){await bootstrap();return} await initStaff(); if(identity.staff&&!identity.staff.mustChangePassword&&roles.value.some(r=>['ORGANIZER','CHECKER','READER'].includes(r))) events.value=(await call('EventList',{page:1,pageSize:100})).items||[] } catch(e){error.value=errorText(e)} finally{ready.value=true} }
async function logout() { if(getPending()){error.value='仍有结果待确认的操作，请先核对。';return} try{await call('Logout',{operationId:newId()}); await router.push('/login')}catch(e){error.value=errorText(e)} }
function changeEvent(id:string){void router.push(roles.value.includes('ORGANIZER')?'/manage/events/'+id+'/statistics':roles.value.includes('CHECKER')?'/check/events/'+id+'/scan':'/read/events/'+id+'/statistics')}
onMounted(load)
watch(()=>identity.staff?.accountId,async(id)=>{if(id&&roles.value.some(r=>['ORGANIZER','CHECKER','READER'].includes(r))){try{events.value=(await call('EventList',{page:1,pageSize:100})).items||[]}catch(e){error.value=errorText(e)}}else events.value=[]})
</script>
<template>
  <div v-if="!ready" class="loading" role="status">正在连接活动服务…</div>
  <template v-else>
    <template v-if="publicPage"><RouterView :key="route.fullPath" /></template>
    <div v-else-if="!identity.staff" class="center-card"><h1>请先登录</h1><p>使用获授权的工作人员账号继续。</p><RouterLink class="button" to="/login">工作人员登录</RouterLink></div>
    <div v-else-if="identity.staff.mustChangePassword && page.id!=='S02'" class="center-card"><h1>请设置新密码</h1><p>首次登录后需要更新密码。</p><RouterLink class="button" to="/account/password">修改密码</RouterLink></div>
    <div v-else-if="!page.roles.some((r:string)=>roles.includes(r))" class="center-card"><h1>此页面暂不可访问</h1><RouterLink to="/account/access">查看我的授权</RouterLink></div>
    <div v-else class="app-shell">
      <aside class="sidebar"><a class="brand" href="/">UniGate<span>活动服务</span></a><label v-if="events.length" class="event-choice">当前活动<select :value="eventId" @change="changeEvent(($event.target as HTMLSelectElement).value)"><option v-for="e in events" :key="e.eventId" :value="e.eventId">{{e.title}}</option></select></label><nav><RouterLink v-for="p in nav.filter(p=>!p.route.includes(':eventId')||eventId)" :key="p.id" :to="href(p)" :class="{active:p.id===page.id}">{{p.title}}</RouterLink></nav><button class="quiet" @click="logout">退出登录</button></aside>
      <main class="workspace"><header class="workspace-header"><div><p class="eyebrow">UniGate</p><h1>{{page.title}}</h1></div><span>{{identity.staff.displayName}}</span></header><p v-if="error" role="alert" class="alert">{{error}}</p><RouterView :key="route.fullPath" /></main>
    </div>
  </template>
</template>
