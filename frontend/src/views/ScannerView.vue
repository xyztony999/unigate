<script setup lang="ts">
import { computed, nextTick,onMounted,onBeforeUnmount,ref,watch } from 'vue'
import { useRoute,onBeforeRouteLeave } from 'vue-router'
import { BrowserQRCodeReader,type IScannerControls } from '@zxing/browser'
import { createGate } from '../lib/gate'
import { display,enumCode } from '../lib/contract'
const route=useRoute(),kiosk=computed(()=>route.name==='P04')
const {gate,connect,resolve,commit,recover,retrySame,reset}=createGate()
const video=ref<HTMLVideoElement>(),token=ref(''),cameraError=ref(''),cameraActive=ref(false),help=ref(false)
let controls:IScannerControls|undefined;let countdown:ReturnType<typeof setTimeout>|undefined;let idle:ReturnType<typeof setTimeout>|undefined;let poll:ReturnType<typeof setInterval>|undefined
const scanning=computed(()=>['READY','UNKNOWN'].includes(gate.state))
function stopCamera(){controls?.stop();controls=undefined;cameraActive.value=false}
async function startCamera(){if(!video.value||!scanning.value)return;cameraError.value='';try{const reader=new BrowserQRCodeReader();controls=await reader.decodeFromVideoDevice(undefined,video.value,(result)=>{if(result&&scanning.value){stopCamera();void resolve(result.getText())}});cameraActive.value=true}catch{cameraError.value='无法使用相机，请使用扫码枪或联系工作人员。'}}
async function scan(){const raw=token.value.trim();token.value='';if(raw)await resolve(raw)}
function activity(){if(idle)clearTimeout(idle);if(kiosk.value&&gate.state==='CONFIRMING')idle=setTimeout(()=>{void reset()},30000)}
watch(()=>gate.state,async(state)=>{if(!['READY','UNKNOWN'].includes(state))stopCamera();if(countdown)clearTimeout(countdown);if(state==='SUCCESS'&&kiosk.value)countdown=setTimeout(()=>{void reset()},5000);activity();if(state==='READY'){await nextTick();document.getElementById('scan-input')?.focus({preventScroll:true})}})
function online(){if(gate.state==='UNKNOWN')void recover();else void connect()}
function offline(){stopCamera();if(gate.state==='SUBMITTING'){gate.state='UNKNOWN';gate.message='结果待确认，请勿重复扫码或提交。'}else if(gate.state!=='UNKNOWN'){gate.state='BLOCKED';gate.message='本机暂时无法检票，请联系工作人员。'}}
onMounted(async()=>{await connect();poll=setInterval(()=>{if(['READY','BLOCKED'].includes(gate.state))void connect()},15000);window.addEventListener('online',online);window.addEventListener('offline',offline)})
onBeforeUnmount(()=>{stopCamera();if(countdown)clearTimeout(countdown);if(idle)clearTimeout(idle);if(poll)clearInterval(poll);window.removeEventListener('online',online);window.removeEventListener('offline',offline);gate.preview=null;gate.result=null})
onBeforeRouteLeave(()=>!['UNKNOWN','SUBMITTING'].includes(gate.state))
</script>
<template>
 <main class="scanner-page" :class="{kiosk}" @pointerdown="activity" @keydown="activity">
  <header class="scan-header"><div><p class="muted">{{kiosk?'自助检票':'工作人员检票'}}</p><h1>{{gate.device?.name||'检票终端'}}</h1></div><span class="badge" :class="{warning:gate.state==='BLOCKED'||gate.state==='UNKNOWN'}">{{gate.state==='BLOCKED'?'暂不可检票':gate.state==='UNKNOWN'?'结果待确认':'检票服务'}}</span></header>
  <div v-if="['READY','RESOLVING','CONFIRMING'].includes(gate.state)" class="scan-columns">
   <section class="camera-panel"><video ref="video" muted playsinline :hidden="!cameraActive"/><div class="scan-frame" aria-hidden="true"><span>▦</span></div><h2>{{gate.state==='CONFIRMING'?'已识别，扫描已暂停':'请将入场码对准扫描区'}}</h2><button v-if="!cameraActive&&gate.state==='READY'" class="camera-button" @click="startCamera">开启相机扫码</button><p v-if="cameraError" role="alert">{{cameraError}}</p><form v-if="gate.state==='READY'" class="scanner-input" @submit.prevent="scan"><label class="sr-only" for="scan-input">入场码</label><input id="scan-input" v-model="token" type="password" autocomplete="off" placeholder="扫码枪输入入场码" required><button>识别</button></form></section>
   <section class="confirm-panel" aria-live="polite">
    <template v-if="gate.state==='CONFIRMING'&&gate.preview"><h2>这次共有几位入场？</h2><p class="muted">剩余可入场</p><div class="remaining">{{gate.preview.remainingCount}}<span>人</span></div><p class="counts">已批准 {{gate.preview.approvedSize}} 人 · 已入场 {{gate.preview.netAdmitted}} 人</p><label class="quantity-label" for="count">本次入场人数</label><div class="stepper"><button aria-label="减少一人" :disabled="gate.count<=1" @click="gate.count--">−</button><input id="count" v-model.number="gate.count" type="number" min="1" :max="gate.preview.remainingCount" step="1" inputmode="numeric"><button aria-label="增加一人" :disabled="gate.count>=gate.preview.remainingCount" @click="gate.count++">＋</button></div><button class="confirm-button" :disabled="!Number.isInteger(gate.count)||gate.count<1||gate.count>gate.preview.remainingCount" @click="commit">确认 {{gate.count}} 人入场</button><button class="quiet" @click="reset">暂不检票，返回扫码</button></template>
    <template v-else><h2>{{gate.state==='RESOLVING'?'正在识别入场码…':'请出示您的入场码'}}</h2><p>同组人员可以分批入场</p><p class="muted">识别后，请确认本次入场人数。</p></template>
   </section>
  </div>
  <section v-else class="gate-result" :class="{success:gate.state==='SUCCESS',warning:gate.state==='UNKNOWN',rejected:gate.state==='REJECTED'}" role="status" aria-live="assertive"><div class="result-symbol">{{gate.state==='SUCCESS'?'✓':gate.state==='SUBMITTING'?'…':gate.state==='KNOWN_REPLAY'?'↺':'!'}}</div><h2>{{gate.state==='SUBMITTING'?'正在确认，请稍候…':gate.message}}</h2><template v-if="gate.result"><p>本次登记 {{gate.result.committedCount}} 人<span v-if="gate.result.remainingCount!==undefined"> · 剩余 {{gate.result.remainingCount}} 人</span></p><p v-if="gate.result.committedAt">{{display(gate.result.committedAt)}}</p></template><template v-if="gate.state==='UNKNOWN'"><p>请联系现场工作人员核对。</p><button @click="recover">查询原操作结果</button><button v-if="gate.pendingRequest" class="secondary" @click="retrySame">重试本次操作</button><div v-if="!gate.preview" class="recovery-camera"><video ref="video" muted playsinline :hidden="!cameraActive"/><button v-if="!cameraActive" class="secondary" @click="startCamera">扫描原组入场码核对</button><p v-if="cameraError">{{cameraError}}</p><form class="scanner-input" @submit.prevent="scan"><label class="sr-only" for="original-ticket">原组入场码</label><input id="original-ticket" v-model="token" type="password" required autocomplete="off" placeholder="重扫原组入场码"><button>核对</button></form></div></template><button v-else-if="gate.state!=='SUBMITTING'" @click="reset">{{gate.state==='SUCCESS'?'完成':gate.state==='BLOCKED'?'重新连接':'返回扫码'}}</button></section>
  <footer class="scan-footer"><template v-if="!kiosk&&!['UNKNOWN','SUBMITTING'].includes(gate.state)"><RouterLink :to="'/check/events/'+gate.eventId+'/lookup'">辅助找票</RouterLink><RouterLink :to="'/check/events/'+gate.eventId+'/records'">本人及本端记录</RouterLink></template><button class="text-button" @click="help=!help">需要帮助</button><p v-if="help">请向现场工作人员出示本屏幕，我们会协助您完成检票。</p></footer>
 </main>
</template>
