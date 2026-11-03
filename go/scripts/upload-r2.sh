#!/usr/bin/env bash
# 手动把发布包上传到 Cloudflare R2（S3 兼容接口）。
#
# 只有你手动运行时才会上传；scripts/build-release.sh 不会调用它。
#
#   bash scripts/upload-r2.sh check       测试：凭据、桶、读写权限是否正常（只写入并删除一个探测文件）
#   bash scripts/upload-r2.sh upload      上传 dist/ 里的发布包：先校验 SHA256SUMS，上传前让你确认，上传后回查大小
#   bash scripts/upload-r2.sh upload -y   同上，跳过确认（非交互环境必须加）
#   bash scripts/upload-r2.sh list        列出桶里该目录下已有的文件
#
# 配置：环境变量，或写进 scripts/.r2.env（参考 scripts/r2.env.example；该文件已被 .gitignore 忽略）
#   R2_ACCOUNT_ID         账号 ID（端点是 https://<账号ID>.r2.cloudflarestorage.com）
#   R2_BUCKET             桶名
#   R2_ACCESS_KEY_ID      R2 API 令牌的 Access Key ID
#   R2_SECRET_ACCESS_KEY  R2 API 令牌的 Secret Access Key
#   可选：R2_PREFIX（桶内目录，缺省 xboard）、R2_ENDPOINT（直接指定端点，优先于账号 ID）、DIST_DIR（缺省 ./dist）
#
# 依赖：curl >= 7.75（--aws-sigv4）、sha256sum 或 shasum、md5sum 或 md5。
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # .../go

if [ -f "$HERE/scripts/.r2.env" ]; then
  set -a
  # shellcheck disable=SC1091
  . "$HERE/scripts/.r2.env"
  set +a
fi

die() { echo "错误：$*" >&2; exit 1; }

usage() {
  sed -n '2,18p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
  exit "${1:-2}"
}

CMD="${1:-}"
[ -n "$CMD" ] || usage
shift || true
case "$CMD" in
  check|upload|list) ;;
  -h|--help|help) usage 0 ;;
  *) echo "未知命令：$CMD" >&2; usage ;;
esac

# ---- 配置检查 ----
missing=()
for v in R2_BUCKET R2_ACCESS_KEY_ID R2_SECRET_ACCESS_KEY; do
  [ -n "${!v:-}" ] || missing+=("$v")
done
if [ -z "${R2_ENDPOINT:-}" ]; then
  if [ -n "${R2_ACCOUNT_ID:-}" ]; then
    R2_ENDPOINT="https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"
  else
    missing+=("R2_ACCOUNT_ID（或 R2_ENDPOINT）")
  fi
fi
if [ "${#missing[@]}" -gt 0 ]; then
  echo "缺少配置：${missing[*]}" >&2
  echo "请设置环境变量，或复制 scripts/r2.env.example 为 scripts/.r2.env 后填写。" >&2
  exit 1
fi
R2_ENDPOINT="${R2_ENDPOINT%/}"
PREFIX="${R2_PREFIX-xboard}"
PREFIX="${PREFIX#/}"; PREFIX="${PREFIX%/}"
DIST="${DIST_DIR:-$HERE/dist}"

# ---- 依赖检查 ----
command -v curl >/dev/null || die "需要 curl"
cv="$(curl --version | head -1 | awk '{print $2}')"
if [ "$(printf '%s\n%s\n' 7.75.0 "$cv" | sort -V | head -1)" != "7.75.0" ]; then
  die "curl $cv 太旧，需要 7.75 或更高（用于 --aws-sigv4 签名）"
fi

if command -v sha256sum >/dev/null; then
  sha_check() { sha256sum -c "$1"; }
  sha_of() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null; then
  sha_check() { shasum -a 256 -c "$1"; }
  sha_of() { shasum -a 256 "$1" | awk '{print $1}'; }
else
  die "需要 sha256sum 或 shasum"
fi
if command -v md5sum >/dev/null; then
  md5_of() { md5sum "$1" | awk '{print $1}'; }
elif command -v md5 >/dev/null; then
  md5_of() { md5 -q "$1"; }
else
  md5_of() { echo ""; }   # 没有 md5 工具时跳过 ETag 对比，只比对大小
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# 带 SigV4 签名的 curl。凭据通过 stdin 的配置传入，不会出现在进程列表或日志里。
s3curl() {
  curl -sS --aws-sigv4 "aws:amz:auto:s3" -K - "$@" \
    <<<"user = \"${R2_ACCESS_KEY_ID}:${R2_SECRET_ACCESS_KEY}\""
}

