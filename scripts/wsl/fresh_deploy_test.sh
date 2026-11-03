#!/bin/bash
# 干净服务器演练：空目录 + 只有发布包里的文件 + 一个不存在的数据库。
D="$HOME/xboard-verify"
PW="$(cat "$D/pw")"
PORT=17041
PASS=0; FAIL=0
ok()  { echo "PASS  $1"; PASS=$((PASS+1)); }
bad() { echo "FAIL  $1  -> $2"; FAIL=$((FAIL+1)); }
has() { if echo "$2" | grep -q -- "$3"; then ok "$1"; else bad "$1" "missing [$3] in: $(echo "$2" | head -c 250)"; fi; }
root() { mariadb --no-defaults --socket="$D/mysql.sock" -uroot "$@"; }
C() { curl --noproxy '*' -s -m 20 "$@"; }

# 干净环境：删库、删目录
root -e "DROP DATABASE IF EXISTS xboard_fresh; GRANT ALL ON *.* TO 'xbv'@'127.0.0.1'; FLUSH PRIVILEGES;"
pkill -f "[l]v-deploy/xboard serve" 2>/dev/null; sleep 1
rm -rf "$HOME/lv-deploy"; mkdir -p "$HOME/lv-deploy/extract"
tar -xzf "$1" -C "$HOME/lv-deploy/extract"
PKG="$(ls -d "$HOME"/lv-deploy/extract/*/)"
cp "$PKG/xboard" "$PKG/.env.example" "$HOME/lv-deploy/"
cd "$HOME/lv-deploy" || exit 1
ls -la | head -8

# 版本与 keygen
has "version prints" "$(./xboard version)" "xboard-go 1.0.0"
KEY="$(./xboard keygen)"
has "keygen format" "$KEY" "^base64:"

# 写 .env（只通过 .env 配置，不用环境变量）
cat > .env <<EOF
APP_KEY=$KEY
LISTEN=0.0.0.0:$PORT
DB_HOST=127.0.0.1
DB_PORT=33306
DB_DATABASE=xboard_fresh
DB_USERNAME=xbv
DB_PASSWORD=$PW
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_DB=14
TRUSTED_PROXIES=127.0.0.1
EOF
redis-cli -n 14 flushdb > /dev/null

# migrate：库不存在 → 自动创建 + 建表 + 种子
OUT="$(./xboard migrate 2>&1)"; echo "$OUT"
has "migrate created the database and tables" "$OUT" "新建 3[0-9] 张表"
T="$(root -N -e "select count(*) from information_schema.tables where table_schema='xboard_fresh'")"
[ "$T" -ge 30 ] && ok "tables present ($T)" || bad "tables present" "$T"
[ "$(root -N -e 'select count(*) from xboard_fresh.v2_subscribe_templates')" = "6" ] && ok "seed: 6 subscribe templates" || bad "seed templates" "?"
OUT2="$(./xboard migrate 2>&1)"
has "second migrate is a no-op" "$OUT2" "已是最新"
[ "$(root -N -e 'select count(*) from xboard_fresh.v2_subscribe_templates')" = "6" ] && ok "seed not duplicated" || bad "seed duplicated" "?"

# install：创建管理员并打印后台地址
INS="$(./xboard install admin@fresh.local 2>&1)"; echo "$INS"
APW="$(echo "$INS" | sed -n 's/^  密码: //p')"
has "install created the admin" "$INS" "管理员已创建"
has "install prints the admin path" "$INS" "后台地址"
SP="$(echo "$INS" | sed -n 's#.*站点/##p' | tr -d '\r ')"
INS2="$(./xboard install 2>&1)"
has "second install skips admin creation" "$INS2" "已存在管理员"

# 启动
nohup ./xboard serve > serve.log 2>&1 &
SP_PID=$!
for i in $(seq 1 50); do C -o /dev/null "http://127.0.0.1:$PORT/healthz" && break; sleep 0.2; done
B="http://127.0.0.1:$PORT"
has "healthz" "$(C $B/healthz)" '"status":"ok"'
grep -q "release" serve.log || true
! grep -q "GIN-debug" serve.log && ok "release mode (no GIN-debug noise)" || bad "release mode" "debug logs present"
has "serve log shows admin path" "$(cat serve.log)" "管理后台: /$SP"

# 前端：没有 public/ 与 theme/ 目录，全部来自内嵌
[ ! -d public ] && [ ! -d theme ] && ok "no public/ or theme/ on disk" || bad "disk dirs" "present"
has "user site shell" "$(C $B/)" "umi.js"
has "admin shell" "$(C $B/$SP)" '<div id="root">'
JS="$(C $B/$SP | sed -n 's#.*src="\(/assets/admin/assets/index-[^"]*\.js\)".*#\1#p' | head -1)"
CODE="$(C -o /dev/null -w '%{http_code}' $B$JS)"
[ "$CODE" = "200" ] && ok "embedded admin bundle served ($JS)" || bad "admin bundle" "$CODE"
CODE="$(C -o /dev/null -w '%{http_code}' $B/theme/Xboard/assets/umi.js)"
[ "$CODE" = "200" ] && ok "embedded user theme served" || bad "user theme" "$CODE"
CODE="$(C -o /dev/null -w '%{http_code}' $B/assets/admin/assets/../../../../etc/passwd)"
[ "$CODE" != "200" ] && ok "static path traversal rejected ($CODE)" || bad "traversal" "$CODE"

# 登录与后台
R="$(C -X POST $B/api/v1/passport/auth/login -d "email=admin@fresh.local&password=$APW")"
TOK="$(echo "$R" | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["auth_data"])' 2>/dev/null)"
[ -n "$TOK" ] && ok "admin login with printed password" || bad "admin login" "$R"
has "guest comm config" "$(C $B/api/v1/guest/comm/config)" "is_email_verify"
A="$B/api/v2/$SP"
has "admin plugin list shows embedded core plugins" "$(C -H "Authorization: $TOK" $A/plugin/getPlugins)" "AlipayF2F"
has "admin theme list shows embedded theme" "$(C -H "Authorization: $TOK" $A/theme/getThemes)" '"name":"Xboard"'
has "admin config fetch" "$(C -H "Authorization: $TOK" $A/config/fetch)" "app_name"
has "subscribe templates seeded (config)" "$(C -H "Authorization: $TOK" $A/config/fetch)" "subscribe_template"

# 全流程：沿用用户旅程脚本（管理员密码来自 install 输出）
python3 "$2" "$B" "$SP" admin@fresh.local "$APW" 2>&1 | grep -v "^PASS" | tail -8
FLOW_RC=${PIPESTATUS[0]}
[ "$FLOW_RC" = "0" ] && ok "user_flows.py full journey on the fresh install" || bad "user_flows.py" "exit $FLOW_RC"

# 优雅退出
kill -TERM $SP_PID; for i in $(seq 1 20); do kill -0 $SP_PID 2>/dev/null || break; sleep 0.5; done
kill -0 $SP_PID 2>/dev/null && bad "SIGTERM exit" "still running" || ok "SIGTERM exit"
echo "================ PASS=$PASS FAIL=$FAIL"
tail -5 serve.log
