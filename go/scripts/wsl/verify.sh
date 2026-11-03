#!/bin/bash
# Go 版端到端验证：私有 MariaDB(33306，PHP 迁移建表) + Redis DB 15 + 本地桩服务。
# 只写 xboard_verify 与 Redis DB 15，不触碰其他数据。
D="$HOME/xboard-verify"
S="$(dirname "$(readlink -f "$0")")"
PW="$(cat "$D/pw")"
PORT=7011
BASE="http://127.0.0.1:$PORT"
PASS=0; FAIL=0
ok()   { echo "PASS  $1"; PASS=$((PASS+1)); }
bad()  { echo "FAIL  $1  -> $2"; FAIL=$((FAIL+1)); }
check(){ if [ "$2" = "$3" ]; then ok "$1"; else bad "$1" "got [$2] want [$3]"; fi; }
has()  { if echo "$2" | grep -q -- "$3"; then ok "$1"; else bad "$1" "missing [$3] in: $(echo "$2" | head -c 300)"; fi; }
sql()  { mariadb --no-defaults --socket="$D/mysql.sock" -uroot xboard_verify -N -B -e "$1"; }
jget() { python3 -c 'import sys,json
d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
    d=d[int(k)] if isinstance(d,list) else d.get(k)
    if d is None: break
print("" if d is None else d)' "$1" 2>/dev/null; }

export DB_HOST=127.0.0.1 DB_PORT=33306 DB_DATABASE=xboard_verify DB_USERNAME=xbv DB_PASSWORD="$PW"
export REDIS_HOST=127.0.0.1 REDIS_PORT=6379 REDIS_DB=15 LISTEN="127.0.0.1:$PORT"
export APP_KEY="base64:dmVyaWZ5LWtleS1mb3ItZ28tcG9ydA=="
export XBOARD_SCHEDULER=1 XBOARD_PUBLIC_DIR="$D/public" XBOARD_ALLOW_PRIVATE_SUBSCRIPTION=1
mkdir -p "$D/public/uploads/payment" && echo "icon-bytes" > "$D/public/uploads/payment/t.txt"

cp "$S/xboard-linux" "$D/xboard" && chmod +x "$D/xboard"
cd "$D"

# ---- 干净起点（只清理验证库）----
redis-cli -n 15 flushdb > /dev/null
sql "DELETE FROM v2_server; DELETE FROM v2_user; DELETE FROM v2_server_group; DELETE FROM v2_settings; DELETE FROM v2_plugins;"
# 设置必须在服务启动前写入：服务进程会缓存设置。
sql "INSERT INTO v2_server_group (name, created_at, updated_at) VALUES ('g1', UNIX_TIMESTAMP(), UNIX_TIMESTAMP());
INSERT INTO v2_settings (name,value,created_at,updated_at) VALUES
 ('node_probe_mode','tcp',NOW(),NOW()), ('server_token','nodetoken123',NOW(),NOW());"

# ---- 桩服务：TCP 端口 9001、订阅文件 8099 ----
python3 -m http.server 9001 --bind 127.0.0.1 > /dev/null 2>&1 &
P1=$!
mkdir -p "$D/stub"
q() { python3 -c 'import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))' "$1"; }
printf 'ss://YWVzLTEyOC1nY206cGFzc3dvcmQ@10.0.0.1:8388#%s\nss://YWVzLTEyOC1nY206cGFzc3dvcmQ@10.0.0.2:8388#%s\nss://YWVzLTEyOC1nY206cGFzc3dvcmQ@10.0.0.3:8388#%s\n' \
  "$(q '日本 东京 01')" "$(q '剩余流量 100GB')" "$(q 'US-West')" | base64 -w0 > "$D/stub/sub.txt"
( cd "$D/stub" && python3 -m http.server 8099 --bind 127.0.0.1 > /dev/null 2>&1 ) &
P2=$!
cat > "$D/stub/fetch_sub.py" <<'PYEOF'
import json, os
out = os.environ["BALALA_OUTFILE"]
assert os.environ["BALALA_EMAIL"] == "stub@example.com"
json.dump({"success": True, "subscribe_url": "http://127.0.0.1:8099/sub.txt",
           "subscriptions": [{"client": "v2rayn", "url": "http://127.0.0.1:8099/sub.txt"}]},
          open(out, "w"))
PYEOF
export XBOARD_BALALA_SCRIPT="$D/stub/fetch_sub.py"

# ---- 启动 Go 服务 ----
"$D/xboard" serve > "$D/go.log" 2>&1 &
GP=$!
for i in $(seq 1 50); do curl -s -o /dev/null "$BASE/api/v1/guest/comm/config" && break; sleep 0.2; done
if kill -0 $GP 2>/dev/null; then ok "go server started"; else bad "go server started" "$(tail -5 "$D/go.log")"; fi

# ---- 注册 / 登录 / 用户信息 ----
R=$(curl -s -X POST "$BASE/api/v1/passport/auth/register" -d "email=verify@example.com&password=Passw0rd-verify")
AUTH=$(echo "$R" | jget data.auth_data)
if [ -n "$AUTH" ]; then ok "register returns auth_data"; else bad "register" "$R"; fi
R=$(curl -s -X POST "$BASE/api/v1/passport/auth/login" -d "email=verify@example.com&password=Passw0rd-verify")
AUTH=$(echo "$R" | jget data.auth_data)
if [ -n "$AUTH" ]; then ok "login returns auth_data"; else bad "login" "$R"; fi
R=$(curl -s -H "Authorization: $AUTH" "$BASE/api/v1/user/info")
check "user/info email" "$(echo "$R" | jget data.email)" "verify@example.com"
R=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: bad" "$BASE/api/v1/user/info")
check "user/info with bad token is rejected" "$R" "403"
R=$(curl -s -X POST "$BASE/api/v1/passport/auth/login" -d "email=verify@example.com&password=wrong-password")
has "login with wrong password fails" "$R" '"status":"fail"'

# ---- 种子数据：用户入组并放开流量、节点 ----
sql "UPDATE v2_user SET group_id=1, transfer_enable=107374182400, expired_at=NULL, is_admin=1 WHERE email='verify@example.com';
INSERT INTO v2_server (type,name,host,port,server_port,rate,\`show\`,group_ids,protocol_settings,created_at,updated_at)
VALUES ('shadowsocks','香港 测试','127.0.0.1','9001',9001,1,1,'[1]','{\"cipher\":\"aes-128-gcm\",\"obfs\":null,\"obfs_settings\":null}',NOW(),NOW());"
TOKEN=$(sql "SELECT token FROM v2_user WHERE email='verify@example.com'")

# ---- 订阅：明文已按设计下线，加密订阅可解密 ----
check "plain subscription is disabled (403)" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/client/subscribe?token=$TOKEN")" "403"
R=$(curl -s "$BASE/s-enc/$TOKEN")
has "encrypted subscription envelope" "$R" '"alg":"aes-256-gcm"'
ENC=$(sql "SELECT IFNULL(enc_key,'') FROM v2_user WHERE email='verify@example.com'")
echo "$R" > "$D/enc.json"
DEC=$(python3 "$S/dec.py" "$D/enc.json" "$ENC" 2>&1)
has "encrypted subscription decrypts and lists the node" "$DEC" "香港 测试"

# ---- 节点端接口（UniProxy，使用 server_token）----
NID=$(sql "SELECT id FROM v2_server ORDER BY id LIMIT 1")
R=$(curl -s "$BASE/api/v1/server/UniProxy/config?token=nodetoken123&node_id=$NID&node_type=shadowsocks")
has "UniProxy config returns server_port" "$R" "9001"
R=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/v1/server/UniProxy/config?token=wrong&node_id=$NID&node_type=shadowsocks")
if [ "$R" != "200" ]; then ok "UniProxy rejects wrong token ($R)"; else bad "UniProxy rejects wrong token" "got 200"; fi

# ---- 管理端（secure_path 分发）----
SP=$(php -r 'echo hash("crc32b", "base64:dmVyaWZ5LWtleS1mb3ItZ28tcG9ydA==");')
A="$BASE/api/v2/$SP"
R=$(curl -s -H "Authorization: $AUTH" "$A/server/manage/getNodes")
has "admin getNodes lists the seeded node" "$R" "香港 测试"
R=$(curl -s -o /dev/null -w "%{http_code}" "$A/server/manage/getNodes")
check "admin without token is rejected" "$R" "403"
R=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: $AUTH" "$BASE/api/v2/wrongpath/server/manage/getNodes")
check "wrong secure_path is 404" "$R" "404"

# 管理端 GET 端点冒烟：全部应返回 status=success（覆盖真实 PHP 表结构上的 SQL）
for ep in plan/fetch notice/fetch coupon/fetch user/fetch ticket/fetch order/fetch knowledge/fetch \
          server/group/fetch server/route/fetch server/machine/fetch config/fetch \
          system/getSystemStatus system/getQueueStats system/getQueueWorkload system/getAuditLog \
          plugin/getPlugins plugin/types theme/getThemes payment/fetch payment/getPaymentMethods \
          stat/getStats stat/getOverride "stat/getRanking?type=server_traffic_rank" "stat/getTrafficRank?type=node" \
          gift-card/templates gift-card/codes gift-card/statistics mail/template/list traffic-reset/logs traffic-reset/stats; do
  R=$(curl -s -w "\n%{http_code}" -H "Authorization: $AUTH" "$A/$ep")
  CODE=$(echo "$R" | tail -1); BODY=$(echo "$R" | sed '$d')
  if [ "$CODE" = "200" ] && echo "$BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); sys.exit(1 if isinstance(d,dict) and d.get("status")=="fail" else 0)' 2>/dev/null; then
    ok "admin GET $ep"; else bad "admin GET $ep" "HTTP $CODE $(echo "$BODY" | head -c 160)"; fi
done

# ---- 配置保存 / 读取往返 ----
R=$(curl -s -X POST -H "Authorization: $AUTH" -H "Content-Type: application/json" -d '{"app_name":"VerifyApp"}' "$A/config/save")
has "config/save ok" "$R" '"status":"success"'
check "config/save persisted to v2_settings" "$(sql "SELECT value FROM v2_settings WHERE name='app_name'")" "VerifyApp"

# ---- 手动测速（TCP）----
UA0=$(sql "SELECT updated_at FROM v2_server ORDER BY id LIMIT 1")
sleep 1
R=$(curl -s -X POST -H "Authorization: $AUTH" -H "Content-Type: application/json" -d '{}' "$A/server/manage/probeLatency")
has "probeLatency started" "$R" "started"
L=""
for i in $(seq 1 40); do
  L=$(sql "SELECT IFNULL(latency_ms,'') FROM v2_server ORDER BY id LIMIT 1")
  [ -n "$L" ] && break; sleep 0.5
done
if [ -n "$L" ]; then ok "tcp probe wrote latency_ms ($L ms)"; else bad "tcp probe wrote latency_ms" "empty"; fi
check "country_code parsed from node name" "$(sql "SELECT country_code FROM v2_server ORDER BY id LIMIT 1")" "HK"
check "probe does not bump updated_at" "$(sql "SELECT updated_at FROM v2_server ORDER BY id LIMIT 1")" "$UA0"
R=$(curl -s -H "Authorization: $AUTH" "$A/server/manage/probeStatus")
has "probeStatus has running=false" "$R" '"running":false'
R=$(curl -s -H "Authorization: $AUTH" "$A/server/manage/probeDiagnose")
has "probeDiagnose reports singbox state" "$R" '"singbox"'

# ---- sing-box 模式：本机无 sing-box → 「没法测」，保留上次结果 ----
sql "UPDATE v2_settings SET value='singbox' WHERE name='node_probe_mode'; UPDATE v2_server SET latency_ms=77, last_latency_at=UNIX_TIMESTAMP();"
"$D/xboard" check:server-latency > "$D/job.log" 2>&1
check "untestable (no sing-box) keeps previous latency" "$(sql 'SELECT latency_ms FROM v2_server ORDER BY id LIMIT 1')" "77"

# ---- 真实 sing-box 探测：本地起一个 shadowsocks 服务端，节点指向它 ----
if [ -x "$D/bin/sing-box" ]; then
  UUID=$(sql "SELECT uuid FROM v2_user WHERE email='verify@example.com'")
  cat > "$D/stub/ss-server.json" <<SSEOF
{"log":{"level":"warn"},
 "inbounds":[{"type":"shadowsocks","tag":"in","listen":"127.0.0.1","listen_port":9388,"method":"aes-128-gcm","password":"$UUID"}],
 "outbounds":[{"type":"direct","tag":"direct"}]}
SSEOF
  "$D/bin/sing-box" run -c "$D/stub/ss-server.json" > "$D/stub/ss-server.log" 2>&1 &
  P3=$!
  sleep 2
  NID=$(sql "SELECT id FROM v2_server WHERE name='香港 测试'")
  sql "UPDATE v2_server SET host='127.0.0.1', port='9388', server_port=9388, latency_ms=NULL, exit_ip=NULL, last_latency_at=NULL, \`show\`=1 WHERE id=$NID;
       INSERT INTO v2_settings (name,value,created_at,updated_at) VALUES ('node_probe_singbox_path','$D/bin/sing-box',NOW(),NOW());
       INSERT INTO v2_settings (name,value,created_at,updated_at) VALUES ('node_probe_test_count','2',NOW(),NOW());
       INSERT INTO v2_settings (name,value,created_at,updated_at) VALUES ('node_probe_ip_echo_url','https://ipwho.is/',NOW(),NOW());"
  "$D/xboard" check:server-latency >> "$D/job.log" 2>&1
  L=$(sql "SELECT IFNULL(latency_ms,'') FROM v2_server WHERE id=$NID")
  if [ -n "$L" ]; then ok "real sing-box probe measured latency (${L} ms)"; else bad "real sing-box probe measured latency" "$(tail -5 "$D/job.log")"; fi
  EX=$(sql "SELECT IFNULL(exit_ip,'') FROM v2_server WHERE id=$NID")
  if [ -n "$EX" ]; then ok "real sing-box probe recorded exit_ip ($EX)"; else bad "real sing-box probe recorded exit_ip" "empty"; fi
  # 密码错误 → 真连不通，应判不可达（latency 置空，非「没法测」）
  sql "UPDATE v2_server SET latency_ms=55, last_latency_at=UNIX_TIMESTAMP() WHERE id=$NID;"
  kill $P3 2>/dev/null; sleep 1
  "$D/xboard" check:server-latency >> "$D/job.log" 2>&1
  check "node down under real probe is marked unreachable (latency NULL)" "$(sql "SELECT IFNULL(latency_ms,'NULL') FROM v2_server WHERE id=$NID")" "NULL"
  sql "DELETE FROM v2_settings WHERE name IN ('node_probe_singbox_path','node_probe_test_count','node_probe_ip_echo_url');"
else
  echo "INFO  ~/xboard-verify/bin/sing-box missing, real probe skipped"
fi
sql "UPDATE v2_settings SET value='tcp' WHERE name='node_probe_mode'"

# ---- balala：抓取 + 导入 ----
sql "INSERT INTO v2_plugins (code,name,version,type,is_enabled,config,created_at,updated_at)
VALUES ('balala_sub','Balala','1.0.0','feature',1,
'{\"email\":\"stub@example.com\",\"password\":\"x\",\"python_bin\":\"python3\",\"auto_import_tag\":\"balala\",\"auto_import_group_ids\":\"1\",\"auto_import_show\":1,\"auto_import_rate\":\"1.5\",\"auto_import_client\":\"v2rayn\"}',NOW(),NOW());"
"$D/xboard" balala-sub:fetch >> "$D/job.log" 2>&1
R=$(redis-cli -n 15 get plugin:balala_sub:latest)
has "balala fetch cached subscribe_url" "$R" "127.0.0.1:8099/sub.txt"
"$D/xboard" balala-sub:import-force >> "$D/job.log" 2>&1
check "balala import created 2 real nodes (info node filtered)" "$(sql "SELECT COUNT(*) FROM v2_server WHERE JSON_CONTAINS(tags,'\"balala\"')")" "2"
check "balala imported rate applied" "$(sql "SELECT rate FROM v2_server WHERE JSON_CONTAINS(tags,'\"balala\"') ORDER BY id LIMIT 1")" "1.50"
check "balala imported show flag" "$(sql "SELECT \`show\` FROM v2_server WHERE JSON_CONTAINS(tags,'\"balala\"') ORDER BY id LIMIT 1")" "1"
R=$(redis-cli -n 15 get plugin:balala_sub:import_status)
has "balala import status ok" "$R" '"ok":true'
"$D/xboard" balala-sub:import-force >> "$D/job.log" 2>&1
check "second import replaces by tag (still 2)" "$(sql "SELECT COUNT(*) FROM v2_server WHERE JSON_CONTAINS(tags,'\"balala\"')")" "2"
R=$(curl -s -H "Authorization: $AUTH" "$BASE/api/v1/balala-sub/latest")
has "balala-sub/latest endpoint" "$R" "subscribe_url"
R=$(curl -s -H "Authorization: $AUTH" "$BASE/api/v1/balala-sub/doctor")
has "balala-sub/doctor endpoint" "$R" '"checks"'
# 自动重导信号：0 个可用节点 + auto_reimport_on_empty
sql "UPDATE v2_plugins SET config=JSON_SET(config,'$.auto_reimport_on_empty',true) WHERE code='balala_sub'; UPDATE v2_server SET host='127.0.0.1', port='9', server_port=9;"
redis-cli -n 15 del plugin:balala_sub:import_requested > /dev/null
"$D/xboard" check:server-latency >> "$D/job.log" 2>&1
R=$(redis-cli -n 15 exists plugin:balala_sub:import_requested)
check "no-available hook wrote the import signal" "$R" "1"

# ---- 国家码回退：不通 + 名字 ----
sql "INSERT INTO v2_server (type,name,host,port,server_port,rate,\`show\`,group_ids,protocol_settings,created_at,updated_at)
VALUES ('shadowsocks','Singapore-X','127.0.0.1','9','9',1,1,'[1]','{\"cipher\":\"aes-128-gcm\"}',NOW(),NOW());"
SGID=$(sql "SELECT id FROM v2_server WHERE name='Singapore-X'")
"$D/xboard" check:server-latency >> "$D/job.log" 2>&1
check "unreachable node gets country from its name" "$(sql "SELECT IFNULL(country_code,'NULL') FROM v2_server WHERE id=$SGID")" "SG"
check "unreachable node latency is NULL" "$(sql "SELECT IFNULL(latency_ms,'NULL') FROM v2_server WHERE id=$SGID")" "NULL"

# ---- 自动显隐：开启后不可达节点被隐藏，但「一个都不显示」时保持现状 ----
SHOWN0=$(sql "SELECT COUNT(*) FROM v2_server WHERE \`show\`=1")
sql "INSERT INTO v2_settings (name,value,created_at,updated_at) VALUES ('server_auto_manage_visibility','1',NOW(),NOW());"
"$D/xboard" check:server-latency >> "$D/job.log" 2>&1
check "auto visibility safety net keeps nodes when none reachable" "$(sql "SELECT COUNT(*) FROM v2_server WHERE \`show\`=1")" "$SHOWN0"

# ---- 上传目录静态服务、WebSocket 握手、调度器心跳 ----
check "/uploads static file served" "$(curl -s "$BASE/uploads/payment/t.txt")" "icon-bytes"
R=$(python3 "$S/ws_check.py" "$PORT" 2>&1)
has "websocket upgrade handshake" "$R" "101"
has "websocket wrong node token gets an error frame" "$R" "error"
sleep 12   # 越过 10 秒鉴权超时：之前这里会 panic 整个进程
if kill -0 $GP 2>/dev/null; then ok "server survives the ws auth timeout"; else bad "server survives the ws auth timeout" "process died"; fi
HB=""
for i in $(seq 1 70); do HB=$(redis-cli -n 15 get SCHEDULE_LAST_CHECK_AT); [ -n "$HB" ] && break; sleep 1; done
if [ -n "$HB" ]; then ok "scheduler heartbeat written"; else bad "scheduler heartbeat" "empty after 70s"; fi
R=$(redis-cli -n 15 get ws_server:heartbeat)
if [ -n "$R" ]; then ok "websocket heartbeat written"; else bad "websocket heartbeat" "empty"; fi
grep -q "panic" "$D/go.log" && bad "no panic in go.log" "$(grep -m1 -A5 panic "$D/go.log")" || ok "no panic in go.log"

# ---- 收尾 ----
kill $GP $P1 $P2 2>/dev/null
wait 2>/dev/null
echo "================ PASS=$PASS FAIL=$FAIL"
echo "--- go.log tail:"; tail -12 "$D/go.log"
echo "--- job.log tail:"; tail -12 "$D/job.log"
