<script setup lang="ts">
import { computed } from 'vue'
import { toZonedInput,fromZonedInput } from '../lib/validation'
import { schema, label, contract, type Obj } from '../lib/contract'
const props=withDefaults(defineProps<{modelValue:Obj;definition:Obj;page?:Obj;operation?:string;prefix?:string;options?:Record<string,Obj[]>;locked?:string[];timeZone?:string}>(),{prefix:'',options:()=>({}),locked:()=>[],timeZone:'Asia/Shanghai'})
const emit=defineEmits<{ 'update:modelValue':[Obj] }>()
const TIMEZONES=['Asia/Shanghai','Asia/Hong_Kong','Asia/Taipei','Asia/Tokyo','Asia/Seoul','Asia/Singapore','Asia/Bangkok','Asia/Kolkata','Asia/Dubai','Europe/London','Europe/Paris','Europe/Berlin','Europe/Moscow','America/New_York','America/Chicago','America/Denver','America/Los_Angeles','America/Sao_Paulo','Australia/Sydney','Pacific/Auckland','UTC']
const fields=computed(()=>Object.entries(schema(props.definition).properties||{}).map(([key,s])=>({key,s:schema(s as Obj),path:props.prefix?props.prefix+'.'+key:key})).filter(f=>!['operationId','page','pageSize','expectedPartyVersion','previewVersion','version','scanContextToken','phoneVerificationProof','acceptedNoticeVersion','clientContextId'].includes(f.key) && !props.locked.includes(f.key)))
function update(key:string,value:any){ emit('update:modelValue',{...props.modelValue,[key]:value}) }
function title(key:string,path:string){return label(path,props.page,props.operation) || label(key)}
function values(s:Obj):string[] {return schema(s).enum||[]}
function enumText(code:string){for(const dict of Object.values(contract.enums)) if(dict[code]) return dict[code];return code}
function choices(key:string,path:string):Obj[]{return props.options[path]||props.options[key]||[]}
function arrayUpdate(key:string,value:string,checked:boolean){const current=props.modelValue[key]||[];update(key,checked?[...current,value]:current.filter((x:string)=>x!==value))}
function dateUpdate(key:string,v:string){update(key,fromZonedInput(v,props.modelValue.timeZone||props.timeZone))}
function localDate(v:any){return toZonedInput(v,props.modelValue.timeZone||props.timeZone)}
function addRow(key:string){update(key,[...(props.modelValue[key]||[]),{}])}
function setRow(key:string,index:number,value:Obj){const arr=[...(props.modelValue[key]||[])];arr[index]=value;update(key,arr)}
</script>
<template>
  <div class="fields-grid">
    <template v-for="f in fields" :key="f.key">
      <fieldset v-if="f.s.type==='object'" class="field-group full"><legend>{{title(f.key,f.path)}}</legend><DynamicFields :model-value="modelValue[f.key]||{}" :definition="f.s" :page="page" :operation="operation" :prefix="f.path" :options="options" :time-zone="modelValue.timeZone||timeZone" @update:model-value="update(f.key,$event)" /></fieldset>
      <fieldset v-else-if="f.s.type==='array' && schema(f.s.items).type==='object'" class="field-group full"><legend>{{title(f.key,f.path)}}</legend><div v-for="(row,index) in modelValue[f.key]||[]" :key="index" class="array-row"><DynamicFields :model-value="row" :definition="f.s.items" :page="page" :operation="operation" :options="options" :prefix="f.path+'[]'" :time-zone="modelValue.timeZone||timeZone" @update:model-value="setRow(f.key,Number(index),$event)"/><button type="button" class="quiet danger" @click="update(f.key,modelValue[f.key].filter((_:unknown,i:number)=>i!==index))">移除</button></div><button type="button" class="secondary" @click="addRow(f.key)">添加一项</button></fieldset>
      <fieldset v-else-if="f.s.type==='array'" class="field-group full"><legend>{{title(f.key,f.path)}}<span v-if="(schema(definition).required||[]).includes(f.key)" class="required"> *</span></legend><label class="check-choice" v-for="value in values(f.s.items)" :key="value"><input type="checkbox" :checked="(modelValue[f.key]||[]).includes(value)" @change="arrayUpdate(f.key,value,($event.target as HTMLInputElement).checked)">{{enumText(value)}}</label><label class="check-choice" v-for="item in choices(f.key,f.path)" :key="item.value"><input type="checkbox" :checked="(modelValue[f.key]||[]).includes(item.value)" @change="arrayUpdate(f.key,item.value,($event.target as HTMLInputElement).checked)">{{item.label}}</label><p v-if="!values(f.s.items).length&&!choices(f.key,f.path).length" class="muted">请先在记录列表中选择适用项。</p></fieldset>
      <label v-else-if="f.s.type==='boolean'" class="toggle-field"><input type="checkbox" :checked="modelValue[f.key]===true" @change="update(f.key,($event.target as HTMLInputElement).checked)"><span>{{title(f.key,f.path)}}</span></label>
      <label v-else class="field" :class="{full:(f.s.maxLength||0)>500}"><span>{{title(f.key,f.path)}}<span v-if="(schema(definition).required||[]).includes(f.key)" class="required"> *</span></span>
        <select v-if="values(f.s).length" :value="modelValue[f.key]||''" :required="(schema(definition).required||[]).includes(f.key)" @change="update(f.key,($event.target as HTMLSelectElement).value)"><option value="">请选择</option><option v-for="value in values(f.s)" :key="value" :value="value">{{enumText(value)}}</option></select>
        <select v-else-if="choices(f.key,f.path).length" :value="modelValue[f.key]||''" :required="(schema(definition).required||[]).includes(f.key)" @change="update(f.key,($event.target as HTMLSelectElement).value)"><option value="">请选择</option><option v-for="item in choices(f.key,f.path)" :key="item.value" :value="item.value">{{item.label}}</option></select>
        <select v-else-if="f.key==='timeZone'" :value="modelValue[f.key]||'Asia/Shanghai'" :required="(schema(definition).required||[]).includes(f.key)" @change="update(f.key,($event.target as HTMLSelectElement).value)"><option v-for="tz in TIMEZONES" :key="tz" :value="tz">{{tz}}</option></select>
        <input v-else-if="f.s.format==='binary'" type="file" accept=".csv,.xlsx" :required="(schema(definition).required||[]).includes(f.key)" @change="update(f.key,($event.target as HTMLInputElement).files?.[0])">
        <input v-else-if="f.s.format==='date-time'" type="datetime-local" :value="localDate(modelValue[f.key])" :required="(schema(definition).required||[]).includes(f.key)" @input="dateUpdate(f.key,($event.target as HTMLInputElement).value)">
        <textarea v-else-if="(f.s.maxLength||0)>500 && !/token|secret|password|pairing/i.test(f.key)" :value="modelValue[f.key]||''" :maxlength="f.s.maxLength" :required="(schema(definition).required||[]).includes(f.key)" rows="4" @input="update(f.key,($event.target as HTMLTextAreaElement).value)" />
        <input v-else :type="f.s.type==='integer'||f.s.type==='number'?'number':/password|secret|token|pairingCode/i.test(f.key)?'password':/email/i.test(f.key)?'email':/phone/i.test(f.key)?'tel':'text'" :value="modelValue[f.key]??''" :min="f.s.minimum" :max="f.s.maximum" :minlength="f.s.minLength" :maxlength="f.s.maxLength" :required="(schema(definition).required||[]).includes(f.key)" :step="f.s.type==='integer'?1:undefined" :autocomplete="/password/i.test(f.key)?'new-password':'off'" @input="update(f.key,($event.target as HTMLInputElement).type==='number'? (($event.target as HTMLInputElement).value===''?undefined:Number(($event.target as HTMLInputElement).value)):($event.target as HTMLInputElement).value)">
        <small v-if="f.s.format==='date-time'" class="muted">时区：{{modelValue.timeZone||timeZone}}</small>
      </label>
    </template>
  </div>
</template>
