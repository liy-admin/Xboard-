"""WebSocket 测试客户端。

用法：ws_client.py <port> <query> <seconds> [hold]
  query    例如 client_token=XXXX 或 node_id=1&token=YYYY
  seconds  最长读取时间；到时间后输出已收到的帧并退出
每收到一个文本帧输出一行 JSON。
"""
import base64
import os
import socket
import struct
import sys
import time

port, query, seconds = int(sys.argv[1]), sys.argv[2], float(sys.argv[3])
s = socket.create_connection(("127.0.0.1", port), timeout=3)
key = base64.b64encode(os.urandom(16)).decode()
s.sendall((
    f"GET /ws?{query} HTTP/1.1\r\nHost: 127.0.0.1\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
    f"Sec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
).encode())
buf = b""
while b"\r\n\r\n" not in buf:
    c = s.recv(4096)
    if not c:
        print("HANDSHAKE-CLOSED")
        sys.exit(0)
    buf += c
head, _, buf = buf.partition(b"\r\n\r\n")
print(head.split(b"\r\n")[0].decode(), flush=True)

end = time.time() + seconds
s.settimeout(0.5)


def need(n):
    global buf
    while len(buf) < n:
        buf += s.recv(4096)


while time.time() < end:
    try:
        need(2)
        b2 = buf[1]
        n = b2 & 0x7F
        off = 2
        if n == 126:
            need(4)
            n = struct.unpack(">H", buf[2:4])[0]
            off = 4
        elif n == 127:
            need(10)
            n = struct.unpack(">Q", buf[2:10])[0]
            off = 10
        need(off + n)
        payload, buf = buf[off:off + n], buf[off + n:]
    except socket.timeout:
        continue
    except Exception:
        break
    print(payload.decode(errors="replace"), flush=True)
