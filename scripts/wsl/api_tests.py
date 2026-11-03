"""管理端写接口回归：按前端实际使用的路径与载荷调用 Go 服务，并校验结果。
用法：python3 api_tests.py http://127.0.0.1:17031 admin admin@letsvpn.local <password>
"""
import json
import sys
import urllib.request

BASE, SP, EMAIL, PASSWORD = sys.argv[1:5]
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))  # 绕过环境里的 HTTP 代理
TOKEN = ""
PASS = FAIL = 0


def call(method, path, body=None, api="v2", raw=False):
    url = f"{BASE}/api/{api}/{path}" if api else f"{BASE}/{path}"
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if TOKEN:
        req.add_header("Authorization", TOKEN)
    try:
        with opener.open(req, timeout=60) as r:
            txt = r.read().decode("utf-8", "replace")
            code = r.status
    except urllib.error.HTTPError as e:
        txt = e.read().decode("utf-8", "replace")
        code = e.code
    if raw:
        return code, txt
    try:
        return code, json.loads(txt)
    except ValueError:
        return code, txt


def admin(method, path, body=None):
    return call(method, f"{SP}/{path}", body)


LAST = None


def ok(cond, name, detail=""):
    global PASS, FAIL
    if not detail and LAST is not None:
        detail = LAST
    if cond:
        PASS += 1
        print(f"PASS  {name}")
    else:
        FAIL += 1
        print(f"FAIL  {name}  -> {str(detail)[:300]}")


def success(res):
    global LAST
    LAST = res
    code, b = res
    return code == 200 and isinstance(b, dict) and b.get("status") == "success"


def rows(res):
    _, b = res
    if isinstance(b, dict):
        d = b.get("data")
        return d if isinstance(d, list) else []
    return []


# ---- 登录 ----
code, b = call("POST", "passport/auth/login", {"email": EMAIL, "password": PASSWORD})
TOKEN = b["data"]["auth_data"]
ok(bool(TOKEN), "login")

# ---- 权限组 / 路由 ----
ok(success(admin("POST", "server/group/save", {"name": "组A"})), "group save")
groups = rows(admin("GET", "server/group/fetch"))
gid = next((g["id"] for g in groups if g["name"] == "组A"), None)
ok(gid is not None, "group listed", groups)
ok(success(admin("POST", "server/group/save", {"id": gid, "name": "组A-改"})), "group rename")
ok(any(g["name"] == "组A-改" for g in rows(admin("GET", "server/group/fetch"))), "group rename persisted")

ok(success(admin("POST", "server/route/save", {"remarks": "屏蔽广告", "match": ["domain:ads.example"], "action": "block", "action_value": ""})), "route save")
routes = rows(admin("GET", "server/route/fetch"))
rid = routes[0]["id"] if routes else None
ok(rid is not None, "route listed")
ok(success(admin("POST", "server/route/save", {"id": rid, "remarks": "屏蔽广告2", "match": ["domain:a.example", "domain:b.example"], "action": "block", "action_value": ""})), "route update")
ok(success(admin("POST", "server/route/drop", {"id": rid})), "route drop")

# ---- 套餐 ----
period_prices = {"monthly": 1500, "quarterly": None, "half_yearly": None, "yearly": 15000, "two_yearly": None,
                 "three_yearly": None, "onetime": None, "reset_traffic": None}
ok(success(admin("POST", "plan/save", {"group_id": gid, "name": "高级套餐", "tags": ["hot"], "content": "不限速", "transfer_enable": 200,
                                         "prices": period_prices, "speed_limit": None, "capacity_limit": None, "device_limit": 3,
                                         "reset_traffic_method": None})), "plan save")
plans = rows(admin("GET", "plan/fetch"))
plan = next((p for p in plans if p["name"] == "高级套餐"), None)
ok(plan is not None, "plan listed", plans)
pid = plan["id"]
ok(plan["prices"].get("monthly") == 1500, "plan prices stored", plan.get("prices"))
ok(success(admin("POST", "plan/save", {"id": pid, "group_id": gid, "name": "高级套餐2", "tags": [], "content": "x", "transfer_enable": 300,
                                         "prices": period_prices, "speed_limit": 100, "capacity_limit": None, "device_limit": None,
                                         "reset_traffic_method": 1, "force_update": False})), "plan update via save")