obj_url() { printf '%s/%s/%s' "$R2_ENDPOINT" "$R2_BUCKET" "$1"; }
key_of() { if [ -n "$PREFIX" ]; then printf '%s/%s' "$PREFIX" "$1"; else printf '%s' "$1"; fi; }

# xml_field <文件> <标签>：取 S3 错误响应里的字段
xml_field() { sed -n "s:.*<$2>\([^<]*\)</$2>.*:\1:p" "$1" | head -1; }

# explain <状态码> <响应体文件> <操作描述>：2xx 返回 0，否则打印原因并返回 1
explain() {
  local code="$1" body="$2" what="$3" ecode emsg
  case "$code" in 2??) return 0 ;; esac
  ecode="$(xml_field "$body" Code)"; emsg="$(xml_field "$body" Message)"
  echo "  ✗ $what失败（HTTP $code${ecode:+，$ecode}）" >&2
  case "$ecode" in
    SignatureDoesNotMatch) echo "    Secret Access Key 不对，或与 Access Key ID 不是同一个令牌。" >&2 ;;
    InvalidAccessKeyId) echo "    Access Key ID 不存在或已删除。" >&2 ;;
    AccessDenied) echo "    令牌没有该桶或该操作的权限（上传需要「对象读和写」）。" >&2 ;;
    NoSuchBucket) echo "    桶名不对，或令牌所属账号下没有这个桶。" >&2 ;;
    *) [ -z "$emsg" ] || echo "    $emsg" >&2 ;;
  esac
  case "$code" in
    403) [ -n "$ecode" ] || echo "    凭据无效，或令牌没有权限。" >&2 ;;
    404) [ -n "$ecode" ] || echo "    桶或对象不存在。" >&2 ;;
  esac
  return 1
}

# 发送请求并返回 HTTP 状态码；连不上时报网络错误。
# http <响应体文件> <curl 参数...>
http() {
  local out="$1"; shift
  s3curl -o "$out" -w '%{http_code}' "$@" || die "无法连接 $R2_ENDPOINT（检查网络 / 代理 / 端点地址）"
}

cmd_check() {
  echo "端点：$R2_ENDPOINT"
  echo "桶：$R2_BUCKET    目录：${PREFIX:-（桶根）}/"
  local code probe n
  code="$(http "$TMP/body" "$R2_ENDPOINT/$R2_BUCKET?list-type=2&max-keys=1000&prefix=${PREFIX:+$PREFIX/}")"
  explain "$code" "$TMP/body" "读取（列出文件）" || exit 1
  # 空桶时 grep 无匹配会返回 1，在 pipefail 下会让脚本悄悄退出，所以这里容错
  n="$({ grep -o '<Key>' "$TMP/body" || true; } | wc -l | tr -d ' ')"
  echo "  ✓ 读取正常（该目录下已有 $n 个文件）"

  probe="$(key_of ".r2-check-$$")"
  code="$(http "$TMP/body" -X PUT --data-binary "r2 check" -H "Content-Type: text/plain" -H "Expect:" "$(obj_url "$probe")")"
  explain "$code" "$TMP/body" "写入（上传探测文件）" || exit 1
  echo "  ✓ 写入正常"

  code="$(http "$TMP/body" -X DELETE "$(obj_url "$probe")")"
  if explain "$code" "$TMP/body" "删除探测文件"; then
    echo "  ✓ 删除正常"
  else
    echo "    探测文件 $probe 没能删除，可手动清理；不影响上传。" >&2
  fi
  echo "测试通过：可以执行 bash scripts/upload-r2.sh upload"
}

cmd_list() {
  local code
  code="$(http "$TMP/body" "$R2_ENDPOINT/$R2_BUCKET?list-type=2&max-keys=1000&prefix=${PREFIX:+$PREFIX/}")"
  explain "$code" "$TMP/body" "列出文件" || exit 1
  # 每个 <Contents> 一行：大小  修改时间  键
  tr -d '\n' <"$TMP/body" | sed 's:<Contents>:\n:g' | sed '1d' | while IFS= read -r blk || [ -n "$blk" ]; do  # 最后一行没有换行符，也要处理
    k="$(printf '%s' "$blk" | sed -n 's:.*<Key>\([^<]*\)</Key>.*:\1:p')"
    s="$(printf '%s' "$blk" | sed -n 's:.*<Size>\([^<]*\)</Size>.*:\1:p')"
    m="$(printf '%s' "$blk" | sed -n 's:.*<LastModified>\([^<]*\)</LastModified>.*:\1:p')"
    printf '%12s  %s  %s\n' "$s" "$m" "$k"
  done
}

