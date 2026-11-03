#!/bin/bash
# 业务流程验证：客户端信号与重拉、支付回调、邮件发送（本地 SMTP 接收端）。
# 依赖 init.sh + migrate.sh 已完成；只写 xboard_verify 与 Redis DB 15。
D="$HOME/xboard-verify"
S="$(dirname "$(readlink -f "$0")")"
PW="$(cat "$D/pw")"
PORT=17021
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
export XBOARD_SCHEDULER=0 XBOARD_PUBLIC_DIR="$D/public"

cp "$S/xboard-linux" "$D/xboard" && chmod +x "$D/xboard"
cd "$D"
redis-cli -n 15 flushdb > /dev/null
sql "DELETE FROM v2_order; DELETE FROM v2_payment; DELETE FROM v2_plan; DELETE FROM v2_ticket_message; DELETE FROM v2_ticket;
     DELETE FROM v2_mail_log; DELETE FROM v2_server; DELETE FROM v2_user; DELETE FROM v2_server_group; DELETE FROM v2_settings;"
rm -rf "$D/mail" "$D/flows.log"

# 设置必须在服务启动前写入（服务进程会缓存设置）
sql "INSERT INTO v2_server_group (name, created_at, updated_at) VALUES ('g1', UNIX_TIMESTAMP(), UNIX_TIMESTAMP());
INSERT INTO v2_settings (name,value,created_at,updated_at) VALUES
 ('node_probe_mode','tcp',NOW(),NOW()), ('server_token','nodetoken123',NOW(),NOW()),
 ('client_ws_enable','1',NOW(),NOW()), ('server_auto_manage_visibility','1',NOW(),NOW()),
 ('email_host','127.0.0.1',NOW(),NOW()), ('email_port','2525',NOW(),NOW()),
 ('email_from_address','noreply@example.com',NOW(),NOW()), ('app_name','FlowApp',NOW(),NOW()),
 ('app_url','http://127.0.0.1:$PORT',NOW(),NOW());"

python3 "$S/smtp_sink.py" 2525 "$D/mail" > /dev/null 2>&1 &
P1=$!
python3 -m http.server 9001 --bind 127.0.0.1 > /dev/null 2>&1 &
P2=$!
"$D/xboard" serve > "$D/flows.log" 2>&1 &
GP=$!
for i in $(seq 1 50); do curl -s -o /dev/null "$BASE/api/v1/guest/comm/config" && break; sleep 0.2; done
if kill -0 $GP 2>/dev/null; then ok "server started"; else bad "server started" "$(tail -5 "$D/flows.log")"; fi

