"""Cross-stack implementation evidence; this tool does not decide Gate 2."""
from pathlib import Path
from datetime import datetime, timezone
import argparse
import hashlib
import json
import re
import sys
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parents[2]
WS = ROOT / 'profiles/unigate/workspace'


def read(path):
    return json.loads(path.read_text(encoding='utf-8'))


def compact(value):
    if isinstance(value, dict):
        return {k: compact(v) for k, v in value.items()
                if k not in ['description', 'example', 'examples', 'notes', 'aa']}
    if isinstance(value, list):
        return [compact(v) for v in value]
    return value


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--write-report', action='store_true')
    args = parser.parse_args()
    oas_path = WS / '02-detailed/api/openapi-unigate.json'
    oas = read(oas_path)
    pages = read(WS / '02-detailed/interaction/page-contracts.json')
    ui = read(ROOT / 'frontend/src/generated/contract.json')
    ops = oas['x-operation-index']
    names = {o['name'] for o in ops}
    checks = []

    def check(name, passed, evidence):
        checks.append(dict(name=name, status='PASS' if passed else 'FAIL', evidence=evidence))

    baseline = read(WS / '05-reports/Gate1-cross-stage-check.json')
    check('design-fingerprints', all(hashlib.sha256((ROOT / p).read_bytes()).hexdigest() == h
          for p, h in baseline['artifacts'].items()), 'All 19 adopted Gate1 design artifacts remain byte-identical')
    check('frontend-contract-fingerprint', ui['sourceSha256'] == hashlib.sha256(oas_path.read_bytes()).hexdigest(),
          'Generated runtime contract points to adopted OpenAPI')
    check('operation-path-auth-status-schema', ui['operations'] == compact(ops),
          f'All {len(ops)} operations, paths, methods, identities, roles, errors and status codes match')
    check('schemas-and-enums', ui['schemas'] == compact(oas['components']['schemas']) and ui['enums'] == oas['x-enum-pool'],
          f'All {len(ui["schemas"])} DTO schemas and {len(ui["enums"])} enum pools match')
    projected = [{k: p[k] for k in ['id', 'group', 'title', 'route', 'roles', 'layout', 'operations', 'primary', 'columns']}
                 | {k: [{f: b[f] for f in ['path', 'label', 'operation', 'visibility'] if f in b} for b in p[k]]
                    for k in ['requestFields', 'responseFields']} for p in pages['pages']]
    check('page-route-fields', ui['pages'] == projected and len({p['route'] for p in ui['pages']}) == len(ui['pages']),
          f'All {len(ui["pages"])} routes, actions, columns and field bindings match PageSpec')
    router = (ROOT / 'frontend/src/router.ts').read_text(encoding='utf-8')
    check('router-contract-binding', 'contract.pages.map' in router and 'path:page.route' in router,
          'Vue Router consumes the validated generated route set')
    types = (ROOT / 'frontend/src/generated/api-types.ts').read_text(encoding='utf-8')
    check('typed-operation-bindings', all(f"{o['name']}: {{ request: {o['request']}; data: {o['data']} }}" in types for o in ops),
          'Every operation has its adopted request/data TypeScript mapping')
    java = {p.name: p.read_text(encoding='utf-8') for p in (ROOT / 'backend/src/main/java/com/unigate').glob('*.java')}
    check('backend-shared-contract', 'getResourceAsStream("/openapi-unigate.json")' in java['Contract.java']
          and all(x in java['Contract.java'] for x in ['op.get("method")', 'op.get("path")', 'op.get("name")'])
          and all(x in java['ApiController.java'] for x in ['contract.routes.get', 'contract.validate', 'return checked(op,']),
          'Server loads the same method/path index and validates requests plus transaction success data')
    packaged = ROOT / 'backend/target/classes/openapi-unigate.json'
    check('backend-packaged-contract', packaged.exists() and packaged.read_bytes() == oas_path.read_bytes(),
          'Compiled backend resource equals adopted API, rather than a separately maintained path list')
    # Static presence is evidence of dispatch coverage, not proof that every branch was executed.
    executable = '\n'.join(java.values())
    literals = set(re.findall(r'"([A-Z][A-Za-z]+)"', executable))
    check('backend-operation-dispatch-presence', names <= literals,
          {'operationCount': len(names), 'missing': sorted(names - literals), 'kind': 'static presence; runtime tested separately'})
    ddl = (ROOT / 'backend/src/main/resources/db/migration/V1__unigate.sql').read_text(encoding='utf-8')
    tables = re.findall(r'create table\s+(ug_\w+)', ddl, re.I)
    check('normalized-migration', len(tables) == 25 and len(set(tables)) == 25,
          f'{len(tables)} normalized PostgreSQL tables; actual Flyway execution covered by database tests')
    reports = list((ROOT / 'backend/target/surefire-reports').glob('TEST-*.xml'))
    tests = dict(tests=0, failures=0, errors=0, skipped=0)
    for report in reports:
        attrs = ET.parse(report).getroot().attrib
        for key in tests:
            tests[key] += int(attrs.get(key, 0))
    check('backend-test-evidence', bool(reports) and tests['tests'] > 0 and not any(tests[k] for k in ['failures', 'errors', 'skipped']),
          tests)
    coverage_path = ROOT / 'backend/target/site/jacoco/jacoco.xml'
    backend_coverage = {}
    if coverage_path.exists():
        classes = ET.parse(coverage_path).getroot().findall('./package/class')
        for layer, suffix in [('services', 'Service'), ('controller', 'Controller')]:
            counters = [next(c for c in item.findall('counter') if c.get('type') == 'LINE')
                        for item in classes if item.get('name').split('$')[0].endswith(suffix)]
            covered = sum(int(c.get('covered')) for c in counters)
            total = sum(int(c.get('covered')) + int(c.get('missed')) for c in counters)
            backend_coverage[layer] = dict(covered=covered, total=total,
                                          percent=round(100 * covered / total, 2) if total else 0)
    check('measured-backend-layer-coverage', backend_coverage.get('services', {}).get('percent', 0) >= 80
          and backend_coverage.get('controller', {}).get('percent', 0) >= 60,
          {'metric': 'JaCoCo LINE; includes nested classes; no core service exclusions', **backend_coverage})
    frontend_coverage_path = ROOT / 'frontend/verification/coverage-summary.json'
    frontend_coverage = read(frontend_coverage_path) if frontend_coverage_path.exists() else {}
    check('measured-vue-component-coverage', frontend_coverage.get('allVue', {}).get('statements', {}).get('pct', 0) >= 60,
          {'metric': 'V8 statements; every Vue file including App; excludes no core implementation',
           'allVue': frontend_coverage.get('allVue'), 'total': frontend_coverage.get('total')})
    source_files = [p for p in (ROOT / 'frontend/src').rglob('*') if p.suffix in ['.ts', '.vue'] and p.name != 'api-types.ts']
    source_files += list((ROOT / 'frontend/tests').rglob('*.ts'))
    source_files += [ROOT / 'frontend/package-lock.json', ROOT / 'frontend/vite.config.ts']
    source_hash = hashlib.sha256()
    for file in sorted(source_files):
        source_hash.update(str(file.relative_to(ROOT / 'frontend')).replace('\\', '/').encode())
        source_hash.update(file.read_bytes())
    check('frontend-coverage-source-fingerprint', source_hash.hexdigest() == frontend_coverage.get('sourceAndTestSha256'),
          'Measured coverage matches current source, tests, lock and runner configuration')
    browser_path = ROOT / '.local/browser-e2e-result.json'
    browser = read(browser_path) if browser_path.exists() else {}
    check('real-browser-workflow-evidence', browser.get('realBackend') and browser.get('realPostgreSQL')
          and browser.get('failed') == 0 and len(browser.get('results', [])) >= 6,
          {'timestamp': browser.get('timestamp'), 'results': browser.get('results', []),
           'kind': 'local development service, Edge headless; no physical-device claim'})
    report = dict(scope='Gate2-cross-stack-evidence', timestamp=datetime.now(timezone.utc).isoformat(),
                  passed=sum(c['status'] == 'PASS' for c in checks), failed=sum(c['status'] == 'FAIL' for c in checks),
                  checks=checks, gateDecisionMadeByThisTool=False,
                  limitations=['Static contract/presence checks do not prove all runtime branches',
                               'Build, component tests, coverage and rule review are separate evidence',
                               'No real provider credentials, Android hardware or capacity certification'])
    if args.write_report:
        (WS / '05-reports/Gate2-cross-stack-check.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({k: report[k] for k in ['scope', 'passed', 'failed', 'gateDecisionMadeByThisTool']}))
    for c in checks:
        if c['status'] == 'FAIL':
            print(json.dumps(c, ensure_ascii=False))
    return bool(report['failed'])


if __name__ == '__main__':
    sys.exit(main())
