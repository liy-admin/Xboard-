# WSL 端到端验证

在 WSL（Ubuntu）中用私有 MariaDB（端口 33306）+ PHP 迁移建表 + Redis DB 15 验证 Go 二进制。只写 `~/xboard-verify` 与 Redis DB 15，不触碰已有数据库。

1. 在 Windows 交叉编译：`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o scripts/wsl/xboard-linux ./cmd/xboard`
2. WSL 内依次执行：`bash init.sh`（起库）、`bash migrate.sh`（PHP 迁移建表）、可选 `bash get_singbox.sh`（下载 sing-box 以验证真连探测）、`bash verify.sh`（跑检查，输出 PASS / FAIL）。
3. 依赖：WSL 里的 `mariadbd`、`redis-server`（已在运行）、`php`（含 vendor）、`python3`；`cryptography` 模块可选（缺失时跳过订阅解密校验）。

`verify.sh` 末尾的 `wait` 会因桩 http.server 子进程而不返回，PASS/FAIL 行已全部输出后可手动结束。