fsize() { wc -c <"$1" | tr -d ' '; }

cmd_upload() {
  local yes=0 a
  for a in "$@"; do
    case "$a" in
      -y|--yes) yes=1 ;;
      *) die "upload 不认识的参数：$a" ;;
    esac
  done

  [ -d "$DIST" ] || die "找不到 $DIST，请先运行 bash scripts/build-release.sh"
  local tars=() f
  for f in "$DIST"/*.tar.gz; do [ -f "$f" ] && tars+=("$f"); done
  [ "${#tars[@]}" -gt 0 ] || die "$DIST 里没有发布包（*.tar.gz），请先运行 bash scripts/build-release.sh"
  [ -f "$DIST/SHA256SUMS" ] || die "$DIST 里没有 SHA256SUMS，请重新运行 bash scripts/build-release.sh"

  echo "==> 校验本地文件（SHA256SUMS）"
  ( cd "$DIST" && sha_check SHA256SUMS ) || die "本地文件与 SHA256SUMS 不一致，已停止，没有上传任何内容"

  # 先传发布包，最后传 SHA256SUMS，保证校验文件指向的对象一定已经存在
  local files=("${tars[@]}" "$DIST/SHA256SUMS")
  echo
  echo "即将上传到 $R2_BUCKET（$R2_ENDPOINT）："
  for f in "${files[@]}"; do
    printf '  %-48s %10s 字节  ->  %s\n' "$(basename "$f")" "$(fsize "$f")" "$(key_of "$(basename "$f")")"
  done
  echo "同名文件会被覆盖。"
  if [ "$yes" -ne 1 ]; then
    [ -t 0 ] || die "非交互环境请加 -y"
    local ans
    read -r -p "确认上传？输入 y 继续：" ans
    [ "$ans" = "y" ] || [ "$ans" = "Y" ] || { echo "已取消，没有上传任何内容。"; exit 1; }
  fi

  local base key ct code size md5 etag hdr
  for f in "${files[@]}"; do
    base="$(basename "$f")"; key="$(key_of "$base")"
    case "$base" in *.tar.gz) ct="application/gzip" ;; *) ct="text/plain" ;; esac
    echo "==> 上传 $base"
    code="$(http "$TMP/body" -T "$f" -H "Content-Type: $ct" -H "Expect:" --retry 3 --retry-delay 2 "$(obj_url "$key")")"
    explain "$code" "$TMP/body" "上传 $base" || exit 1

    # 回查：大小必须一致；单段上传的 ETag 是 MD5，能对上就一起校验
    hdr="$TMP/head"
    code="$(http "$hdr" -I "$(obj_url "$key")")"
    explain "$code" "$hdr" "回查 $base" || exit 1
    size="$(tr -d '\r' <"$hdr" | awk 'tolower($1)=="content-length:" {print $2}' | tail -1)"
    [ "$size" = "$(fsize "$f")" ] || die "回查失败：$base 本地 $(fsize "$f") 字节，桶里 ${size:-未知} 字节"
    etag="$(tr -d '\r"' <"$hdr" | awk 'tolower($1)=="etag:" {print $2}' | tail -1)"
    md5="$(md5_of "$f")"
    if [ -n "$md5" ] && printf '%s' "$etag" | grep -Eq '^[0-9a-fA-F]{32}$'; then
      [ "$(printf '%s' "$etag" | tr 'A-F' 'a-f')" = "$md5" ] || die "回查失败：$base 的 MD5 与桶里的 ETag 不一致"
      echo "    ✓ 已上传并校验（大小、MD5 一致）"
    else
      echo "    ✓ 已上传并校验（大小一致）"
    fi
  done
  echo
  echo "完成：${#files[@]} 个文件已上传到 s3://$R2_BUCKET/${PREFIX:+$PREFIX/}"
}

case "$CMD" in
  check)  cmd_check ;;
  list)   cmd_list ;;
  upload) cmd_upload "$@" ;;
esac
