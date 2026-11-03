#!/bin/bash
# 用 PHP 自带迁移在一个全新的库里建表，再导出结构与迁移自带的种子数据，作为 Go 版内置的 schema。
D="$HOME/xboard-verify"
PW="$(cat "$D/pw")"
SOCK="$D/mysql.sock"
M="mariadb --no-defaults --socket=$SOCK -uroot"
$M -e "DROP DATABASE IF EXISTS xboard_schema; CREATE DATABASE xboard_schema CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; GRANT ALL ON xboard_schema.* TO 'xbv'@'127.0.0.1';"
cd /mnt/d/D/vps/Xboard || exit 1
export DB_CONNECTION=mysql DB_HOST=127.0.0.1 DB_PORT=33306 DB_DATABASE=xboard_schema \
  DB_USERNAME=xbv DB_PASSWORD="$PW" CACHE_DRIVER=array CACHE_STORE=array \
  QUEUE_CONNECTION=sync SESSION_DRIVER=array LOG_CHANNEL=stderr
php artisan migrate --force 2>&1 | tail -3
echo "--- tables with rows:"
$M xboard_schema -N -e "select table_name, table_rows from information_schema.tables where table_schema='xboard_schema' and table_rows>0"
echo "--- migrations rows: $($M xboard_schema -N -e 'select count(*) from migrations')"
OUT=/mnt/d/D/vps/Xboard/internal/schema
mkdir -p "$OUT"
mariadb-dump --no-defaults --socket="$SOCK" -uroot --no-data --skip-comments --skip-add-drop-table --no-tablespaces \
  --compact xboard_schema | sed 's/^CREATE TABLE `/CREATE TABLE IF NOT EXISTS `/' > "$OUT/schema.sql"
mariadb-dump --no-defaults --socket="$SOCK" -uroot --no-create-info --skip-comments --no-tablespaces \
  --compact --complete-insert xboard_schema > "$OUT/seed.sql"
wc -c "$OUT/schema.sql" "$OUT/seed.sql"
grep -c "CREATE TABLE" "$OUT/schema.sql"
