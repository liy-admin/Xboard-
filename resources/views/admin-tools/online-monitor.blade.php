<style>
/* ── Online Monitor Tool ── */
#xb-om-fab {
  position: fixed; right: 24px; bottom: 76px; z-index: 9000;
  display: flex; align-items: center; gap: 8px;
  padding: 0 16px; height: 44px;
  background: #18181b; color: #e4e4e7;
  border: 1px solid #3f3f46; border-radius: 22px;
  font-size: 14px; font-weight: 500; cursor: pointer;
  box-shadow: 0 4px 14px rgba(0,0,0,.4);
  transition: transform .15s, box-shadow .15s; font-family: inherit;
}
#xb-om-fab:hover { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(0,0,0,.5); border-color: #6366f1; }
#xb-om-badge {
  min-width: 20px; height: 20px; padding: 0 6px;
  background: #6366f1; color: #fff; border-radius: 10px;
  font-size: 11px; font-weight: 700; display: inline-flex; align-items: center; justify-content: center;
}

#xb-om-panel {
  position: fixed; top: 50%; left: 50%;
  transform: translate(-50%,-52%) scale(.96);
  z-index: 9002; width: min(640px, 96vw); max-height: 80vh;
  background: #18181b; color: #e4e4e7;
  border: 1px solid #27272a; border-radius: 16px;
  display: none; flex-direction: column; overflow: hidden;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  font-size: 14px; box-shadow: 0 32px 96px rgba(0,0,0,.8);
  transition: transform .2s cubic-bezier(.22,1,.36,1), opacity .2s; opacity: 0;
}
#xb-om-panel.open { display: flex; transform: translate(-50%,-50%) scale(1); opacity: 1; }

#xb-om-hd {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 22px; border-bottom: 1px solid #27272a; flex-shrink: 0;
}
#xb-om-hd h2 { margin: 0; font-size: 15px; font-weight: 600; color: #fafafa; }
#xb-om-hd-right { display: flex; align-items: center; gap: 10px; }
#xb-om-total { font-size: 12px; color: #71717a; }
#xb-om-refresh {
  padding: 5px 12px; border: none; border-radius: 7px; background: #27272a;
  color: #d4d4d8; font-size: 12px; cursor: pointer; font-family: inherit;
}
#xb-om-refresh:hover { background: #3f3f46; }
#xb-om-close2 {
  width: 28px; height: 28px; border: none; background: #27272a; border-radius: 7px;
  color: #a1a1aa; cursor: pointer; display: flex; align-items: center; justify-content: center;
}
#xb-om-close2:hover { background: #3f3f46; color: #fafafa; }

#xb-om-body { overflow-y: auto; flex: 1; min-height: 0; }

#xb-om-table { width: 100%; border-collapse: collapse; }
#xb-om-table th {
  position: sticky; top: 0; background: #18181b; z-index: 1;
  text-align: left; padding: 8px 22px; font-size: 11px; font-weight: 600;
  color: #52525b; text-transform: uppercase; letter-spacing: .06em;
  border-bottom: 1px solid #27272a;
}
#xb-om-table td { padding: 10px 22px; font-size: 13px; border-bottom: 1px solid #1c1c1f; }
#xb-om-table tr:last-child td { border-bottom: none; }
#xb-om-table tr:hover td { background: #1c1c1f; }

