<?php

namespace App\Services;

use App\Models\Server;
use App\Models\User;
use App\Protocols\SingBoxProbe;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

/**
 * 节点探测：测延迟 + 判国家。两种模式：
 *
 *  - singbox（最准确）：面板用 sing-box 核心实际跑一遍该节点，请求 IP 回显服务拿到
 *    「真实出口 IP」，对出口 IP 判国家、以请求耗时作为真实代理延迟。能正确识别中转
 *    节点的落地国（入口在国内、落地在日本的节点会被判为日本）。需面板宿主机装 sing-box。
 *
 *  - tcp（轻量兜底）：对 host 做 TCP 握手测延迟（多次取最低），国家优先从节点名解析
 *    （中转节点名里一般写落地国），解析不到再对 host 做 GeoIP（入口国，可能不准）。
 *
 * 返回：['ok'=>bool,'latency_ms'=>?int,'exit_ip'=>?string,'country_code'=>?string,'error'=>?string]
 */
class NodeProbeService
{
    public function __construct(
        private GeoIpService $geoip,
        private SingBoxBinaryService $singbox,
    ) {
    }

    public function probe(Server $node): array
    {
        $mode = admin_setting('node_probe_mode', 'singbox');
        if ($mode === 'singbox') {
            $result = $this->probeViaSingbox($node);
            // 绝不回退到 TCP 握手：TCP 通只能说明端口开着，不能说明这个节点真的能用
            // （uuid/密码错、协议/传输不匹配、证书失效都照样握手成功），回退会把
            // "链接其实用不了"的节点误判成连通——假阳性。真连测不通就是不可达。
            // 仅在国家为空时用节点名补一下，纯展示用，不参与可达判定。
            if (!$result['ok'] && empty($result['country_code'])) {
                $result['country_code'] = ServerService::parseCountryFromName($node->name);
            }
            return $result;
        }
        return $this->probeViaTcp($node);
    }

    /**
     * 诊断：检查 sing-box 是否就绪（可自动下载/能运行），并对一个样本节点实跑一次，
     * 返回可读的成功/失败信息。供管理端「测试 sing-box」按钮用。
     */
    public function diagnose(?Server $node): array
    {
        $mode = (string) admin_setting('node_probe_mode', 'singbox');

        $binaryInfo = ['ok' => false, 'path' => null, 'version' => null, 'error' => null];
        if (!function_exists('proc_open')) {
            $binaryInfo['error'] = 'proc_open 被禁用，无法启动 sing-box';
        } else {
            $binary = $this->singbox->resolve();
            if ($binary) {
                $binaryInfo['ok'] = true;
                $binaryInfo['path'] = $binary;
                $out = @shell_exec(escapeshellarg($binary) . ' version 2>&1');
                if (is_string($out) && preg_match('/sing-box version (\S+)/', $out, $m)) {
                    $binaryInfo['version'] = $m[1];
                }
            } else {
                $binaryInfo['error'] = 'sing-box 不可用：自动下载失败（容器可能无法访问 GitHub），或未安装。可在下方填自定义路径。';
            }
        }

        $sample = null;
        if ($node) {
            $r = $this->probe($node);
            $sample = [
                'node_id' => $node->id,
                'node_name' => $node->name,
                'host' => $node->host,
                'type' => $node->type,
                'ok' => (bool) ($r['ok'] ?? false),
                'latency_ms' => $r['latency_ms'] ?? null,
                'exit_ip' => $r['exit_ip'] ?? null,
                'country_code' => $r['country_code'] ?? null,
                'tests' => $r['tests'] ?? null,
                'success_tests' => $r['success_tests'] ?? null,
                'untestable' => (bool) ($r['untestable'] ?? false),
                'error' => $r['error'] ?? null,
            ];
        }

        return ['mode' => $mode, 'singbox' => $binaryInfo, 'sample' => $sample];
    }

    // ---------------------------------------------------------------------
    // 轻量模式：TCP 握手 + 名字/GeoIP 判国家
    // ---------------------------------------------------------------------

    private function probeViaTcp(Server $node): array
    {
        $timeoutMs = $this->timeoutMs();
        $transport = $node->protocolTransport();

        $latency = $transport === 'udp'
            ? $this->pingHost($node->host, $timeoutMs)
            : $this->tcpConnectBest($node->host, $this->firstPort($node->port), $timeoutMs);

        // 国家：先从节点名解析（中转节点落地国通常写在名字里），再 GeoIP(host) 兜底
        $cc = ServerService::parseCountryFromName($node->name);
        if (!$cc) {
            $cc = $this->geoip->lookupCountries([$node->host])[$node->host] ?? null;
        }

        return [
            'ok' => $latency !== null,
            'latency_ms' => $latency,
            'min_latency' => $latency,
            'avg_latency' => $latency,
            'median_latency' => $latency,
            'max_latency' => $latency,
            'tests' => 1,
            'success_tests' => $latency !== null ? 1 : 0,
            'exit_ip' => null,
            'country_code' => $cc,
            'error' => $latency === null ? 'tcp/ping 探测失败' : null,
            'untestable' => false,
        ];
    }

