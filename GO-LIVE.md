# Xboard (Go) 上线文档

适用范围：把 Go 版面板上线，包括**全新安装**和**替换正在运行的 PHP 版**两种情况。

- 安装命令与 systemd / nginx 细节见 [DEPLOY.md](DEPLOY.md)。
- 功能清单、与 PHP 的行为差异见 [PROGRESS.md](PROGRESS.md)。
- 本文回答的是：**能不能上、上之前要查什么、怎么切、怎么验、出问题怎么退。**

---

## 0. 先看结论

Go 版是单个二进制（约 15 MB），内嵌管理端、默认用户端主题、核心插件清单和建表 SQL；服务器只需要 MySQL / MariaDB 和 Redis，不需要 PHP、Node。

### 验证状态（上线前请对照）

| 项目 | 状态 | 依据 |
|---|---|---|
| 干净服务器安装（空目录 + 发布包 + 不存在的库） | 已验证 | `scripts/wsl/fresh_deploy_test.sh`，27 项通过 |
| 全新安装上的完整用户旅程 | 已验证 | 同上，65 项通过 |
| 对已由 PHP 迁移过的库执行 `migrate` | 已验证（空操作，表数与行数不变） | 开发期 WSL 演练，见 PROGRESS.md |
| MariaDB 10.11 | 已验证 | WSL 演练环境 |
| **MySQL 8** | **未验证** | 有代码路径（JSON 列还原），没有跑过 |
| **真实支付网关**（EPay、支付宝当面付等） | **未验证** | 只有签名往返与桩测试，没接过真实商户 |
| **真实 SMTP、真实 Telegram Bot** | **未验证** | 只有逻辑层与桩测试 |
| **真实节点的 WebSocket 联调** | **未验证** | 同上 |
| 回滚（Go → PHP） | **未演练** | 本文给出方案，上线前请在预发环境走一遍 |
| 浏览器端逐页验收 | 部分 | 开发期人工看过主要页面，没有自动化 UI 测试 |

> **建议：上线前至少在预发环境（独立库 + 独立 Redis DB）用真实的支付商户、SMTP 和一个真实节点各跑通一次。** 上表中标"未验证"的每一项，都是上线后用户会直接碰到的功能。

---

## 1. 上线前必须决定 / 检查的事（Go / No-Go）

逐项过一遍，任何一项答案是"不确定"，先不要切。

### 1.1 数据库必须是 MySQL / MariaDB

Go 版**不支持 SQLite、PostgreSQL**（大量使用 `JSON_CONTAINS`、`FROM_UNIXTIME` 等 MySQL 语法）。

> 注意：PHP 版的默认 Docker 部署使用 SQLite（`.env` 里 `DB_CONNECTION=sqlite`，库文件在 `.docker/.data/database.sqlite`）。如果生产环境是这种安装，**不能直接切换**，需要先把数据迁到 MySQL，再按本文操作。

```bash
grep -E '^DB_CONNECTION' .env     # PHP 版的 .env；要求是 mysql
```

### 1.2 支付方式必须都在内置网关范围内

Go 内置 7 个网关，`v2_payment.payment` 字段必须是下面之一：

`EPay`、`AlipayF2F`、`Bepusdt`、`MGate`、`BTCPay`、`CoinPayments`、`Coinbase`

```sql
SELECT payment, COUNT(*) AS n, SUM(enable) AS enabled FROM v2_payment GROUP BY payment;
```

出现上面以外的值（例如 Stripe 或第三方 PHP 插件），该支付方式在 Go 版下单时会返回"payment gateway ... is not implemented"。这类支付要么先移植，要么上线前停用。

### 1.3 PHP 插件与主题不会被执行

Go 版**不加载 PHP 代码**。插件、主题只维护数据库记录。已原生实现的：核心支付网关（见 1.2）、Telegram、BalalaSub。其他插件的钩子和定时任务不会运行。

```sql
SELECT code, name, is_enabled FROM v2_plugins;
```

除 `Telegram`、`BalalaSub` 和上面 7 个支付插件之外，如有已启用的插件，请确认它的功能不是你业务依赖的。

### 1.4 品牌与默认值（待你确认）

