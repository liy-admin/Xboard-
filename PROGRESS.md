# Xboard Go 移植进度

最后更新：2026-10-16（第七轮：独立部署能力）。验证：`gofmt -l .` 无输出；`go vet ./...`、`go test ./...` 全部通过；`go build ./cmd/xboard` 产出的二进制在无数据库时打印 `open store: ...` 后以退出码 1 结束；另在 WSL（私有 MariaDB + PHP 迁移建表 + Redis DB 15）上跑通端到端脚本，见「WSL 端到端验证」。`go mod tidy` 在离线环境下因测试依赖无法完成，已手动把 `gorilla/websocket` 改为直接依赖（v1.5.3，来自本地模块缓存）。

本文件记录"已完成 / 未完成 / 与 PHP 行为不一致之处"。未列出的接口视为未实现。

## 第七轮：独立部署（编译后直接丢服务器）

结论：现在**可以**。`bash scripts/build-release.sh` 产出 `xboard-<版本>-linux-<架构>.tar.gz`（约 15 MB），服务器上只需要 MySQL / MariaDB 与 Redis；不需要 PHP、Node，也不需要仓库里的 `public/`、`theme/`、`plugins-core/`。

| 缺口 | 处理 |
|------|------|
| 没有建表能力（依赖 PHP `artisan migrate`） | 内置 `schema.sql` / `seed.sql`（由 PHP 迁移在全新库上导出，36 张表 + 6 个订阅模板 + `migrations` 记录）。`xboard migrate` 幂等：库不存在时创建，已存在的表不动，种子只写入全新库。对已由 PHP 迁移过的库是空操作（已验证表数与行数不变）。导出时 MariaDB 的 `longtext + json_valid` 在真正的 MySQL 上还原为原生 `JSON`；导入期间关闭外键检查（转储按表名排序，引用表可能尚未创建）。 |
| 没有安装 / 创建管理员命令 | `xboard install [邮箱]`（migrate + 创建管理员，已有管理员则跳过，打印后台地址与站点地址提示）、`xboard admin:create 邮箱 [密码]`（已存在则提升为管理员）、`xboard keygen`（生成 APP_KEY）、`xboard version`。 |
| 配置只能靠环境变量 | 自动读取工作目录的 `.env`（`XBOARD_ENV_FILE` 可指定），真实环境变量优先，支持引号与行尾注释；`REDIS_HOST` 可为 unix socket；新增 `APP_TIMEZONE`（缺省 Asia/Shanghai，内嵌 tzdata，最小镜像也能用）与 `TRUSTED_PROXIES`。 |
| 前端资源依赖磁盘 | 管理端构建产物、默认用户端主题、核心插件清单内嵌进二进制（`internal/web`，`scripts/sync-web.sh` 同步）。磁盘优先、内嵌兜底：工作目录放 `public/`、`theme/`、`plugins-core/` 可覆盖。静态路径做了穿越防护。 |
| 生产默认值 | gin 默认 release 模式（`XBOARD_DEBUG=1` 开调试）；GORM 只记录慢查询与错误，`record not found` 不再刷屏（`XBOARD_SQL_DEBUG=1` 打印全部 SQL）；连接池上限；HTTP `ReadHeaderTimeout`；`/healthz` 同时探测数据库与 Redis。 |
| 打包与部署文件 | `scripts/build-release.sh`（同步前端 → gofmt / vet / test → 交叉编译 → 打包，显式写入文件权限：二进制 0755）、`.env.example`、`deploy/xboard.service`（systemd）、`deploy/nginx.conf.example`（含 `/ws`）、`DEPLOY.md`。 |
| `go mod tidy` 无法完成 | 已可执行，`go.mod` / `go.sum` 已整理（`go-sql-driver/mysql` 为直接依赖）。 |

清洁服务器演练（`scripts/wsl/fresh_deploy_test.sh`）：空目录 + 只有发布包里的文件 + 一个不存在的数据库，覆盖 migrate 幂等、install、`.env` 配置、内嵌资源、`/healthz`、release 模式、路径穿越、管理端登录、内嵌插件 / 主题清单，并在这个全新安装上跑完整用户旅程（65 项）与 SIGTERM 退出，全部通过。

演练中发现并修复的缺陷：

1. `migrate` 建表因外键顺序失败（`v2_server` 引用的表尚未创建）。
2. 在 Windows / NTFS 上构建的发布包丢失二进制的可执行位（服务器上 `Permission denied`）。
3. 全新安装没有站点地址时保存支付方式会失败（PHP 同样要求）：`install` 现在会提示先配置站点地址。

仍需注意：

- **只支持 MySQL / MariaDB**。PHP 版支持的 SQLite、PostgreSQL 不支持（大量使用 `JSON_CONTAINS`、`FROM_UNIXTIME` 等 MySQL 语法）；从 PHP 的 SQLite 安装迁移需要先把数据导入 MySQL。
- **MySQL 8 未实测**：演练环境只有 MariaDB 10.11；MySQL 8 路径（JSON 列还原）有代码但没有跑过。
- **后续 PHP 迁移**：内置 schema 对应当前仓库的 55 个迁移。以后 PHP 新增迁移时，需要运行 `scripts/wsl/dump_schema.sh` 重新导出并重新发布。
- **无自动升级迁移**：`migrate` 只建缺失的表，不会给已有表加列。

## 第六轮：前后端联调

目标：把前端与 Go 后端一起跑起来，逐页看效果、修问题。环境为 WSL 内私有 MariaDB（PHP 迁移建表）+ Redis DB 15 + 交叉编译的 Linux 二进制；管理端用重写后的 Lets VPN 样式（`admin-src` 构建到 `public/assets/admin`），用户端用仓库里原有的编译主题（`theme/Xboard`，无源码，未改动）。

新增 / 补齐：