    /** TCP 握手测延迟，连 3 次取最低，避免「首包慢」误判。失败返回 null。 */
    private function tcpConnectBest(string $host, int $port, int $timeoutMs): ?int
    {
        if ($port <= 0) {
            return null;
        }
        $best = null;
        for ($i = 0; $i < 3; $i++) {
            $start = microtime(true);
            $errno = 0;
            $errstr = '';
            $conn = @stream_socket_client("tcp://{$host}:{$port}", $errno, $errstr, $timeoutMs / 1000, STREAM_CLIENT_CONNECT);
            if ($conn !== false) {
                $ms = (int) round((microtime(true) - $start) * 1000);
                $best = $best === null ? $ms : min($best, $ms);
                fclose($conn);
            }
            usleep(150_000);
        }
        return $best;
    }

    /** ICMP ping 测延迟（Linux 风格）；不可用返回 null。 */
    private function pingHost(string $host, int $timeoutMs): ?int
    {
        if (!$this->execUsable()) {
            return null;
        }
        $waitSec = max(1, (int) ceil($timeoutMs / 1000));
        $cmd = sprintf('ping -c 1 -W %d %s 2>/dev/null', $waitSec, escapeshellarg($host));
        $output = [];
        $code = 0;
        @exec($cmd, $output, $code);
        if ($code !== 0) {
            return null;
        }
        foreach ($output as $line) {
            if (preg_match('/time[=<]\s*([\d.]+)\s*ms/i', $line, $m)) {
                return (int) round((float) $m[1]);
            }
        }
        return null;
    }

    // ---------------------------------------------------------------------
    // 精确模式：sing-box 实跑 + 真实出口 IP
    // ---------------------------------------------------------------------

