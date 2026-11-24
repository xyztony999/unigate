"""Read-only UniGate document checks. This is NOT an application or full OAS validator."""
from pathlib import Path
import argparse
import glob
import hashlib
import json
import re
import sys

ROOT = Path(__file__).resolve().parents[2]
WS = ROOT / 'profiles/unigate/workspace'

def _resolve_engine_source():
    home = str(Path.home())
    found = []
    for pat in (home + '/.zcode/cli/plugins/cache/*/ai-project-engine/*/payload/skills/validation/SKILL.md',
                home + '/.codex/plugins/cache/*/ai-project-engine/*/payload/skills/validation/SKILL.md'):
        found += glob.glob(pat)
    if not found:
        return None
    def ver(p):
        m = re.search(r'ai-project-engine/([0-9.]+)/', p.replace('\\', '/'))
        return tuple(int(x) for x in m.group(1).split('.')) if m else (0,)
    return Path(max(found, key=ver))

SOURCE = _resolve_engine_source()

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--write-report', action='store_true')
    args = parser.parse_args()
    api = WS / '02-detailed/api'
    spec = json.loads((api / 'openapi-unigate.json').read_text(encoding='utf-8'))
    schemas = spec['components']['schemas']
    ops = spec['x-operation-index']
    md = (api / 'api-unigate.md').read_text(encoding='utf-8')
    fl = (WS / '00-architecture/function-list.md').read_text(encoding='utf-8')
    results = []

    def check(name, condition, evidence):
        results.append({'name': name, 'status': 'PASS' if condition else 'FAIL', 'evidence': evidence})

    refs = []
    def walk(value):
        if isinstance(value, dict):
            if '$ref' in value: refs.append(value['$ref'])
            for child in value.values(): walk(child)
        elif isinstance(value, list):
            for child in value: walk(child)
    walk(spec)
    check('schema-references', all(r.startswith('#/components/schemas/') and r.rsplit('/',1)[-1] in schemas for r in refs), f'{len(refs)} local references resolve')
    check('operation-uniqueness', len(ops)==len({o['name'] for o in ops})==len(spec['paths']), f'{len(ops)} unique operations and paths')
    check('routes', all(re.fullmatch(r'/api/v1/[a-z]+/[a-z-]+/[a-z-]+',o['path']) for o in ops), 'version/domain/resource/action')
    check('operation-index', all(spec['paths'][o['path']][o['method']]['operationId']==o['name'] for o in ops), 'index matches paths/methods')
    objects = [v for v in schemas.values() if v.get('type')=='object']
    check('closed-objects', all(v.get('additionalProperties') is False and set(v.get('required',[]))<=set(v.get('properties',{})) for v in objects), f'{len(objects)} closed object schemas')
    check('semantic-write-ids', all('operationId' in schemas[o['request']]['required'] for o in ops if o['mutation']), 'all semantic writes require operationId')
    check('write-methods', all(o['method']=='post' for o in ops if o['mutation']), 'writes POST; OAuth GET callback has one-time state exception')
    posts = [(o,spec['paths'][o['path']][o['method']]) for o in ops if o['method']=='post']
    check('post-csrf', all(any(p['name']=='X-CSRF-Token' and p['required'] for p in op.get('parameters',[])) for _,op in posts), 'all POSTs declare CSRF, including anonymous/safe queries')
    check('anonymous-context', all(any(p['name']=='X-Client-Context-ID' and p['required'] for p in op.get('parameters',[])) for o,op in posts if o['auth']=='ANON' and o['mutation']), 'original anonymous context header required')
    check('bootstrap-continuity', 'clientContextId' in schemas['CsrfView']['required'] and '不得把新bootstrap的标记套到旧operationId' in md, 'public continuity marker plus no cross-context automatic replay')
    check('response-envelopes', all(set(schemas[o['response']]['required'])=={'code','message','data','requestId'} for o in ops), 'OK/message/data/requestId; binary download explicit exception')
    check('documented-operations', all(re.search(r'#### 4\.2\.\d+ '+re.escape(o['name'])+r' —',md) for o in ops), 'each operation has a human-readable contract')
    check('six-chapters', all(re.search(r'^## '+str(n)+r'\. .+',md,re.M) for n in range(1,7)), 'six required chapter headings')
    check('pagination', all(schemas[o['request']]['properties']['pageSize']['maximum']==100 for o in ops if 'pageSize' in schemas[o['request']]['properties']), 'ordinary pageSize cap100; gate lookup fixed10')
    check('import-selection', schemas['BatchCommitRequest']['properties']['selectedRowIds'].get('uniqueItems') and schemas['BatchCommitRequest']['properties']['selectedRowIds']['minItems']==1, 'nonempty unique selected row set')
    check('own-party-boundary', all('partyId' not in schemas[n]['properties'] for n in ['MyPartyRequest','MyCredentialRequest']), 'own party derived from capability cookie')
    mine = next((o for o in ops if o['name']=='MyGrantList'),None)
    check('own-grants', mine is not None and 'READER' in mine['roles'] and 'accountId' not in schemas[mine['request']]['properties'], 'FL-R-002 has own-grant read API; cannot request another account')
    check('gate-minimum-data', set(schemas['GateCandidate']['properties']).isdisjoint({'email','phone','fields','ticketToken','queryToken'}), 'lookup response has only masked candidate fields')
    check('reissue-response', 'access' not in schemas['ReissueView']['required'] and 'credential' in schemas['ReissueView']['required'], 'no private access returned unless explicitly rotated')
    check('operation-lookup-action', all('action' in schemas[n]['required'] for n in ['MyOperationRequest','OperationResultRequest']), 'generic result lookup includes original action')
    check('connection-scope', 'allowedEventIds' in schemas['ConnectionSaveRequest']['required'] and 'eventId' in schemas['ResourceListRequest']['properties'], 'event-scoped resource access is explicit')
    rolemap = {'A':'ADMIN','O':'ORGANIZER','C':'CHECKER','R':'READER','P':'PARTICIPANT'}
    links=[]
    for line in fl.splitlines():
        m=re.search(r'(FL-([AOCRP])-\d+)',line)
        aa=re.search(r'\| AA ([0-9.]+) ',line)
        if m and aa: links.append((m[1],rolemap[m[2]],aa[1]))
    check('fl-role-coverage',len(links)==43 and len({x[0] for x in links})==43 and all(any(a in o['aa'] and r in o['roles'] for o in ops) for _,r,a in links),'43 FL functions have a same-AA operation with matching role')
    check('aa-leaves',len({x[2] for x in links})==23 and {x[2] for x in links}<={a for o in ops for a in o['aa']},'23 AA leaves covered')
    enums=spec['x-enum-pool']
    check('enum-values',all(schemas[n+'Code']['enum']==list(v) and set(schemas[n+'Value']['required'])=={'code','label'} for n,v in enums.items()),f'{len(enums)} enum pools match response value objects')
    pooltext=''.join((WS/'01-solution/common-spec'/n).read_text(encoding='utf-8') for n in ['03-common-enums.md','04-business-enum.md'])
    check('enum-pool-files',all('## '+n in pooltext and all('| '+code+' | '+label+' |' in pooltext for code,label in values.items()) for n,values in enums.items()),'every code/label exists in the maintained enum files')
    check('no-stale-protocol-names','countMode=COMPANION_COUNT' not in md and 'TOTAL_COUNT' not in md and 'NEW_PARTY/LINK_EXISTING' not in md,'prose uses COMPANIONS/TOTAL/KEEP_SEPARATE')
    check('gate-auth',spec['paths']['/api/v1/gate/checkin/commit']['post']['security']==[{'StaffSession':[],'DeviceSession':[]},{'DeviceSession':[]}], 'STAFF+DEVICE vs KIOSK-only branch; conditional runtime enforcement still required')
    check('unknown-result-semantics',all(t in md for t in ['showAsNewAdmission=false','不能推断没扣数','同一次操作换组/换端不得作为新写入恢复']),'replay never means a new admission; no new operation on unknown')
    blocks=re.findall(r'```json\n(.*?)\n```',md,re.S)
    try:
        examples=[json.loads(t) for t in blocks]
        valid=len(examples)==3 and examples[0]['operationId']==examples[1]['data']['operationId']==examples[2]['data']['operationId']
    except (ValueError,KeyError,TypeError):
        valid=False
    check('json-examples',valid,'3 JSON examples parse and refer to the same operation, not executed requests')
    ddl=(api/'data-model-unigate.md').read_text(encoding='utf-8')
    check('design-tables',ddl.count('CREATE TABLE ')==25 and all('ug_'+n in ddl for n in ['event','party','operation','admission','admission_correction','deletion_ledger']),'25 design DDL tables; SQL not executed')
    check('source-column-inputs','availableColumns' in schemas['BatchView']['required'] and 'ResourceFieldsRequest' in schemas and 'availableConnections' in schemas['SourceListView']['required'] and set(schemas['SourceConnectionOption']['properties'])=={'connectionId','displayName','provider','accountKind','status'},'mapping has actual columns and event-assigned thin connections, never guessed headers or admin secrets')
    check('admin-event-selector',set(schemas['GrantEventOption']['properties'])=={'eventId','title','startsAt'} and next(o for o in ops if o['name']=='GrantEventOptions')['roles']==['ADMIN'],'authorization target selector reveals no roster/allocation/attendance')
    reg=spec['paths']['/api/v1/public/registration/result']['post']
    check('anonymous-original-result',not next(o for o in ops if o['name']=='RegistrationResult')['mutation'] and any(p['name']=='X-Client-Context-ID' and p['required'] for p in reg['parameters']),'reload can read original result without storing form PII or creating another group')
    check('cleanup-scope-output','scopes' in schemas['CleanupView']['required'] and len(spec['x-enum-pool']['CleanupScope'])==6,'cleanup reports six managed ranges including backup coverage')
    check('ui-state-inputs','hasAdmissionHistory' in schemas['PartyView']['required'] and 'eventStatus' in schemas['DeviceSessionView']['required'] and 'asOf' in schemas['PartyPage']['required'],'history/window/snapshot labels have actual API inputs')
    check('gate-global-operation-key','固定GATE' in md and 'deviceId、组ID、原版本和人数纳入载荷摘要' in md,'same event/action/operation cannot become a new write by changing group/device')

    adopted='## ADR-043：' in (ROOT/'profiles/unigate/knowledge/design-decisions/index.md').read_text(encoding='utf-8')
    source_rows=[]
    if SOURCE is not None and SOURCE.is_file():
        original=SOURCE.read_text(encoding='utf-8')
        source_rows=[{'id':i,'requirement':v} for i,v in re.findall(r'^\| ((?:CH[1-6]-|GL-)[0-9]+[a-z]?) \| (.*?) \|$',original,re.M)]
    inventory={'sourcePath':str(SOURCE) if SOURCE is not None else None,'sourceSha256':hashlib.sha256(SOURCE.read_bytes()).hexdigest() if SOURCE is not None and SOURCE.is_file() else None,
               'knownRows':source_rows,'missingOriginalAssets':['prd-to-api-design/template.md','prd-to-api-design/check.sh','prd-to-api-design/checklist.md'],
               'undefinedInAvailableSource':['CH6 definitions (3 claimed by design skill)']+[f'GL-{i:03d}' for i in [13,14,15,16,17,18,19,20,23,24,25]]}
    check('source-inventory',adopted or (len(source_rows)==164 and len({x['id'] for x in source_rows})==164),
          'ADR-043 project basis adopted: original-engine CH/GL inventory is version-specific and informational (recovered %d rows from %s); without ADR-043 the 164-row original baseline is required'%(len(source_rows), SOURCE if SOURCE is not None else 'unavailable'))
    report={'scope':'project-mechanical-document-checks-only','contractDecision':'ADR-042','projectBasisDecision':'ADR-043' if adopted else None,'projectBasisAdopted':adopted,'apiVersion':spec['info']['version'],
            'operationCount':len(ops),'checks':results,'passed':sum(r['status']=='PASS' for r in results),
            'failed':sum(r['status']=='FAIL' for r in results),'originalEngineD3Passed':False,
            'limitations':['No HTTP/DB/device/provider tests','Not a complete OpenAPI or JSON Schema validator','Original assets/undefined clauses remain unavailable; ADR-043 authorizes the project basis, never an original-engine PASS']}
    if args.write_report:
        reports=WS/'05-reports'
        (reports/'D3-project-check.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
        (reports/'D3-check-source.json').write_text(json.dumps(inventory,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps({k:report[k] for k in ['scope','apiVersion','operationCount','passed','failed','originalEngineD3Passed']},ensure_ascii=False))
    if report['failed']:
        print(json.dumps([r for r in results if r['status']=='FAIL'],ensure_ascii=False))
    return bool(report['failed'])

if __name__=='__main__':
    sys.exit(main())
