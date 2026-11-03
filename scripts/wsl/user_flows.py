"""终端用户旅程：游客配置 → 注册/登录 → 套餐 → 下单/支付回调 → 优惠券 → 工单 → 邀请 → 个人设置 → 节点/订阅。
用法：python3 user_flows.py http://127.0.0.1:17031 admin admin@letsvpn.local <admin_password>
"""
import hashlib
import json
import sys
import time
import urllib.parse
import urllib.request

BASE, SP, ADMIN_EMAIL, ADMIN_PW = sys.argv[1:5]
op = urllib.request.build_opener(urllib.request.ProxyHandler({}))
PASS = FAIL = 0
LAST = None


def req(method, path, body=None, tok="", raw=False, form=False):
    url = BASE + path
    data = None
    headers = {}
    if body is not None:
        if form:
            data = urllib.parse.urlencode(body).encode()
            headers["Content-Type"] = "application/x-www-form-urlencoded"
        else:
            data = json.dumps(body).encode()
            headers["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=data, method=method, headers=headers)
    if tok:
        r.add_header("Authorization", tok)
    try:
        with op.open(r, timeout=60) as resp:
            txt, code = resp.read().decode("utf-8", "replace"), resp.status
    except urllib.error.HTTPError as e:
        txt, code = e.read().decode("utf-8", "replace"), e.code
    if raw:
        return code, txt
    try:
        return code, json.loads(txt)
    except ValueError:
        return code, txt


def ok(cond, name, detail=None):
    global PASS, FAIL
    if cond:
        PASS += 1
        print(f"PASS  {name}")
    else:
        FAIL += 1
        d = detail if detail is not None else LAST
        print(f"FAIL  {name}  -> {str(d)[:300]}")


def good(res):
    global LAST
    LAST = res
    c, b = res
    return c == 200 and isinstance(b, dict) and b.get("status") == "success"


def data(res):
    return res[1].get("data") if isinstance(res[1], dict) else None


# ---------------- 管理端准备：组、套餐、支付、节点 ----------------
c, b = req("POST", "/api/v2/passport/auth/login", {"email": ADMIN_EMAIL, "password": ADMIN_PW})
ADM = b["data"]["auth_data"]
A = f"/api/v2/{SP}"
# 全新安装没有站点地址；支付回调地址依赖它，管理员需要先配置（真实使用时在「系统配置 → 站点」）
good(req("POST", f"{A}/config/save", {"app_url": BASE}, ADM))
good(req("POST", f"{A}/server/group/save", {"name": "用户组"}, ADM))
gid = [g for g in req("GET", f"{A}/server/group/fetch", None, ADM)[1]["data"] if g["name"] == "用户组"][0]["id"]
good(req("POST", f"{A}/plan/save", {"group_id": gid, "name": "月付套餐", "tags": [], "content": "100GB", "transfer_enable": 100,
                                    "prices": {"monthly": 1000, "yearly": 9000}, "speed_limit": None, "capacity_limit": None,
                                    "device_limit": 2, "reset_traffic_method": None}, ADM))
plan = [p for p in req("GET", f"{A}/plan/fetch", None, ADM)[1]["data"] if p["name"] == "月付套餐"][0]
good(req("POST", f"{A}/plan/update", {"id": plan["id"], "show": True, "sell": True, "renew": True}, ADM))
good(req("POST", f"{A}/payment/save", {"name": "易支付", "icon": "", "notify_domain": "", "handling_fee_percent": None, "handling_fee_fixed": None,
                                       "payment": "EPay", "config": {"url": "https://pay.example.com", "pid": "1001", "key": "secretkey", "type": "alipay"}}, ADM))
pay = [p for p in req("GET", f"{A}/payment/fetch", None, ADM)[1]["data"] if p["name"] == "易支付"][0]
good(req("POST", f"{A}/payment/show", {"id": pay["id"]}, ADM)) if not pay.get("enable") else None
good(req("POST", f"{A}/server/manage/save", {"type": "shadowsocks", "name": "测试节点", "host": "127.0.0.1", "port": "9001", "server_port": "9001",
                                              "rate": "1", "group_ids": [str(gid)], "route_ids": [], "tags": [], "show": True,
                                              "protocol_settings": {"cipher": "aes-128-gcm", "obfs": None, "obfs_settings": None}}, ADM))
good(req("POST", f"{A}/coupon/generate", {"name": "九折", "code": "SAVE10", "type": 2, "value": 10, "started_at": int(time.time()) - 100,
                                          "ended_at": int(time.time()) + 86400, "limit_use": None, "limit_use_with_user": None,
                                          "limit_plan_ids": [], "limit_period": [], "generate_count": None}, ADM))
cp = [x for x in req("POST", f"{A}/coupon/fetch", {"current": 1, "pageSize": 20}, ADM)[1]["data"] if x["code"] == "SAVE10"][0]
good(req("POST", f"{A}/coupon/update", {"id": cp["id"], "show": True}, ADM))
good(req("POST", f"{A}/notice/save", {"title": "公告A", "content": "内容A", "img_url": "", "url": "", "show": 1, "tags": []}, ADM))
good(req("POST", f"{A}/knowledge/save", {"title": "教程A", "category": "入门", "language": "zh-CN", "show": 1, "body": "正文 {{siteName}}"}, ADM))

# ---------------- 游客接口 ----------------
for path, key in [("/api/v1/guest/comm/config", "is_email_verify"), ("/api/v1/guest/plan/fetch", None),
                  ("/api/v1/guest/website/config", "downloads"), ("/api/v1/guest/discover/config", "prompt_body"),
                  ("/api/v1/guest/anti_loss/config", "socials"), ("/api/v1/guest/ad-banner/config", "countdown_seconds")]:
    r = req("GET", path)
    d = data(r)
    ok(good(r) and (key is None or (isinstance(d, dict) and key in d)), f"guest GET {path.split('/api/v1/guest/')[1]}")
r = req("GET", "/api/v1/guest/plan/fetch")
ok(any(p["name"] == "月付套餐" for p in data(r)), "guest plan list has the plan", r)
r = req("GET", "/api/v1/guest/notice/fetch")
ok(r[0] == 200 and any(n["title"] == "公告A" for n in r[1]["data"]), "guest notice list", r)

# ---------------- 注册 / 登录 ----------------
EMAIL, PW = "buyer@letsvpn.local", "Buyer-Pass-1"
r = req("POST", "/api/v1/passport/auth/register", {"email": EMAIL, "password": PW})
ok(good(r) and data(r).get("auth_data"), "register", r)
dup = req("POST", "/api/v1/passport/auth/register", {"email": EMAIL, "password": PW})
ok(dup[0] != 200 or dup[1].get("status") == "fail", "duplicate register rejected", dup)
weak = req("POST", "/api/v1/passport/auth/register", {"email": "weak@letsvpn.local", "password": "123"})
ok(weak[1].get("status") == "fail" if isinstance(weak[1], dict) else True, "short password rejected", weak)
r = req("POST", "/api/v1/passport/auth/login", {"email": EMAIL, "password": PW})
TOK = data(r)["auth_data"]
ok(good(r), "login", r)
ok(not good(req("POST", "/api/v1/passport/auth/login", {"email": EMAIL, "password": "wrong"})), "wrong password rejected")

# ---------------- 个人信息 ----------------
r = req("GET", "/api/v1/user/info", None, TOK)
ok(good(r) and data(r)["email"] == EMAIL, "user info", r)
ok(good(req("GET", "/api/v1/user/checkLogin", None, TOK)), "checkLogin")
r = req("GET", "/api/v1/user/getSubscribe", None, TOK)
ok(good(r) and data(r).get("subscribe_url"), "getSubscribe", r)
sub_url = data(r)["subscribe_url"]
ok(good(req("GET", "/api/v1/user/getStat", None, TOK)), "getStat")
ok(good(req("GET", "/api/v1/user/comm/config", None, TOK)), "user comm config")
ok(good(req("GET", "/api/v1/user/notice/fetch", None, TOK)) or req("GET", "/api/v1/user/notice/fetch", None, TOK)[0] == 200, "user notice fetch")
kr = req("GET", "/api/v1/user/knowledge/fetch?language=zh-CN", None, TOK)
ok(good(kr), "knowledge list", kr)
kid = None
for cat in (data(kr) or {}).values():
    for item in cat:
        kid = item["id"]
if kid:
    kd = req("GET", f"/api/v1/user/knowledge/fetch?id={kid}&language=zh-CN", None, TOK)
    ok(good(kd) and "正文" in json.dumps(data(kd), ensure_ascii=False), "knowledge detail", kd)
ok(good(req("GET", "/api/v1/user/knowledge/getCategory", None, TOK)), "knowledge categories")

# ---------------- 套餐 / 订单 / 支付 ----------------
r = req("GET", "/api/v1/user/plan/fetch", None, TOK)
ok(good(r) and any(p["id"] == plan["id"] for p in data(r)), "user plan list", r)
r = req("GET", f"/api/v1/user/plan/fetch?id={plan['id']}", None, TOK)
ok(good(r) and data(r)["id"] == plan["id"], "user plan detail", r)
cc = req("POST", "/api/v1/user/coupon/check", {"code": "SAVE10", "plan_id": plan["id"]}, TOK)
ok(good(cc) and data(cc)["code"] == "SAVE10", "coupon check", cc)
bad_cc = req("POST", "/api/v1/user/coupon/check", {"code": "NOPE", "plan_id": plan["id"]}, TOK)
ok(not good(bad_cc), "invalid coupon rejected", bad_cc)
od = req("POST", "/api/v1/user/order/save", {"plan_id": plan["id"], "period": "monthly", "coupon_code": "SAVE10"}, TOK)
ok(good(od) and data(od), "order save", od)
trade = data(od)
ok(not good(req("POST", "/api/v1/user/order/save", {"plan_id": plan["id"], "period": "monthly"}, TOK)), "second pending order rejected")
r = req("GET", f"/api/v1/user/order/detail?trade_no={trade}", None, TOK)
ok(good(r) and data(r)["trade_no"] == trade and data(r)["total_amount"] == 90000 and data(r)["discount_amount"] == 10000,
   "order detail (coupon 10% off)", r)
ok(good(req("GET", "/api/v1/user/order/fetch", None, TOK)) and len(data(req("GET", "/api/v1/user/order/fetch", None, TOK))) == 1, "order list")
r = req("GET", "/api/v1/user/order/getPaymentMethod", None, TOK)
ok(good(r) and any(m["id"] == pay["id"] for m in data(r)), "payment methods", r)
co = req("POST", "/api/v1/user/order/checkout", {"trade_no": trade, "method": pay["id"]}, TOK)
ok(co[0] == 200 and "pay.example.com" in str(co[1].get("data")) and co[1].get("type") == 1, "checkout returns pay url", co)
st = req("GET", f"/api/v1/user/order/check?trade_no={trade}", None, TOK)
ok(good(st) and data(st) == 0, "order status pending", st)


def epay_sign(params, key):
    raw = "&".join(f"{k}={params[k]}" for k in sorted(params) if params[k] != "")
    return hashlib.md5((raw + key).encode()).hexdigest()


cb = {"out_trade_no": trade, "trade_no": "GW-" + trade[:6], "money": "900.00"}
cb["sign"], cb["sign_type"] = epay_sign(cb, "secretkey"), "MD5"
r = req("GET", f"/api/v1/guest/payment/notify/EPay/{pay['uuid']}?" + urllib.parse.urlencode(cb), raw=True)
ok(r[0] == 200 and r[1] == "success", "payment callback accepted", r)
time.sleep(1)
st = req("GET", f"/api/v1/user/order/check?trade_no={trade}", None, TOK)
# 回调后订单进入处理中(1) 或已完成(3)；调度器每分钟推进
ok(good(st) and data(st) in (1, 3), "order advanced after callback", st)
r = req("GET", "/api/v1/user/info", None, TOK)
for _ in range(80):  # 等调度器的 check:order（每分钟）把订单开通
    st = req("GET", f"/api/v1/user/order/check?trade_no={trade}", None, TOK)
    if data(st) == 3:
        break
    time.sleep(1)
ok(data(st) == 3, "order completed (activated)", st)
r = req("GET", "/api/v1/user/info", None, TOK)
ok(data(r)["plan_id"] == plan["id"] and data(r)["transfer_enable"] == 100 * 1024 ** 3, "plan applied to user", data(r))

# ---------------- 节点 / 订阅 ----------------
req("POST", f"{A}/server/manage/probeLatency", {}, ADM)
for _ in range(30):
    nodes = req("GET", f"{A}/server/manage/getNodes", None, ADM)[1]["data"]
    if nodes and nodes[0].get("last_latency_at"):
        break
    time.sleep(1)
r = req("GET", "/api/v1/user/server/fetch", None, TOK)
ok(r[0] == 200 and len(data(r)) == 1 and "0001-01-01" not in data(r)[0]["cache_key"], "user sees the node after activation", r)
r = req("GET", "/api/v1/user/server/revision", None, TOK)
ok(r[0] == 200 and isinstance(r[1].get("data"), dict) and r[1]["data"].get("revision"), "server revision", r)
token = sub_url.rstrip("/").split("/")[-1].split("token=")[-1]
r = req("GET", f"/s-enc/{token}", raw=True)
ok(r[0] == 200 and "aes-256-gcm" in r[1], "encrypted subscription", r)
ok(req("GET", f"/s/{token}", raw=True)[0] == 403, "plain subscription disabled")
r = req("GET", f"/api/v1/client/app/getVersion?token={token}", raw=True)
ok(r[0] == 200, "client app getVersion", r)
r = req("GET", f"/api/v1/client/app/getConfig?token={token}", raw=True)
ok(r[0] == 200 and "proxies" in r[1], "client app getConfig (clash yaml)", r[1][:150])

# ---------------- 工单 ----------------
r = req("POST", "/api/v1/user/ticket/save", {"subject": "连不上", "level": 1, "message": "节点无法连接"}, TOK)
ok(good(r), "ticket save", r)
tl = req("GET", "/api/v1/user/ticket/fetch", None, TOK)
tid = data(tl)[0]["id"] if good(tl) and data(tl) else None
ok(tid is not None, "ticket list", tl)
td = req("GET", f"/api/v1/user/ticket/fetch?id={tid}", None, TOK)
ok(good(td) and len(data(td)["message"]) == 1, "ticket detail", td)
ar = req("POST", f"{A}/ticket/reply", {"id": tid, "message": "已处理"}, ADM)
ok(good(ar), "admin reply", ar)
r = req("POST", "/api/v1/user/ticket/reply", {"id": tid, "message": "谢谢"}, TOK)
ok(good(r), "user reply", r)
r = req("POST", "/api/v1/user/ticket/close", {"id": tid}, TOK)
ok(good(r), "ticket close", r)

# ---------------- 邀请 / 设备 / 礼品卡 / 其它 ----------------
ok(good(req("GET", "/api/v1/user/invite/save", None, TOK)), "invite code generate")
r = req("GET", "/api/v1/user/invite/fetch", None, TOK)
ok(good(r) and len(data(r)["codes"]) >= 1, "invite fetch", r)
ok(req("GET", "/api/v1/user/invite/details", None, TOK)[0] == 200, "invite details")
ok(good(req("GET", "/api/v1/user/stat/getTrafficLog", None, TOK)), "traffic log")
ok(good(req("GET", "/api/v1/user/devices/fetch", None, TOK)) or req("GET", "/api/v1/user/devices/fetch", None, TOK)[0] == 200, "devices fetch")
ok(good(req("GET", "/api/v1/user/gift-card/types", None, TOK)) or req("GET", "/api/v1/user/gift-card/types", None, TOK)[0] == 200, "gift card types")
ok(good(req("GET", "/api/v1/user/getActiveSession", None, TOK)), "active sessions")
ok(good(req("GET", "/api/v1/user/lucky_draw/info", None, TOK)) or req("GET", "/api/v1/user/lucky_draw/info", None, TOK)[0] == 200, "lucky draw info")

# ---------------- 设置 / 安全 ----------------
ok(good(req("POST", "/api/v1/user/update", {"remind_expire": 0, "remind_traffic": 0}, TOK)), "user update reminders")
r = req("GET", "/api/v1/user/info", None, TOK)
ok(data(r)["remind_expire"] in (0, False) and data(r)["remind_traffic"] in (0, False), "reminders persisted", data(r))
r = req("POST", "/api/v1/user/changePassword", {"old_password": PW, "new_password": "New-Pass-2"}, TOK)
ok(good(r), "changePassword", r)
ok(not good(req("POST", "/api/v1/passport/auth/login", {"email": EMAIL, "password": PW})), "old password no longer works")
r = req("POST", "/api/v1/passport/auth/login", {"email": EMAIL, "password": "New-Pass-2"})
ok(good(r), "new password works", r)
TOK2 = data(r)["auth_data"]
r = req("GET", "/api/v1/user/resetSecurity", None, TOK2)
ok(good(r) and data(r) != sub_url, "resetSecurity rotates subscription", r)
ql = req("POST", "/api/v1/user/getQuickLoginUrl", {}, TOK2)
ok(good(ql) and "verify=" in str(data(ql)), "quick login url", ql)

# ---------------- 节点端 API（UniProxy）----------------
r = req("GET", "/api/v1/server/UniProxy/user?token=nodetoken123&node_id=1&node_type=shadowsocks", raw=True)
ok(r[0] in (200, 400, 422, 500), "uniproxy reachable", r[0])

print(f"================ PASS={PASS} FAIL={FAIL}")
sys.exit(1 if FAIL else 0)
