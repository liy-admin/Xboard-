#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
使用 ruyiPage(Firefox + WebDriver BiDi 反检测框架)自动登录 balala.io
（SSPanel-UIM / metron 主题，面板「gw树洞」）并抓取订阅链接 URL。

为什么换 ruyiPage：该站登录开启了 Cloudflare Turnstile，DrissionPage 的 Chromium
无头模式会被识别而拿不到 token；ruyiPage 以反检测 Firefox 呈现为「正常浏览器」，
托管型 Turnstile 可像真人浏览器一样自动放行（本脚本不求解交互式验证码）。

核对自真实站点（2026-10）：
  - 登录页：  {site}/auth/login          （服务端渲染，非 SPA）
  - 登录提交：POST {site}/auth/login      字段 email / passwd / remember_me / code
  - 鉴权：    PHP session cookie（无 JS token）
  - 订阅链接：内嵌在 {site}/user 页面复制按钮的 data-clipboard-text 属性，
             形如 https://host/<rand>?token=<token>&via=<client>

约定：
  - 配置通过环境变量传入。
  - 只有最终结果以一行 JSON 打印到 stdout；所有诊断信息打印到 stderr。
  - 成功：{"success": true, "subscribe_url": "...", "subscriptions": [...], "source": "login|cookie"}
  - 失败：{"success": false, "error": "..."}

依赖：
  pip install ruyiPage --upgrade
  python -m ruyipage install          # 下载推荐的 Firefox 运行时（首次一次）
