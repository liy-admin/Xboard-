"""最小 SMTP 接收端：把每封邮件原样写到目录里（<seq>.eml），并在文件头加一行 X-Rcpt。"""
import os
import socketserver
import sys
import threading

OUT = sys.argv[2]
PORT = int(sys.argv[1])
os.makedirs(OUT, exist_ok=True)
lock = threading.Lock()
seq = [0]


class H(socketserver.StreamRequestHandler):
    def send(self, line):
        self.wfile.write((line + "\r\n").encode())
        self.wfile.flush()

    def handle(self):
        self.send("220 sink ESMTP")
        rcpts = []
        while True:
            raw = self.rfile.readline()
            if not raw:
                return
            line = raw.decode(errors="replace").strip()
            up = line.upper()
            if up.startswith("EHLO"):
                self.wfile.write(b"250-sink\r\n250 8BITMIME\r\n")
                self.wfile.flush()
            elif up.startswith("HELO"):
                self.send("250 sink")
            elif up.startswith("MAIL"):
                rcpts = []
                self.send("250 OK")
            elif up.startswith("RCPT"):
                rcpts.append(line.split(":", 1)[1].strip().strip("<>"))
                self.send("250 OK")
            elif up == "DATA":
                self.send("354 End data with <CR><LF>.<CR><LF>")
                data = []
                while True:
                    l = self.rfile.readline()
                    if not l or l in (b".\r\n", b".\n"):
                        break
                    data.append(l)
                with lock:
                    seq[0] += 1
                    n = seq[0]
                with open(os.path.join(OUT, "%04d.eml" % n), "wb") as f:
                    f.write(("X-Rcpt: %s\r\n" % ",".join(rcpts)).encode())
                    f.write(b"".join(data))
                self.send("250 OK queued")
            elif up == "QUIT":
                self.send("221 bye")
                return
            else:
                self.send("250 OK")


class S(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True


S(("127.0.0.1", PORT), H).serve_forever()
