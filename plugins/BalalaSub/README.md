# Balala 订阅抓取插件 (balala_sub)

基于 [ruyiPage](https://github.com/LoseNine/ruyipage)（Firefox + WebDriver BiDi 反检测框架）
自动登录 `balala.io`，抓取账号的**订阅链接 URL**，结果写入缓存，供 Artisan 命令或 API 读取。

> **站点实测结论（2026-10 核对）**：balala.io 是 **SSPanel-UIM**（metron 主题，面板「gw树洞」），
> 并非 V2board/Xboard。脚本已按真实结构编写：
> - 登录页 `/auth/login`（服务端渲染），POST 字段 `email` / `passwd` / `remember_me`
> - 鉴权用 PHP session cookie（无 JS token）
> - 订阅链接内嵌在 `/user` 页面复制按钮的 `data-clipboard-text`，形如 `https://host/<rand>?token=...&via=<client>`

## ⚠️ 登录开启了 Cloudflare Turnstile 验证码

该站 `loginConfig.switch.turnstile = true`，登录需人机验证。实测中 **DrissionPage 的 Chromium
无头模式拿不到 Turnstile token**（被识别为自动化），因此改用 **ruyiPage**：它以反检测 Firefox
呈现为「正常浏览器」，托管型 Turnstile 可像真人浏览器一样**自动放行**。

- **自动登录**：脚本填入账号密码，等待 Turnstile 隐藏域被自动填充后提交。
- **Cookie 兜底（最稳）**：若自动登录仍被拦，在浏览器手动登录一次，把请求头里的整串 `Cookie`
  粘到插件配置的「会话 Cookie」字段，脚本直接带 Cookie 抓取。**Cookie 优先于账号密码。**

> 本插件依赖浏览器的反检测特性让托管挑战自动放行，不会求解交互式验证码。

## 工作方式

```
Artisan 命令 (PHP)  ──env──▶  python/fetch_sub.py (ruyiPage 驱动反检测 Firefox)
      │                               │
      │                   Cookie 或 账号密码登录 → 读 /user 抓 data-clipboard-text 链接
      ▼                               ▼
  写入缓存  ◀────── 一行 JSON (stdout) ──┘
```

## 前置依赖 ⚠️

抓取脚本需要在**实际执行命令的那个环境**里具备：

- Python 3.8+
- `pip install -r plugins/BalalaSub/python/requirements.txt`（即 ruyiPage）
### 环境自动安装 🚀

**安装插件时会自动在后台执行 `pip install ruyiPage`**（见 `Plugin::install()`），
日志写到 `storage/logs/balala_sub_setup.log`。因此新装插件通常无需手动装 Python 依赖。

> 前提：容器/主机已有 Python 3 + pip。Alpine 版 Xboard 镜像默认不含 Python，
> 需自行在镜像里加 `apk add python3 py3-pip`，否则自动安装会在日志里记失败。

手动安装 / 重装环境（已安装过的插件、或需要同时下载 Firefox 运行时）：

```bash
php artisan balala-sub:setup            # 仅 pip 安装 ruyiPage
php artisan balala-sub:setup --firefox  # 顺带下载 Firefox 运行时（glibc 环境才可用）
```

### Firefox 内核

一个可用的 Firefox 内核，三选一：
  1. **后台上传**（推荐）：插件配置页「Firefox 内核」字段点「上传内核」，上传 ruyiPage 配套
     Firefox 包，服务器自动解压到 `storage/app/balala_sub/firefox/` 并填好 `chrome_path`。
     - Windows 用 `.zip`，Linux 服务器用 `.tar.xz` / `.tar.gz`（容器内需有 `tar`，xz 需 `xz-utils`）
  2. `python -m ruyipage install` 下载官方运行时；
  3. 手动把 `chrome_path` 填成已有的 firefox 可执行文件绝对路径。

> **后台上传大文件需放宽 PHP 上传上限**（Firefox 包约 70~120MB），否则会报
> `The POST data is too large.`。本仓库已在 `.docker/php/zz-xboard.ini` 设置
> `upload_max_filesize=200M` / `post_max_size=210M`，**改完需重建/重启容器并
> `php artisan octane:reload` 生效**。非 Docker 部署请在自己的 php.ini 里同样调大。

> Xboard 常跑在 Docker(Octane) 容器里。方案：① 在容器镜像里加装 Python + ruyiPage + Firefox 运行时；
> ② 在宿主机/单独采集机上跑该命令，并把 `python_bin` 指到对应 Python。

## 安装 / 启用

1. 把 `BalalaSub/` 放到 `plugins/` 目录下（已就位）。
2. 后台插件管理里安装并启用「Balala 订阅抓取」，或命令行启用。
3. 在插件配置里填写：
   - **站点地址** `site_url`（默认 `https://balala.io`）
   - **登录邮箱** `email`
   - **登录密码** `password`
   - **Python 可执行文件** `python_bin`（如 `python3` 或绝对路径）
   - **Chrome 路径** `chrome_path`（留空自动探测）
   - **无头模式** `headless`（服务器环境开启）

## 使用

```bash
# 抓取并写入缓存
php artisan balala-sub:fetch

# 仅输出一行 JSON（便于脚本调用），不写缓存
php artisan balala-sub:fetch --json --no-cache

# 临时覆盖账号
php artisan balala-sub:fetch --email=a@b.com --password=secret
```

API（需后台管理员鉴权，`admin` 中间件）：

```
GET  /api/v1/balala-sub/latest    # 读取最近一次缓存结果
POST /api/v1/balala-sub/refresh   # 触发一次抓取并返回结果
```

定时抓取：可在 `Plugin.php` 的 `schedule()` 里加
`$schedule->command('balala-sub:fetch')->hourly();`

## 本地单独调试 Python

账号密码方式（有头，方便观察 Turnstile 是否放行）：

```bash
BALALA_URL=https://balala.io \
BALALA_EMAIL=a@b.com \
BALALA_PASSWORD=secret \
BALALA_HEADLESS=0 \
python plugins/BalalaSub/python/fetch_sub.py
```

Cookie 方式（绕过验证码，最稳）：

```bash
BALALA_URL=https://balala.io \
BALALA_COOKIE='sid=xxxx; other=yyyy' \
python plugins/BalalaSub/python/fetch_sub.py
```

## 环境变量一览

| 变量 | 说明 |
|---|---|
| `BALALA_URL` | 站点根地址，默认 `https://balala.io` |
| `BALALA_EMAIL` / `BALALA_PASSWORD` | 账号密码登录 |
| `BALALA_COOKIE` | 已登录会话 Cookie（优先于账号密码，绕过验证码） |
| `BALALA_HEADLESS` | `1` 无头 / `0` 有界面 |
| `BALALA_CHROME_PATH` | firefox.exe 路径，留空用 ruyipage 下载的运行时 |
| `BALALA_TIMEOUT` | 超时秒数，默认 60 |
