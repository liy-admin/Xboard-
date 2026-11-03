import base64
import hashlib
import json
import sys

try:
    from cryptography.hazmat.primitives.ciphers.aead import AESGCM
except Exception as e:
    print("no-cryptography-module", e)
    sys.exit(0)

env = json.load(open(sys.argv[1]))
key = hashlib.sha256(sys.argv[2].encode()).digest()
nonce = base64.b64decode(env["n"])
ct = base64.b64decode(env["c"]) + base64.b64decode(env["t"])
print(AESGCM(key).decrypt(nonce, ct, None).decode())