- 站点名称缺省值已从 `XBoard` 改为 `Lets VPN`（影响订阅标题、邮件发件名）；可在后台修改。已有 `app_name` 设置的库不受影响。
- 管理端界面已重新设计为 Lets VPN 风格并重新构建。**这是开发过程中额外做的改动，不属于"沿用原前端"的范围**，上线前请确认保留还是回退。

### 1.5 代码状态

- Go 版已作为仓库内容提交（分支 `go-rewrite`）；替换前的 PHP 版完整保存在标签 `php-final`，随时可以 `git checkout php-final` 找回。
- 上线前请给要发布的提交打版本标签（如 `v1.0.0`），再用同样的版本号构建，线上出问题时才能对应到具体代码。
- 用 `VERSION=x.y.z bash scripts/build-release.sh` 构建，版本号会写进二进制（`./xboard version` 可查）。

### 1.6 不要把开发环境的东西带上线

- 开发库里有测试账号、测试节点、测试管理员密码，**生产不要复用开发库**。
- 本地开发用的 `.env`（监听 `0.0.0.0`、连开发库）**不要**拷到服务器。发布包只带 `.env.example`。
- `storage/`、`xboard.exe`、`dist/` 不在发布包里，也不需要上传。

---

## 2. 构建与交付物

前置：需要 Node.js / npm（脚本会自动构建管理端前端；已构建过可设 `SKIP_ADMIN_BUILD=1` 跳过）。

```bash
VERSION=1.0.0 bash scripts/build-release.sh          # 默认 linux/amd64 + linux/arm64
TARGETS="linux/amd64" VERSION=1.0.0 bash scripts/build-release.sh
```

脚本依次执行：同步前端资源到 `internal/web/dist` → `gofmt` / `go vet` / `go test` → 交叉编译（`CGO_ENABLED=0`）→ 打包到 `dist/xboard-<版本>-linux-<架构>.tar.gz`。**检查不通过会中止，不会产出包。**

发布包内容：

```
xboard                      # 二进制（0755）
.env.example
DEPLOY.md
GO-LIVE.md                  # 本文
deploy/xboard.service       # systemd 单元
deploy/nginx.conf.example   # 反向代理示例（含 /ws）
```

> 在 Windows / NTFS 上构建时文件没有可执行位，脚本会显式写入权限，所以请用脚本打包，不要手工压缩。

### 上传到 Cloudflare R2（手动）

构建**不会**自动上传。测试通过后，由你手动执行 `scripts/upload-r2.sh`。

一次性配置：

1. 在 Cloudflare 控制台创建 R2 API 令牌（R2 → 管理 R2 API 令牌，权限选「对象读和写」，可限定到这一个桶），记下 Access Key ID 和 Secret Access Key（Secret 只显示一次）。
2. `cp scripts/r2.env.example scripts/.r2.env`，填入桶名和这两个值。该文件已被 `.gitignore` 忽略，不要提交。账号 ID 对应的端点是 `https://<账号ID>.r2.cloudflarestorage.com`。

每次发布：

```bash
bash scripts/build-release.sh
bash scripts/upload-r2.sh check      # 测试：凭据、桶、读写权限（只写入并删除一个探测文件）
bash scripts/upload-r2.sh upload     # 先校验 SHA256SUMS，列出将上传的文件并让你确认，上传后回查大小与 MD5
bash scripts/upload-r2.sh list       # 查看桶里已有的文件
```

说明：

- 文件默认放在桶内 `xboard/` 目录，可用 `R2_PREFIX` 修改；同名文件会被覆盖。
- 发布包先传，`SHA256SUMS` 最后传，保证校验文件指向的对象一定已经存在。
- 本地文件与 `SHA256SUMS` 对不上时直接中止，不会上传任何内容。
- 非交互环境（CI）需加 `-y` 跳过确认。
- R2 的 S3 端点是私有的，不能直接下载。服务器取包需要给桶绑定公开域名（或开启 r2.dev），或使用带签名的链接。
- 该脚本依赖 curl 7.75 或更高版本（`--aws-sigv4` 签名）。

---

## 3. 环境准备

