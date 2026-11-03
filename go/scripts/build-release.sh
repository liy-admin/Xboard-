#!/usr/bin/env bash
# 构建可直接丢到服务器的发布包（单个二进制 + 配置样例 + 部署文件）。
#
#   bash scripts/build-release.sh                 # 默认 linux/amd64 + linux/arm64
#   TARGETS="linux/amd64" bash scripts/build-release.sh
#   VERSION=1.2.3 bash scripts/build-release.sh
#
# 前置：管理端已构建（admin-src/source 下 npm run build → public/assets/admin）。
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # .../go
cd "$HERE"

VERSION="${VERSION:-$(date +%Y%m%d)-$(git rev-parse --short HEAD 2>/dev/null || echo local)}"
TARGETS="${TARGETS:-linux/amd64 linux/arm64}"
OUT="$HERE/dist"

echo "==> 同步前端资源"
bash scripts/sync-web.sh

echo "==> 检查（gofmt / vet / test）"
test -z "$(gofmt -l . | grep -v '^internal/web/dist' || true)" || { echo "gofmt 未通过"; gofmt -l .; exit 1; }
go vet ./...
go test ./... 2>&1 | grep -v "no test files"

rm -rf "$OUT"
mkdir -p "$OUT"
for t in $TARGETS; do
  os="${t%/*}"; arch="${t#*/}"
  name="xboard-${VERSION}-${os}-${arch}"
  dir="$OUT/$name"
  mkdir -p "$dir/deploy"
  echo "==> 构建 $os/$arch"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" -o "$dir/xboard" ./cmd/xboard
  cp .env.example "$dir/.env.example"
  cp deploy/xboard.service deploy/nginx.conf.example "$dir/deploy/"
  cp DEPLOY.md GO-LIVE.md "$dir/"
  # 在 Windows 上检出的文件可能带 CRLF：发布包里的配置 / 部署文件必须是 LF，
  # 否则 .env 的值会带上回车符、systemd 单元与脚本在 Linux 上出错。
  for f in "$dir/.env.example" "$dir"/*.md "$dir"/deploy/*; do
    tr -d '\r' < "$f" > "$f.tmp" && mv "$f.tmp" "$f"
  done
  # 在 NTFS / Windows 上构建时文件没有可执行位，所以用 python 显式写入权限：
  # 二进制 0755，其余文件 0644，属主 root:root。
  PY="$(command -v python3 || command -v python || true)"
  if [ -n "$PY" ]; then
    "$PY" - "$OUT" "$name" <<'PYEOF'
import os, sys, tarfile
out, name = sys.argv[1], sys.argv[2]
root = os.path.join(out, name)
with tarfile.open(os.path.join(out, name + ".tar.gz"), "w:gz") as tf:
    def add(path, arc, mode):
        ti = tf.gettarinfo(path, arc)
        ti.uid = ti.gid = 0
        ti.uname = ti.gname = "root"
        ti.mode = mode
        if os.path.isdir(path):
            tf.addfile(ti)
        else:
            with open(path, "rb") as f:
                tf.addfile(ti, f)
    add(root, name, 0o755)
    for dirpath, dirs, files in os.walk(root):
        for d in sorted(dirs):
            full = os.path.join(dirpath, d)
            add(full, os.path.relpath(full, out).replace(os.sep, "/"), 0o755)
        for f in sorted(files):
            full = os.path.join(dirpath, f)
            add(full, os.path.relpath(full, out).replace(os.sep, "/"), 0o755 if f == "xboard" else 0o644)
PYEOF
  else
    tar -C "$OUT" --mode=755 -czf "$OUT/$name.tar.gz" "$name"
  fi
  ( cd "$OUT" && sha256sum "$name.tar.gz" >> SHA256SUMS )
  rm -rf "$dir"
done

echo "==> 完成："
ls -lh "$OUT"
echo
echo "测试通过后，手动上传到 Cloudflare R2："
echo "  bash scripts/upload-r2.sh check     # 先测试凭据与读写权限"
echo "  bash scripts/upload-r2.sh upload    # 再上传（会让你确认）"
