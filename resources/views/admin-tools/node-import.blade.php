<style>
/* ── Node Batch Import Tool ── */
#xb-nim-fab {
  position: fixed; right: 24px; bottom: 24px; z-index: 9000;
  display: flex; align-items: center; gap: 8px;
  padding: 0 18px; height: 44px;
  background: #6366f1; color: #fff;
  border: none; border-radius: 22px;
  font-size: 14px; font-weight: 500; cursor: pointer;
  box-shadow: 0 4px 14px rgba(99,102,241,.45);
  transition: transform .15s, box-shadow .15s; font-family: inherit;
}
#xb-nim-fab:hover { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(99,102,241,.55); }

#xb-nim-backdrop {
  display: none; position: fixed; inset: 0;
  background: rgba(0,0,0,.55); z-index: 9001; backdrop-filter: blur(4px);
}
#xb-nim-backdrop.open { display: block; }

#xb-nim-panel {
  position: fixed; top: 50%; left: 50%;
  transform: translate(-50%,-52%) scale(.96);
  z-index: 9002; width: min(900px, 96vw); max-height: 88vh;
  background: #18181b; color: #e4e4e7;
  border: 1px solid #27272a; border-radius: 16px;
  display: none; flex-direction: column; overflow: hidden;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  font-size: 14px;
  box-shadow: 0 32px 96px rgba(0,0,0,.8);
  transition: transform .2s cubic-bezier(.22,1,.36,1), opacity .2s; opacity: 0;
}
#xb-nim-panel.open { display: flex; transform: translate(-50%,-50%) scale(1); opacity: 1; }

/* Header */
#xb-nim-hd {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 22px; border-bottom: 1px solid #27272a; flex-shrink: 0;
}
#xb-nim-hd h2 { margin: 0; font-size: 15px; font-weight: 600; color: #fafafa; }
.xb-close-btn {
  width: 28px; height: 28px; border: none; background: #27272a; border-radius: 7px;
  color: #a1a1aa; cursor: pointer; display: flex; align-items: center; justify-content: center;
  transition: background .12s, color .12s;
}
.xb-close-btn:hover { background: #3f3f46; color: #fafafa; }

/* Body: two-column */
#xb-nim-body {
  display: grid; grid-template-columns: 1fr 290px; overflow: hidden; flex: 1; min-height: 0;
}
#xb-nim-left  { padding: 18px 22px; border-right: 1px solid #27272a; overflow-y: auto; display: flex; flex-direction: column; gap: 14px; }
#xb-nim-right { padding: 18px 18px; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }

/* Tabs */
.xb-tabs { display: flex; gap: 3px; background: #27272a; border-radius: 8px; padding: 3px; width: fit-content; }
.xb-tab {
  padding: 6px 14px; border: none; border-radius: 6px;
  background: transparent; color: #71717a; cursor: pointer;
  font-size: 13px; font-weight: 500; transition: background .12s, color .12s; font-family: inherit;
}
.xb-tab.active { background: #3f3f46; color: #fafafa; }

/* Inputs */
.xb-label { font-size: 12px; color: #71717a; margin-bottom: 5px; display: block; }
.xb-textarea {
  width: 100%; min-height: 170px; background: #09090b; border: 1px solid #27272a;
  border-radius: 8px; color: #e4e4e7; font-size: 12.5px; padding: 10px 12px;
  resize: vertical; font-family: 'Menlo','Consolas','Courier New',monospace; line-height: 1.6;
  box-sizing: border-box; transition: border-color .12s;
}
.xb-textarea:focus { outline: none; border-color: #6366f1; }
.xb-textarea::placeholder { color: #3f3f46; }
.xb-input-row { display: flex; gap: 8px; }
.xb-input {
  flex: 1; background: #09090b; border: 1px solid #27272a; border-radius: 8px;
  color: #e4e4e7; font-size: 13px; padding: 8px 12px; font-family: inherit;
  transition: border-color .12s; box-sizing: border-box;
}
.xb-input:focus { outline: none; border-color: #6366f1; }
.xb-input::placeholder { color: #3f3f46; }
.xb-number {
  width: 100%; background: #09090b; border: 1px solid #27272a; border-radius: 8px;
  color: #e4e4e7; font-size: 13px; padding: 7px 10px; font-family: inherit; box-sizing: border-box;
}
.xb-number:focus { outline: none; border-color: #6366f1; }
.xb-select {
  width: 100%; background: #09090b; border: 1px solid #27272a; border-radius: 8px;
  color: #e4e4e7; font-size: 13px; padding: 7px 10px; font-family: inherit;
}
.xb-select:focus { outline: none; border-color: #6366f1; }

/* Buttons */
.xb-btn {
  padding: 7px 16px; border: none; border-radius: 8px; cursor: pointer;
  font-size: 13px; font-weight: 500; font-family: inherit; transition: background .12s, opacity .12s;
  white-space: nowrap; display: inline-flex; align-items: center; gap: 6px;
}
.xb-btn-primary   { background: #6366f1; color: #fff; }
.xb-btn-primary:hover   { background: #5558e3; }
.xb-btn-primary:disabled { background: #27272a; color: #52525b; cursor: not-allowed; }
.xb-btn-secondary { background: #27272a; color: #d4d4d8; }
.xb-btn-secondary:hover { background: #3f3f46; }
.xb-btn-secondary:disabled { opacity: .5; cursor: not-allowed; }
.xb-btn-ghost { background: transparent; border: 1px solid #3f3f46; color: #a1a1aa; }
.xb-btn-ghost:hover { background: #27272a; }

/* Section label */
.xb-section {
  font-size: 11px; font-weight: 600; color: #52525b; text-transform: uppercase;
  letter-spacing: .06em; margin: 2px 0 8px;
}

/* Groups */
#xb-nim-groups {
  background: #09090b; border: 1px solid #27272a; border-radius: 8px;
  max-height: 100px; overflow-y: auto; padding: 4px;
}
.xb-group-item {
  display: flex; align-items: center; gap: 8px;
  padding: 5px 8px; border-radius: 6px; cursor: pointer; font-size: 13px; user-select: none;
}
.xb-group-item:hover { background: #18181b; }
.xb-group-item input { accent-color: #6366f1; cursor: pointer; }

/* Preview */
#xb-nim-preview { display: flex; flex-direction: column; gap: 3px; max-height: 220px; overflow-y: auto; }
.xb-prev-item {
  display: flex; align-items: center; gap: 7px;
  padding: 6px 8px; border-radius: 6px; background: #09090b; font-size: 12px;
}
.xb-prev-item.ok  { border-left: 2px solid #22c55e; }
.xb-prev-item.err { border-left: 2px solid #ef4444; }
.xb-badge {
  flex-shrink: 0; padding: 1px 5px; border-radius: 4px;
  font-size: 10px; font-weight: 700; letter-spacing: .02em;
}
.xb-b-vmess      { background:#1e3a8a;color:#93c5fd }
.xb-b-vless      { background:#4c1d95;color:#c4b5fd }
.xb-b-trojan     { background:#064e3b;color:#6ee7b7 }
.xb-b-shadowsocks{ background:#78350f;color:#fcd34d }
.xb-b-hysteria   { background:#831843;color:#fbcfe8 }
.xb-b-tuic       { background:#0c4a6e;color:#7dd3fc }
.xb-b-err        { background:#450a0a;color:#fca5a5 }
.xb-prev-name { flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; color:#e4e4e7 }
.xb-prev-host { color:#52525b; font-size:11px; flex-shrink:0 }

/* Result box */
#xb-nim-result {
  padding: 10px 12px; border-radius: 8px; background: #09090b;
  border: 1px solid #27272a; font-size: 13px; display: none; line-height: 1.6;
}
.c-ok  { color: #22c55e }
.c-err { color: #ef4444 }
.c-muted { color: #71717a }

/* Footer */
#xb-nim-ft {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 22px; border-top: 1px solid #27272a; flex-shrink: 0; gap: 10px;
}
.xb-ft-status { display: flex; align-items: center; gap: 6px; color: #71717a; font-size: 12px; }
.xb-ft-btns   { display: flex; gap: 8px; }

/* Spinner */
.xb-spin {
  width: 14px; height: 14px; border: 2px solid #3f3f46;
  border-top-color: #6366f1; border-radius: 50%; flex-shrink: 0;
  animation: xb-rot .65s linear infinite;
}
@keyframes xb-rot { to { transform: rotate(360deg); } }

/* Token warning */
#xb-nim-warn {
  padding: 9px 12px; background: #450a0a; border: 1px solid #7f1d1d;
  border-radius: 8px; color: #fca5a5; font-size: 13px; display: none;
}

.xb-empty-state { color: #52525b; font-size: 12px; padding: 14px 0; text-align: center; }
.xb-field { margin-bottom: 10px; }
.xb-field label { display: block; font-size: 12px; color: #71717a; margin-bottom: 5px; }
.xb-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }

/* Scrollbar */
:is(#xb-nim-left,#xb-nim-right,#xb-nim-preview,#xb-nim-groups)::-webkit-scrollbar { width: 4px; }
:is(#xb-nim-left,#xb-nim-right,#xb-nim-preview,#xb-nim-groups)::-webkit-scrollbar-thumb { background:#3f3f46; border-radius:2px; }

@media (max-width:640px) {
  #xb-nim-body { grid-template-columns: 1fr; }
  #xb-nim-left { border-right: none; border-bottom: 1px solid #27272a; }
}
</style>

<!-- ── Floating action button ── -->
<button id="xb-nim-fab" title="批量导入节点">
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round">
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
    <polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
  </svg>
  批量导入节点
</button>

<div id="xb-nim-backdrop"></div>

<!-- ── Panel ── -->
<div id="xb-nim-panel" role="dialog" aria-modal="true" aria-label="批量导入节点">

  <div id="xb-nim-hd">
    <h2>批量导入节点</h2>
    <button class="xb-close-btn" id="xb-nim-close">
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
        <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
      </svg>
    </button>
  </div>

  <div id="xb-nim-body">

    <!-- LEFT: input -->
    <div id="xb-nim-left">
      <div id="xb-nim-warn">请先登录管理后台，再使用此工具。</div>

      <div class="xb-tabs" role="tablist">
        <button class="xb-tab active" data-tab="links" role="tab">粘贴链接</button>
        <button class="xb-tab" data-tab="sub" role="tab">订阅链接</button>
      </div>

      <div id="xb-nim-tab-links">
        <label class="xb-label">每行一条代理链接，支持 vmess / vless / trojan / ss / hysteria2 / tuic</label>
        <textarea id="xb-nim-links" class="xb-textarea"
          placeholder="vmess://eyJ2IjoyLCJwcyI6Ii4uLiJ9&#10;vless://uuid@host:443?type=tcp&security=reality#名称&#10;trojan://pass@host:443?sni=x.com#名称&#10;ss://YWVzLTI1Ni1nY206cGFzc0AxLjIuMy40OjgwODA#名称&#10;hysteria2://auth@host:443?sni=x.com#名称&#10;tuic://uuid:pass@host:443?sni=x.com#名称"></textarea>
      </div>

      <div id="xb-nim-tab-sub" style="display:none">
        <label class="xb-label">订阅地址（支持 Base64 编码或明文的代理链接列表）</label>
        <input id="xb-nim-sub-url" class="xb-input" type="url" placeholder="https://example.com/sub/your-token" />
        <p style="margin:6px 0 0;font-size:11.5px;color:#52525b">点击"解析预览"将自动拉取订阅内容并解析</p>
      </div>
    </div>

    <!-- RIGHT: settings + preview -->
    <div id="xb-nim-right">
      <div class="xb-section">导入设置</div>

      <div class="xb-field">
        <label>权限组 <span style="color:#3f3f46">(导入后可在节点详情修改)</span></label>
        <div id="xb-nim-groups"><div class="xb-empty-state">加载中...</div></div>
      </div>

      <div class="xb-grid-2">
        <div class="xb-field">
          <label>倍率</label>
          <input id="xb-nim-rate" class="xb-number" type="number" value="1" min="0" step="0.1" />
        </div>
        <div class="xb-field">
          <label>显示状态</label>
          <select id="xb-nim-show" class="xb-select">
            <option value="0">隐藏</option>
            <option value="1">显示</option>
          </select>
        </div>
      </div>

      <div class="xb-section" style="margin-top:4px">
        解析预览
        <span id="xb-nim-cnt" style="font-weight:400;color:#71717a;text-transform:none;letter-spacing:0;margin-left:4px;font-size:12px"></span>
      </div>

      <div id="xb-nim-preview"><div class="xb-empty-state">点击"解析预览"查看节点列表</div></div>

      <div id="xb-nim-result"></div>
    </div>
  </div>

  <div id="xb-nim-ft">
    <div class="xb-ft-status" id="xb-nim-status"></div>
    <div class="xb-ft-btns">
      <button class="xb-btn xb-btn-ghost" id="xb-nim-cancel">取消</button>
      <button class="xb-btn xb-btn-secondary" id="xb-nim-parse">解析预览</button>
      <button class="xb-btn xb-btn-primary" id="xb-nim-import" disabled>导入节点</button>
    </div>
  </div>
</div>

<script>
(function () {
  'use strict';

  // ── Auth & API ──────────────────────────────────────────────────
  // auth_data already contains "Bearer xxx", use it directly
  function getAuthHeader() {
    const raw = sessionStorage.getItem('Xboard_access_token')
             || localStorage.getItem('Xboard_access_token')
             || '';
    return raw;
  }

  function apiBase() {
    const s = window.settings || {};
    const base = (s.base_url || '/').replace(/\/$/, '');
    return base + '/api/v2/' + (s.secure_path || '');
  }

  async function post(path, body) {
    const r = await fetch(apiBase() + path, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': getAuthHeader(),
      },
      body: JSON.stringify(body),
    });
    const json = await r.json();
    if (!r.ok) throw new Error(json.message || ('HTTP ' + r.status));
    return json;
  }

  async function get(path) {
    const r = await fetch(apiBase() + path, {
      headers: { 'Authorization': getAuthHeader() },
    });
    const json = await r.json();
    if (!r.ok) throw new Error(json.message || ('HTTP ' + r.status));
    return json;
  }

  // ── Elements ────────────────────────────────────────────────────
  const $ = id => document.getElementById(id);
  const fab      = $('xb-nim-fab');
  const backdrop = $('xb-nim-backdrop');
  const panel    = $('xb-nim-panel');
  const warn     = $('xb-nim-warn');
  const closeBtn = $('xb-nim-close');
  const cancelBtn= $('xb-nim-cancel');
  const parseBtn = $('xb-nim-parse');
  const importBtn= $('xb-nim-import');
  const linksEl  = $('xb-nim-links');
  const subUrlEl = $('xb-nim-sub-url');
  const groupsEl = $('xb-nim-groups');
  const rateEl   = $('xb-nim-rate');
  const showEl   = $('xb-nim-show');
  const previewEl= $('xb-nim-preview');
  const cntEl    = $('xb-nim-cnt');
  const resultEl = $('xb-nim-result');
  const statusEl = $('xb-nim-status');

  let tab = 'links';
  let parsedServers = [];
  let groupsLoaded = false;

  // ── Draggable FAB ───────────────────────────────────────────────
  (function initDrag() {
    let dragging = false, moved = false;
    let startX, startY, startRight, startBottom;

    const savedPos = (() => { try { return JSON.parse(localStorage.getItem('xb-nim-fab-pos') || 'null'); } catch { return null; } })();
    if (savedPos) {
      fab.style.right  = Math.max(8, Math.min(window.innerWidth  - fab.offsetWidth  - 8, savedPos.right))  + 'px';
      fab.style.bottom = Math.max(8, Math.min(window.innerHeight - fab.offsetHeight - 8, savedPos.bottom)) + 'px';
    }

    fab.addEventListener('mousedown', e => {
      dragging = true; moved = false;
      startX = e.clientX; startY = e.clientY;
      const rect = fab.getBoundingClientRect();
      startRight  = window.innerWidth  - rect.right;
      startBottom = window.innerHeight - rect.bottom;
      fab.style.transition = 'none';
      e.preventDefault();
    });

    document.addEventListener('mousemove', e => {
      if (!dragging) return;
      const dx = e.clientX - startX, dy = e.clientY - startY;
      if (Math.abs(dx) > 4 || Math.abs(dy) > 4) moved = true;
      const r = Math.max(8, Math.min(window.innerWidth  - fab.offsetWidth  - 8, startRight  - dx));
      const b = Math.max(8, Math.min(window.innerHeight - fab.offsetHeight - 8, startBottom - dy));
      fab.style.right  = r + 'px';
      fab.style.bottom = b + 'px';
      localStorage.setItem('xb-nim-fab-pos', JSON.stringify({ right: r, bottom: b }));
    });

    document.addEventListener('mouseup', () => {
      dragging = false;
      fab.style.transition = '';
    });

    fab.addEventListener('click', e => {
      if (moved) { moved = false; return; }
      open();
    });
  }());

  // ── Open / close ────────────────────────────────────────────────
  function open() {
    backdrop.classList.add('open');
    panel.classList.add('open');
    warn.style.display = getAuthHeader() ? 'none' : 'block';
    if (!groupsLoaded) loadGroups();
  }
  function close() {
    backdrop.classList.remove('open');
    panel.classList.remove('open');
  }

  backdrop.addEventListener('click', close);
  closeBtn.addEventListener('click', close);
  cancelBtn.addEventListener('click', close);
  document.addEventListener('keydown', e => { if (e.key === 'Escape' && panel.classList.contains('open')) close(); });

  // ── Tabs ────────────────────────────────────────────────────────
  document.querySelectorAll('.xb-tab').forEach(t => {
    t.addEventListener('click', () => {
      tab = t.dataset.tab;
      document.querySelectorAll('.xb-tab').forEach(x => x.classList.toggle('active', x === t));
      $('xb-nim-tab-links').style.display = tab === 'links' ? '' : 'none';
      $('xb-nim-tab-sub').style.display   = tab === 'sub'   ? '' : 'none';
      resetPreview();
    });
  });

  // ── Load groups ─────────────────────────────────────────────────
  async function loadGroups() {
    try {
      const res = await get('/server/group/fetch');
      const list = res.data || [];
      if (!list.length) {
        groupsEl.innerHTML = '<div class="xb-empty-state">暂无权限组</div>';
        return;
      }
      groupsEl.innerHTML = list.map(g =>
        `<label class="xb-group-item">
          <input type="checkbox" value="${g.id}">
          <span>${esc(g.name)}</span>
        </label>`
      ).join('');
      groupsLoaded = true;
    } catch {
      groupsEl.innerHTML = '<div class="xb-empty-state" style="color:#ef4444">加载权限组失败</div>';
    }
  }

  function selectedGroups() {
    return Array.from(groupsEl.querySelectorAll('input:checked')).map(el => +el.value);
  }

  // ── Parse preview ───────────────────────────────────────────────
  parseBtn.addEventListener('click', async () => {
    resetPreview();
    setBusy(true, '解析中...');
    try {
      const body = tab === 'links'
        ? { type: 'links', content: linksEl.value }
        : { type: 'subscription', url: subUrlEl.value.trim() };

      const res = await post('/server/manage/parseNodes', body);
      const { servers = [], parse_errors = [] } = res.data || {};

      parsedServers = servers;
      renderPreview(servers, parse_errors);
      importBtn.disabled = servers.length === 0;
      importBtn.textContent = servers.length > 0 ? `导入 ${servers.length} 个节点` : '导入节点';
    } catch (e) {
      previewEl.innerHTML = `<div class="xb-empty-state" style="color:#ef4444">解析出错: ${esc(e.message)}</div>`;
    } finally {
      setBusy(false);
    }
  });

  // ── Import ──────────────────────────────────────────────────────
  importBtn.addEventListener('click', async () => {
    if (importBtn.disabled) return;
    importBtn.disabled = true;
    parseBtn.disabled  = true;
    setBusy(true, `正在导入 ${parsedServers.length} 个节点...`);
    resultEl.style.display = 'none';

    try {
      const gids = selectedGroups();
      const defaults = {
        rate: parseFloat(rateEl.value) || 1,
        show: parseInt(showEl.value),
        ...(gids.length ? { group_ids: gids } : {}),
      };
      const body = tab === 'links'
        ? { type: 'links', content: linksEl.value, defaults }
        : { type: 'subscription', url: subUrlEl.value.trim(), defaults };

      const res = await post('/server/manage/batchImport', body);
      const { created = 0, failed = 0, parse_errors = [], results = [] } = res.data || {};

      renderPreview(results, parse_errors);
      showResult(created, failed, parse_errors.length);

      // Signal the React app to refresh node list
      window.dispatchEvent(new CustomEvent('xb-nodes-imported', { detail: { created } }));
    } catch (e) {
      resultEl.innerHTML = `<span class="c-err">导入出错: ${esc(e.message)}</span>`;
      resultEl.style.display = 'block';
    } finally {
      setBusy(false);
      parseBtn.disabled  = false;
      importBtn.disabled = parsedServers.length === 0;
    }
  });

  // ── Render helpers ──────────────────────────────────────────────
  const BADGE_CLS = {
    vmess: 'xb-b-vmess', vless: 'xb-b-vless', trojan: 'xb-b-trojan',
    shadowsocks: 'xb-b-shadowsocks', hysteria: 'xb-b-hysteria', tuic: 'xb-b-tuic',
  };
  const TYPE_LBL = {
    vmess: 'VMess', vless: 'VLESS', trojan: 'Trojan',
    shadowsocks: 'SS', hysteria: 'HY2', tuic: 'TUIC',
  };

  function renderPreview(servers, errors) {
    const rows = servers.map(s => {
      if (s.success === false) {
        return `<div class="xb-prev-item err">
          <span class="xb-badge xb-b-err">ERR</span>
          <span class="xb-prev-name">${esc(s.error || '创建失败')}</span>
        </div>`;
      }
      const t   = s.type || '';
      const lbl = TYPE_LBL[t] || t.toUpperCase();
      const cls = BADGE_CLS[t] || '';
      const host = s.host ? `${s.host}:${s.port}` : (s.id ? `#${s.id}` : '');
      return `<div class="xb-prev-item ok">
        <span class="xb-badge ${cls}">${lbl}</span>
        <span class="xb-prev-name">${esc(s.name || '')}</span>
        <span class="xb-prev-host">${esc(host)}</span>
      </div>`;
    });

    errors.forEach(e => {
      rows.push(`<div class="xb-prev-item err">
        <span class="xb-badge xb-b-err">SKIP</span>
        <span class="xb-prev-name">${esc((e.line ? `第${e.line}行: ` : '') + (e.error || ''))}</span>
      </div>`);
    });

    const ok = servers.filter(s => s.success !== false).length;
    const bad = servers.filter(s => s.success === false).length + errors.length;
    cntEl.textContent = ok ? `(${ok} 个节点${bad ? `，${bad} 个跳过` : ''})` : (bad ? `(${bad} 条无法识别)` : '');
    previewEl.innerHTML = rows.length ? rows.join('') : '<div class="xb-empty-state">未解析到任何节点</div>';
  }

  function showResult(created, failed, skipped) {
    const parts = [];
    if (created) parts.push(`<span class="c-ok">✓ 成功导入 ${created} 个节点</span>`);
    if (failed)  parts.push(`<span class="c-err">✗ ${failed} 个失败</span>`);
    if (skipped) parts.push(`<span class="c-muted">跳过 ${skipped} 条无法识别</span>`);
    resultEl.innerHTML = parts.join('　');
    resultEl.style.display = 'block';
  }

  function resetPreview() {
    parsedServers = [];
    previewEl.innerHTML = '<div class="xb-empty-state">点击"解析预览"查看节点列表</div>';
    cntEl.textContent = '';
    importBtn.disabled = true;
    importBtn.textContent = '导入节点';
    resultEl.style.display = 'none';
    resultEl.innerHTML = '';
  }

  function setBusy(on, text = '') {
    const icon = on ? '<div class="xb-spin"></div>' : '';
    statusEl.innerHTML = icon + (text ? `<span>${esc(text)}</span>` : '');
    parseBtn.disabled  = on;
  }

  function esc(s) {
    return String(s ?? '')
      .replace(/&/g,'&amp;').replace(/</g,'&lt;')
      .replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }
}());
</script>