| 组件 | 要求 |
|---|---|
| 系统 | Linux amd64 / arm64 |
| 数据库 | MySQL 5.7+ / 8.x 或 MariaDB 10.4+，字符集 `utf8mb4` |
| Redis | 5.0+ |
| 反向代理 | nginx / Caddy，必须放行 `/ws` 的 Upgrade 头 |
| 可选 | `sing-box`（真连测速；也可在后台「在线下载」）、`python3`（BalalaSub 抓取）、`xz-utils`（上传 `.tar.xz`） |

建库与账号：

```sql
CREATE DATABASE xboard CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'xboard'@'127.0.0.1' IDENTIFIED BY '<强密码>';
GRANT ALL ON xboard.* TO 'xboard'@'127.0.0.1';
```

网络与端口：

- Go 版 HTTP 和 WebSocket **共用同一个监听地址**（`LISTEN`），WebSocket 路径是 `/ws`。PHP 的 Docker 方案是 Caddy 在 7001 上把 `/ws` 转给 8076 的 ws-server；Go 版没有 8076 这个独立端口，**不需要再开放 8076**。
- 放在反向代理后面时 `LISTEN=127.0.0.1:7001`，只让代理访问。直接对外才用 `0.0.0.0`。
- `TRUSTED_PROXIES` 缺省信任回环和内网网段；代理在别的网段时必须显式配置，否则真实 IP（注册限流、日志）会取错。

---

## 4. 路径 A：全新安装

```bash
sudo useradd -r -m -d /opt/xboard xboard
sudo tar -xzf xboard-<版本>-linux-amd64.tar.gz -C /tmp
sudo cp /tmp/xboard-<版本>-linux-amd64/{xboard,.env.example} /opt/xboard/
sudo mkdir -p /opt/xboard/deploy && sudo cp /tmp/xboard-<版本>-linux-amd64/deploy/* /opt/xboard/deploy/
cd /opt/xboard
sudo -u xboard cp .env.example .env
./xboard keygen                       # 把输出写进 .env 的 APP_KEY
sudo -u xboard vi .env                # 填 DB_* / REDIS_* / LISTEN / APP_KEY
sudo -u xboard ./xboard install admin@example.com    # 建表 + 创建管理员，打印后台地址与随机密码
sudo cp deploy/xboard.service /etc/systemd/system/
sudo systemctl daemon-reload && sudo systemctl enable --now xboard
curl -s http://127.0.0.1:7001/healthz
```

随后**立刻**：

1. 记下 `install` 输出的管理员密码，登录后改掉；
2. 后台「站点配置」先填 `app_url`（订阅地址、支付回调、邮件链接都依赖它；全新库没填时保存支付方式会失败）；
3. 按第 6 节的清单配置。

---

## 5. 路径 B：从 PHP 版切换

核心事实：**Go 版直接读写 PHP 版的同一个 MySQL 库，不需要导数据。**

- 表结构一致，`./xboard migrate` 对已有库是空操作，不会改动任何已存在的表。
- 登录 token 兼容（Sanctum `id|token`），旧密码算法（md5 / sha256 / 带盐 / `$2y$`）兼容，**用户不会被踢下线**。
- 订阅 token、用户 UUID 不变，用户手里的订阅链接继续有效。
- 订阅路径不变：加密订阅 `/{subscribe_path}-enc/{token}`；明文订阅 `/{subscribe_path}/{token}` 与 PHP 版一样返回 403。
- 节点对接路径不变：`/api/v1/server/UniProxy/*`（用后台的 `server_token`），节点 agent 不需要改。
- 支付回调路径不变：`/api/v1/guest/payment/notify/{method}/{uuid}`。
- 端口可以沿用 7001，`/ws` 与 HTTP 共用，对反向代理和节点来说是原地替换。

### B0. 切换前预检（至少提前一天）