| 类别 | 内容 |
|------|------|
| 网页外壳 | `GET /` 渲染用户端主题外壳（读 `public/theme`，首次访问自动从 `theme/` 复制）；`GET /{secure_path}` 渲染管理端外壳（按 Vite `manifest.json` 收集入口脚本与样式）；`/assets`、`/theme`、`/uploads` 静态服务。此前 Go 完全不提供页面，前端无法跑起来。 |
| 游客接口 | `/api/v1/guest/`：`comm/config`、`plan/fetch`、`notice/fetch`、`website/config`、`discover/config`、`anti_loss/config`、`ad-banner/config`。此前整个 guest 分组缺失，用户端登录页依赖的 `comm/config` 会 404。 |
| 客户端 App | `/api/v1/client/app/getVersion`、`getConfig`（需 token）。 |
| 邮件链接登录 | `POST /passport/auth/loginWithMailLink`（v1 / v2），5 分钟一次性链接，60 秒限频。 |
| 管理端补齐 | `config/getEmailTemplate`、`getThemeTemplate`、`setTelegramWebhook`、`uploadDiscoverHtml`；`coupon/drop`；`notice/update`；`server/manage/pluginSubscription` / `pluginDoctor` / `pluginSetup` / `pluginImport`。与 PHP `AdminRoute` 逐条对照后，仅剩 `user/setInviteUser`、`system/getQueueMasters`、`update/*` 三类（PHP 中没有对应控制器方法或路由已注释）。 |
| 请求方法 | PHP 用 `any` 的列表接口（`order/fetch`、`user/fetch`、`ticket/fetch`、`coupon/fetch`、`gift-card/*`、`stat/getStatUser`、`system/getAuditLog` 等）同时接受 GET / POST；`generateEchKey`、`machine/getToken`、`installCommand` 同时接受 GET / POST。前端用 POST 提交 body，原先 Go 只注册了 GET，这些页面全部 404。 |
| 过滤与分页 | 新 `admin_filter.go` 对应 PHP `QueryOperators`：`{id, value, logic}`，值支持数组（IN）、`op:value`（eq / gt / gte / lt / lte / like / notlike / null / notnull）、默认 LIKE；`total_used` 计算列；`plan` / `invite_user` / `group` 关联过滤；参数同时读查询串与 JSON body。`paramsMW` 把 body 的标量参数镜像到查询串，使 `c.Query` 在 POST 下同样生效。 |
| 用户信息 | `user/info` 补齐 `remind_expire`、`remind_traffic`、`discount`、`commission_rate`、`telegram_id`，与 PHP select 一致。 |
| 用户生成 | `user/generate` 重写：参数 `email_suffix`（前端 / PHP 一致，原先误用 `email_domain`），三种模式（前缀 + 数量、单个、随机），密码缺省为邮箱，批量返回 `{code, message, data}` 或 CSV。 |
| 管理端前端 | Lets VPN 品牌与样式重写（见 `admin-src/source/assets/styles/theme/letsvpn.css`、`components/Brand.jsx`、登录页分栏布局）；Go 内 `app_name` 缺省值改为 `Lets VPN`。 |

联调中发现并修复的缺陷：

1. `v2_plan` / `v2_payment` 创建不写 `created_at`（NOT NULL 无默认），套餐与支付方式无法新建。
2. `v2_order.commission_status` 为 NOT NULL，模型用可空指针插入 NULL，管理端分配订单（以及任何走该模型的下单）失败。
3. 支付方式表单按插件目录名查找，与网关名大小写不同（`EPay` / `Epay`、`MGate` / `Mgate`、`BTCPay` / `Btcpay`、`AlipayF2F` / `AlipayF2f`），表单取不到。
4. 节点 `group_ids` / `route_ids` 被前端以字符串回传（编辑节点时 `map(String)`），`JSON_CONTAINS(group_ids, 数字)` 匹配不到，节点对用户组不可见。现在保存时统一转整数。
5. 管理端创建节点不写 `created_at` / `updated_at`，`cache_key` 带 `0001-01-01`。
6. 分页参数 `current` / `pageSize` 只读查询串，POST body 里的分页被忽略。
7. 进程不响应 SIGTERM、WebSocket 注册表未创建、验证码邮件正文为空等（见第五轮 WSL 验证章节）。

验证脚本（均在 `scripts/wsl/`）：`api_tests.py`（管理端全部写接口，前端同款载荷，100 项）、`user_flows.py`（终端用户全流程，含下单 / 支付回调 / 开通 / 工单 / 邀请 / 改密 / 订阅，65 项）、`verify.sh`（75 项）、`verify_flows.sh`（客户端信号、支付回调、邮件、优雅退出，40 项）、`run_stack.sh`（起完整本地栈）。

## 第五轮：本轮完成

| 类别 | 内容 |
|------|------|
| 设备体验 | `POST /api/v1/passport/auth/deviceTrial`：同一设备只能领取一次，之后凭设备码重新登录同一影子账号（`@device.local`）。绑定类型保留 PHP 行为（`trial`）。 |
| 审计日志 | 管理端写操作经 `auditMW` 记录到 `v2_admin_audit_log`（动作名 `资源_方法`）；请求参数在 handler 执行前捕获，过滤 password / token / secret / key / api_key。`getAuditLog` 已提供。 |
| 邮件队列 | 进程内 `jobs.MailQueue`：2 个工人、失败最多 3 次（线性退避）、保留最近 100 条失败记录、统计（已处理、失败、每分钟、待处理、运行中）。`getQueueStats` / `getQueueWorkload` / `getHorizonFailedJobs` 基于它。 |
| 插件记录管理 | `plugins` 表的扫描、安装、卸载、启用、停用、配置读写、上传、删除、升级记录（扫描 `plugins-core` 与 `plugins`，或 `XBOARD_PLUGINS_DIR`）。只维护数据库记录，不执行 PHP 插件代码。 |
| 主题管理 | 主题列表、上传、删除、主题配置读写（存于 `theme_<name>` 设置）。目录为 `theme` 或 `XBOARD_THEME_DIR`。 |
| 支付方式 | `getPaymentForm`：由 `plugins-core/*/Plugin.php` 的 `form()` 生成 `payGatewayForms`（`internal/api/payform_map.go`）。`uploadIcon`：2MB 上限，按内容嗅探类型，写入 `public/uploads/payment/`。 |
| 静态上传 | `/uploads` 由 gin 提供（目录为 `XBOARD_PUBLIC_DIR` 下的 `uploads`），与图标上传返回的 URL 对应。 |
| Telegram 插件 | 插件未启用时机器人不响应（`pluginEnabled("telegram")`）。欢迎语文案可配置，缺省值来自内嵌的 `telegram_plugin.json`（与 `plugins-core/Telegram/config.json` 一致）。启动时注册命令菜单（`setMyCommands`）。 |
| balala 抓取 | `internal/jobs/balala.go`：以同一环境变量调用插件的 `python/fetch_sub.py`，读取输出文件 JSON，结果写入 Redis 缓存 `balala_sub:latest`。CLI：`xboard balala-sub:fetch`。`server:import-subscriptions` 中的预设勾选 `refetch_balala` 时先抓取，失败回退预设链接。 |
| balala 管理接口 | `GET /api/v1/balala-sub/latest`、`POST /refresh`、`GET /doctor`、`POST /upload-firefox`（管理员鉴权，插件未启用返回 400「插件未启用」）。`upload-firefox` 支持 zip 与 tar.gz，拒绝路径穿越，写入配置 `chrome_path`。 |
| 节点国家码 | `internal/jobs/geoip.go`：host → IPv4 → ip-api.com 批量接口（每批 ≤100）。测速成功且缺少 `country_code` 的节点批量补充；查询失败保持为空，下次再试。 |
| sing-box 诊断 | `GET /server/manage/probeDiagnose`：按 `node_probe_singbox_path`（非裸命令）、再按 PATH 中的 `sing-box`，执行 `version` 检测（5 秒超时），并对样本节点（`id` 参数，缺省取第一个展示节点）实跑一次快速探测（1 次、请求超时 5 秒），返回 ok / path / version / error / sample。 |
| 节点探测（重写） | `internal/jobs/probe.go` 对应 `NodeProbeService`：`node_probe_mode=singbox`（缺省）起只含被测节点的 sing-box（`sub.ProbeOutbound` 复用订阅的 outbound 生成），经本地 socks5h 发真实请求测延迟，取最小/平均/中位/最大，附出口 IP 与国家，不回退 TCP；`tcp` 模式取 3 次最低，UDP 类协议（hysteria / tuic）用 ICMP ping。「没法测」（无 sing-box、无可用用户、协议不支持）保留上次结果，不判不可达。 |
| 测速任务（对齐 PHP） | `check:server-latency`：手动触发忽略自动开关；可用性按「agent 在线或实测结果仍新鲜」判定，新鲜度窗口至少覆盖 1.5 个测速周期；自动显隐按「每国延迟最低的可达节点 + 无国家的可达节点」，一个都不显示时保持现状；变更才推送客户端 `subscription.updated`；全量探测 0 个可用节点时按插件配置 `auto_reimport_on_empty` 写入 balala 导入信号；探测字段用 `UpdateColumns` 写入，不改 `updated_at`；新增 `v2_server.exit_ip`。 |
| 国家码名称回退 | `ParseCountryFromName`（旗帜 emoji → 中文关键词 → 英文名/国家码按单词边界），顺序与 PHP `ServerService::parseCountryFromName` 一致。 |
| balala 自动导入 | `balala-sub:import`（调度表中每分钟一次，插件启用才执行）：导入信号 / 开关 + 间隔判断、按客户端选链接、过滤信息条目（到期、流量、官网等，支持 `info_node_keywords`）、按标识整批替换、权限组 / 倍率 / 显示可配置、状态写 Redis。CLI：`balala-sub:import`、`balala-sub:import-force`。 |
| .tar.xz 上传 | `upload-firefox` 的 `.tar.xz` / `.tar.bz2` / `.tar` 交给系统 `tar`（xz 需 xz-utils），与 PHP 一致。 |