    private function probeViaSingbox(Server $node): array
    {
        // $untestable=true 表示「根本没测成」（协议不支持 / 该组没有可用用户 /
        // sing-box 起不来等基础设施问题），不是节点本身不通——调用方据此保留上次
        // 的测速结果，而不是把节点判成不可达（否则会误伤能用的节点）。
        // 只有真正测了却不通（$untestable=false）才算节点不可达。
        $fail = fn(string $err, bool $untestable = true) => [
            'ok' => false, 'latency_ms' => null,
            'min_latency' => null, 'avg_latency' => null, 'median_latency' => null, 'max_latency' => null,
            'tests' => (int) admin_setting('node_probe_test_count', 3), 'success_tests' => 0,
            'exit_ip' => null, 'country_code' => null, 'error' => $err, 'untestable' => $untestable,
        ];

        if (!function_exists('proc_open')) {
            return $fail('proc_open 不可用，无法启动 sing-box');
        }
        // 自动解析/下载 sing-box 二进制（宿主机没装就从官方下载并缓存）
        $binary = $this->singbox->resolve();
        if (!$binary) {
            return $fail('sing-box 不可用（自动下载失败，可在后台填自定义路径）');
        }

        // 1) 取一个有权限访问该节点的真实用户，复用订阅那套逻辑生成 outbound
        $outbound = $this->buildOutbound($node);
        if (!$outbound) {
            return $fail('无法为该节点生成 sing-box outbound（无可用用户或协议不受支持）');
        }
        $outbound['tag'] = 'proxy';

        // 预解析服务器域名为 IP 填回 server：否则 sing-box 要先解析该域名才能连节点，
        // 而 DNS 查询在 route.final=proxy 下会被塞进「还没连上的节点」，造成死锁超时。
        // 域名改成 IP 后，连接节点时不再需要 DNS。若节点是 TLS 且没显式 SNI，则把原
        // 域名补到 tls.server_name，避免 SNI 变成 IP 导致握手失败。
        $origServer = $outbound['server'] ?? $node->host;
        if (!filter_var($origServer, FILTER_VALIDATE_IP)) {
            $ip = @gethostbyname($origServer);
            if ($ip && $ip !== $origServer && filter_var($ip, FILTER_VALIDATE_IP)) {
                $outbound['server'] = $ip;
                if (isset($outbound['tls']) && is_array($outbound['tls']) && empty($outbound['tls']['server_name'])) {
                    $outbound['tls']['server_name'] = $origServer;
                }
            }
        }

        // 2) 分配本地端口 + 写临时配置
        $port = $this->freeLocalPort();
        if (!$port) {
            return $fail('无法分配本地端口');
        }
        // dns 块必须有：socks5h 会把回显域名（ip-api.com）交给 sing-box 解析，没有
        // dns 服务器就无法解析。server 已是 IP，连节点不触发 DNS，所以这里不设 detour：
        // 回显域名在节点连上后经代理解析（也正好反映出口侧）。
        $dnsServer = (string) admin_setting('node_probe_dns', '1.1.1.1');
        // 关键：outbounds 里**只有被测节点**，没有 direct。这样代理握手/TLS/DNS 失败时
        // 请求直接失败，绝不会偷偷走直连造成「虚假成功」（真连测速的核心要求）。
        $config = [
            'log' => ['level' => 'error'],
            'dns' => ['servers' => [['type' => 'udp', 'tag' => 'r', 'server' => $dnsServer]]],
            'inbounds' => [[
                'type' => 'mixed',
                'tag' => 'in',
                'listen' => '127.0.0.1',
                'listen_port' => $port,
            ]],
            'outbounds' => [$outbound],
            'route' => ['final' => 'proxy'],
        ];
        $cfgPath = tempnam(sys_get_temp_dir(), 'sbprobe_') . '.json';
        file_put_contents($cfgPath, json_encode($config, JSON_UNESCAPED_UNICODE));

        $descriptors = [0 => ['pipe', 'r'], 1 => ['pipe', 'w'], 2 => ['pipe', 'w']];
        $proc = @proc_open([$binary, 'run', '-c', $cfgPath], $descriptors, $pipes);
        if (!is_resource($proc)) {
            @unlink($cfgPath);
            return $fail("无法启动 sing-box（检查 node_probe_singbox_path：{$binary}）");
        }
        foreach ([1, 2] as $i) {
            if (isset($pipes[$i])) {
                stream_set_blocking($pipes[$i]);
            }
        }

        try {
            // 3) 等本地入站端口就绪
            if (!$this->waitPort($port, 6000)) {
                $stderr = isset($pipes[2]) ? stream_get_contents($pipes[2]) : '';
                return $fail('sing-box 入站端口未就绪：' . trim(mb_substr($stderr, 0, 300)));
            }

            $proxy = "socks5h://127.0.0.1:{$port}";
            $reqTimeoutSec = max(2, (int) ceil(((int) admin_setting('node_probe_request_timeout_ms', 10000)) / 1000));
            $latUrl = (string) admin_setting('node_probe_latency_url', 'https://www.gstatic.com/generate_204');
            $testCount = max(1, min((int) admin_setting('node_probe_test_count', 3), 10));

            // 4) 真连延迟：通过本地代理把轻量请求（generate_204）经被测节点发出去，连测
            // 多次。每次都从「开始发请求」计时到「收到有效响应」，不含 Core 启动时间。
            $latencies = [];
            $lastErr = null;
            for ($i = 0; $i < $testCount; $i++) {
                $t0 = microtime(true);
                try {
                    $resp = Http::withOptions(['proxy' => $proxy, 'timeout' => $reqTimeoutSec])->get($latUrl);
                    if ($resp->status() < 400) {
                        $latencies[] = (int) round((microtime(true) - $t0) * 1000);
                    } else {
                        $lastErr = 'HTTP ' . $resp->status();
                    }
                } catch (\Throwable $e) {
                    // 代理握手 / TLS / DNS / 超时失败 —— 记为该次失败（不会走直连，见上方配置）
                    $lastErr = $e->getMessage();
                }
            }

            $successTests = count($latencies);
            if ($successTests === 0) {
                // 真的测了、但一次都没通 —— 这才是节点不可达
                return $fail('真连测速失败：' . ($lastErr ?? '无成功请求'), false);
            }
            $stats = $this->latencyStats($latencies);

            // 5) 出口 IP + 国家：best-effort，失败不影响「可达」结论（国家回退到节点名解析）
            $exitIp = null;
            $cc = null;
            try {
                $echoUrl = (string) admin_setting('node_probe_ip_echo_url', 'http://ip-api.com/json/?fields=status,countryCode,query');
                $echo = Http::withOptions(['proxy' => $proxy, 'timeout' => $reqTimeoutSec])->get($echoUrl);
                if ($echo->ok()) {
                    [$exitIp, $cc] = $this->parseEcho($echo->json(), $echo->body());
                }
            } catch (\Throwable $e) {
                // 忽略：出口信息拿不到不影响可达判定
            }
            if (!$cc && $exitIp) {
                $cc = $this->geoip->lookupCountries([$exitIp])[$exitIp] ?? null;
            }
            if (!$cc) {
                $cc = ServerService::parseCountryFromName($node->name);
            }

            return [
                'ok' => true,
                'latency_ms' => $stats['min'], // 头部「真连延迟」用最优值
                'min_latency' => $stats['min'],
                'avg_latency' => $stats['avg'],
                'median_latency' => $stats['median'],
                'max_latency' => $stats['max'],
                'tests' => $testCount,
                'success_tests' => $successTests,
                'exit_ip' => $exitIp,
                'country_code' => $cc ? strtoupper($cc) : null,
                'error' => null,
            ];
        } catch (\Throwable $e) {
            return $fail('sing-box 探测异常：' . $e->getMessage());
        } finally {
            foreach ($pipes as $p) {
                if (is_resource($p)) {
                    fclose($p);
                }
            }
            if (is_resource($proc)) {
                @proc_terminate($proc);
                @proc_close($proc);
            }
            @unlink($cfgPath);
        }
    }