# ---- 用户与节点 ----
curl -s -X POST "$BASE/api/v1/passport/auth/register" -d "email=flow@example.com&password=Passw0rd-flow" > /dev/null
R=$(curl -s -X POST "$BASE/api/v1/passport/auth/login" -d "email=flow@example.com&password=Passw0rd-flow")
AUTH=$(echo "$R" | jget data.auth_data)
sql "UPDATE v2_user SET group_id=1, transfer_enable=107374182400, expired_at=NULL, is_admin=1 WHERE email='flow@example.com';
INSERT INTO v2_server (type,name,host,port,server_port,rate,\`show\`,group_ids,protocol_settings,created_at,updated_at) VALUES
 ('shadowsocks','香港 流程','127.0.0.1','9001',9001,1,1,'[1]','{\"cipher\":\"aes-128-gcm\"}',NOW(),NOW()),
 ('shadowsocks','Singapore-F','127.0.0.1','9','9',1,1,'[1]','{\"cipher\":\"aes-128-gcm\"}',NOW(),NOW());"
TOKEN=$(sql "SELECT token FROM v2_user WHERE email='flow@example.com'")
UID_=$(sql "SELECT id FROM v2_user WHERE email='flow@example.com'")
NID=$(sql "SELECT id FROM v2_server WHERE name='香港 流程'")
SP=$(php -r 'echo hash("crc32b", "base64:dmVyaWZ5LWtleS1mb3ItZ28tcG9ydA==");')
A="$BASE/api/v2/$SP"

# =============== 1. 信号给客户端 / 客户端重拉节点 ===============
R=$(curl -s -H "Authorization: $AUTH" "$BASE/api/v1/user/server/fetch")
N0=$(echo "$R" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(len(d["data"]))')
check "client sees 2 nodes before changes" "$N0" "2"
REV0=$(curl -s -H "Authorization: $AUTH" "$BASE/api/v1/user/server/revision" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print(d.get("revision") if isinstance(d,dict) else d)')
ETAG=$(curl -s -D - -o /dev/null -H "Authorization: $AUTH" "$BASE/api/v1/user/server/fetch" | grep -i '^etag:' | tr -d '\r' | awk '{print $2}')
if [ -n "$ETAG" ]; then ok "server/fetch returns an ETag"; else bad "server/fetch returns an ETag" "none"; fi
check "ETag revalidation returns 304" "$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: $AUTH" -H "If-None-Match: $ETAG" "$BASE/api/v1/user/server/fetch")" "304"

python3 "$S/ws_client.py" $PORT "client_token=$TOKEN" 40 > "$D/ws_client.out" 2>&1 &
WC=$!
sleep 2
has "client websocket authenticated" "$(cat "$D/ws_client.out")" "auth.success"

# 节点上线（NodeWorker 路径）→ node.online
python3 "$S/ws_client.py" $PORT "node_id=$NID&token=nodetoken123" 6 > "$D/ws_node.out" 2>&1 &
sleep 4
has "node websocket authenticated" "$(cat "$D/ws_node.out")" "auth.success"
sleep 1
has "client got subscription.updated (node.online)" "$(cat "$D/ws_client.out")" "node.online"
sleep 5   # 节点连接结束 → node.offline（3 秒节流之外）
has "client got subscription.updated (node.offline)" "$(cat "$D/ws_client.out")" "node.offline"

# 自动显隐变更（CLI 进程 → Redis client:push → 服务进程 → 客户端）
"$D/xboard" check:server-latency >> "$D/flows.log" 2>&1
sleep 2
has "client got subscription.updated (visibility)" "$(cat "$D/ws_client.out")" "visibility"
R=$(curl -s -H "Authorization: $AUTH" "$BASE/api/v1/user/server/fetch")
N1=$(echo "$R" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(len(d["data"]))')
check "client re-pull returns only the reachable node after the signal" "$N1" "1"
REV1=$(curl -s -H "Authorization: $AUTH" "$BASE/api/v1/user/server/revision" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; print(d.get("revision") if isinstance(d,dict) else d)')
if [ "$REV0" != "$REV1" ] && [ -n "$REV1" ]; then ok "revision changed after the node set changed"; else bad "revision changed" "rev0=$REV0 rev1=$REV1"; fi
check "old ETag no longer matches (200)" "$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: $AUTH" -H "If-None-Match: $ETAG" "$BASE/api/v1/user/server/fetch")" "200"
kill $WC 2>/dev/null

# =============== 2. 支付回调 ===============
sql "INSERT INTO v2_plan (name, group_id, transfer_enable, prices, sell, \`show\`, renew, sort, created_at, updated_at)
     VALUES ('FlowPlan', 1, 200, '{\"monthly\":1000}', 1, 1, 1, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());
INSERT INTO v2_payment (uuid, payment, name, config, enable, created_at, updated_at)
     VALUES ('flowpay01', 'EPay', 'EPay-flow', '{\"url\":\"http://127.0.0.1:9\",\"pid\":\"1001\",\"key\":\"secretkey\",\"type\":\"alipay\"}', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());"
PLAN=$(sql "SELECT id FROM v2_plan WHERE name='FlowPlan'")
sql "INSERT INTO v2_order (user_id, plan_id, type, period, trade_no, total_amount, status, created_at, updated_at)
     VALUES ($UID_, $PLAN, 1, 'monthly', 'FLOWTRADE001', 1000, 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());"
N="$BASE/api/v1/guest/payment/notify/EPay/flowpay01"
BADQ=$(python3 "$S/epay_sign.py" wrongkey out_trade_no=FLOWTRADE001 trade_no=GW123 money=10.00)
R=$(curl -s -o /dev/null -w "%{http_code}" "$N?$BADQ")
if [ "$R" != "200" ]; then ok "callback with a wrong signature is rejected ($R)"; else bad "wrong signature rejected" "got 200"; fi
check "order stays pending after a bad callback" "$(sql "SELECT status FROM v2_order WHERE trade_no='FLOWTRADE001'")" "0"
GOODQ=$(python3 "$S/epay_sign.py" secretkey out_trade_no=FLOWTRADE001 trade_no=GW123 money=10.00)
R=$(curl -s "$N?$GOODQ")
check "valid callback answers success" "$R" "success"
sleep 1
ST=$(sql "SELECT status FROM v2_order WHERE trade_no='FLOWTRADE001'")
if [ "$ST" = "1" ] || [ "$ST" = "3" ]; then ok "order advanced after callback (status $ST)"; else bad "order advanced after callback" "status=$ST"; fi
"$D/xboard" check:order >> "$D/flows.log" 2>&1
check "order is completed after check:order" "$(sql "SELECT status FROM v2_order WHERE trade_no='FLOWTRADE001'")" "3"
check "callback_no recorded" "$(sql "SELECT callback_no FROM v2_order WHERE trade_no='FLOWTRADE001'")" "GW123"
check "user got the plan" "$(sql "SELECT plan_id FROM v2_user WHERE id=$UID_")" "$PLAN"
check "user traffic quota applied (200 GB)" "$(sql "SELECT transfer_enable FROM v2_user WHERE id=$UID_")" "214748364800"
R=$(curl -s "$N?$GOODQ")
check "replayed callback is idempotent (success)" "$R" "success"
check "order still completed after replay" "$(sql "SELECT status FROM v2_order WHERE trade_no='FLOWTRADE001'")" "3"

# =============== 3. 邮件发送 ===============
count_mail() { ls "$D/mail" 2>/dev/null | wc -l; }
mailtext() { python3 -c '
import sys, email
from email.header import decode_header, make_header
m = email.message_from_bytes(open(sys.argv[1], "rb").read())
print("SUBJECT:", str(make_header(decode_header(m.get("Subject", "")))))
for part in m.walk():
    if part.get_content_maintype() == "text":
        print(part.get_payload(decode=True).decode("utf-8", "replace"))
' "$1"; }
M0=$(count_mail)
R=$(curl -s -X POST "$BASE/api/v1/passport/comm/sendEmailVerify" -d "email=newcomer@example.com")
has "sendEmailVerify accepted" "$R" "true"
for i in $(seq 1 30); do [ "$(count_mail)" -gt "$M0" ] && break; sleep 0.5; done
LASTMAIL=$(ls -t "$D/mail"/*.eml 2>/dev/null | head -1)
if [ -n "$LASTMAIL" ]; then ok "verification mail reached the SMTP server"; else bad "verification mail reached the SMTP server" "no mail"; fi
has "mail is addressed to the requester" "$(head -1 "$LASTMAIL" 2>/dev/null)" "newcomer@example.com"
CODE=$(redis-cli -n 15 get EMAIL_VERIFY_CODE_newcomer@example.com | tr -d '"')
if [ -n "$CODE" ]; then has "mail body contains the stored verify code" "$(mailtext "$LASTMAIL")" "$CODE"; else bad "verify code stored in redis" "empty"; fi
has "mail has the from address" "$(cat "$LASTMAIL" 2>/dev/null)" "noreply@example.com"
check "mail log row written" "$(sql "SELECT COUNT(*) FROM v2_mail_log WHERE email='newcomer@example.com' AND error IS NULL")" "1"
R=$(curl -s -X POST "$BASE/api/v1/passport/comm/sendEmailVerify" -d "email=newcomer@example.com")
has "second request inside 60s is throttled" "$R" '"status":"fail"'

# 管理员群发
M1=$(count_mail)
R=$(curl -s -X POST -H "Authorization: $AUTH" -H "Content-Type: application/json" \
  -d "{\"scope\":\"selected\",\"user_ids\":[$UID_],\"subject\":\"Hello FlowSubject\",\"content\":\"Body FlowBody\"}" "$A/user/sendMail")
has "admin sendMail accepted" "$R" '"status":"success"'
for i in $(seq 1 30); do [ "$(count_mail)" -gt "$M1" ] && break; sleep 0.5; done
LASTMAIL=$(ls -t "$D/mail"/*.eml | head -1)
has "admin mail delivered with subject" "$(cat "$LASTMAIL")" "Hello FlowSubject"
has "admin mail delivered to the selected user" "$(head -1 "$LASTMAIL")" "flow@example.com"

# 工单回复通知
sql "INSERT INTO v2_ticket (user_id, subject, level, status, reply_status, created_at, updated_at) VALUES ($UID_, 'Flow ticket', 1, 0, 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());"
TID=$(sql "SELECT id FROM v2_ticket WHERE subject='Flow ticket'")
sql "INSERT INTO v2_ticket_message (user_id, ticket_id, message, created_at, updated_at) VALUES ($UID_, $TID, 'help me', UNIX_TIMESTAMP(), UNIX_TIMESTAMP());"
M2=$(count_mail)
R=$(curl -s -X POST -H "Authorization: $AUTH" -H "Content-Type: application/json" -d "{\"id\":$TID,\"message\":\"We fixed it FlowReply\"}" "$A/ticket/reply")
has "admin ticket reply accepted" "$R" '"status":"success"'
for i in $(seq 1 30); do [ "$(count_mail)" -gt "$M2" ] && break; sleep 0.5; done
if [ "$(count_mail)" -gt "$M2" ]; then ok "ticket reply notification mail delivered"; else bad "ticket reply notification mail delivered" "no new mail"; fi
check "ticket marked replied" "$(sql "SELECT reply_status FROM v2_ticket WHERE id=$TID")" "0"

# 失败路径：SMTP 不可达 → 重试后记入失败列表，服务不崩
sql "UPDATE v2_settings SET value='2599' WHERE name='email_port';"
redis-cli -n 15 flushdb > /dev/null   # 清掉验证码限流键
kill $GP 2>/dev/null; wait $GP 2>/dev/null
for i in $(seq 1 20); do ss -ltn | grep -q ":$PORT " || break; sleep 0.5; done
"$D/xboard" serve >> "$D/flows.log" 2>&1 &
GP=$!
for i in $(seq 1 50); do curl -s -o /dev/null "$BASE/api/v1/guest/comm/config" && break; sleep 0.2; done
R=$(curl -s -X POST "$BASE/api/v1/passport/comm/sendEmailVerify" -d "email=failcase@example.com")
has "request still accepted when SMTP is down" "$R" "true"
for i in $(seq 1 40); do E=$(sql "SELECT COUNT(*) FROM v2_mail_log WHERE email='failcase@example.com' AND error IS NOT NULL"); [ "$E" -ge 1 ] && break; sleep 1; done
if [ "${E:-0}" -ge 1 ]; then ok "failed delivery is recorded in v2_mail_log"; else bad "failed delivery recorded" "none"; fi
if kill -0 $GP 2>/dev/null; then ok "server survives SMTP failure"; else bad "server survives SMTP failure" "died"; fi
grep -q "panic" "$D/flows.log" && bad "no panic in flows.log" "$(grep -m1 -A5 panic "$D/flows.log")" || ok "no panic in flows.log"

kill -TERM $GP 2>/dev/null
GONE=0
for i in $(seq 1 20); do kill -0 $GP 2>/dev/null || { GONE=1; break; }; sleep 0.5; done
if [ $GONE = 1 ]; then ok "server exits on SIGTERM (graceful shutdown)"; else bad "server exits on SIGTERM" "still running after 10s"; kill -9 $GP 2>/dev/null; fi
kill $P1 $P2 2>/dev/null
echo "================ PASS=$PASS FAIL=$FAIL"
echo "--- flows.log tail:"; tail -12 "$D/flows.log"