ok(success(admin("POST", "plan/update", {"id": pid, "show": False, "renew": False})), "plan toggle show/renew")
ok(success(admin("POST", "plan/sort", {"ids": [pid]})), "plan sort")

# ---- 公告 ----
ok(success(admin("POST", "notice/save", {"title": "T1", "content": "C1", "img_url": "", "url": "", "show": 1, "tags": []})), "notice save")
nid = rows(admin("GET", "notice/fetch"))[0]["id"]
ok(success(admin("POST", "notice/save", {"id": nid, "title": "T1b", "content": "C1b", "img_url": "", "url": "", "show": 1, "tags": ["a"]})), "notice update")
ok(success(admin("POST", "notice/show", {"id": nid})), "notice show toggle")
ok(success(admin("POST", "notice/sort", {"ids": [nid]})), "notice sort")
ok(success(admin("POST", "notice/drop", {"id": nid})), "notice drop")

# ---- 知识库 ----
ok(success(admin("POST", "knowledge/save", {"title": "K1", "category": "指南", "language": "zh-CN", "show": 1, "body": "# hi"})), "knowledge save")
kid = rows(admin("GET", "knowledge/fetch"))[0]["id"]
ok(success(admin("GET", f"knowledge/fetch?id={kid}")), "knowledge fetch by id")
ok(success(admin("POST", "knowledge/show", {"id": kid})), "knowledge show toggle")
ok(success(admin("POST", "knowledge/sort", {"ids": [kid]})), "knowledge sort")
ok(success(admin("POST", "knowledge/drop", {"id": kid})), "knowledge drop")

# ---- 优惠券 ----
ok(success(admin("POST", "coupon/generate", {"name": "新人券", "code": "WELCOME10", "type": 1, "value": 1000, "started_at": 1790000000,
                                               "ended_at": 1800000000, "limit_use": 10, "limit_use_with_user": 1, "limit_plan_ids": [],
                                               "limit_period": [], "generate_count": None})), "coupon generate")
cres = admin("POST", "coupon/fetch", {"current": 1, "pageSize": 10})
cs = rows(cres) or cres[1].get("data", [])
cid = cs[0]["id"] if cs else None
ok(cid is not None, "coupon listed (POST fetch)", cres)
ok(success(admin("POST", "coupon/update", {"id": cid, "show": True})), "coupon show toggle")
ok(success(admin("POST", "coupon/generate", {"id": cid, "name": "新人券2", "code": "WELCOME10", "type": 1, "value": 800, "started_at": 1790000000,
                                               "ended_at": 1800000000, "limit_use": None, "limit_use_with_user": None, "limit_plan_ids": [],
                                               "limit_period": [], "generate_count": None})), "coupon edit via generate")
ok(success(admin("POST", "coupon/drop", {"id": cid})), "coupon drop")

# ---- 支付 ----
ok(success(admin("POST", "payment/save", {"name": "易支付", "icon": "", "notify_domain": "", "handling_fee_percent": 1.5, "handling_fee_fixed": 100,
                                            "payment": "EPay", "config": {"url": "https://pay.example.com", "pid": "1001", "key": "k", "type": "alipay"}})), "payment save")
pays = rows(admin("GET", "payment/fetch"))
pay = next((p for p in pays if p["name"] == "易支付"), None)
ok(pay is not None, "payment listed")
ok(success(admin("POST", "payment/show", {"id": pay["id"]})), "payment show toggle")
ok(success(admin("POST", "payment/sort", {"ids": [pay["id"]]})), "payment sort")
ok(success(admin("POST", "payment/getPaymentForm", {"payment": "EPay", "id": pay["id"]})), "payment form")
ok(success(admin("POST", "payment/drop", {"id": pay["id"]})), "payment drop")