    /** 复用订阅逻辑：取可用用户 -> getAvailableServers -> 生成单节点 outbound。 */
    private function buildOutbound(Server $node): ?array
    {
        $groupIds = $node->group_ids ?? [];
        if (empty($groupIds)) {
            return null;
        }
        $user = User::whereIn('group_id', $groupIds)
            ->where('banned', 0)
            ->where(function ($q) {
                $q->whereNull('expired_at')->orWhere('expired_at', '>=', time());
            })
            ->first();
        if (!$user) {
            return null;
        }

        $servers = ServerService::getAvailableServers($user);
        $target = collect($servers)->firstWhere('id', $node->id);
        if (!$target) {
            return null;
        }

        try {
            return (new SingBoxProbe($user, [$target]))->buildSingleOutbound();
        } catch (\Throwable $e) {
            Log::warning('NodeProbeService: 生成 outbound 失败', ['node' => $node->id, 'message' => $e->getMessage()]);
            return null;
        }
    }

    /** 解析 IP 回显响应：返回 [exitIp, countryCode|null]。 */
    private function parseEcho(?array $json, string $body): array
    {
        if (is_array($json)) {
            // ip-api.com: {status, countryCode, query}
            $ip = $json['query'] ?? $json['ip'] ?? ($json['YourFuckingIPAddress'] ?? null);
            $cc = $json['countryCode'] ?? $json['country_code'] ?? ($json['country'] ?? null);
            if ($ip) {
                return [$ip, is_string($cc) && strlen($cc) === 2 ? $cc : null];
            }
        }
        // 纯文本回显（只有 IP）
        $body = trim($body);
        if (filter_var($body, FILTER_VALIDATE_IP)) {
            return [$body, null];
        }
        return [null, null];
    }

    // ---------------------------------------------------------------------
    // 工具
    // ---------------------------------------------------------------------

    private function timeoutMs(): int
    {
        $t = (int) admin_setting('server_latency_timeout_ms', 2000);
        return max(200, min($t, 10000));
    }

    /** 从若干次延迟样本算 min/avg/median/max（毫秒，整数）。 */
    private function latencyStats(array $latencies): array
    {
        sort($latencies);
        $n = count($latencies);
        $median = $n % 2
            ? $latencies[intdiv($n, 2)]
            : (int) round(($latencies[$n / 2 - 1] + $latencies[$n / 2]) / 2);
        return [
            'min' => $latencies[0],
            'max' => $latencies[$n - 1],
            'avg' => (int) round(array_sum($latencies) / $n),
            'median' => (int) $median,
        ];
    }

    private function execUsable(): bool
    {
        if (!function_exists('exec')) {
            return false;
        }
        $disabled = array_map('trim', explode(',', (string) ini_get('disable_functions')));
        return !in_array('exec', $disabled, true);
    }

    private function firstPort(string|int|null $port): int
    {
        if ($port === null) {
            return 0;
        }
        $port = (string) $port;
        if (str_contains($port, '-')) {
            $port = explode('-', $port)[0];
        }
        return (int) trim($port);
    }

    /** 让系统分配一个空闲本地端口。 */
    private function freeLocalPort(): ?int
    {
        $sock = @stream_socket_server('tcp://127.0.0.1:0', $errno, $errstr);
        if (!$sock) {
            return null;
        }
        $name = stream_socket_get_name($sock);
        fclose($sock);
        $port = (int) substr($name, strrpos($name, ':') + 1);
        return $port ?: null;
    }

    /** 轮询等待本地端口可连接。 */
    private function waitPort(int $port, int $timeoutMs): bool
    {
        $deadline = microtime(true) + $timeoutMs / 1000;
        while (microtime(true) < $deadline) {
            $c = @stream_socket_client("tcp://127.0.0.1:{$port}", $errno, $errstr, 0.3);
            if ($c !== false) {
                fclose($c);
                return true;
            }
            usleep(200_000);
        }
        return false;
    }
}
