#!/bin/bash
# 在 WSL 中启动一个私有 MariaDB（端口 33306，数据目录在 ~/xboard-verify），不触碰任何已有数据库。
set -e
D="$HOME/xboard-verify"
mkdir -p "$D"
if [ ! -f "$D/pw" ]; then
  openssl rand -hex 12 > "$D/pw"
  chmod 600 "$D/pw"
fi
PW="$(cat "$D/pw")"
if [ ! -d "$D/mysql" ]; then
  mariadb-install-db --no-defaults --datadir="$D/mysql" --auth-root-authentication-method=normal > "$D/install.log" 2>&1
fi
if [ ! -S "$D/mysql.sock" ]; then
  setsid nohup mariadbd --no-defaults --datadir="$D/mysql" --socket="$D/mysql.sock" \
    --port=33306 --bind-address=127.0.0.1 --pid-file="$D/mysql.pid" > "$D/mysql.log" 2>&1 < /dev/null &
fi
for i in $(seq 1 60); do
  if mariadb --no-defaults --socket="$D/mysql.sock" -uroot -e "select 1" > /dev/null 2>&1; then break; fi
  sleep 0.5
done
mariadb --no-defaults --socket="$D/mysql.sock" -uroot -e "
CREATE DATABASE IF NOT EXISTS xboard_verify CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'xbv'@'127.0.0.1' IDENTIFIED BY '$PW';
ALTER USER 'xbv'@'127.0.0.1' IDENTIFIED BY '$PW';
GRANT ALL ON xboard_verify.* TO 'xbv'@'127.0.0.1';
FLUSH PRIVILEGES;"
echo "mariadb ready on 127.0.0.1:33306 (db=xboard_verify user=xbv)"