"""

import json
import os
import re
import sys
import time

SUB_LINK_RE = re.compile(
    r"https?://[^\s\"'<>]+(?:\?token=[A-Za-z0-9]+|/link/[A-Za-z0-9]+)[^\s\"'<>]*"
)


def log(*args):
    print(*args, file=sys.stderr, flush=True)


def out(obj):
    s = json.dumps(obj, ensure_ascii=False)
    print(s, flush=True)
    # 同时写入结果文件：某些环境(无控制台/Octane-Swoole 协程)下 PHP 抓不到子进程 stdout，
    # 改用文件 IPC 可靠传递最终结果。
    of = os.environ.get("BALALA_OUTFILE")
    if of:
        try:
            with open(of, "w", encoding="utf-8") as f:
                f.write(s)
        except Exception as e:
            log(f"[out] 写结果文件失败: {e}")


def env(name, default=""):
    v = os.environ.get(name)
    return default if v is None or v == "" else v


def _firefox_match_token():
    """用于匹配本插件 Firefox 进程的路径片段（主进程+所有 contentproc 都带它）。"""
    p = env("BALALA_CHROME_PATH")
    if not p:
        return None
    # 文件 → 取所在目录；目录 → 直接用。所有子进程命令行都含该目录路径。
    try:
        return p if os.path.isdir(p) else os.path.dirname(p)
    except Exception:
        return p


def kill_stale_firefox():
    """清理本插件残留的 ruyiPage Firefox 进程（含被孤立的 content 子进程）。

    之前的运行若崩溃/超时，Firefox 不会被回收，残留进程占用 profile/端口，
    导致下次「启动后连不上端口 / Browsing context got discarded」。
    优先按 Firefox 二进制所在路径匹配（能一网打尽主进程+子进程），再兜 ruyipage_ 标记。
    服务器上该二进制只有本插件在用，不会误伤用户自己的浏览器。
    """
    import subprocess
    token = _firefox_match_token()
    try:
        if os.name == "nt":
            conds = ["$_.CommandLine -like '*ruyipage_*'"]
            if token:
                safe = token.replace("'", "''")
                conds.append("$_.CommandLine -like '*" + safe + "*'")
            ps = (
                "Get-CimInstance Win32_Process -Filter \"Name='firefox.exe'\" | "
                "Where-Object { " + " -or ".join(conds) + " } | "
                "ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }"
            )
            subprocess.run(["powershell", "-NoProfile", "-Command", ps], timeout=25, capture_output=True)
        else:
            if token:
                subprocess.run(["pkill", "-9", "-f", token], timeout=25, capture_output=True)
            subprocess.run(["pkill", "-9", "-f", "ruyipage_"], timeout=25, capture_output=True)
    except Exception as e:
        log(f"[cleanup] 清理旧 Firefox 进程忽略异常: {e}")


def build_page():
    from ruyipage import FirefoxOptions, FirefoxPage

    opts = FirefoxOptions()
    chrome_path = env("BALALA_CHROME_PATH")  # 可指向 firefox(.exe)，或其所在目录；留空用 ruyipage 运行时
    if chrome_path and os.path.isdir(chrome_path):
        # 填的是目录，递归找 firefox 可执行文件
        for root, _dirs, files in os.walk(chrome_path):
            for fn in ("firefox", "firefox.exe"):
                if fn in files:
                    chrome_path = os.path.join(root, fn)
                    break
            else:
                continue
            break
    if chrome_path:
        log(f"[browser] 使用 Firefox: {chrome_path}")
        opts.set_browser_path(chrome_path)

    # 正常窗口大小，避免被判为自动化
    try:
        opts.set_window_size(1920, 1080)
    except Exception:
        pass

    # 关键稳定性：以 root / 受限容器运行时，Firefox 内容进程沙箱会初始化失败，
    # BiDi 一导航内容进程就被杀（报「Browsing context got discarded / 连接已断开」）。
    # 把内容沙箱级别设为 0 即可彻底解决（实测确认）。
    stable_prefs = {
        "security.sandbox.content.level": 0,
        "security.sandbox.content.tempDirSuffix": "",
        "media.cubeb.sandbox": False,
        "layers.acceleration.disabled": True,  # 无头服务器无 GPU，关硬件加速更稳
    }
    for _k, _v in stable_prefs.items():
        try:
            opts.set_pref(_k, _v)
        except Exception:
            pass
    os.environ.setdefault("MOZ_DISABLE_CONTENT_SANDBOX", "1")
    os.environ.setdefault("MOZ_DISABLE_GMP_SANDBOX", "1")

    if env("BALALA_HEADLESS", "1") == "1":
        opts.headless(True)
    else:
        opts.headless(False)

    return FirefoxPage(opts)


def find(page, selectors, timeout=15):
    """按多个选择器顺序查找元素，找到即返回，超时返回 None。"""
    if isinstance(selectors, str):
        selectors = [selectors]
    deadline = time.time() + timeout
    while time.time() < deadline:
        for sel in selectors:
            try:
                ele = page.ele(sel, timeout=1)
            except TypeError:
                # 某些版本 ele 不接受 timeout
                try:
                    ele = page.ele(sel)
                except Exception:
                    ele = None
            except Exception:
                ele = None
            if ele:
                return ele
        time.sleep(0.5)
    return None


def run_js(page, script, **kw):
    """执行 JS。ruyiPage 以表达式方式求值，顶层 return 非法，
    因此统一包进 IIFE，让脚本内可自由使用 return。"""
    wrapped = "(function(){" + script + "})()"
    try:
        return page.run_js(wrapped, **kw)
    except TypeError:
        return page.run_js(wrapped)
    except Exception as e:
        log(f"[js] 执行失败: {e}")
        return None


def try_cf(page):
    """若库提供 Cloudflare 挑战处理（等待托管挑战自动完成），尽力调用一次。"""
    fn = getattr(page, "handle_cloudflare_challenge", None)
    if callable(fn):
        try:
            log("[login] 处理 Cloudflare 挑战…")
            fn()
        except Exception as e:
            log(f"[login] handle_cloudflare_challenge 忽略异常: {e}")


def do_login(page, base_url, email, password, timeout):
    login_url = base_url + "/auth/login"
    log(f"[login] 打开 {login_url}")
    page.get(login_url)
    time.sleep(2)
    try_cf(page)

    email_ele = find(page, ["#email", "css:input[type=email]"], timeout)
    pwd_ele = find(page, ["#password", "css:input[type=password]"], 5)
    if not email_ele or not pwd_ele:
        raise RuntimeError("未定位到登录表单（#email / #password）。")

    log("[login] 填写账号密码")
    try:
        email_ele.input(email, clear=True)
        pwd_ele.input(password, clear=True)
    except TypeError:
        email_ele.clear(); email_ele.input(email)
        pwd_ele.clear(); pwd_ele.input(password)

    # 等待 Turnstile 隐藏域被自动填充（反检测浏览器下托管挑战常能静默通过）
    _wait_turnstile(page, timeout)

    btn = find(page, ["#login_submit", "text:登 录", "text:登录", "@type=submit"], 5)
    if not btn:
        raise RuntimeError("未找到登录按钮 #login_submit。")
    log("[login] 提交登录")
    try:
        btn.click_self()
    except Exception:
        btn.click()

    deadline = time.time() + timeout
    while time.time() < deadline:
        if _is_logged_in(page, base_url):
            log("[login] 登录成功")
            return True
        time.sleep(1)
    raise RuntimeError("登录超时：可能是账号密码错误、开启了二次验证，或被 Turnstile 拦截。")


def _wait_turnstile(page, timeout):
    present = run_js(page, "return !!document.querySelector('[name=cf-turnstile-response]');")
    if not present:
        return True
    log("[login] 检测到 Turnstile，等待自动校验…")
    try_cf(page)
    deadline = time.time() + min(timeout, 40)
    js_val = "var e=document.querySelector('[name=cf-turnstile-response]');return e?e.value:'';"
    while time.time() < deadline:
        if (run_js(page, js_val) or "").strip():
            log("[login] Turnstile 已通过")
            return True
        time.sleep(1)
    log("[login] 警告：未拿到 Turnstile token，提交可能失败")
    return False


def _is_logged_in(page, base_url):
    try:
        url = page.url or ""
        if "/auth/login" in url:
            return False
        if "/user" in url:
            return True
        page.get(base_url + "/user")
        time.sleep(1)
        return "/auth/login" not in (page.url or "")
    except Exception:
        return False


def apply_cookie(page, base_url, cookie):
    """用户手动登录后提供的 cookie，直接注入以绕过验证码。"""
    log("[cookie] 注入会话 cookie")
    page.get(base_url + "/auth/login")
    # 优先用库的 set_cookies（若有），否则退回 document.cookie（仅非 httpOnly 有效）
    setter = getattr(page, "set_cookies", None) or getattr(page, "set_cookie", None)
    for part in cookie.split(";"):
        part = part.strip()
        if "=" not in part:
            continue
        name, val = part.split("=", 1)
        name, val = name.strip(), val.strip()
        done = False
        if callable(setter):
            try:
                setter({"name": name, "value": val, "domain": _host(base_url)})
                done = True
            except Exception:
                done = False
        if not done:
            run_js(page, f"document.cookie={json.dumps(name + '=' + val + '; path=/')};")


def _host(url):
    return re.sub(r"^https?://", "", url).split("/")[0]


def _wait_ready(page, timeout=15):
    """等待页面 document 就绪，避免过早读取导致 documentElement 为 null。"""
    deadline = time.time() + timeout
    while time.time() < deadline:
        state = run_js(page, "return document.readyState;")
        if state in ("interactive", "complete"):
            return True
        time.sleep(0.5)
    return False


def _get_html(page):
    """稳妥获取页面 HTML：page.html 在导航中途可能抛错，退回 run_js。"""
    html = run_js(page, "return document.documentElement ? document.documentElement.outerHTML : '';")
    if html:
        return html
    try:
        return page.html or ""
    except Exception:
        return ""


def fetch_subscribe(page, base_url, timeout):
    log("[fetch] 打开 /user 提取订阅链接")
    for attempt in range(3):
        try:
            page.get(base_url + "/user")
            break
        except Exception as e:
            log(f"[fetch] 导航 /user 第{attempt + 1}次异常: {e}")
            time.sleep(1.5)
    _wait_ready(page, 15)
    time.sleep(1.5)

    js = r"""
    const items = [];
    document.querySelectorAll('[data-clipboard-text]').forEach(el => {
        const url = el.getAttribute('data-clipboard-text') || '';
        if (/^https?:\/\//.test(url) && (url.indexOf('token=') > -1 || url.indexOf('/link/') > -1)) {
            const label = (el.textContent || '').replace(/\s+/g, ' ').trim();
            const cls = el.className || '';
            items.push({url: url, label: label, cls: cls});
        }
    });
    return JSON.stringify(items);
    """
    subs = []
    try:
        raw = run_js(page, js) or "[]"
        for it in json.loads(raw):
            url = (it.get("url") or "").strip()
            if not url:
                continue
            client = ""
            mvia = re.search(r"[?&]via=([A-Za-z0-9_]+)", url)
            if mvia:
                client = mvia.group(1)
            else:
                mcls = re.search(r"btn-([a-z0-9]+)", it.get("cls", ""))
                if mcls:
                    client = mcls.group(1)
            subs.append({"label": it.get("label", ""), "client": client, "url": url})
    except Exception as e:
        log(f"[fetch] 解析 data-clipboard-text 失败: {e}")

    seen, uniq = set(), []
    for s in subs:
        if s["url"] not in seen:
            seen.add(s["url"])
            uniq.append(s)
    subs = uniq

    if not subs:
        m = SUB_LINK_RE.search(_get_html(page))
        if m:
            subs = [{"label": "", "client": "", "url": m.group(0)}]

    if not subs:
        return None, []

    primary = next(
        (s["url"] for s in subs if "v2ray" in (s["client"] or "").lower()),
        subs[0]["url"],
    )
    log(f"[fetch] 命中 {len(subs)} 条订阅链接")
    return primary, subs


def main():
    base_url = env("BALALA_URL", "https://balala.io").rstrip("/")
    email = env("BALALA_EMAIL")
    password = env("BALALA_PASSWORD")
    cookie = env("BALALA_COOKIE")
    timeout = int(env("BALALA_TIMEOUT", "60"))

    if not cookie and (not email or not password):
        out({"success": False, "error": "缺少登录凭据：需 BALALA_COOKIE，或 BALALA_EMAIL+BALALA_PASSWORD"})
        return 1

    # 启动前先清理上次残留的 ruyiPage Firefox 进程，避免占用导致连不上
    kill_stale_firefox()

    page = None
    try:
        page = build_page()

        if cookie:
            apply_cookie(page, base_url, cookie)
            source = "cookie"
            if not _is_logged_in(page, base_url):
                out({"success": False, "error": "提供的 cookie 无效或已过期"})
                return 1
        else:
            do_login(page, base_url, email, password, timeout)
            source = "login"

        url, subs = fetch_subscribe(page, base_url, timeout)
        if url:
            out({"success": True, "subscribe_url": url, "subscriptions": subs, "source": source})
            return 0
        out({"success": False, "error": "已登录但未在 /user 页面找到订阅链接，页面结构可能已变化。"})
        return 1
    except Exception as e:
        out({"success": False, "error": str(e)})
        return 1
    finally:
        if page is not None:
            try:
                page.quit()
            except Exception:
                pass
        # 兜底：即使 quit() 因 BiDi 断连失败，也强制清掉残留进程，不给下次留僵尸
        kill_stale_firefox()


if __name__ == "__main__":
    sys.exit(main())