.xb-om-email { color: #e4e4e7; font-weight: 500; }
.xb-om-uid   { color: #52525b; font-size: 11px; margin-top: 2px; }
.xb-om-bar-wrap { display: flex; align-items: center; gap: 8px; }
.xb-om-bar-bg { flex: 1; height: 4px; background: #27272a; border-radius: 2px; }
.xb-om-bar-fill { height: 4px; border-radius: 2px; background: #6366f1; transition: width .3s; }
.xb-om-bar-fill.warn { background: #f59e0b; }
.xb-om-bar-fill.full { background: #ef4444; }
.xb-om-cnt { font-size: 12px; color: #a1a1aa; white-space: nowrap; }
.xb-om-exp  { font-size: 12px; color: #52525b; }
.xb-om-exp.expired { color: #ef4444; }

#xb-om-empty { padding: 48px 22px; text-align: center; color: #52525b; font-size: 13px; }
#xb-om-loading { padding: 48px 22px; text-align: center; }

#xb-om-ft {
  padding: 10px 22px; border-top: 1px solid #27272a; flex-shrink: 0;
  font-size: 11px; color: #3f3f46; display: flex; justify-content: space-between;
}

::-webkit-scrollbar { width: 4px; }
::-webkit-scrollbar-thumb { background: #3f3f46; border-radius: 2px; }
</style>

<button id="xb-om-fab" title="在线用户监控">
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="10"/><path d="M12 8v4l3 3"/>
  </svg>
  在线用户
  <span id="xb-om-badge">-</span>
</button>

<div id="xb-om-panel" role="dialog" aria-modal="true">
  <div id="xb-om-hd">
    <h2>在线用户监控</h2>
    <div id="xb-om-hd-right">
      <span id="xb-om-total"></span>
      <button id="xb-om-refresh">刷新</button>
      <button id="xb-om-close2">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>
  </div>

  <div id="xb-om-body">
    <div id="xb-om-loading">
      <div class="xb-spin" style="margin:0 auto;width:20px;height:20px;border-width:2px"></div>
    </div>
    <table id="xb-om-table" style="display:none">
      <thead>
        <tr>
          <th>用户</th>
          <th>在线设备</th>
          <th>到期时间</th>
        </tr>
      </thead>
      <tbody id="xb-om-tbody"></tbody>
    </table>
    <div id="xb-om-empty" style="display:none">当前没有在线用户</div>
  </div>

  <div id="xb-om-ft">
    <span id="xb-om-updated"></span>
    <span>设备数据每 5 分钟更新一次</span>
  </div>
</div>

<script>
(function () {
  'use strict';

  function getAuthHeader() {
    return sessionStorage.getItem('Xboard_access_token')
        || localStorage.getItem('Xboard_access_token')
        || '';
  }

  function apiBase() {
    const s = window.settings || {};
    return (s.base_url || '/').replace(/\/$/, '') + '/api/v2/' + (s.secure_path || '');
  }

  async function apiGet(path) {
    const r = await fetch(apiBase() + path, {
      headers: { 'Authorization': getAuthHeader() },
    });
    const json = await r.json();
    if (!r.ok) throw new Error(json.message || 'HTTP ' + r.status);
    return json;
  }

  function esc(s) {
    return String(s ?? '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }

  const fab      = document.getElementById('xb-om-fab');
  const panel    = document.getElementById('xb-om-panel');
  const closeBtn = document.getElementById('xb-om-close2');
  const refreshBtn = document.getElementById('xb-om-refresh');
  const badge    = document.getElementById('xb-om-badge');
  const loading  = document.getElementById('xb-om-loading');
  const table    = document.getElementById('xb-om-table');
  const tbody    = document.getElementById('xb-om-tbody');
  const empty    = document.getElementById('xb-om-empty');
  const totalEl  = document.getElementById('xb-om-total');
  const updatedEl= document.getElementById('xb-om-updated');

  let isOpen = false;
  let pollTimer = null;

  // ── Draggable FAB ───────────────────────────────────────────────
  (function initDrag() {
    let dragging = false, moved = false;
    let startX, startY, startRight, startBottom;

    const savedPos = (() => { try { return JSON.parse(localStorage.getItem('xb-om-fab-pos') || 'null'); } catch { return null; } })();
    if (savedPos) {
      fab.style.right  = Math.max(8, savedPos.right)  + 'px';
      fab.style.bottom = Math.max(8, savedPos.bottom) + 'px';
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
      localStorage.setItem('xb-om-fab-pos', JSON.stringify({ right: r, bottom: b }));
    });

    document.addEventListener('mouseup', () => {
      dragging = false;
      fab.style.transition = '';
    });

    fab.addEventListener('click', e => {
      if (moved) { moved = false; return; }
      togglePanel();
    });
  }());

  function togglePanel() {
    isOpen = !isOpen;
    panel.classList.toggle('open', isOpen);
    if (isOpen) {
      loadData();
      pollTimer = setInterval(loadData, 30000);
    } else {
      clearInterval(pollTimer);
    }
  }

  closeBtn.addEventListener('click', () => { isOpen = false; panel.classList.remove('open'); clearInterval(pollTimer); });
  refreshBtn.addEventListener('click', loadData);
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && isOpen) { isOpen = false; panel.classList.remove('open'); clearInterval(pollTimer); }
  });

  async function loadData() {
    refreshBtn.disabled = true;
    try {
      const res = await apiGet('/user/getOnlineUsers');
      const { total = 0, users = [] } = res.data || {};

      badge.textContent = total;
      totalEl.textContent = `共 ${total} 人在线`;
      updatedEl.textContent = '更新于 ' + new Date().toLocaleTimeString();

      loading.style.display = 'none';

      if (!users.length) {
        table.style.display = 'none';
        empty.style.display = 'block';
        return;
      }

      empty.style.display = 'none';
      table.style.display = '';

      tbody.innerHTML = users.map(u => {
        const limit     = u.device_limit || 0;
        const pct       = limit > 0 ? Math.min(100, Math.round(u.device_count / limit * 100)) : 0;
        const barClass  = pct >= 100 ? 'full' : pct >= 80 ? 'warn' : '';
        const barStyle  = limit > 0 ? `width:${pct}%` : 'width:100%;background:#3f3f46';
        const limitText = limit > 0 ? `/ ${limit}` : '无限制';

        const expTs  = u.expired_at;
        const expStr = expTs ? new Date(expTs * 1000).toLocaleDateString() : '长期有效';
        const expCls = (expTs && expTs < Date.now() / 1000) ? 'expired' : '';

        return `<tr>
          <td>
            <div class="xb-om-email">${esc(u.email)}</div>
            <div class="xb-om-uid">UID ${u.user_id}</div>
          </td>
          <td>
            <div class="xb-om-bar-wrap">
              <div class="xb-om-bar-bg"><div class="xb-om-bar-fill ${barClass}" style="${barStyle}"></div></div>
              <span class="xb-om-cnt">${u.device_count} ${limitText}</span>
            </div>
          </td>
          <td><span class="xb-om-exp ${expCls}">${expStr}</span></td>
        </tr>`;
      }).join('');
    } catch (e) {
      badge.textContent = '!';
      totalEl.textContent = '加载失败: ' + e.message;
    } finally {
      refreshBtn.disabled = false;
    }
  }

  // Poll badge count in background every 60s
  setInterval(async () => {
    try {
      const res = await apiGet('/user/getOnlineUsers');
      badge.textContent = res.data?.total ?? 0;
    } catch {}
  }, 60000);

  // Initial badge load
  apiGet('/user/getOnlineUsers').then(res => { badge.textContent = res.data?.total ?? 0; }).catch(() => {});
}());
</script>