## 第四轮：已完成

| 类别 | 内容 |
|------|------|
| 节点 WebSocket | `/ws` 通道（与 PHP NodeWorker 协议一致）：节点 / 宿主机（`node_id`/`machine_id`+`token`）与客户端（`client_token`）鉴权、`auth.success`/`error`、10 秒鉴权超时、`sync.config`/`sync.users`/`sync.devices`/`sync.nodes`/`sync.user.delta` 下发、`node.status`/`report.devices`/`request.devices`/`pong` 处理、心跳键 `ws_server:heartbeat`、55 秒 ping、10 秒待下发设备节点处理、订阅 Redis `node:push` 与 `client:push`、`client_ws_enable` 下的 3 秒节流广播。 |
| 节点测速 | TCP 握手模式的 `check:server-latency`（按 `server_latency_probe_interval` 节流，支持端口范围与自动显隐）；后台 `probeLatency`/`probeStatus`（Redis 状态）。 |
| 节点管理 | `copy`、`resetTraffic`、`batchResetTraffic`、`batchDelete`、`batchUpdate`、`batchSave`、`batchImport`、`parseNodes`（分享链接与订阅内容解析）、`importSubscriptions`（GET/POST）、`syncStats`、`generateEchKey`（X25519，字节布局与 PHP 一致）。 |
| 节点导入 | `internal/nodeparse`：vmess/vless/trojan/ss（SIP002 与整体 base64）/hysteria2/hy2/hysteria/tuic 解析；订阅下载仅允许公网地址；`server:import-subscriptions` 每小时按预设间隔导入，按标签整批替换。 |
| 订单 | `update`（commission_status）、`assign`（按用户分配订单，类型判断与 PHP 一致，邀请佣金基数通过 `shop.ApplyInvite`）。 |
| 统计 | `getOverride`、`getServerLastRank`、`getServerYesterdayRank`、`getStatUser`、`getRanking`（server_traffic_rank / user_consumption_rank / invite_rank）、`getTrafficRank`（含上一周期增长率）。 |
| 礼品卡 | `export-codes`（TXT）、`statistics`（总量、按天、按模板类型）。 |
| 工单 Telegram | 用户创建或回复工单后通知绑定的管理员；管理员回复 Telegram 提醒即可在工单下回复。 |
| Telegram 机器人 | `POST /api/v1/guest/telegram/webhook`（`access_token` 为 md5(bot_token)）；命令 `/start`、`/bind`、`/traffic`、`/getlatesturl`、`/unbind`；入群申请按绑定与可用性通过/拒绝；付款成功通知（`shop.PaidHook`）。 |
| 系统 | `getSystemStatus`：数据库、Redis、WebSocket 心跳、定时任务时间。 |

## 进度总表

| # | 模块 | 状态 |
|---|------|------|
| 1 | 用户侧 | 完成 |
| 2 | 管理端 V2 | 完成，除下文列出的插件 / 主题执行与 Horizon 相关项 |
| 3 | 节点服务端 | 完成（HTTP 接口 + WebSocket 通道） |
| 4 | 定时任务与 CLI | 完成（11 个调度任务 + `balala-sub:fetch` CLI） |
| 5 | 邮件 / 验证码 / Telegram | 完成 |
| 6 | 订阅协议 | 完成 |

## 1. 用户侧

路由挂在 `/api/v1/user` 与 `/api/v1/passport` 下，路径与 PHP 一致。设备体验（`deviceTrial`）已完成。

## 2. 管理端 V2

路由前缀为 `secure_path`（缺省 `crc32b(APP_KEY)`），通过 `gin.NoRoute` 分发。鉴权为 `is_admin`，未通过返回 `403 {"message":"Unauthorized"}`。

未完成（见「仍未完成」）：
- PHP 插件与主题的真实执行（Go 版只维护记录）。
- `UpdateController`：PHP 中相关路由已注释停用，Go 版不注册。
- `SystemController::getQueueMasters`：依赖 Horizon，不适用。
- `UserController::setInviteUser`：PHP 中没有该方法实现，路由本身即不可用，Go 版不注册。

## 3. 节点服务端

- `/api/v2/server`：`handshake`、`report`、`config`、`user`、`push`、`alive`、`alivelist`、`status`。
- `/api/v2/server/machine`：`nodes`、`status`。
- `/ws`：节点、宿主机、客户端 WebSocket 通道。

## 4. 定时任务与 CLI

用法：`xboard`（或 `xboard serve`）启动 HTTP、WebSocket 与调度器（`XBOARD_SCHEDULER=0` 关闭调度器）；`xboard schedule` 仅运行调度器；`xboard <job>` 立即执行一次。

| 任务 | 频率 | 说明 |
|------|------|------|
| check:order | 每分钟 | 待支付超 2 小时取消，处理中调用 Open |
| check:commission | 每分钟 | 超 3 天自动确认，按分销比例发放 |
| check:ticket | 每分钟 | 管理员已回复且 24 小时未跟进的工单自动关闭 |
| check:traffic-exceeded | 每分钟 | 超额用户推送移除（通过 `node:push` 下发） |
| reset:traffic | 每分钟 | 到期的流量重置 |
| reset:log | 每日 00:00 | 清理过期的统计与审计日志 |
| cleanup:online-status | 每 5 分钟 | 清零超时在线数 |
| check:server-latency | 每分钟（内部按 `server_latency_probe_interval` 节流） | TCP 测速，成功节点补充国家码 |
| server:import-subscriptions | 每小时 | 按预设间隔导入订阅（可选 balala 重抓） |
| xboard:statistics | 每日 00:10 | 统计昨日数据写入 v2_stat |
| cleanup:cancelled-orders | 每日 03:30 | 删除已取消超过 N 天的订单（后台开关，缺省关闭；仅调度器触发） |
| send:remindMail | 每日 11:30 | 到期前一天提醒、流量超 80% 提醒 |
| balala-sub:import | 每分钟（插件启用才执行） | BalalaSub 插件注册的任务，内部按开关 / 间隔 / 导入信号判断 |

