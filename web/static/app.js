const state = { cases: [], current: null };
const $ = (selector) => document.querySelector(selector);
const statusNames = { draft: '草稿', returned: '待整改', review_pending: '待复核', approved: '已批准', frozen: '已冻结' };
const eventNames = { CaseCreated: '案卷已建立', RevisionSubmitted: '修订与测量证据已提交', FindingResolved: '质量发现已整改', ReviewDecided: '独立复核决定已记录', CaseFrozen: '案卷已冻结并签发凭据' };

async function api(path, options = {}) {
  const response = await fetch(path, options);
  const contentType = response.headers.get('content-type') || '';
  const body = contentType.includes('json') ? await response.json() : await response.text();
  if (!response.ok) throw new Error(typeof body === 'string' ? body.trim() : body.message || '请求失败');
  return body;
}

function commandKey(prefix) {
  return `${prefix}-${Date.now()}-${crypto.getRandomValues(new Uint32Array(1))[0]}`;
}

function toast(message, error = false) {
  const element = $('#toast');
  element.textContent = message;
  element.classList.toggle('error', error);
  element.classList.add('show');
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => element.classList.remove('show'), 3200);
}

function escapeHTML(value) {
  const span = document.createElement('span');
  span.textContent = String(value ?? '');
  return span.innerHTML;
}

async function loadCases(selectID = state.current?.caseID) {
  state.cases = await api('/api/cases');
  $('#case-list').innerHTML = state.cases.length
    ? state.cases.map((item) => `
      <button class="case-entry ${item.caseID === selectID ? 'selected' : ''}" data-case-id="${escapeHTML(item.caseID)}" type="button">
        <span>${escapeHTML(item.archiveUnit)}</span><strong>${escapeHTML(item.title)}</strong>
        <small>${statusNames[item.status] || item.status} · v${item.version}</small>
      </button>`).join('')
    : '<p class="rail-empty">案卷簿尚无记录</p>';
}

async function selectCase(caseID) {
  state.current = await api(`/api/cases/${encodeURIComponent(caseID)}`);
  $('#empty-state').hidden = true;
  $('#case-workspace').hidden = false;
  renderCurrent();
  await loadTimeline();
  await loadCases(caseID);
}

function renderCurrent() {
  const item = state.current;
  $('#case-unit').textContent = item.archiveUnit;
  $('#case-title').textContent = item.title;
  $('#case-meta').textContent = `创建者 ${item.creator} · 案卷号 ${item.caseID}`;
  $('#case-status').textContent = statusNames[item.status] || item.status;
  $('#case-status').dataset.status = item.status;
  $('#case-version').textContent = `VERSION ${String(item.version).padStart(2, '0')}`;
  document.querySelectorAll('.workflow-strip span').forEach((step) => step.classList.toggle('active', step.dataset.stage === item.status));
  const revision = item.revisions?.find((entry) => entry.revisionID === item.currentRevisionID);
  renderFindings(revision?.findings || []);
  $('#revision-form button').disabled = item.status === 'frozen';
  $('#review-form button').disabled = item.status !== 'review_pending';
  $('#freeze-button').disabled = item.status !== 'approved';
  $('#verify-button').disabled = item.status !== 'frozen';
  $('#manifest-link').hidden = item.status !== 'frozen';
  $('#manifest-link').href = `/api/cases/${encodeURIComponent(item.caseID)}/manifest`;
  if (item.status !== 'frozen') $('#verification-result').textContent = '等待冻结发布';
}

function renderFindings(findings) {
  const blocking = findings.filter((item) => item.severity === 'blocking' && item.status === 'open').length;
  const notice = findings.filter((item) => item.severity === 'notice' && item.status === 'open').length;
  $('#findings-summary').innerHTML = `
    <div><strong>${blocking}</strong><span>阻断项</span></div>
    <div><strong>${notice}</strong><span>提示项</span></div>
    <div><strong>${findings.filter((item) => item.status === 'resolved').length}</strong><span>已整改</span></div>`;
  $('#findings-list').innerHTML = findings.length
    ? findings.map((finding) => `
      <article class="finding ${finding.severity} ${finding.status}">
        <div><span>第 ${finding.pageNumber} 页 · ${escapeHTML(finding.ruleCode)}</span><strong>${escapeHTML(finding.evidence)}</strong></div>
        ${finding.status === 'open' ? `<button data-finding-id="${escapeHTML(finding.findingID)}" type="button">登记整改</button>` : `<small>${escapeHTML(finding.resolutionNote || '已整改')}</small>`}
      </article>`).join('')
    : '<p class="muted">当前修订未产生质量发现。</p>';
}

