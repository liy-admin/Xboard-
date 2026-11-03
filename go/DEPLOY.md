# Xboard (Go) 部署指南

> 上线（含从 PHP 版切换、验证、回滚）请看 [GO-LIVE.md](GO-LIVE.md)。本文只讲安装与运行。

单个二进制，**内嵌**管理端、用户端默认主题与核心插件清单，建表 SQL 也在二进制里。服务器上只需要 MySQL / MariaDB 和 Redis。

## 1. 准备

| 组件 | 要求 |
|------|------|
| 数据库 | MySQL 5.7+ / 8.x 或 MariaDB 10.4+（**不支持 SQLite / PostgreSQL**） |
| Redis | 5.0+ |
| 系统 | Linux（amd64 / arm64），无需安装 PHP、Node |
| 可选 | `python3`（BalalaSub 抓取）、`sing-box`（真连测速）、`xz-utils`（上传 .tar.xz） |

创建数据库账号（库不存在时程序也会尝试创建，需要 `CREATE` 权限）：

```sql
CREATE DATABASE xboard CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'xboard'@'127.0.0.1' IDENTIFIED BY '换成强密码';
GRANT ALL ON xboard.* TO 'xboard'@'127.0.0.1';
```

## 2. 安装

```bash
sudo useradd -r -m -d /opt/xboard xboard
sudo tar -xzf xboard-<版本>-linux-amd64.tar.gz -C /tmp
sudo cp /tmp/xboard-<版本>-linux-amd64/{xboard,.env.example} /opt/xboard/
cd /opt/xboard
sudo -u xboard cp .env.example .env
./xboard keygen                 # 把输出写进 .env 的 APP_KEY
sudo -u xboard vi .env          # 填 DB_* / REDIS_* / LISTEN
```

## 3. 初始化（建表 + 管理员）

```bash
sudo -u xboard ./xboard install admin@example.com
```

输出会包含管理员账号、随机密码和后台地址（`/<secure_path>`，缺省是 APP_KEY 的 crc32b，可在后台「系统配置」里改）。

- 只建表：`./xboard migrate`（幂等，已有数据不动）；`serve` / `schedule` 启动时也会自动执行一次（`XBOARD_AUTO_MIGRATE=0` 可关闭），所以空库直接启动即可，之后用 `./xboard admin:create 邮箱` 创建管理员
- 再加一个管理员 / 重置密码：`./xboard admin:create 邮箱 [密码]`
- 已有 PHP 版的 MySQL 库：直接指向它即可，`migrate` 不会改动已存在的表（表结构与 PHP 迁移一致）

## 4. 运行

```bash
sudo cp deploy/xboard.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now xboard
journalctl -u xboard -f
```

健康检查：`GET /healthz`（数据库与 Redis 都正常返回 200）。

反向代理与 HTTPS：参考 `deploy/nginx.conf.example`（`/ws` 需要 Upgrade 头）。

## 5. 升级

```bash
sudo systemctl stop xboard
sudo cp xboard /opt/xboard/xboard      # 覆盖二进制
sudo -u xboard /opt/xboard/xboard migrate
sudo systemctl start xboard
```

## 6. 多实例

调度器默认随 `serve` 启动，任务用 Redis 锁去重，多实例也不会重复执行。想只让一台机器跑调度，其余设置 `XBOARD_SCHEDULER=0`；也可以单独 `./xboard schedule`。

## 7. 与 PHP 版并存 / 差异

- 变量名、表结构、Redis 键与 PHP 版一致，可原地替换；Laravel 缓存前缀不同，切换时让两边不要同时写同一个 Redis。
- 前端资源「磁盘优先、内嵌兜底」：在工作目录放 `public/`、`theme/`、`plugins-core/` 可覆盖内嵌版本（用于自定义主题 / 新增插件清单）。
- 不执行 PHP 插件代码；BalalaSub、Telegram、核心支付网关已原生实现。完整差异见 `PROGRESS.md`。
- 缺省站点名称为 `Lets VPN`，可在后台修改。
