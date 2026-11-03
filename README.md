# Xboard (Go)

代理订阅面板，**Go 语言实现**。单个二进制即可部署：内嵌管理端、默认用户端主题、核心插件清单和建表 SQL，服务器上只需要 MySQL / MariaDB 和 Redis，不需要 PHP 或 Node。

> 本仓库原为基于 Laravel 的 PHP 版 Xboard。现已整体替换为 Go 版：接口路径、数据库表结构、登录 token 与 PHP 版兼容，可以原地替换。
> 替换前的 PHP 版完整保存在标签 **`php-final`**：`git checkout php-final`。

## 特性

- 用户、套餐、订单、优惠券、礼品卡、邀请与佣金、工单、公告、知识库
- 支付网关：EPay、支付宝当面付、Bepusdt、MGate、BTCPay、CoinPayments、Coinbase
- 订阅输出：sing-box、Clash / Clash.Meta / Stash、Surge、Surfboard、Loon、QuantumultX、Shadowrocket、SIP008、通用 base64
- 节点对接（UniProxy）、WebSocket 推送、节点真连测速（sing-box）、按国家去重
- 内置调度器（订单、佣金、流量重置、统计、提醒邮件等），无需 cron / 队列进程
- 邮件、验证码、Telegram 机器人
- 管理端（React）在 `admin-src/`，用户端为内嵌的编译主题

## 快速开始

需要 Go（版本见 `go.mod`）、MySQL / MariaDB、Redis；构建管理端需要 Node.js。

```bash
cp .env.example .env          # 填 DB_* / REDIS_* / LISTEN
go run ./cmd/xboard keygen    # 把输出写进 .env 的 APP_KEY
go run ./cmd/xboard install admin@example.com   # 建库建表并创建管理员
go run ./cmd/xboard           # 启动（HTTP + WebSocket + 调度器）
```

打发布包（linux/amd64 + linux/arm64，含管理端构建、gofmt / vet / test）：

```bash
VERSION=1.0.0 bash scripts/build-release.sh
TARGETS="linux/amd64" VERSION=1.0.0 bash scripts/build-release.sh   # 只打一个平台
```

产物在 `dist/xboard-<版本>-linux-<架构>.tar.gz`。

## 文档

| 文档 | 内容 |
|---|---|
| [GO-LIVE.md](GO-LIVE.md) | **上线文档**：上线前检查、从 PHP 版切换、冒烟验证、回滚、运维、安全清单 |
| [DEPLOY.md](DEPLOY.md) | 安装与运行（systemd、nginx、升级、多实例） |
| [PROGRESS.md](PROGRESS.md) | 功能清单，以及与 PHP 版的行为差异 |
| [AUDIT.md](AUDIT.md) | 管理端接口与 PHP 版的对照审计 |
| [docs/api.md](docs/api.md) | 接口说明 |

## 目录

```
cmd/xboard        入口（serve / schedule / migrate / install / admin:create / keygen / version / 定时任务）
internal/         后端实现（api、shop、sub、jobs、notify、nodeparse、singbox、schema、web ...）
admin-src/source  管理端前端源码（React + Vite）
theme/Xboard      用户端默认主题（编译产物，无源码）
plugins-core/     核心插件清单（config.json / README，供后台显示）
plugins/          BalalaSub 抓取脚本等运行时资源
deploy/           systemd 单元与 nginx 示例
scripts/          构建与发布脚本；scripts/wsl 为 WSL 端到端验证脚本
```

## 限制

- 只支持 MySQL / MariaDB（不支持 SQLite、PostgreSQL）。
- 不执行 PHP 插件与主题，只维护数据库记录。已原生实现的：核心支付网关、Telegram、BalalaSub。
- 完整列表见 [GO-LIVE.md](GO-LIVE.md) 第 11 节。

## 许可证

见 [LICENSE](LICENSE)。