async function loadTimeline() {
  const events = await api(`/api/cases/${encodeURIComponent(state.current.caseID)}/timeline`);
  $('#event-count').textContent = `${events.length} 条事件`;
  $('#timeline').innerHTML = events.map((event) => `
    <li><time>${new Date(event.at).toLocaleString('zh-CN')}</time><strong>${eventNames[event.type] || event.type}</strong><span>${escapeHTML(event.actor)} · ${escapeHTML(event.role)}</span></li>`).join('');
}

async function run(action, success) {
  try { await action(); toast(success); } catch (error) { toast(error.message, true); }
}

$('#case-form').addEventListener('submit', (event) => {
  event.preventDefault();
  run(async () => {
    const data = Object.fromEntries(new FormData(event.currentTarget));
    data.idempotencyKey = commandKey('web-case');
    const item = await api('/api/cases', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) });
    event.currentTarget.reset();
    event.currentTarget.creator.value = '档案管理员';
    await selectCase(item.caseID);
  }, '案卷已建立并写入审计记录');
});

$('#revision-form').addEventListener('submit', (event) => {
  event.preventDefault();
  run(async () => {
    const data = Object.fromEntries(new FormData(event.currentTarget));
    const revision = state.current.revisions?.find((entry) => entry.revisionID === state.current.currentRevisionID);
    const payload = {
      actor: data.actor, role: 'conservator', expectedVersion: state.current.version,
      idempotencyKey: commandKey('web-revision'), parentRevisionID: revision?.revisionID || '',
      metadata: { period: data.period },
      pages: [{ pageNumber: Number(data.pageNumber), clarity: Number(data.clarity), skew: Number(data.skew), cropRatio: Number(data.cropRatio), colorTarget: data.colorTarget === 'on' }]
    };
    state.current = await api(`/api/cases/${encodeURIComponent(state.current.caseID)}/revisions`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
    renderCurrent(); await loadTimeline(); await loadCases(state.current.caseID);
  }, '修订已保存，质量规则已重新计算');
});

$('#findings-list').addEventListener('click', (event) => {
  const button = event.target.closest('[data-finding-id]');
  if (!button) return;
  const note = window.prompt('请填写整改说明：');
  if (!note?.trim()) return;
  run(async () => {
    state.current = await api(`/api/cases/${encodeURIComponent(state.current.caseID)}/findings`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ findingID: button.dataset.findingId, actor: '文物保护员', role: 'conservator', expectedVersion: state.current.version, idempotencyKey: commandKey('web-finding'), note: note.trim() })
    });
    renderCurrent(); await loadTimeline(); await loadCases(state.current.caseID);
  }, '整改证据已登记');
});

$('#review-form').addEventListener('submit', (event) => {
  event.preventDefault();
  run(async () => {
    const data = Object.fromEntries(new FormData(event.currentTarget));
    state.current = await api(`/api/cases/${encodeURIComponent(state.current.caseID)}/review`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ actor: data.actor, role: 'reviewer', approved: data.approved === 'true', note: data.note, expectedVersion: state.current.version, idempotencyKey: commandKey('web-review') })
    });
    renderCurrent(); await loadTimeline(); await loadCases(state.current.caseID);
  }, '复核决定已记录');
});

$('#freeze-button').addEventListener('click', () => run(async () => {
  const result = await api(`/api/cases/${encodeURIComponent(state.current.caseID)}/freeze`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ actor: '发布复核员', role: 'reviewer', expectedVersion: state.current.version, idempotencyKey: commandKey('web-freeze') })
  });
  state.current = result.case;
  renderCurrent();
  $('#verification-result').textContent = `清单 ${result.manifest.manifestID}\n算法 ${result.credential.algorithm}\n摘要 ${result.manifest.contentDigest}`;
  await loadTimeline(); await loadCases(state.current.caseID);
}, '发布物已冻结，验证凭据已签发'));

$('#verify-button').addEventListener('click', () => run(async () => {
  const result = await api(`/api/cases/${encodeURIComponent(state.current.caseID)}/verify`);
  $('#verification-result').textContent = result.valid ? `✓ 凭据有效\n${result.reason || '清单摘要与签名一致'}` : `✕ 凭据无效\n${result.reason}`;
}, '完整性验证已完成'));

$('#case-list').addEventListener('click', (event) => {
  const entry = event.target.closest('[data-case-id]');
  if (entry) run(() => selectCase(entry.dataset.caseId), '案卷已载入');
});
$('#refresh-button').addEventListener('click', () => run(() => loadCases(), '案卷簿已刷新'));
$('#new-case-button').addEventListener('click', () => $('#case-form input[name="title"]').focus());
run(() => loadCases(), '案卷簿已载入');