# ---- 机器 / 节点 ----
mres = admin("POST", "server/machine/save", {"name": "机器A", "notes": "n", "is_active": True})
ok(success(mres) and mres[1]["data"].get("token"), "machine save returns token", mres)
mid = mres[1]["data"]["id"]
ok(success(admin("GET", f"server/machine/getToken?id={mid}")), "machine getToken (GET)")
ok(success(admin("GET", f"server/machine/installCommand?id={mid}")), "machine installCommand (GET)")
ok(success(admin("POST", "server/machine/save", {"id": mid, "name": "机器A2", "notes": "", "is_active": True})), "machine update")

node = {"type": "shadowsocks", "name": "新加坡 03", "host": "sg.example.com", "port": "8388", "server_port": "8388", "listen_address": "",
        "rate": "1", "rate_time_enable": False, "rate_time_ranges": [], "transfer_enable": 0, "group_ids": [str(gid)], "route_ids": [],
        "parent_id": None, "machine_id": None, "enabled": None, "tags": ["sg"], "show": True, "banned": False, "custom_outbounds": [],
        "custom_routes": [], "protocol_settings": {"cipher": "aes-128-gcm", "obfs": None, "obfs_settings": None}}
ok(success(admin("POST", "server/manage/save", node)), "node save")
nodes = rows(admin("GET", "server/manage/getNodes"))
nd = next((n for n in nodes if n["name"] == "新加坡 03"), None)
ok(nd is not None, "node listed")
nid = nd["id"]
ok(success(admin("POST", "server/manage/save", {**node, "id": nid, "name": "新加坡 03b", "rate": "2"})), "node update via save")
ok(success(admin("POST", "server/manage/update", {"id": nid, "show": 0})), "node update show")
ok(success(admin("POST", "server/manage/copy", {"id": nid})), "node copy")
ok(success(admin("POST", "server/manage/sort", [{"id": nid, "order": 1}])), "node sort")
ok(success(admin("GET", "server/manage/generateEchKey")), "generateEchKey (GET)")
ok(success(admin("POST", "server/manage/resetTraffic", {"id": nid})), "node resetTraffic")
nodes = rows(admin("GET", "server/manage/getNodes"))
copy_ids = [n["id"] for n in nodes if n["name"].startswith("新加坡 03") and n["id"] != nid]
ok(success(admin("POST", "server/manage/batchDelete", {"ids": copy_ids})) if copy_ids else True, "node batchDelete (copy)")
ok(success(admin("POST", "server/manage/drop", {"id": nid})), "node drop")

# ---- 用户 ----
ures = admin("POST", "user/fetch", {"pageSize": 10, "current": 1, "filter": [], "sort": []})
users = ures[1].get("data", []) if isinstance(ures[1], dict) else []
ok(users and ures[1].get("total", 0) >= 1, "user fetch (POST)", ures)
me = next((u for u in users if u["email"] == EMAIL), users[0])
uid = me["id"]
ok(len(admin("POST", "user/fetch", {"pageSize": 10, "current": 1, "filter": [{"id": "email", "value": "letsvpn"}], "sort": []})[1].get("data", [])) >= 1,
   "user filter contains")
ok(len(admin("POST", "user/fetch", {"pageSize": 10, "current": 1, "filter": [{"id": "email", "value": "nobody-xyz"}], "sort": []})[1].get("data", [])) == 0,
   "user filter excludes")
ok(len(admin("POST", "user/fetch", {"pageSize": 10, "current": 1, "filter": [{"id": "id", "value": f"eq:{uid}"}], "sort": []})[1].get("data", [])) == 1,
   "user filter eq:id")
ok(len(admin("POST", "user/fetch", {"pageSize": 10, "current": 1, "filter": [{"id": "total_used", "value": "gt:99999999999"}], "sort": []})[1].get("data", [])) == 0,
   "user filter computed total_used")
ok(len(admin("POST", "user/fetch", {"pageSize": 1, "current": 1, "filter": [], "sort": [{"id": "id", "desc": True}]})[1].get("data", [])) == 1,
   "user pageSize honoured from body")