| # | 检查 | 命令 / 方法 | 通过标准 |
|---|---|---|---|
| 1 | 数据库类型 | 见 1.1 | MySQL / MariaDB |
| 2 | 支付网关 | 见 1.2 | 全部在内置 7 个里 |
| 3 | 插件 | 见 1.3 | 无业务依赖的未移植插件 |
| 4 | 取得 PHP 的 `APP_KEY` | 看 PHP 的 `.env` | 原样（含 `base64:` 前缀）写进 Go 的 `.env`；**不要换**，后台默认路径（crc32b）和加密订阅依赖它 |
| 5 | 预发演练 | 用生产库的备份还原到预发库，在预发跑完 B3 与第 7 节 | 全部通过 |
| 6 | 回滚预案 | 第 B4 节 | 已在预发走通 |
| 7 | 通知 | 提前告知用户有短暂维护窗口 | — |

### B1. 备份（必须）

```bash
# 数据库（一致性快照）
mysqldump --single-transaction --routines -u<用户> -p <库名> > xboard-$(date +%F-%H%M).sql

# PHP 版的配置与上传文件
cp .env .env.bak-$(date +%F)
tar czf php-files-$(date +%F).tgz storage public/uploads plugins 2>/dev/null
```

把备份放到**另一台机器**，并实际试一次还原。

### B2. 部署 Go 但先不启动对外服务

```bash
# 在服务器上按路径 A 的前几步放好二进制与 deploy/ 文件，然后配置 .env：
#   DB_*          与 PHP 相同的库
#   APP_KEY       与 PHP 完全一致
#   REDIS_HOST/PORT  指向一个 Redis —— 见下方说明
#   REDIS_DB      用一个与 PHP 不同的库号（例如 PHP 用 0，Go 用 5），避免互相读到对方的键
#   LISTEN        127.0.0.1:7001（沿用 7001 时先确保 PHP 已停）
./xboard version
```

**关于 Redis：** Laravel 的缓存键带前缀，Go 的键名不带，两边不互通，所以不要让两个版本同时写同一个 Redis DB。切换后在线状态、缓存、国家快照等会重新生成（节点按推送间隔上报后恢复，预期几分钟内），这是**预期行为**，不是故障。PHP 的 Docker 方案里 Redis 在容器内（`/data`），Go 版需要一个独立可用的 Redis。

### B3. 切换（维护窗口，尽量短）

1. **排空 PHP 队列**：在 Horizon 面板确认没有待处理任务（邮件、订单处理）。Go 的邮件队列在进程内，**不会接手 PHP 队列里未发的任务**。
2. **停 PHP**：Docker 部署执行 `docker compose stop`。PHP 的定时任务是跑在 Octane 容器里的（带 Redis 锁），停容器即停调度；不要留任何还在跑 `schedule:run` 的进程或 cron，否则两边会重复执行定时任务。需要停的进程：Octane、Horizon、ws-server，以及占用同一端口的旧 Caddy。
3. **启动 Go**：
   ```bash
   sudo -u xboard ./xboard migrate        # 预期：已有表不动（空操作）
   sudo systemctl enable --now xboard
   journalctl -u xboard -f                # 应看到 scheduler started，无 fatal
   curl -s http://127.0.0.1:7001/healthz  # {"database":"up","redis":"up","status":"ok",...}
   ```
4. **接入流量**：沿用同一个 7001 与同一个反向代理即可；换机器的话再改 DNS 或上游。
5. **立即执行第 7 节冒烟验证。** 任何一项失败，直接走 B4 回滚，不要在窗口内排查。

> 风险提示：窗口期内节点上报的流量可能丢失，窗口越短越好；这也是要先在预发走通全流程的原因。

### B4. 回滚（Go → PHP）

触发条件：冒烟验证失败、healthz 异常、登录或订阅大面积失败。

1. `sudo systemctl stop xboard`
2. 启动 PHP（`docker compose start`，或你原来的启动方式），保持 PHP 版使用原来的 Redis 与 `.env`。
3. 确认 PHP 的订阅、登录、节点上报恢复。
4. **数据库不需要还原**：两个版本共用同一套表结构，Go 期间写入的订单、用户、节点数据 PHP 都能读。只有在 Go 运行期间发生了你不希望保留的数据变更时，才用 B1 的备份还原。
5. Go 为新功能写入的少量设置项（例如 `country_refresh_time`）PHP 会忽略，无需清理。

