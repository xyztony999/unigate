"""Independent cross-stage document assertions for Gate 1; manual review is also required."""
from pathlib import Path
from datetime import datetime, timezone
import argparse
import hashlib
import json
import re
import sys

ROOT = Path(__file__).resolve().parents[2]
WS = ROOT / 'profiles/unigate/workspace'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--write-report', action='store_true')
    args = parser.parse_args()
    files = {
        'ba': WS / '00-architecture/business-arch.md',
        'aa': WS / '00-architecture/application-arch.md',
        'fl': WS / '00-architecture/function-list.md',
        'prd': WS / '02-detailed/prd/unigate-prd.md',
        'domain': WS / '02-detailed/prd/unigate-domain-model.md',
        'api': WS / '02-detailed/api/api-unigate.md',
        'oas': WS / '02-detailed/api/openapi-unigate.json',
        'data': WS / '02-detailed/api/data-model-unigate.md',
        'pages': WS / '02-detailed/interaction/page-contracts.json',
        'overview': WS / '02-detailed/interaction/README.md',
    }
    for group in ['shared', 'admin', 'organizer', 'checker', 'reader', 'participant']:
        files[group] = WS / f'02-detailed/interaction/{group}/unigate/unigate.md'
    for name in ['03-common-enums.md', '04-business-enum.md', '05-select-service-pool.md']:
        files[name] = WS / '01-solution/common-spec' / name
    doc = {k: p.read_text(encoding='utf-8') for k, p in files.items()}
    spec = json.loads(doc['oas'])
    pages = json.loads(doc['pages'])
    ops = {o['name']: o for o in spec['x-operation-index']}
    results = []

    def check(name, condition, evidence):
        results.append({'name': name, 'status': 'PASS' if condition else 'FAIL', 'evidence': evidence})

    active = json.loads((ROOT / '.claude/settings.json').read_text(encoding='utf-8'))['env']
    check('active-chain', active['ACTIVE_PROFILE'] == 'unigate' and active['ACTIVE_STACKS'] == 'springboot-vue-web',
          'Business unigate and Spring Boot/Vue stack explicitly bound; engine-cli integrity evaluated separately')
    decisions = (ROOT / 'profiles/unigate/knowledge/design-decisions/index.md').read_text(encoding='utf-8')
    check('confirmation-chain', all(f'## ADR-{n}：' in decisions for n in ['039', '040', '041', '042', '043', '044'])
          and pages['status'] == 'confirmed' and pages['confirmation']['sevenItems'] == list(range(1, 8)),
          'Online scope, architecture, requirements, API, project basis, seven-item interaction confirmation recorded')
    ba3 = set(re.findall(r'^##### (2\.\d+\.\d+) ', doc['ba'], re.M))
    ba4 = set(re.findall(r'^###### (2\.\d+\.\d+\.\d+) ', doc['ba'], re.M))
    aa_refs = re.findall(r'追溯 BA (2\.\d+\.\d+(?:\.\d+)?)。', doc['aa'])
    check('ba-aa-map', len(ba3) == 6 and len(ba4) == 17 and set(aa_refs) == ba4
          and len(aa_refs) == len(set(aa_refs)), 'All 17 BA activities map once into AA; 6 BA capabilities retained per ADR-047')
    aa_leaves = set(re.findall(r'^##### (2\.\d+\.\d+\.\d+) ', doc['aa'], re.M))
    fl = re.findall(r'^\| (FL-[AOCRP]-\d+).*?\| AA (2\.\d+\.\d+\.\d+) ', doc['fl'], re.M)
    page_fl = {f for p in pages['pages'] for f in p['fl']}
    check('aa-fl-api-pages-map', len(fl) == len({f for f, _ in fl}) == 43 and len(aa_leaves) == 23
          and {a for _, a in fl} == aa_leaves == {a for o in ops.values() for a in o['aa']}
          and page_fl == {f for f, _ in fl}, '23 AA leaves / 43 FL functions map to API and 33 pages without new modules')
    check('prd-acceptance', len(set(re.findall(r'\| (AC-\d+) \|', doc['prd']))) == 16
          and all(role in doc['prd'] for role in ['系统管理员', '活动负责人', '检票员', '校方只读', '参与者']),
          '16 PRD acceptance scenarios and all five roles present; qualitative completeness manually reviewed')
    roots = ['Event', 'RegistrationParty', 'ImportBatch', 'SourceConnection', 'Device', 'ExportJob', 'CleanupRun', 'Account']
    check('domain-api-roots', all(n in doc['domain'] and n in doc['api'] for n in roots)
          and all(n in doc['domain'] for n in ['聚合根', '值对象', '过程数据', '子流程 I/O', '关联领域对象']),
          'Eight confirmed aggregate names (ADR-047) and five model categories trace to API service design')
    check('api-completeness', all(o['request'] in spec['components']['schemas'] and o['response'] in spec['components']['schemas']
          and o['data'] in spec['components']['schemas'] for o in ops.values()) and bool(spec.get('x-errors'))
          and all(re.fullmatch(r'/api/v1/[a-z]+/[a-z-]+/[a-z-]+', o['path']) for o in ops.values()),
          'All 85 operations have request/response/data schemas, shared errors, and stack-conforming paths')
    check('page-types', {'management_list', 'settings_form', 'detail_cards', 'review_workspace', 'import_steps', 'scanner', 'kiosk'}
          <= {p['layout'] for p in pages['pages']}, 'List/form/detail/review plus import/scanner/kiosk types covered')
    used = {a['operation'] for p in pages['pages'] for a in p['actions']}
    check('page-action-exactness', used | {p['operation'] for p in pages['protocolOnly']} == set(ops)
          and all(a['path'] == ops[a['operation']]['path'] and a['method'].lower() == ops[a['operation']]['method']
                  for p in pages['pages'] for a in p['actions']), 'Every page action matches API; OAuth callback remains protocol-only')
    states = spec['x-enum-pool']['EventStatus']
    check('event-state-cross-stage', all(s in doc['prd'] and s in doc['api'] for s in states)
          and 'ARCHIVED不代表所有副本已删除' in doc['prd'] and '先锁Event并ARCHIVED/停写' in doc['data'],
          'All event states present; access archival occurs before physical cleanup, whose progress remains separate')
    check('critical-guards', '仅READY' in ops['ReviewDecide']['notes'] and '仅READY' in ops['BatchCommit']['notes']
          and '仅CHECKIN_OPEN' in ops['GateWalkIn']['notes'] and '仅CHECKIN_OPEN' in ops['GateIncrease']['notes']
          and '仅OPEN/PAUSED' in ops['GateCorrect']['notes'], 'Preparation/review/import vs onsite increase/correction windows explicit')
    check('scanner-transition-evidence', {r[0] for r in pages['scannerStates']} ==
          {'READY', 'RESOLVING', 'CONFIRMING', 'SUBMITTING', 'SUCCESS', 'KNOWN_REPLAY', 'REJECTED', 'UNKNOWN', 'BLOCKED'}
          and all(len(row) == 4 and all(row) for row in pages['scannerStates']),
          'Each of nine scanner states has trigger, restrictions, and next-step design')
    check('in-progress-preserves-operation', all('OPERATION_IN_PROGRESS沿原ID' in a['failure']
          for p in pages['pages'] for a in p['actions'] if a['mutation'])
          and 'OPERATION_IN_PROGRESS' in doc['overview'],
          '409 in-progress is distinguished from definite business rejection; it cannot create a new operation')
    check('owner-only-special-actions', all(ops[name]['roles'] == ['ORGANIZER'] for name in
          ['GateWalkIn', 'GateIncrease', 'GateCorrect', 'CredentialReissue', 'CredentialRevoke', 'ExportCreate', 'ExportDownload'])
          and all(not ops[a['operation']]['mutation'] for p in pages['pages'] if p['group'] == 'reader' for a in p['actions']),
          'Special writes/export remain organizer-only; reader remains read-only')
    check('enum-single-source', len(spec['x-enum-pool']) == 29 and all(
          sum(len(re.findall(r'^## ' + re.escape(name) + r'\s*$', doc[f], re.M))
              for f in ['03-common-enums.md', '04-business-enum.md']) == 1 for name in spec['x-enum-pool']),
          'Each of 29 enums has one authoritative pool section; code/label contents checked by D3 validator')
    markdown = [k for k in files if k not in ['oas', 'pages', '03-common-enums.md', '04-business-enum.md', '05-select-service-pool.md']]
    check('artifact-changelogs', all(re.search(r'^## (?:\d+\. )?更新记录$', doc[k], re.M) and '2026-10-21' in doc[k] for k in markdown),
          'All 14 primary Markdown design artifacts have dated changelogs')
    d3 = json.loads((WS / '05-reports/D3-project-check.json').read_text(encoding='utf-8'))
    d4 = json.loads((WS / '05-reports/D4-project-check.json').read_text(encoding='utf-8'))
    check('phase-check-evidence', d3['passed'] == 37 and d3['failed'] == 0 and d3['projectBasisAdopted']
          and d4['passed'] == 26 and d4['failed'] == 0 and d4['interactionConfirmed']
          and d4['apiSha256'] == hashlib.sha256(files['oas'].read_bytes()).hexdigest()
          and d4['pageContractSha256'] == hashlib.sha256(files['pages'].read_bytes()).hexdigest(),
          'Current D3/D4 check evidence and API/page fingerprints match; these are document-only assertions')
    index = (WS / 'INDEX.md').read_text(encoding='utf-8')
    implementation_authorized = '## ADR-046：' in decisions
    check('design-index-readiness', index.startswith('---\n') and index.rstrip().endswith('---')
          and len(re.findall(r'^    status: complete$', index, re.M)) == 5
          and 'interaction: {status: complete' in index
          and (implementation_authorized or ('backend: {status: todo' in index
          and 'frontend: {status: todo' in index and 'gate2: {status: todo' in index)),
          'D0-D4 complete; later implementation state allowed only with recorded ADR-046 authorization')
    check('phase-authorization', implementation_authorized or
          (not (ROOT / 'backend').exists() and not (ROOT / 'frontend').exists()),
          'Design-only workspace required before pipeline; ADR-046 explicitly authorizes later implementation')
    report = {
        'scope': 'Gate1-independent-cross-stage-document-assertions',
        'timestamp': datetime.now(timezone.utc).isoformat(), 'basis': ['ADR-043', 'ADR-044', 'ADR-047'],
        'passed': sum(r['status'] == 'PASS' for r in results), 'failed': sum(r['status'] == 'FAIL' for r in results),
        'checks': results,
        'artifacts': {str(p.relative_to(ROOT)).replace('\\', '/'): hashlib.sha256(p.read_bytes()).hexdigest() for p in files.values()},
        'gateDecisionMadeByThisTool': False,
        'limitations': ['Final Gate1 requires manual review and a separate signed-off design report',
                        'Not a complete OpenAPI validator, application test, device check, or original-engine checklist PASS'],
    }
    if args.write_report:
        (WS / '05-reports/Gate1-cross-stage-check.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({k: report[k] for k in ['scope', 'passed', 'failed', 'gateDecisionMadeByThisTool']}))
    if report['failed']:
        print(json.dumps([r for r in results if r['status'] == 'FAIL'], ensure_ascii=False))
    return bool(report['failed'])


if __name__ == '__main__':
    sys.exit(main())