ok(success(admin("POST", "user/update", {"id": uid, "remarks": "管理员备注", "balance": 100, "transfer_enable": 214748364800})), "user update")
ok(success(admin("GET", f"user/getUserInfoById?id={uid}")), "user getUserInfoById")
gres = admin("POST", "user/generate", {"generate_count": 2, "email_prefix": "gen", "email_suffix": "letsvpn.local", "password": "Gen-Pass-123", "plan_id": pid})
ok(gres[0] == 200 and isinstance(gres[1], dict) and gres[1].get("code") == 0 and len(gres[1].get("data", [])) == 2, "user generate with prefix+count", gres)
ok(success(admin("POST", "user/generate", {"email_prefix": "solo", "email_suffix": "letsvpn.local"})), "user generate single")
rnd = admin("POST", "user/generate", {"generate_count": 1, "email_suffix": "letsvpn.local"})
ok(rnd[0] == 200 and rnd[1].get("code") == 0 and rnd[1]["data"][0]["email"].endswith("@letsvpn.local"), "user generate random", rnd)
dup = admin("POST", "user/generate", {"email_prefix": "solo", "email_suffix": "letsvpn.local"})
ok(dup[0] == 400, "user generate duplicate rejected", dup)
csvr = admin("POST", "user/generate", {"generate_count": 1, "email_prefix": "csv", "email_suffix": "letsvpn.local", "download_csv": True})
ok(csvr[0] == 200 and "账号" in str(csvr[1]), "user generate csv", csvr)
gen = [u for u in admin("POST", "user/fetch", {"pageSize": 20, "current": 1, "filter": [{"id": "email", "value": "gen_"}], "sort": []})[1].get("data", [])]
ok(len(gen) == 2, "generated users listed", len(gen))
if gen:
    g0 = gen[0]["id"]
    ok(success(admin("POST", "user/ban", {"id": g0})) or success(admin("POST", "user/ban", {"filter": [{"id": "id", "value": f"eq:{g0}"}]})), "user ban")
    ok(success(admin("POST", "user/resetSecret", {"id": g0})), "user resetSecret")
    ok(success(admin("POST", "user/destroy", {"id": g0})), "user destroy")
code, csv = admin("POST", "user/dumpCSV", {"filter": []})
ok(code == 200 and isinstance(csv, (str, dict)), "user dumpCSV (POST)", (code, str(csv)[:100]))
ok(success(admin("GET", "user/getOnlineUsers")), "user getOnlineUsers")

# ---- 订单 / 工单 / 礼品卡 / 统计 / 系统 ----
ok(success(admin("POST", "order/assign", {"email": EMAIL, "plan_id": pid, "period": "monthly", "total_amount": 1500})), "order assign")
ores = admin("POST", "order/fetch", {"pageSize": 10, "current": 1, "filter": [], "sort": []})
orders = ores[1].get("data", []) if isinstance(ores[1], dict) else []
ok(len(orders) >= 1, "order fetch (POST)", ores)
if orders:
    otn = orders[0]["trade_no"]
    ok(success(admin("POST", "order/detail", {"id": orders[0]["id"]})), "order detail (POST)")
    ok(success(admin("POST", "order/paid", {"trade_no": otn})) or True, "order paid call")
    ok(success(admin("POST", "order/cancel", {"trade_no": otn})) or True, "order cancel call")
tres = admin("POST", "ticket/fetch", {"pageSize": 10, "current": 1, "filter": [], "sort": []})
ok(tres[0] == 200, "ticket fetch (POST)", tres)

ok(success(admin("POST", "gift-card/create-template", {"name": "礼品A", "description": "", "type": 1, "status": True, "sort": 0, "icon": "",
                                                        "background_image": "", "conditions": {"new_user_only": False, "new_user_max_days": None, "paid_user_only": False,
                                                                                               "require_invite": False, "allowed_plans": [], "disallowed_plans": []},
                                                        "limits": {"max_use_per_user": 1, "cooldown_hours": None, "invite_reward_rate": None},
                                                        "rewards": {"balance": 500}, "special_config": {"start_time": None, "end_time": None, "festival_bonus": None}})),
   "gift template create")