`check:server-latency` 的测速间隔节流只在调度器内生效；CLI 手动执行 `xboard check:server-latency` 立即运行。

CLI 专用：`balala-sub:fetch`、`balala-sub:import-force`（不在调度表中）。

未完成：除 BalalaSub 外的插件内定时任务（Go 不加载 PHP 插件）。

## 5. 邮件、验证码、Telegram

已完成：SMTP（ssl / tls / 明文）、DB 模板渲染、提醒邮件、统计、邮箱验证码、邀请码访问计数、注册与找回密码中的验证码与邮箱规则、图形与 Turnstile、reCAPTCHA v2/v3、Telegram Bot API 客户端、webhook 与内置命令、命令菜单注册（启动时）、欢迎语可配置（插件配置，缺省值内嵌）、插件未启用时停止响应、邮件队列（进程内，2 工人，3 次重试）。

## 6. 订阅协议

Surge、Surfboard、Loon、QuantumultX、Shadowrocket、SIP008 已接入 `internal/api/client.go` 分发表。Surge 与 Surfboard 的默认模板是从 `resources/rules/` 复制到 `internal/sub/` 的，修改 PHP 侧模板时需同步。

## 仍未完成

1. **PHP 插件与主题的真实执行**：Go 版只维护 `v2_plugins` 表记录与主题文件，不加载 PHP 插件代码，也不执行插件的钩子（BalalaSub 的抓取、导入、测速无可用信号已按其行为原生实现）。其他插件的定时任务同样不会运行。
2. **Horizon 相关**：`getQueueMasters` 不适用；队列为进程内实现，进程退出时未发送的邮件丢失。
3. **sing-box 托管副本自动下载**：PHP 的 `SingBoxBinaryService::ensureManaged` 未移植，需要宿主机安装 sing-box 或在 `node_probe_singbox_path` 指定。真连探测本身已在 WSL 用真实 sing-box 1.14.3 验证（本地 shadowsocks 服务端：测得延迟、记录出口 IP、节点下线后判为不可达）。
4. **balala 抓取脚本端到端**：`fetch_sub.py` 依赖 Python、浏览器内核与真实账号。Go 侧的调用、环境变量、输出文件契约、导入与缓存已用桩脚本验证，但未对真实站点验证。
5. **maxmind**：`geoip_method=maxmind` 在 PHP 中同样回退到 ip-api，Go 一致；没有本地库查询。
7. **Telegram / SMTP / 支付网关 / 真实节点 WebSocket 联调**：未接真实的外部服务；这些路径只有逻辑层与桩测试。

## 与 PHP 行为不一致（汇总）

1. **Redis / 缓存键前缀**：Laravel Cache 使用 `cache.prefix`，Go 直接使用键名。两边共用同一 Redis 时键名不互通，需整体切换。
2. **邮件兜底正文**：无数据库模板时输出简版 HTML，不复现 Blade 视图（已确认不需要）。
3. **后端文案**：统一中文（已确认）。
4. **邮件发送**：PHP 通过 Laravel 队列 + Horizon 异步发送；Go 为进程内队列，重启丢失未发送任务。
5. **统计时区**：定时任务按服务器本地时区计算日界，PHP 为 Asia/Shanghai。
6. **节点在线判断**：`is_online` 为 Redis 中 5 分钟内有探测或上报，或存在 `node_ws_alive:<id>`。WebSocket 连接建立时同时写入该键。
7. **节点测速**：仅 TCP 握手。失败时 `latency_ms` 置空；sing-box 模式的 `exit_ip` 不写入。
8. **节日倍率**：PHP 的 `calculateActualRewards` 会作用到 `plan_id`，Go 跳过 `plan_id`（修正）。
9. **佣金舍入**：金额按四舍五入写入（MySQL 写 INT 列的行为）。单笔订单出错时 PHP 中断命令，Go 记录后继续。
10. **设备绑定类型**：按确认保留 PHP 行为（落库为默认值 `trial`）。
11. **Loon Hysteria2**：无 SNI 时 PHP 输出字面量 `(null)`，Go 省略该字段（修正）。
12. **Shadowrocket**：`h2` / `httpupgrade` 的 host 为字符串时，PHP 的 `$host[0]` 取首字符，Go 保留整串（修正）。
13. **requirements 的 base_version**：沿用 `internal/sub/servers.go` 既有设计，不参与过滤。
14. **旧设备数据**：切换前旧版 Go 的 `ALIVE_IP_USER_*` 在 `alivelist` 中仍读取，最多 10 分钟。之后以 `user_devices:*` 为准。
15. **流量重置 metadata**：手动重置的 `reason` 与 `admin_id` 通过事后更新最近一条重置日志实现，PHP 则在同一事务内随日志写入。
16. **配置缺省值**：`secure_path` 与 `email_whitelist_suffix` 的缺省值在 `configDefaultOverrides` 中手工实现。
17. **宿主机 token**：Go 使用 64 位十六进制，PHP 的 `generateToken` 长度未核对。
18. **用户会话列表**：不返回 token 哈希（PHP 返回）。
19. **统计排行的默认时段**：`getRanking` 默认最近 30 天，`getTrafficRank` 默认最近 7 天；PHP 的默认时段在服务层设置，未逐项核对。
20. **礼品卡统计**：按天统计使用 MySQL 的 `DATE(FROM_UNIXTIME(...))`，与 PHP 的多数据库分支相同，仅在 MySQL 上验证。
21. **WebSocket 地址**：PHP 的 Workerman 监听独立端口，Go 与 HTTP 共用监听地址，握手返回的 `ws_url` 为 `{scheme}://{host}/ws`，与之一致。
22. **审计日志捕获时机**：PHP 在控制器内记录；Go 的 `auditMW` 在 handler 执行前捕获请求参数（已过滤敏感键）。
23. **插件与主题**：PHP 会加载并执行插件代码、主题文件；Go 版只维护数据库记录（启停、配置、上传、删除）。
24. **Telegram 欢迎语**：PHP 在绑定引导与命令列表之间缺少分隔，Go 补上 `\n\n`（修正）。
25. **sing-box 诊断**：PHP 缺失时自动下载托管副本；Go 只检测已安装的 sing-box，不下载。
26. **balala 压缩包格式**：zip 与 tar.gz 在进程内解压（拒绝路径穿越）；`.tar.xz` 等交给系统 `tar`，与 PHP 一致。
27. **测速国家码**：ip-api 批量查询、名称回退与 PHP 一致；maxmind 两边均未实现（回退到 ip-api）。
28. **静态上传**：PHP 由 Web 服务器提供 `public/uploads`；Go 由 gin 在 `/uploads` 下提供同一目录。
29. **订阅 URL 私有地址限制**：与 PHP 一致拒绝回环 / 内网地址；Go 额外提供 `XBOARD_ALLOW_PRIVATE_SUBSCRIPTION=1` 仅供本机联调放开（生产不要设置）。
30. **明文订阅**：`/api/v1/client/subscribe` 与 `/{subscribe_path}/{token}` 保持原实现下线（403），只提供加密订阅 `/{subscribe_path}-enc/{token}`。
31. **测速间隔**：PHP 的间隔在 Kernel 调度表里决定；Go 在调度器内用 Redis `server_latency_next` 节流，CLI 手动执行不受限。
32. **「没法测」的判定（修正）**：PHP 在一次都没通时，只要 sing-box 进程已退出或日志非空就判为没法测；而 info 级别日志里总有启动行，导致真正不通的节点永远被当成没法测。Go 只在进程已退出或日志含 `FATAL`（配置 / 启动失败）时才判为没法测，否则判为不可达。
33. **出口 IP 回显**：缺省回显地址 `http://ip-api.com/json/...` 在本次 WSL 环境经代理访问被对端直接断开，属于对端 / 网络限制；Go 与 PHP 一致，回显失败只让 `exit_ip` 为空，不影响可达判定，可在后台把 `node_probe_ip_echo_url` 换成其他回显服务（接受 JSON 的 `ip`/`query` 与纯文本 IP）。
34. **客户端 getConfig**：PHP 把节点合并进 `resources/rules/app.clash.yaml` 旧模板；Go 直接返回订阅用的 Clash 生成结果。
35. **站点名称缺省值**：`app_name` 缺省由 `XBoard` 改为 `Lets VPN`（订阅标题、邮件发件名）；`frontend_theme` 缺省仍为 `Xboard`（主题目录名）。
36. **节点 ID 列表**：保存节点时 `group_ids` / `route_ids` 的数字字符串转为整数（PHP 原样保存，字符串会使 JSON_CONTAINS 匹配失败）。

