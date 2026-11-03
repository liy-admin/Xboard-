#!/usr/bin/env bash
# 把前端构建产物与核心插件清单同步到 internal/web/dist，供 go:embed 内嵌进二进制。
# 在仓库根目录执行：bash scripts/sync-web.sh
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # 仓库根目录
ROOT="$HERE"
DIST="$HERE/internal/web/dist"

rm -rf "$DIST"
mkdir -p "$DIST/assets" "$DIST/theme" "$DIST/plugins-core"
: > "$DIST/.keep"

if [ -d "$ROOT/public/assets/admin" ]; then
  cp -R "$ROOT/public/assets/admin" "$DIST/assets/admin"
else
  echo "警告：缺少 $ROOT/public/assets/admin（先在 admin-src/source 执行 npm run build）" >&2
fi

if [ -d "$ROOT/theme/Xboard" ]; then
  cp -R "$ROOT/theme/Xboard" "$DIST/theme/Xboard"
else
  echo "警告：缺少 $ROOT/theme/Xboard" >&2
fi

if [ -d "$ROOT/plugins-core" ]; then
  for d in "$ROOT"/plugins-core/*/; do
    n="$(basename "$d")"
    mkdir -p "$DIST/plugins-core/$n"
    for f in config.json README.md readme.md; do
      [ -f "$d$f" ] && cp "$d$f" "$DIST/plugins-core/$n/$f"
    done
  done
fi

echo "web assets synced: $(du -sh "$DIST" | cut -f1)"