不能回滚的部分：Go 进程内队列里未发出的邮件（进程停止即丢失）。

> 本回滚方案**尚未演练**，请在预发走一遍。

---

## 6. 上线后必配项（后台）

这些设置缺了会直接影响用户。切换自 PHP 的库里大部分已有值，请**逐项核对而不是假设**。

| 项 | 设置 / 位置 | 说明 |
|---|---|---|
| 站点地址 | `app_url` | 订阅地址、支付回调、邮件链接的基础地址。必须是用户和支付商能访问到的外网地址 |
| 订阅地址 | `subscribe_url`、`subscribe_path` | 多域名时逗号分隔；`subscribe_path` 缺省 `s` |
| 后台路径 | `secure_path` | 缺省是 `APP_KEY` 的 crc32b；建议改成自己的随机路径 |
| 节点通信 | `server_token`、`server_pull_interval`、`server_push_interval` | 节点 agent 用同一个 token，**不要改**，除非同步更新所有节点 |
| WebSocket | `server_ws_enable`、`server_ws_url`、`client_ws_enable` | `client_ws_enable` 关闭时后台改动不会推送给客户端 |
| 权限组 / 套餐 / 节点 | 后台「权限组」「套餐」「节点」 | **每个套餐必须设权限组，节点也要设权限组**，否则用户订阅里看不到节点。后台套餐页与节点页会提示"未设权限组" |
| 注册策略 | `stop_register`、`email_verify`、`invite_force`、`register_limit_by_ip_enable` | 按运营需要 |
| 体验套餐 | `try_out_plan_id`、`try_out_hour` | 设备首次注册发放试用 |
| 网页登录 | `user_web_login_enable` | 关闭后非管理员只能用 App 登录（无 `device_id` 视为网页端） |
| 邮件 | 后台「邮件」设置 | 配好后发一封测试邮件 |
| 支付 | 后台「支付方式」 | 每个方式的 `notify_domain`（回调域名）要能被支付商访问 |
| Telegram | 后台 Telegram 机器人设置 | 配 Bot Token 后确认 webhook 生效 |
| 流量重置 | `reset_traffic_method` | 套餐未单独设置时跟随它 |
| 测速 | `node_probe_mode`、`node_probe_singbox_path`、`node_probe_singbox_auto_download`、`node_probe_singbox_version` | `singbox` 模式需要 sing-box：点后台「在线下载」，或手动安装并填路径。服务器在国内连不上 GitHub 时，设环境变量 `XBOARD_SINGBOX_DOWNLOAD_BASE` 指向镜像（路径需为 `<地址>/v<版本>/<包名>`） |
| 国家快照 | `country_refresh_time` | 每日自动更新时刻，缺省 04:00，改后无需重启 |
| 国家角标 | `country_tags` | 客户端国家列表的角标，如 `HK:推荐,JP:高速`；只写国家码时显示「推荐」，缺省为空（不显示）。改后立即生效，无需重启或等快照更新 |
| 已取消订单清理 | `order_cancelled_cleanup_enable`、`order_cancelled_cleanup_days` | 后台「订单管理」页顶部的卡片里设置；缺省**关闭**，天数缺省 30（范围 1–3650）。打开前先看卡片上的「待清理」数量；删除不可恢复，建议先备份库 |
| 设备数 | `default_device_limit` | 新用户默认设备数 |

---

## 7. 冒烟验证清单

把 `BASE` 换成对外的地址。**每一项都要在切换窗口内做完。**

