"""Document-only PageSpec checks, using the ADR-043 project basis; never runs the product."""
from pathlib import Path
import argparse,json,re,sys,hashlib

ROOT=Path(__file__).resolve().parents[2]
WS=ROOT/'profiles/unigate/workspace'

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--write-report',action='store_true')
    args=parser.parse_args()
    spec=json.loads((WS/'02-detailed/api/openapi-unigate.json').read_text(encoding='utf-8'))
    data=json.loads((WS/'02-detailed/interaction/page-contracts.json').read_text(encoding='utf-8'))
    schemas=spec['components']['schemas']; ops={o['name']:o for o in spec['x-operation-index']}; pages=data['pages']; results=[]
    def check(name,ok,evidence): results.append({'name':name,'status':'PASS' if ok else 'FAIL','evidence':evidence})
    def resolve(schema):
        while '$ref' in schema: schema=schemas[schema['$ref'].rsplit('/',1)[-1]]
        return schema
    def field(schema,path):
        for part in path.split('.'):
            schema=resolve(schema)
            key=part.replace('[]','')
            if schema.get('type')!='object' or key not in schema.get('properties',{}): return None
            schema=schema['properties'][key]
            if part.endswith('[]'):
                schema=resolve(schema)
                if schema.get('type')!='array': return None
                schema=schema['items']
        return schema
    def type_text(s):
        if '$ref' in s: return s['$ref'].rsplit('/',1)[-1]
        if s.get('type')=='array': return 'array<'+type_text(s['items'])+'>'
        return '/'.join(s['type']) if isinstance(s.get('type'),list) else str(s.get('type'))
    check('module-input',data['module']=='unigate' and data['apiVersion']==spec['info']['version'] and data['basis']=='ADR-043','module/app/API and adopted basis consistent')
    check('page-routes',len(pages)==33 and len({p['id'] for p in pages})==len({p['route'] for p in pages})==33,'33 unique page IDs and routes')
    check('role-groups',set(p['group'] for p in pages)=={'shared','admin','organizer','checker','reader','participant'},'five business roles plus shared documents; kiosk is a participant mode')
    used={a['operation'] for p in pages for a in p['actions']}; protocol={p['operation'] for p in data['protocolOnly']}
    check('operation-coverage',set(ops)==used|protocol and used.isdisjoint(protocol),f'all {len(ops)} API operations mapped; OAuth callback protocol-only')
    check('action-contracts',all(a['operation'] in ops and a['path']==ops[a['operation']]['path'] and a['method'].lower()==ops[a['operation']]['method'] for p in pages for a in p['actions']),'all Action methods/paths exactly match OpenAPI')
    check('action-identity',all(a['auth']==ops[a['operation']]['auth'] and a['roles']==ops[a['operation']]['roles'] for p in pages for a in p['actions']),'auth/role upper bounds copied without expansion')
    bridge={('O10','SessionBootstrap'),('O10','SmsStart'),('O10','SmsVerify')}
    check('staff-role-boundaries',all(set(p['roles'])&set(a['roles']) or a['auth'] in ['ANON','DEVICE'] or (p['id'],a['operation']) in bridge for p in pages for a in p['actions']),'ANON/device subflows explicit; STAFF/GATE preserve caller role')
    valid_fields=[]; valid_types=[]; bounds=[]
    for p in pages:
        for direction in ['request','response']:
            for f in p[direction+'Fields']:
                o=ops[f['operation']]; root=schemas[o['request'] if direction=='request' else o['data']]
                found=field(root,f['path'])
                valid_fields.append(found is not None)
                if found is not None:
                    valid_types.append(f['type']==type_text(found))
                    bounds.append(all(found.get(k)==v for k,v in f.get('limits',{}).items()))
    check('field-paths',all(valid_fields),f'{len(valid_fields)} request/response bindings resolve in source schemas')
    check('field-types',all(valid_types),'all binding types match source schema references/primitives')
    check('field-limits',all(bounds),'all declared limits/defaults are identical to source')
    check('array-input-controls',all(f['component'] in ['multi_select','authorized_multi_select','valid_row_selection'] for p in pages for f in p['requestFields'] if f['type'].startswith('array<') and f['visibility']=='input'),'array-valued roles/fields/columns/event scopes use multiple selection, not a scalar control')
    check('list-columns',all(field(schemas[ops[c['operation']]['data']], 'items[].'+c['path']) is not None for p in pages for c in p['columns']),'every visible list column resolves in the actual response')
    enums=spec['x-enum-pool']
    check('dictionary-binding',all(f.get('enum') in enums and f.get('dictionary','').endswith('#'+f['enum']) for p in pages for k in ['requestFields','responseFields'] for f in p[k] if f.get('enum')),f'{len(enums)} shared dictionary pools, no invented page enum')
    fl=(WS/'00-architecture/function-list.md').read_text(encoding='utf-8')
    required=set(re.findall(r'^\| (FL-[AOCRP]-\d+) ',fl,re.M)); mapped={f for p in pages for f in p['fl']}
    check('fl-page-coverage',len(required)==43 and required==mapped,'all 43 confirmed functions map to at least one page')
    check('entry-exit-accessibility',all(p.get('entry') and p.get('exit') and p.get('accessibility') for p in pages),'every page states entry/exit/touch/accessibility requirements')
    check('state-coverage',all({'initial','loading','empty','ready','submitting','success','rejected','unknown','forbidden','expired'}<=set(p['states']) for p in pages),'all pages define load/empty/error/unknown/expiry states')
    forbidden={'GateWalkIn','GateIncrease','GateCorrect','CredentialReissue','CredentialRevoke','ExportCreate','ExportDownload','PartyChange','PartyCancel','ReviewDecide'}
    check('checker-no-privileged-write',all(not(forbidden & set(p['operations'])) for p in pages if p['group']=='checker'),'checkers have no increase/correction/reissue/export/approval')
    check('reader-read-only',all(not ops[o]['mutation'] for p in pages if p['group']=='reader' for o in p['operations']),'reader pages expose only safe reads, no export')
    kiosk=next(p for p in pages if p['layout']=='kiosk')
    check('kiosk-scope',set(kiosk['operations'])=={'DeviceDetail','TicketResolve','GateCommit','GateOperation'},'kiosk has only current device/group/operation API, no roster lookup')
    check('kiosk-reset',kiosk['successResetSeconds']==5 and kiosk['idleResetSeconds']==30 and kiosk['unknownAutoReset'] is False,'5s success / 30s idle; unknown never auto-clears pending operation')
    check('gate-submit-guard',all('remainingCount' in a['visibleWhen'] and 'CHECKIN_OPEN' in a['visibleWhen'] for p in pages for a in p['actions'] if a['operation']=='GateCommit'),'count/window/unknown guard declared for each scanner')
    overview=(WS/'02-detailed/interaction/README.md').read_text(encoding='utf-8')
    check('replay-never-release','showAsNewAdmission=true' in overview and '不重复绿灯' in overview and any(x[0]=='KNOWN_REPLAY' for x in data['scannerStates']),'original-result query never displayed as a new release')
    check('registration-reload',all('RegistrationResult' in p['operations'] for p in pages if p['id'] in ['P01','P02']) and '不需持久化原表单PII' in overview,'original anonymous result recovery avoids durable form PII')
    check('sensitive-fields',all(f['visibility'] in ['system','memory_only'] or f['component']=='password' for p in pages for k in ['requestFields','responseFields'] for f in p[k] if f.get('sensitive')),'secret values are system/memory/password bindings, not list columns')
    rolefiles=[WS/'02-detailed/interaction'/g/'unigate/unigate.md' for g in ['shared','admin','organizer','checker','reader','participant']]
    check('spec-documents',all(p.is_file() and '## 更新记录' in p.read_text(encoding='utf-8') and ('| 版本 | v'+data['version']+' |') in p.read_text(encoding='utf-8') for p in rolefiles),'six PageSpec documents with matching version metadata and changelog')
    check('page-headings',all(('## '+p['id']+' '+p['title']) in (WS/'02-detailed/interaction'/p['group']/'unigate/unigate.md').read_text(encoding='utf-8') for p in pages),'all 33 route specifications emitted into readable documents')
    decision=(ROOT/'profiles/unigate/knowledge/design-decisions/index.md').read_text(encoding='utf-8')
    confirmation=data.get('confirmation',{})
    confirmed=(data['status']=='confirmed' and confirmation.get('decision')=='ADR-044'
               and confirmation.get('selectedOption')==1 and confirmation.get('sevenItems')==list(range(1,8))
               and '## ADR-044：' in decision)
    report={'scope':'D4-project-mechanical-document-checks','basis':'ADR-043','version':data['version'],'pageCount':len(pages),'operationCount':len(ops),'fieldBindingCount':len(valid_fields),'passed':sum(r['status']=='PASS' for r in results),'failed':sum(r['status']=='FAIL' for r in results),'checks':results,'interactionConfirmed':confirmed,'confirmationDecision':confirmation.get('decision'),'gate1EvaluatedByThisTool':False,'originalFxChecklistPassed':False,'apiSha256':hashlib.sha256((WS/'02-detailed/api/openapi-unigate.json').read_bytes()).hexdigest(),'pageContractSha256':hashlib.sha256((WS/'02-detailed/interaction/page-contracts.json').read_bytes()).hexdigest(),'limitations':['Documents only, not HTTP/Vue/device tests','Gate 1 is assessed independently','Original FX full checklist is not claimed']+([] if confirmed else ['Seven-item user review still required'])}
    if args.write_report: (WS/'05-reports/D4-project-check.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps({k:report[k] for k in ['scope','pageCount','operationCount','fieldBindingCount','passed','failed','interactionConfirmed','gate1EvaluatedByThisTool']}))
    if report['failed']: print(json.dumps([r for r in results if r['status']=='FAIL'],ensure_ascii=False))
    return bool(report['failed'])

if __name__=='__main__': sys.exit(main())
