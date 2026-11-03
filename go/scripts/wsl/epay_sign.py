"""按易支付规则签名并输出回调查询串：epay_sign.py <key> k=v k=v ..."""
import hashlib
import sys
import urllib.parse

key = sys.argv[1]
p = dict(a.split("=", 1) for a in sys.argv[2:])
raw = "&".join(f"{k}={p[k]}" for k in sorted(p) if p[k] != "" and k not in ("sign", "sign_type"))
p["sign"] = hashlib.md5((raw + key).encode()).hexdigest()
p["sign_type"] = "MD5"
print(urllib.parse.urlencode(p))
