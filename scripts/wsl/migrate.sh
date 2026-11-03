#!/bin/bash
# 用 PHP 自带的迁移在私有 MariaDB 上建表（环境变量覆盖 .env，不修改任何 PHP 文件）。
D="$HOME/xboard-verify"
PW="$(cat "$D/pw")"
cd /mnt/d/D/vps/Xboard || exit 1
export DB_CONNECTION=mysql DB_HOST=127.0.0.1 DB_PORT=33306 DB_DATABASE=xboard_verify \
  DB_USERNAME=xbv DB_PASSWORD="$PW" CACHE_DRIVER=array CACHE_STORE=array \
  QUEUE_CONNECTION=sync SESSION_DRIVER=array LOG_CHANNEL=stderr
php artisan migrate --force 2>&1 | tail -25
echo "--- tables:"
mariadb --no-defaults --socket="$D/mysql.sock" -uroot xboard_verify -e "show tables" | wc -l