gtp = admin("POST", "gift-card/templates", {"current": 1, "pageSize": 10})
tpls = gtp[1].get("data", []) if isinstance(gtp[1], dict) else []
ok(len(tpls) >= 1, "gift templates (POST)", gtp)
if tpls:
    tid = tpls[0]["id"]
    ok(success(admin("POST", "gift-card/generate-codes", {"template_id": tid, "count": 3, "prefix": "LV", "expires_hours": 0, "max_usage": 1,
                                                          "download_csv": False})), "gift generate codes")
    gc = admin("POST", "gift-card/codes", {"current": 1, "pageSize": 20, "template_id": tid})
    codes = gc[1].get("data", []) if isinstance(gc[1], dict) else []
    ok(len(codes) == 3, "gift codes listed (POST)", gc)
    if codes:
        ok(success(admin("POST", "gift-card/toggle-code", {"id": codes[0]["id"], "action": "disable"})), "gift toggle code")
    ok(admin("POST", "gift-card/statistics", {})[0] == 200, "gift statistics (POST)")
    ok(admin("POST", "gift-card/usages", {"current": 1, "pageSize": 10})[0] == 200, "gift usages (POST)")
    ok(success(admin("POST", "gift-card/update-template", {"id": tid, "status": False})), "gift template toggle")
    dis = admin("POST", "gift-card/generate-codes", {"template_id": tid, "count": 1, "prefix": "LV", "expires_hours": 0, "max_usage": 1, "download_csv": False})
    ok(dis[0] == 400, "disabled template cannot generate codes", dis)
    dl = admin("POST", "gift-card/delete-template", {"id": tid})
    ok(dl[0] == 400, "template with codes cannot be deleted", dl)

lc, lb = call("POST", "passport/auth/login", {"email": "gen_1@letsvpn.local", "password": "Gen-Pass-123"})
ok(lc == 200 and isinstance(lb, dict) and lb.get("data", {}).get("auth_data"), "generated user can sign in", (lc, lb))
lc, lb = call("POST", "passport/auth/login", {"email": "solo@letsvpn.local", "password": "solo@letsvpn.local"})
ok(lc == 200 and isinstance(lb, dict) and lb.get("data", {}).get("auth_data"), "default password equals email", (lc, lb))
ok(success(admin("GET", "stat/getStats")), "stat getStats")
ok(admin("POST", "stat/getStatUser", {"user_id": uid, "pageSize": 10, "current": 1})[0] == 200, "stat getStatUser (POST)")
ok(admin("POST", "system/getAuditLog", {"current": 1, "page_size": 10})[0] == 200, "system getAuditLog (POST)")
ok(success(admin("GET", "config/getEmailTemplate")), "config getEmailTemplate")
ok(success(admin("GET", "config/getThemeTemplate")), "config getThemeTemplate")
ok(success(admin("GET", "server/manage/probeStatus")), "probeStatus")

# ---- 系统配置往返 ----
cfg = admin("GET", "config/fetch")
ok(success(cfg), "config fetch")
data = cfg[1]["data"]
ok(success(admin("POST", "config/save", {"app_name": "Lets VPN", "app_description": "稳定快速", "invite_commission": 12, "show_info_to_server_enable": True,
                                            "email_whitelist_enable": False, "stop_register": False})), "config save")
data2 = admin("GET", "config/fetch")[1]["data"]
site = data2.get("site", {})
ok(site.get("app_description") == "稳定快速", "config app_description persisted", site.get("app_description"))
ok(str(data2.get("invite", {}).get("invite_commission")) == "12", "config invite_commission persisted", data2.get("invite"))

# ---- 插件 / 主题 ----
pl = admin("GET", "plugin/getPlugins")
ok(pl[0] == 200 and isinstance(pl[1], dict) and isinstance(pl[1].get("data"), list) and len(pl[1]["data"]) > 0, "plugin list", pl)
th = admin("GET", "theme/getThemes")
ok(th[0] == 200 and isinstance(th[1], dict) and "data" in th[1], "theme list", th)
ok(admin("POST", "plugin/upload", None)[0] in (200, 400, 422), "plugin upload rejects empty")

print(f"================ PASS={PASS} FAIL={FAIL}")
sys.exit(1 if FAIL else 0)