## WSL 端到端验证

环境：Ubuntu 24.04（WSL2）；私有 MariaDB 10.11（端口 33306，数据目录在家目录，不触碰已有数据库）；表结构由 PHP 自带迁移（`php artisan migrate`，环境变量覆盖 `.env`）创建，共 37 张表；Redis 使用 DB 15；Go 二进制由 Windows 交叉编译（`GOOS=linux`）。脚本做了：注册 / 登录 / 用户信息、加密订阅解密、UniProxy 节点配置、管理端约 30 个 GET 端点冒烟（真实表结构上的 SQL）、配置保存往返、TCP 测速与国家码、sing-box 缺失时保留上次结果、真实 sing-box 真连探测与下线判定、balala 抓取 + 导入（桩脚本 + 本地订阅服务）、无可用信号、自动显隐安全网、`/uploads`、WebSocket 握手与鉴权超时、调度器与 WebSocket 心跳。

验证过程中发现并修复的真实缺陷（单元测试都没覆盖到）：

1. **插件表名错误**：PHP 的表是 `v2_plugins`，Go 全部写成 `plugins`，插件管理与 Telegram 插件启用判断在真实库上都会失败。已修正，且写入改用 `NOW()`。
2. **TIMESTAMP 列写入整数**：`v2_server`、`v2_plugins` 等表的 `created_at` / `updated_at` 是 TIMESTAMP，Go 的节点导入、订阅导入、balala 导入以 Unix 时间戳写入会报 `Incorrect datetime value`。已改为 `time.Now()`。
3. **WebSocket 鉴权超时 panic**：鉴权失败的连接被关闭后，10 秒超时协程向已关闭的通道投递，`send on closed channel` 使整个进程崩溃。`enqueue` / `close` 现在共用一把锁并带关闭标记，附回归测试。
4. **CLI 测速被间隔节流**：`xboard check:server-latency` 误用了调度器的间隔节流，手动执行常被跳过。节流只保留在调度器内。
5. **鉴权失败的 error 帧丢失**：连接关闭时立即关闭底层 socket，排队中的 `error` 帧没来得及写出，节点只看到连接被断开。现在由写协程把队列写完后再关闭连接。

6. **「没法测」判定过宽**：见差异 32，日志非空就把不通的节点当成没法测，节点下线后延迟永远不更新。

最终结果：75 项检查全部通过、0 项失败，其中包含用真实 sing-box 1.14.3 的探测（`scripts/wsl/get_singbox.sh` 下载到 `~/xboard-verify/bin`，本地起 shadowsocks 服务端作为被测节点）。

## 测试覆盖

- `internal/api`：邀请/推荐/抽奖纯函数、知识库占位符、客服信封往返、礼品卡规则、工单与设备辅助函数、节点参数解析、管理端参数解析、套餐资源格式化、订阅地址构造、有序分组序列化、节点 revision（与 PHP sha1 一致）、路由构造（用户侧与管理端，检查重复注册）、shell 转义、邮箱脱敏、增长率、金额千分位、邮件模板必填变量、WebSocket 帧格式与宿主机路由、连接队列满时丢弃、PEM 分行、balala 压缩包解压（拒绝路径穿越）、Firefox 可执行文件定位、WebSocket 连接关闭后投递不 panic。
- `internal/jobs`：佣金分配、推荐时长、舍入、调度器到期判断、任务与调度表一致性、流量提醒阈值、GeoIP 解析（IP 字面量、ip-api 桩服务）、balala 布尔转换与脚本路径覆盖、国家名解析、balala 导入辅助（信息条目过滤、权限组解析、按客户端选链接）、探测辅助（回显解析、延迟统计、端口解析、sing-box 缺失）。
- `internal/nodeparse`：分享链接各协议解析、SIP002 与整体 base64、IPv6 主机端口、订阅 base64 解码、错误行记录。
- `internal/notify`：占位符、转义与 nl2br、模板渲染、Telegram / 验证码（httptest）、邮箱验证码（内存存储）、邮件报文。
- `internal/sub`：六个协议的代表性输出、plugin_opts 解析、有序查询串。

上述 WSL 脚本覆盖了真实库上的主要读写路径；真实节点 WebSocket 会话、支付回调、邮件发送、Telegram 仍只有逻辑层测试。

## 可用国家（无会员用户可见）

- 用户端：`GET /api/v1/user/server/countries`（登录即可，不要求会员，附带 `has_plan`）与 `GET /api/v1/guest/server/countries`（免登录，无 `has_plan`）。只返回国家、可用节点数与最优延迟，不含节点地址和凭据；`/user/server/fetch` 对无会员用户仍返回空列表。
- 快照存 Redis（键 `country_snapshot`），缺失时由请求按需重建。
- 每日自动更新时刻由后台设置 `country_refresh_time`（HH:MM，服务器本地时区，缺省 04:00）决定，调度器每次检查都重新读取，修改后无需重启。
- 国家角标由后台设置 `country_tags`（如 `HK:推荐,JP:高速`）决定，国家接口顶层下发 `tags`（国家码 → 文字，没设置时为空对象）。标签不进快照也不依赖快照，读接口时现取，改后立即生效；快照为空或缺新国家时角标照常下发。
- 约定：新功能只做后台按钮 + 调度器，不提供命令行入口。`refresh:countries` 只能由调度器触发，`xboard refresh:countries` 不可用（`jobs.IsCLI`）。
- 「可用」= 已显示（`show=1`）的主节点中，在线上报或最近一次测速成功且未过期。国家码为空时从节点名解析。
- 新快照没有任何可用国家而旧快照有内容时，沿用旧快照并标记 `kept`（多半是探测异常）。
- 后台：「测速设置」的「可用国家」卡片显示上次更新时间与列表，可改每日更新时间，「立即更新」先对全部节点测速再生成快照（异步，可轮询）。接口：`POST /server/manage/refreshCountries`、`GET /server/manage/countriesStatus`。
- 已知限制：调度器没有补跑——到点时服务没在运行，当天就不会更新（列表可能滞后到下一次触发或手动更新）。免登录接口在快照缺失时会按需查库重建一次。
- 与 PHP 的差异：PHP 没有此功能，属于新增。

