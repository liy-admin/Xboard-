#!/bin/bash
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUTPUT="/d/D/vps/Xboard/xboard-deploy.tar.gz"

echo ">>> 编译前端..."
cd "$PROJECT_DIR/admin-src/source"
npm run build

echo ">>> 打包项目..."
cd "$PROJECT_DIR"
tar \
  --exclude='.git' \
  --exclude='admin-src' \
  --exclude='.docker/.data' \
  --exclude='_tmp*' \
  --exclude='.claude' \
  --exclude='compose*' \
  -czf "$OUTPUT" .

echo ">>> 完成: $OUTPUT ($(du -sh "$OUTPUT" | cut -f1))"
