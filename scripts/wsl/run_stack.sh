#!/bin/bash
# 启动完整本地栈：私有 MariaDB（已建表）+ Go 服务（直接使用仓库的 public/ 与 theme/）。
# 监听 127.0.0.1:17031；管理端地址 /admin；种子管理员见输出。
D="$HOME/xboard-verify"
S="$(dirname "$(readlink -f "$0")")"
PORT=17031
PW="$(cat "$D/pw")"
BASE="http://127.0.0.1:$PORT"
ADMIN_EMAIL="admin@letsvpn.local"
ADMIN_PASS="LetsVpn-Admin-1"
sql() { mariadb --no-defaults --socket="$D/mysql.sock" -uroot xboard_verify -N -B -e "$1"; }

bash "$S/init.sh" | tail -1

# 停掉旧的服务进程
pgrep -f "[x]board-verify/xboard" | xargs -r kill 2>/dev/null
sleep 1
pgrep -f "[x]board-verify/xboard" | xargs -r kill -9 2>/dev/null
sleep 1

cp "$S/xboard-linux" "$D/xboard" && chmod +x "$D/xboard"

# 干净的业务数据（只动验证库）
redis-cli -n 15 flushdb > /dev/null
for t in v2_order v2_plan v2_ticket_message v2_ticket v2_mail_log v2_server v2_user v2_server_group v2_settings v2_plugins          v2_coupon v2_notice v2_knowledge v2_payment v2_server_route v2_server_machine v2_invite_code v2_commission_log          v2_gift_card_code v2_gift_card_template v2_gift_card_usage v2_stat v2_stat_user v2_stat_server v2_log          v2_admin_audit_log v2_traffic_reset_logs v2_device_binding v2_lucky_draw_log v2_referral_duration_log; do
  sql "DELETE FROM $t;" 2>/dev/null
done
sql "INSERT INTO v2_server_group (name, created_at, updated_at) VALUES ('默认组', UNIX_TIMESTAMP(), UNIX_TIMESTAMP());
INSERT INTO v2_settings (name,value,created_at,updated_at) VALUES
 ('secure_path','admin',NOW(),NOW()), ('app_name','Lets VPN',NOW(),NOW()),
 ('app_url','$BASE',NOW(),NOW()), ('server_token','nodetoken123',NOW(),NOW()),
 ('node_probe_mode','tcp',NOW(),NOW()), ('email_verify','0',NOW(),NOW());"

export DB_HOST=127.0.0.1 DB_PORT=33306 DB_DATABASE=xboard_verify DB_USERNAME=xbv DB_PASSWORD="$PW"
export REDIS_HOST=127.0.0.1 REDIS_PORT=6379 REDIS_DB=15 LISTEN="0.0.0.0:$PORT"
export APP_KEY="base64:dmVyaWZ5LWtleS1mb3ItZ28tcG9ydA=="
export XBOARD_PUBLIC_DIR=/mnt/d/D/vps/Xboard/public XBOARD_THEME_DIR=/mnt/d/D/vps/Xboard/theme
export XBOARD_PLUGINS_DIR=/mnt/d/D/vps/Xboard/plugins-core

cd "$D"
setsid nohup "$D/xboard" serve > "$D/stack.log" 2>&1 < /dev/null &
for i in $(seq 1 60); do curl -s -o /dev/null "$BASE/" && break; sleep 0.3; done

# 管理员：走正常注册接口创建，再提权
curl -s -X POST "$BASE/api/v1/passport/auth/register" -d "email=$ADMIN_EMAIL&password=$ADMIN_PASS" > /dev/null
sql "UPDATE v2_user SET is_admin=1, group_id=1, transfer_enable=107374182400, expired_at=NULL WHERE email='$ADMIN_EMAIL';"

# 示例节点（本机端口，TCP 测速可通）
python3 -m http.server 9001 --bind 127.0.0.1 > /dev/null 2>&1 &
sql "INSERT INTO v2_server (type,name,host,port,server_port,rate,\`show\`,group_ids,protocol_settings,created_at,updated_at) VALUES
 ('shadowsocks','香港 01','127.0.0.1','9001',9001,1,1,'[1]','{\"cipher\":\"aes-128-gcm\"}',NOW(),NOW()),
 ('vmess','日本 东京 02','127.0.0.1','9','9',1.5,1,'[1]','{\"tls\":0,\"network\":\"tcp\"}',NOW(),NOW());"
sql "INSERT INTO v2_plan (name, group_id, transfer_enable, prices, sell, \`show\`, renew, sort, content, created_at, updated_at)
     VALUES ('基础套餐', 1, 100, '{\"monthly\":1000,\"yearly\":10000}', 1, 1, 1, 1, '100GB 流量 / 月', UNIX_TIMESTAMP(), UNIX_TIMESTAMP());"

echo "stack ready: $BASE  admin=$BASE/admin  user=$ADMIN_EMAIL"
pgrep -af "[x]board-verify/xboard" | head -2