## 流量为 0 视为无限流量

- 约定：`v2_user.transfer_enable = 0` 表示无限流量。PHP 的 `UserService::isAvailable` 要求该值非零，导致「有套餐但流量额度为 0」的账号（如免费/体验套餐）下载订阅一律 403；Go 版不再因此拒绝。
- 判断集中在 `shop.IsAvailable`（未封禁且未过期）；`shop.UnlimitedTraffic` 判断是否无限；`shop.HasPlan`（有套餐且可用）用于「是否会员」——流量为 0 不等于有会员，`/user/server/countries` 的 `has_plan` 用它，没套餐的账号仍是 false。
- 节点用户列表（`availableUsers`，UniProxy / WebSocket 推送共用）改为 `(transfer_enable = 0 OR u + d < transfer_enable)`，无限流量的用户不会被节点踢掉。`check:traffic-exceeded` 本来就排除额度为 0 的用户。
- 展示：订阅里的「剩余流量」提示、Telegram 的流量信息、邮件占位符 `user.transfer_left` 与用户导出表，无限流量显示「无限」而不是 0 或负数。
- 订阅响应头 `subscription-userinfo` 仍是 `total=0`（沿用，客户端按 0 处理）。
- 没有套餐的用户（plan_id 为空）下载加密订阅现在返回 200，但解密后没有任何真实节点；本项目客户端本来就会跳过没有套餐的用户，不受影响。

## 后台改动推送给客户端 / 套餐权限组提示

- 后台改节点（保存、更新、删除、排序、复制、批量操作、重置流量）、改套餐（保存、更新、删除）、改权限组和路由、批量导入节点后，通知在线客户端重新拉取订阅（中间件 `pushAfter`，仅在接口成功时触发）。
- 合并发送：最后一次改动后 2 秒才推一次，连续操作只推一次，推出时已是最终状态。经 Redis `client:push` 发布，多实例都能转发。
- 定时导入（`server:import-subscriptions`）与 Balala 导入在实际导入节点后也会推送。
- 修复：后台触发的任务（手动测速、导入）原先没有推送器，测速导致的显隐变化不会通知客户端，现已补上。
- 是否真正发出仍由「客户端推送」开关（`client_ws_enable`）决定；关闭时节点管理页会提示。
- 后台提示：套餐管理页对没设权限组的套餐显示「未设权限组」标记、顶部横幅和表单提示；节点管理页显示没设权限组的套餐（含有效用户数）与「有套餐但没有权限组」的用户数（`/server/manage/syncStats` 新增 `plans_without_group`、`users_without_group`、`client_push_enabled`）。
- 与 PHP 的差异：PHP 只在节点上下线、测速显隐时推送，后台编辑不推送；提示为新增。

## BalalaSub 插件安装（修复）

现象：在后台上传 / 安装 / 启用 BalalaSub 后"没有效果"。在全新实例上用真实的 `BalalaSub.zip` 复现，发现三处与 PHP 不一致：

1. **上传目录错误**：上传的插件被解到 `plugins-core/`（`pluginDirs()` 默认第一项），列表里变成受保护的核心插件、无法删除。现在上传到第一个非核心目录，缺省 `plugins/`（`pluginUploadDir`）。
2. **安装不拉起环境**：PHP 的 `Plugin::install()` 会在后台执行 `pip install ruyiPage`（日志 `storage/logs/balala_sub_setup.log`）；Go 只写了一行记录。现在安装 `balala_sub` 时在后台安装，失败只写日志、不影响插件安装。
3. **Python 解析**：`python_bin` 缺省 `python` 在多数 Linux 上不存在。现在与 PHP 的 `EnvDoctor::resolvePython` 一致：未配置或为默认 `python` 时，Linux / macOS 用 `python3`，Windows 用 `python`。「一键安装环境」与后台自动安装都带 PEP 668 重试（遇到 externally-managed-environment 自动加 `--break-system-packages`）。

仍需手动：Firefox 内核（在插件配置里上传，或填 `chrome_path`）。装完点「开始自检」确认。已经上传到 `plugins-core/` 的旧目录需手动删除后重新上传。

## 已保存订阅可编辑 / 删除所属节点；测速可停止

- **编辑已保存订阅**：在「批量导入 → 订阅链接」点选一个已保存订阅，会出现编辑面板，可更换链接、改名、改标识 / 默认参数 / 定时设置，点「保存修改」写回同一个预设（id 不变）。改链接或「从 Balala 插件获取」不再取消选中。
- **删除所属节点**：编辑面板显示带该预设**已保存**标识的节点数，「删除所属节点」二次确认后删除这些节点，预设本身保留。没有填写标识的预设无法定位节点，按钮不可用。接口：`GET /server/manage/countByTag?tag=`、`POST /server/manage/deleteByTag`（删除会触发客户端推送）。空标识一律拒绝，不会退化成删除全部。
- **停止测速**：测速进度条右侧有「停止」按钮（`POST /server/manage/probeStop`）。停止标记存在 Redis（`server_latency_probe_stop`），多实例下调度器在别的机器上也能收到。每个节点开始前检查，**正在测的那个节点会测完**（sing-box 模式最多十几秒）。手动停止的一轮不做自动显隐、也不触发「0 个可用」的自动重导；状态里 `stopped=true`，页面显示「上次测速已手动停止（已测 x/y）」。「可用国家 → 立即更新」中途停止时，快照仍会用库里现有结果生成。
- 验证：私有 MariaDB 新库 + 真实订阅，逐请求验证了改链接（id 不变）、按标识删除（只删 32 个 PaoPao，Other 不动，预设保留）、停止（2/33 时停止，1 秒后停在 3/33，下一轮不受影响）；并在浏览器里实际操作了编辑面板、删除确认和停止按钮。

## 服务器详情：一键创建节点

