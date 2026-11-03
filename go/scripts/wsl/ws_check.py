import base64
import os
import socket
import struct
import sys

port = int(sys.argv[1])
s = socket.create_connection(("127.0.0.1", port), timeout=5)
key = base64.b64encode(os.urandom(16)).decode()
req = (
    "GET /ws?node_id=1&token=wrong HTTP/1.1\r\n"
    "Host: 127.0.0.1\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
    f"Sec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
)
s.sendall(req.encode())
buf = b""
while b"\r\n\r\n" not in buf:
    chunk = s.recv(4096)
    if not chunk:
        break
    buf += chunk
head, _, rest = buf.partition(b"\r\n\r\n")
print(head.split(b"\r\n")[0].decode())


def recv_more(sock, pending):
    chunk = sock.recv(4096)
    if not chunk:
        raise EOFError("connection closed by server")
    return pending + chunk


def read_frame(sock, pending):
    while len(pending) < 2:
        pending = recv_more(sock, pending)
    b1, b2 = pending[0], pending[1]
    n = b2 & 0x7F
    off = 2
    if n == 126:
        while len(pending) < 4:
            pending = recv_more(sock, pending)
        n = struct.unpack(">H", pending[2:4])[0]
        off = 4
    while len(pending) < off + n:
        pending = recv_more(sock, pending)
    return pending[off:off + n].decode(errors="replace")


try:
    print(read_frame(s, rest))
except Exception as e:  # 超时也输出，便于排查
    print("no-frame", e)