```bash
BASE=https://panel.example.com

# 1) 健康：数据库与 Redis 都为 up
curl -s $BASE/healthz

# 2) 公共配置可读
curl -s $BASE/api/v1/guest/comm/config

# 3) 登录（用一个真实的普通账号），记下返回的 auth_data
curl -s -X POST $BASE/api/v1/passport/auth/login \
  -H 'Content-Type: application/json' -d '{"email":"<邮箱>","password":"<密码>"}'

# 4) 带登录态访问（旧 token 也应有效，用户不会掉线）
curl -s -H "Authorization: <auth_data>" $BASE/api/v1/user/info
curl -s -H "Authorization: <auth_data>" $BASE/api/v1/user/plan/fetch
curl -s -H "Authorization: <auth_data>" $BASE/api/v1/user/server/fetch

# 5) 加密订阅（返回 {v,alg,n,c,t}）；明文订阅应是 403
curl -s $BASE/<subscribe_path>-enc/<用户token>
curl -s -o /dev/null -w '%{http_code}\n' $BASE/<subscribe_path>/<用户token>

# 6) 节点对接：用真实节点 ID 与 server_token
curl -s "$BASE/api/v1/server/UniProxy/config?token=<server_token>&node_id=<id>&node_type=<type>"

# 7) 支付回调路由是通的：400 / 422 都说明路由到了（404 / 502 才是代理或路由问题）
curl -s -o /dev/null -w '%{http_code}\n' $BASE/api/v1/guest/payment/notify/EPay/<支付方式uuid>

# 8) WebSocket 经反向代理可达：不应是 404 / 502
curl -s -o /dev/null -w '%{http_code}\n' -H 'Connection: Upgrade' -H 'Upgrade: websocket' \
  -H 'Sec-WebSocket-Version: 13' -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' $BASE/ws
```

人工验证（浏览器 / 客户端）：

- [ ] 管理端登录（`/<secure_path>`），首页与用户列表、订单列表能打开
- [ ] 一个真实用户用客户端拉取订阅并连上节点
- [ ] **真实下一笔订单**：下单 → 支付 → 回调 → 套餐开通（用金额最小的套餐）
- [ ] 一封真实邮件（验证码或测试邮件）
- [ ] 节点管理页能看到节点在线状态与流量上报
- [ ] 后台改一个节点名称，在线客户端收到更新（确认推送链路）

---

## 8. 上线后 24 小时观察

| 时间 | 看什么 | 方法 |
|---|---|---|
| 切换后 10 分钟 | 日志无 fatal，调度器启动，节点陆续上报 | `journalctl -u xboard -f` |
| 1 小时 | 订单没有卡在"开通中" | 见下方 SQL |
| 当天 | 定时任务在跑 | 日志与下方 SQL |
| 次日 00:10 之后 | 昨日统计已写入 | 下方 SQL |
| 次日 | 提醒邮件按时发出（11:30） | `v2_mail_log` |

```sql
-- 卡住的订单：状态 1（开通中）持续存在说明开通失败；0 是待支付（正常，超 2 小时会被自动取消）
SELECT status, COUNT(*) FROM v2_order GROUP BY status;
SELECT trade_no, status, FROM_UNIXTIME(updated_at) FROM v2_order WHERE status = 1 ORDER BY updated_at;

-- 统计是否按日写入（xboard:statistics 每日 00:10）
SELECT FROM_UNIXTIME(MAX(record_at)) FROM v2_stat WHERE record_type = 'd';

-- 邮件发送失败
SELECT FROM_UNIXTIME(created_at) AS at, email, template_name, error FROM v2_mail_log WHERE error IS NOT NULL ORDER BY id DESC LIMIT 20;
```

出现以下任一情况，评估是否回滚：登录大面积失败、订阅大面积 4xx/5xx、支付回调持续失败、`healthz` 反复非 200、订单持续卡在开通中。

---

## 9. 日常运维

**备份。** 每天 `mysqldump --single-transaction`，异地保存，定期演练还原。应用本身无状态（上传文件在 `public/uploads`，也要备份）。

**升级。**

```bash
sudo systemctl stop xboard
sudo cp xboard /opt/xboard/xboard              # 覆盖二进制
sudo -u xboard /opt/xboard/xboard migrate      # 只补缺失的表，不会给已有表加列
sudo systemctl start xboard
```

升级前先备份，并保留上一个版本的二进制以便回退。

**日志。** 输出到 stdout，由 systemd 收集：`journalctl -u xboard -f`。排障时可临时设 `XBOARD_DEBUG=1`（gin 调试日志）、`XBOARD_SQL_DEBUG=1`（打印全部 SQL），用完记得关。

**多实例。** 调度器默认随 `serve` 启动，任务用 Redis 锁去重，多实例不会重复执行。想只让一台跑调度，其余设 `XBOARD_SCHEDULER=0`；也可以单独 `./xboard schedule`。

