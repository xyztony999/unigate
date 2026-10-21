import { readFileSync,writeFileSync,mkdirSync,readdirSync } from 'node:fs'
import { createHash } from 'node:crypto'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
const root=fileURLToPath(new URL('../',import.meta.url))
const summary=JSON.parse(readFileSync(path.join(root,'coverage/coverage-summary.json'),'utf8'))
const entries=Object.entries(summary).filter(([name])=>name!=='total')
const metrics=['statements','branches','functions','lines']
function aggregate(accept){const rows=entries.filter(([name])=>accept(name.replaceAll('\\','/'))).map(([,row])=>row);return Object.fromEntries(metrics.map(metric=>{const total=rows.reduce((n,r)=>n+r[metric].total,0),covered=rows.reduce((n,r)=>n+r[metric].covered,0);return[metric,{total,covered,pct:total?Math.round(covered/total*10000)/100:100}]}))}
function files(dir){return readdirSync(dir,{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(path.join(dir,entry.name)):[path.join(dir,entry.name)])}
const evidenceFiles=[...files(path.join(root,'src')).filter(file=>/\.(ts|vue)$/.test(file)&&!file.endsWith('api-types.ts')),...files(path.join(root,'tests')).filter(file=>file.endsWith('.ts')),path.join(root,'package-lock.json'),path.join(root,'vite.config.ts')].sort()
const hash=createHash('sha256');for(const file of evidenceFiles){hash.update(path.relative(root,file).replaceAll('\\','/'));hash.update(readFileSync(file))}
const report={schemaVersion:1,measuredAt:new Date().toISOString(),command:'npm run test:coverage',provider:'@vitest/coverage-v8 4.1.0',node:process.version,scope:'All src/**/*.ts and src/**/*.vue, including untested App/main/router. Only generated API type aliases excluded because they have no runtime code.',sourceAndTestSha256:hash.digest('hex'),total:summary.total,allVue:aggregate(name=>name.endsWith('.vue')),components:aggregate(name=>name.includes('/src/components/')),views:aggregate(name=>name.includes('/src/views/')),lib:aggregate(name=>name.includes('/src/lib/')),zeroStatementCoverage:entries.filter(([,v])=>v.statements.pct===0).map(([name])=>path.relative(root,name).replaceAll('\\','/')),limitations:['Component tests use controlled API responses; these values do not include real-backend browser E2E coverage.','Android camera/scanner/kiosk OS behavior and real third-party account/delivery paths remain device/provider validation tasks.','No synthetic threshold is configured by the active stack; counts are measured, not inferred from passing tests.']}
mkdirSync(path.join(root,'verification'),{recursive:true});writeFileSync(path.join(root,'verification/coverage-summary.json'),JSON.stringify(report,null,2)+'\n');console.log('Saved verification/coverage-summary.json')
