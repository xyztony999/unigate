import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
const root = path.resolve(import.meta.dirname, '../..');
const require = createRequire(path.join(root, '.local/browser/package.json'));
const { chromium } = require('playwright');
const local = path.join(root, '.local');
const runtimeFile = path.join(local, 'runtime.json');
const config = JSON.parse(fs.readFileSync(runtimeFile, 'utf8'));
const evidence = path.join(local, 'browser-evidence');
fs.mkdirSync(evidence, { recursive: true });
const results = [];
const id = () => crypto.randomUUID();
const base = process.env.UNIGATE_WEB_URL || 'http://localhost:5173';
const browser = await chromium.launch({ channel: 'msedge', headless: true });
const contexts = [];
async function pageFor(viewport = { width: 1440, height: 1000 }) {
  const context = await browser.newContext({ viewport }); contexts.push(context);
  const page = await context.newPage(); page.on('dialog', d => d.accept());
  return page;
}
async function call(page, op, payload = {}) {
  const result = await page.evaluate(async ({ op, payload }) => {
    const api = await import('/src/lib/api.ts');
    try {
      // During Vite HMR the app may import a timestamped module instance. Restore this
      // test client's auth from its real HttpOnly session rather than borrowing UI memory.
      const { operations } = await import('/src/lib/contract.ts');
      const auth = operations[op].auth;
      if (auth === 'STAFF' && !api.identity.staffCsrf) await api.call('SessionMe', {});
      if (['DEVICE', 'GATE'].includes(auth) && !api.identity.deviceCsrf) await api.call('DeviceDetail', {});
      return { ok: true, data: await api.call(op, payload) };
    }
    catch (e) { return { ok: false, code: e.code, status: e.status, message: e.message }; }
  }, { op, payload });
  assert.equal(result.ok, true, `${op}: ${result.code} ${result.message}`);
  return result.data;
}
async function login(page, password) {
  await page.goto(base + '/login');
  await page.getByLabel('登录账号', { exact: true }).fill(config.bootstrapUsername);
  await page.getByLabel('密码', { exact: true }).fill(password);
  await page.getByRole('button', { name: '登录', exact: true }).click();
  await page.waitForURL(url => !url.pathname.endsWith('/login'));
}
function pass(name) { results.push({ name, status: 'PASS' }); console.log('PASS ' + name); }
try {
  const admin = await pageFor();
  await login(admin, config.e2eAdminPassword || config.bootstrapPassword);
  if (admin.url().includes('/account/password')) {
    const next = 'UG-' + crypto.randomBytes(22).toString('base64url');
    config.pendingTestPassword = next;
    fs.writeFileSync(runtimeFile, JSON.stringify(config, null, 2) + '\n');
    await admin.getByLabel(/原密码|当前密码/).fill(config.e2eAdminPassword || config.bootstrapPassword);
    await admin.getByLabel('新密码', { exact: false }).fill(next);
    const response = admin.waitForResponse(r => r.url().endsWith('/identity/account/password-change') && r.request().method() === 'POST');
    await admin.getByRole('button', { name: '确认提交', exact: true }).click();
    // Persist before asserting navigation so a failed UI assertion cannot lose a changed password.
    const changed = await response;
    assert.equal(changed.status(), 200, 'Password change HTTP success');
    config.e2eAdminPassword = next; delete config.pendingTestPassword;
    fs.writeFileSync(runtimeFile, JSON.stringify(config, null, 2) + '\n');
    await admin.waitForURL('**/login?passwordChanged=1');
    pass('password-change-logs-out-and-navigates');
    await login(admin, next);
  }
  await admin.waitForURL('**/manage/events');
  pass('staff-browser-login');
  await admin.getByRole('button', { name: '创建活动草稿', exact: true }).click();
  const dialog = admin.getByRole('dialog');
  const start = new Date(Date.now() + 2 * 3600_000);
  const localTime = date => new Date(date.getTime() + 8 * 3600_000).toISOString().slice(0, 16);
  await dialog.getByLabel('活动名称', { exact: false }).fill('联调验收-' + Date.now());
  await dialog.getByLabel('开始时间', { exact: false }).fill(localTime(start));
  await dialog.getByLabel('结束时间', { exact: false }).fill(localTime(new Date(Date.now() + 5 * 3600_000)));
  await dialog.getByLabel('报名开始', { exact: false }).fill(localTime(new Date(Date.now() - 3600_000)));
  await dialog.getByLabel('报名截止', { exact: false }).fill(localTime(new Date(Date.now() + 3600_000)));
  await dialog.getByLabel('活动时区', { exact: false }).fill('Asia/Shanghai');
  await dialog.getByLabel(/总名额|人数容量/).fill('10');
  await dialog.getByRole('button', { name: '确认提交', exact: true }).click();
  await admin.waitForURL(/\/manage\/events\/[0-9a-f-]+\/settings$/);
  const eventId = admin.url().split('/').at(-2);
  pass('browser-create-event-and-success-navigation');
  const old = await call(admin, 'PolicyGet', { eventId });
  await call(admin, 'PolicyUpdate', { operationId: id(), eventId, policy: {
    fields: { name: { enabled: true, required: true }, phone: { enabled: false, required: false }, email: { enabled: false, required: false }, companionCount: { enabled: true, required: false } },
    approvalMode: 'IMMEDIATE', phoneVerificationEnabled: false, selfEditEnabled: false, selfCancelEnabled: false, retentionDays: 30, version: old.version,
  } });
  let event = await call(admin, 'EventDetail', { eventId });
  event = await call(admin, 'EventTransition', { operationId: id(), eventId, version: event.version, action: 'PUBLISH' });
  const participant = await pageFor({ width: 390, height: 844 });
  await participant.goto(`${base}/e/${eventId}/register`);
  await participant.getByLabel('姓名', { exact: false }).fill('联调家长');
  await participant.getByLabel('随行人数', { exact: false }).fill('2');
  await participant.getByRole('checkbox').check();
  await participant.getByRole('button', { name: '提交报名', exact: true }).click();
  await participant.waitForURL('**/me/party');
  await participant.getByRole('img', { name: '本组入场二维码' }).waitFor();
  const credential = await call(participant, 'MyCredential');
  const party = await call(participant, 'MyParty');
  assert.equal(party.approvedSize, 3);
  await participant.screenshot({ path: path.join(evidence, 'participant-ticket.png'), fullPage: true });
  pass('mobile-browser-registration-and-group-qr');
  event = await call(admin, 'EventDetail', { eventId });
  await call(admin, 'EventTransition', { operationId: id(), eventId, version: event.version, action: 'OPEN' });
  const device = await call(admin, 'DeviceRegister', { operationId: id(), eventId, name: '新生见面会自助入口', mode: 'KIOSK' });
  const kiosk = await pageFor({ width: 1280, height: 800 });
  await kiosk.goto(`${base}/terminal/activate#` + new URLSearchParams({ deviceId: device.device.deviceId, pairingCode: device.pairingCode }));
  await kiosk.getByRole('button', { name: '激活终端', exact: true }).click();
  await kiosk.waitForURL('**/terminal/kiosk');
  await kiosk.getByLabel('入场码', { exact: true }).fill(credential.ticketToken);
  await kiosk.getByRole('button', { name: '识别', exact: true }).click();
  await kiosk.getByRole('button', { name: '确认 1 人入场', exact: true }).waitFor();
  await kiosk.getByRole('button', { name: '增加一人', exact: true }).click();
  assert.equal(await kiosk.evaluate(() => document.documentElement.scrollHeight <= innerHeight), true, 'Landscape kiosk fits the viewport without vertical scrolling');
  await kiosk.screenshot({ path: path.join(evidence, 'kiosk-landscape-confirm.png'), fullPage: true });
  const actualColumns = await kiosk.locator('.scan-columns').evaluate(el => getComputedStyle(el).gridTemplateColumns);
  assert.equal(actualColumns.split(' ').length, 2, 'Landscape kiosk has two actual columns');
  await kiosk.getByRole('button', { name: '确认 2 人入场', exact: true }).click();
  await kiosk.getByRole('button', { name: '完成', exact: true }).waitFor();
  assert.match(await kiosk.locator('.gate-result').innerText(), /2/);
  await kiosk.getByRole('button', { name: '完成', exact: true }).click();
  await kiosk.getByLabel('入场码', { exact: true }).waitFor();
  assert.equal(await kiosk.evaluate(() => document.activeElement?.id), 'scan-input', 'Wedge input regains focus for the next group');
  pass('kiosk-activation-landscape-and-split-admission');
  await kiosk.getByLabel('入场码', { exact: true }).fill(credential.ticketToken);
  await kiosk.getByRole('button', { name: '识别', exact: true }).click();
  await kiosk.getByRole('button', { name: '确认 1 人入场', exact: true }).waitFor();
  let intercepted;
  const pattern = '**/api/v1/gate/checkin/commit';
  await kiosk.route(pattern, async route => {
    intercepted = route.request().postDataJSON();
    const committed = await route.fetch();
    assert.equal(committed.status(), 200);
    await route.abort('failed');
  });
  await kiosk.getByRole('button', { name: '确认 1 人入场', exact: true }).click();
  await kiosk.getByRole('button', { name: '查询原操作结果', exact: true }).waitFor();
  await kiosk.unroute(pattern);
  assert.ok(intercepted.operationId);
  await kiosk.screenshot({ path: path.join(evidence, 'kiosk-unknown.png'), fullPage: true });
  await kiosk.getByRole('button', { name: '查询原操作结果', exact: true }).click();
  await kiosk.waitForFunction(() => document.querySelector('.gate-result')?.textContent?.includes('已经记录'));
  assert.equal(await kiosk.locator('.gate-result.success').count(), 0, 'Original result never reopens green admission');
  const replay = await call(kiosk, 'GateCommit', intercepted);
  assert.equal(replay.replayed, true); assert.equal(replay.showAsNewAdmission, false);
  const stats = await call(admin, 'Statistics', { eventId });
  assert.equal(stats.netAdmissions, 3);
  pass('committed-response-loss-original-id-recovery-and-no-double-count');
  const storage = await participant.evaluate(() => ({ session: { ...sessionStorage }, local: { ...localStorage } }));
  assert.ok(!JSON.stringify(storage).includes('联调家长'));
  assert.ok(!JSON.stringify(storage).includes(credential.ticketToken));
  pass('no-roster-pii-or-ticket-secret-in-browser-storage');
  const readFailures = [];
  admin.on('response', async response => {
    if (response.url().includes('/api/v1/') && response.status() >= 400)
      readFailures.push({ status: response.status(), path: new URL(response.url()).pathname });
  });
  const readPages = [
    '/account/access', '/admin/accounts', '/admin/connections', '/admin/jobs',
    ...['settings', 'parties', 'reviews', 'sources', 'gate', 'onsite', 'records',
      'statistics', 'exports', 'lifecycle', 'jobs', 'access', 'audit'].map(p => `/manage/events/${eventId}/${p}`),
    `/manage/events/${eventId}/parties/${party.partyId}`,
  ];
  for (const route of readPages) {
    await admin.goto(base + route);
    await admin.waitForLoadState('networkidle');
    const errors = await admin.getByRole('alert').allTextContents();
    assert.deepEqual(errors, [], `Management page ${route} should load without an API error`);
  }
  assert.deepEqual(readFailures, [], 'Management initial reads obey their exact request contracts');
  pass('eighteen-management-pages-load-real-api-without-errors');
  let importEvent = await call(admin, 'EventCreate', {
    operationId: id(), title: '名单导入验收-' + Date.now(), timeZone: 'Asia/Shanghai', capacity: 10,
    startsAt: start.toISOString(), endsAt: new Date(Date.now() + 5 * 3600_000).toISOString(),
    registrationStartsAt: new Date(Date.now() - 3600_000).toISOString(),
    registrationEndsAt: new Date(Date.now() + 3600_000).toISOString(),
  });
  const importId = importEvent.eventId;
  const importPolicy = await call(admin, 'PolicyGet', { eventId: importId });
  await call(admin, 'PolicyUpdate', { operationId: id(), eventId: importId, policy: {
    fields: { name: { enabled: true, required: true }, phone: { enabled: false, required: false }, email: { enabled: false, required: false }, companionCount: { enabled: true, required: false } },
    approvalMode: 'IMMEDIATE', phoneVerificationEnabled: false, selfEditEnabled: false, selfCancelEnabled: false, retentionDays: 30, version: importPolicy.version,
  } });
  importEvent = await call(admin, 'EventDetail', { eventId: importId });
  await call(admin, 'EventTransition', { operationId: id(), eventId: importId, version: importEvent.version, action: 'PUBLISH' });
  await admin.goto(`${base}/manage/events/${importId}/imports`);
  await admin.waitForLoadState('networkidle');
  assert.equal(await admin.getByRole('dialog').count(), 0, 'Initial import screen does not ask customers for an unknown batch UUID');
  await admin.getByRole('button', { name: '上传名单', exact: true }).first().click();
  const uploadDialog = admin.getByRole('dialog');
  await uploadDialog.locator('input[type=file]').setInputFiles({ name: 'parents.csv', mimeType: 'text/csv', buffer: Buffer.from('\uFEFF姓名,随行人数\n导入家长甲,1\n导入家长乙,0\n') });
  await uploadDialog.getByLabel(/来源/).fill('个人问卷导出名单');
  await uploadDialog.getByRole('button', { name: '确认提交', exact: true }).click();
  await uploadDialog.waitFor({ state: 'hidden' });
  await admin.getByRole('button', { name: '映射校验预览', exact: true }).click();
  const mappingDialog = admin.getByRole('dialog');
  await mappingDialog.getByLabel('姓名列', { exact: false }).selectOption('姓名');
  await mappingDialog.getByLabel('人数列', { exact: false }).selectOption('随行人数');
  await mappingDialog.getByLabel(/人数含义|人数列含义/).selectOption('COMPANIONS');
  await mappingDialog.getByLabel(/资格/).selectOption('ELIGIBLE');
  await mappingDialog.getByRole('button', { name: '确认提交', exact: true }).click();
  await mappingDialog.waitFor({ state: 'hidden' });
  await admin.getByRole('button', { name: '预览行与错误', exact: true }).click();
  await admin.getByLabel('选择第1行', { exact: true }).check();
  await admin.getByLabel('选择第2行', { exact: true }).check();
  await admin.getByRole('button', { name: '选中行整批提交', exact: true }).click();
  await admin.getByRole('dialog').getByRole('button', { name: '确认提交', exact: true }).click();
  await admin.getByRole('dialog').waitFor({ state: 'hidden' });
  const imported = await call(admin, 'PartyList', { eventId: importId, page: 1, pageSize: 20 });
  assert.equal(imported.total, 2);
  assert.deepEqual(imported.items.map(p => p.approvedSize).sort(), [1, 2]);
  pass('browser-csv-upload-map-select-and-atomic-group-import');
  const report = { timestamp: new Date().toISOString(), target: base, browser: 'Microsoft Edge headless', realBackend: true, realPostgreSQL: true, results, failed: 0, limitations: ['Scanner wedge input exercised; no physical camera/Android/kiosk OS verification', 'No real external provider account or outbound notification'] };
  fs.writeFileSync(path.join(local, 'browser-e2e-result.json'), JSON.stringify(report, null, 2) + '\n');
} catch (error) {
  for (let i = 0; i < contexts.length; i++) {
    const page = contexts[i].pages().at(-1);
    if (page) { console.error('PAGE ' + i + ' ' + page.url() + '\n' + (await page.locator('body').innerText()).slice(0, 2500)); await page.screenshot({ path: path.join(evidence, `failure-${i}.png`), fullPage: true }); }
  }
  fs.writeFileSync(path.join(local, 'browser-e2e-result.json'), JSON.stringify({ timestamp: new Date().toISOString(), results, failed: 1, error: error.message }, null, 2) + '\n');
  throw error;
} finally { await browser.close(); }