**重启的影响：**

- 进程内邮件队列里未发送的邮件会丢失（重启前确认队列已空，或接受这个风险）。
- 调度器**没有补跑**：到点时服务没运行，那次任务就跳过（例如每日国家快照会滞后到下一次触发或手动更新）。
- WebSocket 连接会断开，客户端和节点会自动重连。

**PHP 迁移新增后。** 内置建表 SQL 对应当时仓库的 55 个 PHP 迁移。以后 PHP 侧再新增迁移，需要运行 `scripts/wsl/dump_schema.sh` 重新导出并重新发版；`migrate` 不会给已有表加列。

---

## 10. 安全清单

- [ ] `APP_KEY` 已设置且妥善保管（`.env` 权限 `600`，属主 `xboard`）
- [ ] 管理员密码已改，不用默认或安装时输出的随机密码
- [ ] `secure_path` 已改成不易猜的路径
- [ ] `LISTEN=127.0.0.1:...`，仅反向代理可访问；防火墙只开放 80 / 443
- [ ] 全站 HTTPS；`TRUSTED_PROXIES` 配置正确
- [ ] 数据库账号只授权本库，密码强度足够，不对公网开放；Redis 设密码且不对公网开放
- [ ] `XBOARD_ALLOW_PRIVATE_SUBSCRIPTION` **未设置**（它会放开订阅导入对内网地址的限制，仅供本机联调）
- [ ] `XBOARD_DEBUG`、`XBOARD_SQL_DEBUG` 未开启
- [ ] 开发 / 测试账号、测试节点不在生产库中
- [ ] 反向代理的 `client_max_body_size 600m`（用于后台上传插件 / 主题）只对管理端路径生效更稳妥，不要全站放开

---

## 11. 已知限制与风险

| 项 | 影响 | 说明 |
|---|---|---|
| 不执行 PHP 插件与主题 | 依赖未移植插件的功能失效 | 见 1.3 |
| 仅支持 MySQL / MariaDB | SQLite / PostgreSQL 部署无法直接切换 | 见 1.1 |
| MySQL 8 未实测 | 首次在 MySQL 8 上运行可能暴露问题 | 预发先验证 |
| 邮件队列在进程内 | 重启丢失未发邮件 | PHP 用 Laravel 队列 + Horizon |
| 调度器无补跑 | 停机期间的定时任务跳过 | 订单、佣金等按状态机推进，下次运行会继续处理 |
| `migrate` 不改已有表 | 升级若有加列需求需手工处理 | 见第 9 节 |
| 真实支付 / SMTP / Telegram / 节点 WS 未联调 | 上线后第一批真实交易是首次验证 | 见第 0 节，务必预发验证 |
| 明文订阅下线 | 只提供加密订阅，需要客户端支持 | 与 PHP 版当前行为一致 |
| 导入节点需要权限组 | 导入后没设组的节点用户看不到 | 导入时选权限组，或事后在节点管理里批量设置 |
| 测速依赖 sing-box 与网络 | 纯 IPv6 节点在没有 IPv6 的探测机器上会显示不可用 | 属于探测环境限制 |

完整的与 PHP 行为差异（36 条）见 [PROGRESS.md](PROGRESS.md) 的"与 PHP 行为不一致"一节。

---

## 附录 A：环境变量

在工作目录 `.env` 中配置，真实环境变量优先；变量名与 PHP 版一致。

