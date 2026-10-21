import { schema,label,type Obj } from './contract'
export function validate(value:Obj,definition:Obj,path=''):string[] {
 const def=schema(definition),errors:string[]=[]
 const required=[...(def.required||[])]
 for(const rule of def.allOf||[]){const test=rule.if?.properties||{};if(Object.entries(test).every(([k,v])=>value[k]===(v as Obj).const))required.push(...(rule.then?.required||[]))}
 for(const [key,deps] of Object.entries(def.dependentRequired||{}))if(value[key]!==undefined)required.push(...deps as string[])
 for(const key of [...new Set(required)] as string[])if(value[key]===undefined||value[key]===''||value[key]===null)errors.push(/Id$|Version$|^version$/.test(key)?'请先选择对应记录并刷新后操作。':`请填写${label(path?path+'.'+key:key)}。`)
 for(const [key,raw] of Object.entries(def.properties||{})){
  const v=value[key];if(v===undefined||v==='')continue
  const field=schema(raw as Obj),name=label(path?path+'.'+key:key)
  if(field.enum&&!field.enum.includes(v))errors.push(`请选择有效的${name}。`)
  if(field.type==='integer'&&(!Number.isInteger(v)||v<field.minimum||v>field.maximum))errors.push(`${name}须为规定范围内的整数。`)
  if(field.type==='string'&&field.format!=='binary'){
   if(typeof v!=='string'||v.length<(field.minLength||0)||v.length>(field.maxLength||Infinity))errors.push(`${name}的长度不符合要求。`)
   if(field.format==='date-time'&&Number.isNaN(Date.parse(v)))errors.push(`请选择有效的${name}。`)
   if(key==='timeZone'){try{new Intl.DateTimeFormat('zh-CN',{timeZone:v})}catch{errors.push('请输入有效的活动时区，例如 Asia/Shanghai。')}}
  }
  if(field.type==='array'){
   if(!Array.isArray(v)||v.length<(field.minItems||0)||v.length>(field.maxItems||Infinity))errors.push(`请核对${name}的选择数量。`)
   else if(field.uniqueItems&&new Set(v).size!==v.length)errors.push(`${name}不能重复选择。`)
  }
  if(field.type==='object'&&v&&typeof v==='object')errors.push(...validate(v,field,path?path+'.'+key:key))
 }
 return [...new Set(errors)]
}
export function toZonedInput(value:string,zone:string){
 if(!value)return ''
 try{const parts=new Intl.DateTimeFormat('sv-SE',{timeZone:zone,year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hourCycle:'h23'}).formatToParts(new Date(value));const get=(p:string)=>parts.find(x=>x.type===p)?.value;return `${get('year')}-${get('month')}-${get('day')}T${get('hour')}:${get('minute')}`}catch{return ''}
}
export function fromZonedInput(value:string,zone:string){
 if(!value)return undefined
 let guess=Date.parse(value+'Z'),target=guess
 for(let i=0;i<4;i++){const local=toZonedInput(new Date(guess).toISOString(),zone);if(!local)return undefined;const difference=target-Date.parse(local+'Z');if(difference===0)return new Date(guess).toISOString();guess+=difference}
 return undefined
}
