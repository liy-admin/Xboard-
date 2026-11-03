#!/bin/bash
# 把官方 sing-box 下载到 ~/xboard-verify/bin（仅验证用，不安装到系统）。
set -e
D="$HOME/xboard-verify"
mkdir -p "$D/bin"
if [ -x "$D/bin/sing-box" ]; then "$D/bin/sing-box" version | head -1; exit 0; fi
TAG=$(curl -sSIL -m 20 -o /dev/null -w '%{url_effective}' https://github.com/SagerNet/sing-box/releases/latest | sed 's#.*/tag/##')
VER="${TAG#v}"
echo "latest: $TAG"
cd "$D/bin"
curl -sSL -m 180 -o sb.tgz "https://github.com/SagerNet/sing-box/releases/download/${TAG}/sing-box-${VER}-linux-amd64.tar.gz"
tar -xzf sb.tgz
cp sing-box-*/sing-box ./sing-box
chmod +x sing-box
rm -rf sb.tgz sing-box-*
./sing-box version | head -1