| 变量 | 说明 |
|---|---|
| `LISTEN` | 监听地址，如 `127.0.0.1:7001` |
| `APP_KEY` | **必填**。`./xboard keygen` 生成；切换自 PHP 时沿用其值 |
| `APP_TIMEZONE` | 缺省 `Asia/Shanghai`（日界、统计、定时任务按此计算） |
| `DB_HOST` `DB_PORT` `DB_DATABASE` `DB_USERNAME` `DB_PASSWORD` | MySQL / MariaDB |
| `REDIS_HOST` `REDIS_PORT` `REDIS_PASSWORD` `REDIS_DB` | Redis；`REDIS_HOST` 也可以是 unix socket 路径 |
| `TRUSTED_PROXIES` | 逗号分隔；只有来自这些地址的请求才采信 `X-Forwarded-For` |
| `XBOARD_SCHEDULER` | `0` 关闭内置调度器 |
| `XBOARD_AUTO_MIGRATE` | `0` 关闭 `serve` / `schedule` 启动时的自动建表 |
| `XBOARD_ENV_FILE` | 指定 `.env` 路径 |
| `XBOARD_PUBLIC_DIR` `XBOARD_THEME_DIR` `XBOARD_PLUGINS_DIR` | 磁盘覆盖目录，存在时优先于内嵌资源 |
| `XBOARD_STORAGE_DIR` | 存储目录，缺省 `storage`（自动下载的 sing-box 放在其下 `app/singbox/<版本>/`） |
| `XBOARD_SINGBOX_DOWNLOAD_BASE` | sing-box 下载镜像地址（国内服务器常用） |
| `XBOARD_BALALA_SCRIPT` | BalalaSub 抓取脚本路径 |
| `XBOARD_ALLOW_PRIVATE_SUBSCRIPTION` | 仅本机联调；生产不要设置 |
| `XBOARD_DEBUG` `XBOARD_SQL_DEBUG` | 调试日志 / 打印全部 SQL；生产不要常开 |

## 附录 B：命令行

| 命令 | 作用 |
|---|---|
| `xboard` / `xboard serve` | 启动 HTTP + WebSocket + 调度器 |
| `xboard schedule` | 只运行调度器 |
| `xboard migrate` | 建库建表（幂等，已有数据不动） |
| `xboard install [邮箱]` | `migrate` + 创建管理员（已有管理员则跳过），打印后台地址 |
| `xboard admin:create 邮箱 [密码]` | 创建管理员；邮箱已存在则提升为管理员 |
| `xboard keygen` | 生成 `APP_KEY` |
| `xboard version` | 打印版本 |
| `xboard <任务名>` | 立即执行一次定时任务，如 `check:order`、`reset:traffic` |

## 附录 C：内置定时任务

| 任务 | 频率 | 作用 |
|---|---|---|
| `check:order` | 每分钟 | 待支付超 2 小时取消；处理中的订单继续开通 |
| `check:commission` | 每分钟 | 超 3 天自动确认，按分销比例发放 |
| `check:ticket` | 每分钟 | 管理员已回复且 24 小时无跟进的工单自动关闭 |
| `check:traffic-exceeded` | 每分钟 | 超额用户从节点移除 |
| `reset:traffic` | 每分钟 | 到期的流量重置 |
| `reset:log` | 每日 00:00 | 清理过期统计与审计日志 |
| `cleanup:online-status` | 每 5 分钟 | 清零超时的在线数 |
| `check:server-latency` | 每分钟（按 `server_latency_probe_interval` 节流） | 节点测速，补充国家码 |
| `server:import-subscriptions` | 每小时 | 按预设间隔导入订阅 |
| `xboard:statistics` | 每日 00:10 | 统计昨日数据 |
| `cleanup:cancelled-orders` | 每日 03:30 | 删除「已取消」且取消时间超过 N 天的订单；后台「订单管理」里的开关控制，**缺省关闭**；只由调度器触发，没有命令行入口 |
| `send:remindMail` | 每日 11:30 | 到期前一天、流量超 80% 的提醒邮件 |
| `balala-sub:import` | 每分钟（插件启用才执行） | BalalaSub 导入 |

## 附录 D：PHP 与 Go 的进程对照（切换时停什么）

| PHP（Docker 方案） | Go |
|---|---|
| Caddy（监听 7001，`/ws` 转 8076） | 反向代理 → Go 的 `LISTEN`，`/ws` 同端口 |
| Octane（HTTP） | `xboard serve` |
| Octane tick 里的 `schedule:run`（带 Redis 锁） | 内置调度器（Redis 锁去重） |
| Horizon + Redis 队列 | 进程内邮件队列 |
| ws-server（8076） | 并入 `serve`，路径 `/ws` |
| 容器内 Redis | 需要一个独立的 Redis（建议使用不同的 `REDIS_DB`） |