- 入口：后台「服务器管理」→ 服务器详情 →「关联节点」右侧的「一键创建节点」。表单嵌在详情窗口里：权限组（只有一个时自动勾选）、节点地址、端口、名称（可选，缺省为服务器名）、Reality 目标站点、是否对用户显示。
- 创建内容：VLESS + Reality（`xtls-rprx-vision`、uTLS chrome），X25519 密钥对与 8 字节 short_id 由面板生成（`realityKeyPair`，格式与 `xray x25519` 一致），目标站点缺省 `www.apple.com`，节点绑定到该服务器并启用。**不要用 `www.microsoft.com`**：它目前的 TLS 握手与节点内核（sing-box 1.14.0-alpha.2 / Xray 26.3.27）的 Reality 不兼容，服务端报 `REALITY: processed invalid connection`、客户端 EOF；同一组密钥换成 apple / mozilla / cloudflare 均可连通（2026-10-22 实测，两种内核 × Xray 与 sing-box 客户端）。协议设置经 `sub.NormalizeProtocolSettings` 规范化，`group_ids` 存整数（与 `nodeSave` 一致）。
- 节点地址缺省为服务器最近一次连接面板的来源 IP：宿主机鉴权（`authByMachine`、`machineAuth`、WebSocket `wsAuthMachine`）时由 `recordMachineContact` 记录到 Redis `machine_last_ip:<id>`（30 天），`machine/fetch` 返回 `last_ip`，详情里显示「最近 IP」。不加数据库列：`migrate` 不会给已有表加列。服务器从未连接过面板时需手填地址。
- 校验：至少一个存在的权限组；地址为 IP 或主机名；端口 1–65535；目标站点必须是域名；同一服务器上已有节点占用同一端口时拒绝（提示占用的节点）。
- 下发：创建后经 Redis `node:push` 发布 `{machine_id, event: "sync.nodes"}`（`pushMachineNodes`），在线的宿主机立即重新发现并启动新节点；同时经 `pushAfter("node")` 通知客户端。
- 接口：`POST /server/machine/quickNode`（管理员；写入审计日志）。
- 顺带修复：前端详情里的「关联节点」以 `id` 传参，后端只读 `machine_id`，列表一直为空（负载历史同样）；`machine/nodes`、`machine/history` 现在同时接受 `id`。安装命令改用 xnode 安装脚本（`liy-admin/xnode`），可用设置 `machine_installer_url` 覆盖。
- 验证：`go test ./...` 通过（新增 Reality 密钥、权限组解析、主机名、`id` 兼容的单测）；WSL 私有 MariaDB 上 22 项接口检查（报错文案逐项核对）；xnode 以该服务器身份连接后拉到节点并启动 sing-box；运行中新建节点，xnode 在 1 秒内经 WebSocket 收到 `sync.nodes` 并启动；浏览器里实际操作了表单（端口冲突提示、创建成功、列表刷新）。
- 与 PHP 的差异：PHP 没有此功能，属于新增。

## 服务器详情：节点增删改查、实测、导出链接

- 详情里的「关联节点」（`admin-src/source/views/config/MachineNodes.jsx`）：列表显示 ID、名称、类型与安全（Reality / TLS 及目标站点）、地址:端口（监听端口不同时另列）、可见、已激活；每行可编辑（复用节点管理的完整表单 `NodeFormDialog`）、取消关联（节点保留）、删除（二次确认）；顶部可一键创建、「新增节点到此服务器」（完整表单，预选该服务器、地址缺省为最近 IP）、「关联已有节点」（从未绑定的节点里多选）。详情窗口加宽到 `max-w-4xl`。
- 即时下发：节点的 `save`、`update`、`drop`、`copy`、`batchDelete`、`batchUpdate` 加了 `pushMachineAfterNode` 中间件（`internal/api/machine_node_push.go`）：改动前查出节点原来绑定的服务器，成功后向新旧服务器都发 `sync.nodes`（新增、删除、启停、换绑立即生效）；单个节点仍托管且启用时再发 `sync.config`（改配置立即生效）。此前只有一键创建会推送，其余改动要等宿主机轮询（默认 60 秒）。
- `machine/nodes` 多返回 `server_port`、`protocol_settings`、`group_ids`、`machine_id`。
- 实测节点：每行「测试」与顶部「全部测试」调用 `POST /server/machine/testNode`（`machine_node_tools.go`），同步执行与定时测速同一套 `jobs.ProbeNode`（sing-box 模式：起只含该节点的 sing-box，经节点请求 3 次；TCP 模式只测端口），结果显示在节点名下（可用 / 延迟中位数 / 成功次数 / 出口 IP，或失败原因；Reality 失败时提示换目标站点）。面板上没有 sing-box 时返回「无法测试」并指向在线下载。测的是面板服务器到节点的路径，界面有说明。
- 导出链接：「导出链接」调用 `GET /server/machine/nodeLinks`，用 `sub.ShareLink`（与通用订阅逐行内容同一套 `genVless` 等）生成服务器上全部节点的分享链接；缺省每个节点用权限组里第一个可用用户（`jobs.FindProbeUser`，与测速同规则），可填用户邮箱；用户不在节点权限组时给出警告，协议没有通用链接格式时给出说明。弹窗里可逐条复制、复制全部、下载 .txt。
- 验证（WSL）：xnode 以服务器身份运行后，apple 目标站点的节点「测试」显示可用 · 约 80ms · 3/3，microsoft 的显示不可用及原因；面板内「在线下载」sing-box 1.14.2 后可测，下载前显示「无法测试」。导出的链接用 sing-box 客户端实连：apple 节点 HTTP 204，microsoft 节点失败。指定不存在的邮箱提示「用户不存在」。`ShareLink` 有单测（Reality 参数齐全、不含私钥）。
- 管理端外壳 `/{secure_path}` 加 `Cache-Control: no-cache`：外壳引用带哈希的入口文件，升级后浏览器缓存的旧外壳会继续加载旧前端。
- 验证：WSL 本地栈里在浏览器实际操作：编辑（改名保存、列表刷新）、切换可见 / 已激活、取消关联、关联已有、删除、完整创建表单预填；同时订阅 `node:push`，每一步都收到对应的 `sync.nodes` / `sync.config`（停用、取消关联、删除后列表里不再有该节点）。

## 修复：后台编辑节点会关掉 TLS / Reality

- 现象：在后台编辑并保存任何节点后，`protocol_settings.tls` 变成 0（Reality / TLS 被关掉，客户端连不上）；hysteria 的 `version` 等整数字段同样被清零；值为 0 的布尔字段被当成 true。
- 原因：管理端请求体用 `UseNumber` 解析，数字是 `json.Number`；`sub.phpInt` 与 `sub.Truthy` 不认识该类型（前者返回 0，后者返回 true）。`nodeSave` 保存前经 `NormalizeProtocolSettings` 规范化，于是被改写。
- 修复：两个函数都处理 `json.Number`（按字符串 / 数值规则）。附单测 `TestNormalizeAcceptsJSONNumber`、`TestTruthyJSONNumber`；WSL 上经管理端 `manage/save` 修改 Reality 目标站点后 `tls` 保持 2，并用 Xray / sing-box 客户端实际连通。
- 已被改坏的节点：需在后台把节点重新设为 Reality（或在库里把 `protocol_settings.tls` 改回 2）。

## 已取消订单自动清理

