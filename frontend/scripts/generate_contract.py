"""Generate checked-in UI contract and TypeScript DTOs from the adopted OpenAPI/PageSpec."""
import hashlib
import json
from pathlib import Path

root = Path(__file__).resolve().parents[2]
oas_path = root / 'profiles/unigate/workspace/02-detailed/api/openapi-unigate.json'
pages_path = root / 'profiles/unigate/workspace/02-detailed/interaction/page-contracts.json'
oas = json.loads(oas_path.read_text(encoding='utf-8'))
pages = json.loads(pages_path.read_text(encoding='utf-8'))
out = root / 'frontend/src/generated'
out.mkdir(exist_ok=True, parents=True)
def compact(value):
    if isinstance(value, dict):
        return {k: compact(v) for k, v in value.items() if k not in ['description', 'example', 'examples', 'notes', 'aa']}
    if isinstance(value, list):
        return [compact(v) for v in value]
    return value
ui_pages = [{k: p[k] for k in ['id', 'group', 'title', 'route', 'roles', 'layout', 'operations', 'primary', 'columns']}
            | {k: [{f: b[f] for f in ['path', 'label', 'operation', 'visibility'] if f in b} for b in p[k]]
               for k in ['requestFields', 'responseFields']} for p in pages['pages']]
payload = {'schemas': compact(oas['components']['schemas']), 'operations': compact(oas['x-operation-index']),
           'enums': oas['x-enum-pool'], 'pages': ui_pages,
           'sourceSha256': hashlib.sha256(oas_path.read_bytes()).hexdigest()}
(out / 'contract.json').write_text(json.dumps(payload, ensure_ascii=False, separators=(',', ':')), encoding='utf-8')

def ts(s):
    if '$ref' in s:
        return s['$ref'].split('/')[-1]
    if 'enum' in s:
        return ' | '.join(json.dumps(x, ensure_ascii=False) for x in s['enum'])
    if 'oneOf' in s or 'anyOf' in s:
        return ' | '.join(ts(x) for x in s.get('oneOf', s.get('anyOf', [])))
    t = s.get('type')
    if isinstance(t, list):
        return ' | '.join(ts({**s, 'type': x}) for x in t)
    if t == 'object':
        required = s.get('required', [])
        return '{ ' + '; '.join(json.dumps(k) + ('' if k in required else '?') + ': ' + ts(v)
                                 for k, v in s.get('properties', {}).items()) + ' }'
    if t == 'array':
        return 'Array<' + ts(s['items']) + '>'
    return {'string': 'string', 'integer': 'number', 'number': 'number', 'boolean': 'boolean', 'null': 'null'}.get(t, 'unknown')

lines = ['// Generated from the adopted OpenAPI. Run npm run generate:contract to refresh.']
for name, schema in oas['components']['schemas'].items():
    lines.append('export type ' + name + ' = ' + ts(schema))
lines.append('export interface OperationMap {')
for op in oas['x-operation-index']:
    lines.append(f"  {op['name']}: {{ request: {op['request']}; data: {op['data']} }}")
lines.append('}')
(out / 'api-types.ts').write_text('\n'.join(lines) + '\n', encoding='utf-8')
print(f"Generated {len(oas['x-operation-index'])} operations / {len(pages['pages'])} pages")
