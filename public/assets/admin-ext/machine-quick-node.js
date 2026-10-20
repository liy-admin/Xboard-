/*
 * 服务器详情页「一键创建节点」
 *
 * 后台前端是编译产物（xboard-admin-dist），无法直接改源码，这个脚本在
 * 服务器（机器）详情抽屉里「新增节点到此服务器」按钮旁边加一个按钮，
 * 调用 POST /api/v2/{secure_path}/server/machine/quickNode。
 *
 * 表单直接嵌在抽屉里：抽屉是模态对话框，会锁定焦点并屏蔽外部的鼠标事件，
 * 挂在外面的弹窗无法输入。
 */
(function () {
  'use strict';

  // 官方按钮在各语言下的文案（locales/*.js 中的 add_node_to_server）
  var ANCHOR_LABELS = ['新增节点到此服务器', 'Add node to this server', 'Добавить узел на этот сервер'];
  var MARK = 'data-quick-node';

  var zh = /^zh/i.test(localStorage.getItem('i18nextLng') || navigator.language || '');
  var T = zh ? {
    button: '一键创建节点',
    title: '一键创建节点（VLESS + Reality）',
    hint: '面板会生成 Reality 密钥并把节点绑定到这台服务器，节点端会自动拉取并启动。创建后可在节点管理里修改。',
    groups: '权限组',
    noGroups: '还没有权限组，请先在「权限组管理」中创建。',
    host: '节点地址',
    hostHint: '默认是这台服务器最近连接面板时的 IP；在 NAT 后面或想用域名时请修改。',
    port: '端口',
    name: '节点名称（可选）',
    sni: 'Reality 目标站点',
    show: '创建后对用户显示',
    submit: '创建',
    cancel: '取消',
    creating: '创建中…',
    needGroup: '请至少选择一个权限组',
    needHost: '请填写节点地址',
    done: function (id, host, port) {
      return '已创建节点 #' + id + '（' + host + ':' + port + '）。节点端会自动启动：开启 WebSocket 时几秒内，否则最长约 1 分钟。';
    },
    loadFailed: '加载失败：'
  } : {
    button: 'One-click node',
    title: 'Create a node (VLESS + Reality)',
    hint: 'The panel generates Reality keys and binds the node to this server; the server picks it up and starts it. You can edit it later in node management.',
    groups: 'Groups',
    noGroups: 'No groups yet. Create one in group management first.',
    host: 'Node address',
    hostHint: 'Defaults to the IP this server last contacted the panel from. Change it if the server is behind NAT or you want a domain.',
    port: 'Port',
    name: 'Node name (optional)',
    sni: 'Reality target site',
    show: 'Visible to users',
    submit: 'Create',
    cancel: 'Cancel',
    creating: 'Creating…',
    needGroup: 'Select at least one group',
    needHost: 'Enter the node address',
    done: function (id, host, port) {
      return 'Created node #' + id + ' (' + host + ':' + port + '). The server starts it automatically: within seconds with WebSocket, otherwise within about a minute.';
    },
    loadFailed: 'Failed to load: '
  };

  function accessToken() {
    try {
      return JSON.parse(localStorage.getItem('XBOARD_ACCESS_TOKEN')).value || '';
    } catch (e) {
      return '';
    }
  }

  function api(path, body) {
    var opts = {
      method: body ? 'POST' : 'GET',
      headers: { Authorization: accessToken(), Accept: 'application/json' }
    };
    if (body) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    return fetch('/api/v2/' + window.settings.secure_path + path, opts).then(function (res) {
      return res.json().catch(function () { return {}; }).then(function (json) {
        if (!res.ok || json.status === 'fail') {
          throw new Error(json.message || ('HTTP ' + res.status));
        }
        return json.data;
      });
    });
  }

  function currentMachineId() {
    var m = location.hash.match(/[?&]machine_id=(\d+)/);
    return m ? parseInt(m[1], 10) : null;
  }

  function el(tag, style, text) {
    var node = document.createElement(tag);
    if (style) node.style.cssText = style;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  var INPUT_STYLE = 'width:100%;box-sizing:border-box;height:32px;padding:0 8px;border-radius:6px;' +
    'border:1px solid hsl(var(--input, 214 32% 91%));background:transparent;color:inherit;font-size:13px;';
  var LABEL_STYLE = 'display:block;font-size:12px;font-weight:500;margin:10px 0 4px;';
  var HINT_STYLE = 'font-size:12px;color:hsl(var(--muted-foreground, 215 16% 47%));margin-top:4px;';

  function field(label, input, hint) {
    var wrap = el('div');
    wrap.appendChild(el('label', LABEL_STYLE, label));
    wrap.appendChild(input);
    if (hint) wrap.appendChild(el('div', HINT_STYLE, hint));
    return wrap;
  }

  function textInput(value, placeholder, type) {
    var input = el('input', INPUT_STYLE);
    input.type = type || 'text';
    if (value !== undefined && value !== null) input.value = value;
    if (placeholder) input.placeholder = placeholder;
    return input;
  }

  function buildPanel(machineId, anchor) {
    // 所在的头部容器是横向 flex，占满整行以免被挤到半宽
    var panel = el('div', 'flex:1 1 100%;width:100%;box-sizing:border-box;margin-top:12px;padding:14px;border-radius:8px;' +
      'border:1px solid hsl(var(--border, 214 32% 91%));text-align:left;');
    panel.setAttribute(MARK, 'panel');
    panel.appendChild(el('div', 'font-weight:600;font-size:14px;', T.title));
    panel.appendChild(el('div', HINT_STYLE, T.hint));

    var body = el('div');
    var status = el('div', 'font-size:13px;margin-top:10px;');
    panel.appendChild(body);
    panel.appendChild(status);

    Promise.all([api('/server/group/fetch'), api('/server/machine/fetch')]).then(function (res) {
      var groups = res[0] || [];
      var machine = (res[1] || []).filter(function (m) { return m.id === machineId; })[0] || {};

      var groupBox = el('div', 'display:flex;flex-wrap:wrap;gap:6px 14px;');
      var groupInputs = [];
      if (!groups.length) {
        groupBox.appendChild(el('div', HINT_STYLE, T.noGroups));
      }
      groups.forEach(function (g) {
        var label = el('label', 'display:inline-flex;align-items:center;gap:6px;font-size:13px;cursor:pointer;');
        var cb = el('input');
        cb.type = 'checkbox';
        cb.value = String(g.id);
        if (groups.length === 1) cb.checked = true;
        groupInputs.push(cb);
        label.appendChild(cb);
        label.appendChild(document.createTextNode(g.name + ' (#' + g.id + ')'));
        groupBox.appendChild(label);
      });

      var host = textInput(machine.last_ip || '', '203.0.113.10 / node.example.com');
      var port = textInput('443', '443', 'number');
      var name = textInput('', machine.name || '');
      var sni = textInput('www.microsoft.com', 'www.microsoft.com');
      var show = el('input');
      show.type = 'checkbox';
      show.checked = true;
      var showLabel = el('label', 'display:inline-flex;align-items:center;gap:6px;font-size:13px;margin-top:10px;cursor:pointer;');
      showLabel.appendChild(show);
      showLabel.appendChild(document.createTextNode(T.show));

      body.appendChild(field(T.groups, groupBox));
      body.appendChild(field(T.host, host, T.hostHint));
      var row = el('div', 'display:grid;grid-template-columns:1fr 1fr;gap:10px;');
      row.appendChild(field(T.port, port));
      row.appendChild(field(T.name, name));
      body.appendChild(row);
      body.appendChild(field(T.sni, sni));
      body.appendChild(showLabel);

      var actions = el('div', 'display:flex;gap:8px;margin-top:14px;');
      var submit = el('button', null, T.submit);
      submit.type = 'button';
      submit.className = anchor.className;
      var cancel = el('button', 'padding:0 12px;height:32px;border-radius:6px;font-size:13px;cursor:pointer;' +
        'border:1px solid hsl(var(--border, 214 32% 91%));background:transparent;color:inherit;', T.cancel);
      cancel.type = 'button';
      actions.appendChild(submit);
      actions.appendChild(cancel);
      body.appendChild(actions);

      cancel.addEventListener('click', function () { panel.remove(); });
      submit.addEventListener('click', function () {
        var groupIds = groupInputs.filter(function (cb) { return cb.checked; })
          .map(function (cb) { return parseInt(cb.value, 10); });
        if (!groupIds.length) { status.textContent = T.needGroup; return; }
        if (!host.value.trim()) { status.textContent = T.needHost; return; }

        submit.disabled = true;
        submit.textContent = T.creating;
        status.textContent = '';
        api('/server/machine/quickNode', {
          machine_id: machineId,
          group_ids: groupIds,
          host: host.value.trim(),
          port: parseInt(port.value, 10) || 443,
          name: name.value.trim() || null,
          server_name: sni.value.trim() || null,
          show: show.checked
        }).then(function (node) {
          body.remove();
          status.style.color = 'hsl(142 71% 35%)';
          status.textContent = T.done(node.id, node.host, node.port);
          // 重新加载以刷新抽屉里的节点列表（地址中带 machine_id，会自动重新打开抽屉）
          setTimeout(function () { location.reload(); }, 2500);
        }).catch(function (err) {
          submit.disabled = false;
          submit.textContent = T.submit;
          status.style.color = 'hsl(0 72% 51%)';
          status.textContent = err.message;
        });
      });
    }).catch(function (err) {
      status.style.color = 'hsl(0 72% 51%)';
      status.textContent = T.loadFailed + err.message;
    });

    return panel;
  }

  function inject() {
    var machineId = currentMachineId();
    if (!machineId || !window.settings || !window.settings.secure_path) return;

    var dialogs = document.querySelectorAll('[role="dialog"]');
    for (var i = 0; i < dialogs.length; i++) {
      var dialog = dialogs[i];
      if (dialog.querySelector('[' + MARK + ']')) continue;

      var anchor = null;
      var buttons = dialog.querySelectorAll('button');
      for (var j = 0; j < buttons.length; j++) {
        if (ANCHOR_LABELS.indexOf(buttons[j].textContent.trim()) !== -1) { anchor = buttons[j]; break; }
      }
      if (!anchor) continue;

      var btn = el('button', null, T.button);
      btn.type = 'button';
      btn.className = anchor.className;
      btn.setAttribute(MARK, 'button');
      btn.addEventListener('click', function (e) {
        e.preventDefault();
        e.stopPropagation();
        var container = this.parentElement;
        var existing = container.parentElement.querySelector('[' + MARK + '="panel"]');
        if (existing) { existing.remove(); return; }
        container.insertAdjacentElement('afterend', buildPanel(currentMachineId(), this));
      });
      anchor.insertAdjacentElement('afterend', btn);
    }
  }

  // 用 setTimeout 节流而不是 requestAnimationFrame：后台标签页里 rAF 不会执行
  var scheduled = false;
  new MutationObserver(function () {
    if (scheduled) return;
    scheduled = true;
    setTimeout(function () { scheduled = false; inject(); }, 50);
  }).observe(document.documentElement, { childList: true, subtree: true });
  inject();
})();
