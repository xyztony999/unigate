<script setup lang="ts">
import { computed } from 'vue'
import { display,label,secret,readPath,type Obj } from '../lib/contract'
const props=defineProps<{value:any;columns?:Obj[];selectable?:boolean;selected?:string[];references?:Record<string,string>}>()
const emit=defineEmits<{select:[Obj];toggle:[Obj,boolean]}>()
const rows=computed(()=>Array.isArray(props.value)?props.value:props.value?.items)
const internalField=(key:string)=>secret(key)||/(Id|Version|Digest)$/.test(key)||key==='version'
const keys=computed(()=>{
 if(props.columns?.length) return props.columns.filter(c=>c.visible!==false).map(c=>({key:c.path.replace(/^items\[\]\./,''),label:c.label==='errors'?'校验问题':c.label}))
 const sample=rows.value?.[0]||{}
 return Object.keys(sample).filter(k=>!internalField(k)).slice(0,9).map(k=>({key:k,label:label(k)}))
})
const detailKeys=computed(()=>Object.keys(props.value||{}).filter(k=>!internalField(k)&&!['items','availableColumns','availableConnections'].includes(k)))
function rowId(row:Obj){return row.rowId||row.partyId||row.eventId||row.accountId||row.recordId||row.jobId}
function cell(row:Obj,key:string){const value=readPath(row,key);if(/Id$/.test(key)&&value)return props.references?.[value]||row.title||row.partyReference||row.displayName||'相关记录';return display(value)}
</script>
<template>
 <div v-if="value===undefined||value===null" class="empty">暂无记录，刷新后可重新获取。</div>
 <template v-else-if="rows">
  <p v-if="!rows.length" class="empty">暂无符合条件的记录，可调整查询条件后重试。</p>
  <div v-else class="table-scroll"><table><thead><tr><th v-if="selectable">选择</th><th v-for="c in keys" :key="c.key">{{c.label}}</th><th>查看</th></tr></thead><tbody><tr v-for="(row,index) in rows" :key="rowId(row)||index"><td v-if="selectable"><input type="checkbox" :aria-label="'选择第'+(Number(index)+1)+'行'" :checked="selected?.includes(rowId(row))" @change="emit('toggle',row,($event.target as HTMLInputElement).checked)"></td><td v-for="c in keys" :key="c.key">{{cell(row,c.key)}}</td><td><button type="button" class="text-button" @click="emit('select',row)">详情</button></td></tr></tbody></table></div>
  <p v-if="value.total!==undefined" class="muted">共 {{value.total}} 条</p>
 </template>
 <dl v-else-if="typeof value==='object'" class="details"><template v-for="k in detailKeys" :key="k"><dt>{{label(k)}}</dt><dd>{{display(value[k])}}</dd></template></dl>
 <p v-else>{{display(value)}}</p>
</template>