- 入口：后台「订单管理」页顶部的卡片——开关、保留天数、待清理数量预览、「立即清理」（带确认框）、上次清理结果。
- 规则：只删状态为「已取消」且**取消时间**（`updated_at`）早于 N 天前的订单，不看下单时间，所以很早下单、昨天才取消的订单不会被误删。已支付、待支付、开通中、已折抵的订单不受影响；状态条件写在每条 `DELETE` 里，清理过程中订单状态变了也不会被误删。
- 缺省：开关**关闭**（删除数据必须由管理员主动打开），保留 30 天，范围 1–3650；非法天数后端返回 422，不会悄悄改成别的值。
- 调度：`cleanup:cancelled-orders`，每天 03:30（服务器本地时区）；关闭时在访问数据库之前直接返回。按约定只由调度器与后台按钮触发，`xboard cleanup:cancelled-orders` 不可用（`jobs.IsCLI` 为 false）。
- 分批：每批 1000 条，单次最多 20 万条，超出的部分留给下一次（结果里标记 `truncated`）。
- 记录：上次结果（时间、删除数、天数、自动/手动、错误）存在 `v2_settings.order_cancelled_cleanup_last`，重启与清空 Redis 都不丢。
- 接口（管理员）：`GET /order/cleanupStatus`、`POST /order/cleanupSave`、`POST /order/cleanupRun`；员工账号与匿名请求返回 403。
- 验证：临时空库上的 35 项端到端检查（真实路由 + 管理员鉴权；各状态与新旧日期的保留 / 删除；非法天数；开关开 / 关；2500 条跨 3 批）全部通过，并在浏览器里完整操作过一遍（保存 → 立即清理 → 确认 → 列表刷新）。临时库已删除。
- 与 PHP 的差异：PHP 没有此功能，属于新增。
- 已知限制：删除不可恢复；订单被删后，该订单对应的佣金记录（`v2_commission_log`）仍按订单号保留，不会跟着删除。


## 用户管理：筛选与在线设备

修复了用户管理页的 5 个问题。每一项都在真实 MariaDB + Redis 上用集成测试验证过，并在浏览器里点过界面。

| 问题 | 原因 | 修复 |
|------|------|------|
| 「在线设备」列永远是 `0 / N` | 设备明细只写 Redis（`user_devices:<uid>`），而列表、筛选、仪表盘读的是 `v2_user.online_count / last_online_at`。写回这一步缺失 | 新增 `syncOnlineCount`（`internal/api/device_online.go`），在 HTTP `/alive`、WebSocket 上报、节点下线、WS 上报里用户消失、用户踢设备时回写 |
| 在线数过期后不会清零 | `cleanup:online-status` 拿 Unix 整数去比较 `timestamp` 列，MySQL 会把整数当成 `YYYYMMDDHHMMSS`，条件永远不成立（真实库上验证：旧实现下 20 分钟前的用户仍是 3） | 改为传时间值；阈值由 10 分钟改为 300 秒（与 Redis TTL 对齐） |
| 筛选项被静默忽略 | 前端提供了 `online_count` / `uuid` / `token` / `remarks` / `invite_user_id` / `is_staff`，后端白名单没有，直接跳过 | 补进白名单；测试直接解析前端 `userFilter.js` 的 `FIELDS`，保证前端能选的字段后端都支持 |
| 「批量封禁」不可用 | Go 的 `/user/ban` 只是按 `id` 切换单个用户；前端发的是 `{scope, user_ids, filter}`，返回「用户不存在」 | 实现按范围批量封禁（all / filtered / selected），保留按 `id` 切换的旧用法 |
| 按筛选结果批量操作可能作用到全部用户 | 被跳过的筛选项让 `filtered` 范围变成「全部用户」，影响群发邮件、导出 CSV、批量封禁 | 未知筛选字段返回 422；`filtered` 范围必须至少有一个有效条件 |

筛选模型（后端 `internal/api/admin_filter.go`，前端 `views/config/userFilter.js` + `UserFilterBar.jsx`）：

- 叶子条件 `{id, value:"op:val"}`，操作符 eq / neq / gt / gte / lt / lte / like / notlike / prefix / suffix / null / notnull；数组值为 IN。`neq` 对 NULL 安全；LIKE 通配符 `% _ \` 会转义。
- 嵌套分组 `{any:[...]}`（或）/ `{all:[...]}`（且），深度 ≤ 4、叶子 ≤ 60；未知字段、过深、过多一律 422。
- 筛选条（替换了原来的高级筛选弹窗）：邮箱搜索、「＋ 添加筛选」（字段 / 条件 / 值三步，套餐可多选，带预览）、条件标签（点击编辑、✕ 删除）、「满足全部 / 满足任一」切换、快捷筛选按钮（在线中 / 无套餐 / 已过期 / 7 天内到期 / 已封禁，可叠加、再点取消）、「保存为常用」（浏览器本地，最多 12 个）。
- 字段 20 个：邮箱、ID、套餐、流量、已用、在线设备、设备上限、到期时间、注册时间、最后登录、余额、佣金、UUID、Token、备注、邀请人邮箱 / ID、状态、管理员、员工。
- 日期条件：当天、之前、之后、介于、未来 N 天内、过去 N 天内、早于 N 天前、已过期、未过期；按本地时区整天比较。
- 批量操作（群发邮件、导出 CSV、批量封禁）沿用同一份条件。在线设备数超过上限时标红。

行为说明：

1. 批量封禁**不会封禁管理员和员工账号**；先取出命中的 ID 再分批更新，避免 `or` 逻辑改变 `AND is_admin = 0` 的优先级。
2. 在线数有效期 300 秒，与 Redis 设备记录 TTL 一致。

测试：

- 单测：`countDistinctDevices`、白名单与前端字段一致、未知字段拒绝、生成的 SQL（含分组、新操作符、限制）、批量封禁参数校验、清理任务的比较参数类型。
- 前端纯逻辑：`cd admin-src/source && node --test views/config/userFilter.test.mjs`（13 项，TZ 无关）。
- 集成测试（默认不运行，需要真实库）：
  `DB_HOST=... DB_DATABASE=... DB_USERNAME=... DB_PASSWORD=... REDIS_DB=15 go test -tags integration ./internal/api -run Integration -v`
  覆盖在线数随设备变化、在线筛选、过期清零、批量封禁保护管理员（含 `or` 逻辑）、前端筛选条件在真实库上的语义。

仍未处理：

- 没有套餐的用户在列表里到期时间显示为「长期有效」（到期为空的显示问题，数据本身没错）。
- 没有套餐、没有任何活动的账号不会被自动清理。

## 换套餐时剩余时间按价值折算叠加

- 问题：换套餐（订单类型「升级」）时，到期时间被重置为「现在 + 新周期」，原来剩余的会员时间丢失；用户来回切换套餐时表现为时间被覆盖、无法累加。
- 现在：开通时把旧套餐剩余的秒数按「旧套餐日单价 ÷ 新套餐日单价」折算成新套餐的时长，叠加在新周期之前（`shop.ConvertRemaining`、`buyByPeriod`）。换套餐订单按原价支付，不再折抵金额。
- 日单价：优先用本次购买周期的价格（如都按周付），旧套餐没有该周期时取其各周期中最低的日单价（`shop.PlanDailyPrice`）。
- 边界：旧套餐已删除时剩余时间原样保留；旧套餐没有定价（免费试用）时剩余时间不折算，避免用试用时长换付费天数；同一套餐续费仍从当前到期时间往后累加。
- 一次性（无到期时间）套餐没有时长可叠加，仍由 `surplus_enable` 控制按剩余流量价值折抵金额；后台该开关的说明已同步。
- 礼品卡兑换本来就是从当前到期时间往后累加，未改动；换套餐的礼品卡按卡面天数原样叠加，不做价值折算。
- 验证：独立测试库上走真实「下单 → 支付回调 → 开通」：青铜周付新购 7 天 → 续费 14 天 → 换铂金周付 16.33 天（改动前 7 天）→ 换回青铜季付 147.27 天（改动前 92 天）。
