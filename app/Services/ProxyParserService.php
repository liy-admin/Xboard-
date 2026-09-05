<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class ProxyParserService
{
    private const SCHEMES = ['vmess://', 'vless://', 'trojan://', 'ss://', 'hysteria2://', 'hy2://', 'hysteria://', 'tuic://'];

    /**
     * Parse proxy URIs from pasted text.
     */
    public function parseFromText(string $text): array
    {
        $lines = preg_split('/\r?\n/', trim($text));
        $servers = [];
        $errors  = [];

        foreach ($lines as $i => $line) {
            $line = trim($line);
            if ($line === '' || str_starts_with($line, '#')) {
                continue;
            }
            $server = $this->parseLine($line);
            if ($server === null) {
                $errors[] = ['line' => $i + 1, 'content' => mb_substr($line, 0, 80), 'error' => '无法识别的格式'];
            } else {
                $servers[] = $server;
            }
        }

        return ['servers' => $servers, 'errors' => $errors];
    }

    /**
     * Reject URLs pointing to private/loopback/link-local networks or non-HTTP schemes.
     */
    private function assertPublicUrl(string $url): void
    {
        $parsed = parse_url($url);
        if (!$parsed || !in_array(strtolower($parsed['scheme'] ?? ''), ['http', 'https'], true)) {
            throw new \InvalidArgumentException('订阅 URL 必须使用 http 或 https 协议');
        }
        $host = $parsed['host'] ?? '';
        if ($host === '') {
            throw new \InvalidArgumentException('订阅 URL 缺少主机名');
        }
        // Resolve hostname to IP for private range check
        $ip = filter_var($host, FILTER_VALIDATE_IP) ? $host : gethostbyname($host);
        if ($ip === $host && !filter_var($ip, FILTER_VALIDATE_IP)) {
            // gethostbyname returned the same string → resolution failed; allow (server may resolve differently)
            return;
        }
        if (filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_NO_PRIV_RANGE | FILTER_FLAG_NO_RES_RANGE) === false) {
            throw new \InvalidArgumentException('订阅 URL 不允许指向私有或保留地址');
        }
    }

    /**
     * Fetch a subscription URL and parse all proxy links inside.
     */
    public function parseFromSubscription(string $url): array
    {
        try {
            $this->assertPublicUrl($url);
            $response = Http::timeout(15)
                ->withHeaders(['User-Agent' => 'ClashForAndroid/2.5.12'])
                ->get($url);

            if (!$response->successful()) {
                return ['servers' => [], 'errors' => [['error' => "获取订阅失败: HTTP {$response->status()}"]]];
            }

            $body = $response->body();
            $body = $this->tryDecodeBase64($body);

            return $this->parseFromText($body);
        } catch (\Exception $e) {
            Log::error('ProxyParserService subscription fetch error', ['url' => $url, 'error' => $e->getMessage()]);
            return ['servers' => [], 'errors' => [['error' => '获取订阅失败: ' . $e->getMessage()]]];
        }
    }

    /**
     * If the body looks base64-encoded and decodes to proxy links, return decoded; otherwise original.
     */
    private function tryDecodeBase64(string $body): string
    {
        $trimmed = trim($body);
        // Only attempt if it looks like a single base64 blob (no line breaks containing ://)
        if (!str_contains($trimmed, '://')) {
            $decoded = base64_decode($trimmed, strict: false);
            if ($decoded !== false && str_contains($decoded, '://')) {
                return $decoded;
            }
        }
        return $body;
    }

    private function parseLine(string $line): ?array
    {
        return match (true) {
            str_starts_with($line, 'vmess://')     => $this->parseVmess($line),
            str_starts_with($line, 'vless://')     => $this->parseVless($line),
            str_starts_with($line, 'trojan://')    => $this->parseTrojan($line),
            str_starts_with($line, 'ss://')        => $this->parseSs($line),
            str_starts_with($line, 'hysteria2://') => $this->parseHysteria2($line),
            str_starts_with($line, 'hy2://')       => $this->parseHysteria2($line),
            str_starts_with($line, 'hysteria://')  => $this->parseHysteria($line),
            str_starts_with($line, 'tuic://')      => $this->parseTuic($line),
            default => null,
        };
    }

    // -------------------------------------------------------------------------
    // Per-protocol parsers
    // -------------------------------------------------------------------------

    private function parseVmess(string $link): ?array
    {
        try {
            $b64  = substr($link, strlen('vmess://'));
            $json = base64_decode($b64, strict: false);
            if (!$json) return null;

            $d = json_decode($json, true);
            if (!$d || empty($d['add'])) return null;

            $tls = (($d['tls'] ?? '') === 'tls') ? 1 : 0;
            $net = $d['net'] ?? 'tcp';

            return [
                'type'        => 'vmess',
                'code'        => $this->extractUuid($d['id'] ?? null),
                'name'        => $d['ps'] ?? $d['add'],
                'host'        => $d['add'],
                'port'        => (int) ($d['port'] ?? 443),
                'server_port' => (int) ($d['port'] ?? 443),
                'rate'        => 1,
                'protocol_settings' => [
                    'tls'              => $tls,
                    'network'          => $net,
                    'network_settings' => $this->buildNetworkSettings($net, $d),
                    'tls_settings'     => [
                        'server_name'    => ($d['sni'] ?? '') ?: (($d['host'] ?? '') ?: null),
                        'allow_insecure' => false,
                    ],
                ],
            ];
        } catch (\Exception) {
            return null;
        }
    }

    private function parseVless(string $link): ?array
    {
        try {
            // vless://uuid@host:port?params#name
            $url = parse_url($link);
            if (!$url || empty($url['host'])) return null;

            parse_str($url['query'] ?? '', $params);
            $name     = isset($url['fragment']) ? urldecode($url['fragment']) : $url['host'];
            $code     = $this->extractUuid(isset($url['user']) ? rawurldecode($url['user']) : null);
            $security = $params['security'] ?? 'none';
            $net      = $params['type'] ?? 'tcp';
            $tls      = in_array($security, ['tls', 'reality'], true) ? 1 : 0;
            $insecure = (($params['allowInsecure'] ?? $params['insecure'] ?? '0') === '1');

            $ps = [
                'tls'              => $tls,
                'network'          => $net,
                'network_settings' => $this->buildNetworkSettings($net, $params),
                'flow'             => $params['flow'] ?? null,
                'tls_settings'     => [
                    'server_name'    => $params['sni'] ?? null,
                    'allow_insecure' => $insecure,
                ],
            ];

            if ($security === 'reality') {
                $ps['reality_settings'] = [
                    'public_key'  => $params['pbk'] ?? null,
                    'short_id'    => $params['sid'] ?? null,
                    'server_name' => $params['sni'] ?? null,
                ];
            }

            return [
                'type'        => 'vless',
                'code'        => $code,
                'name'        => $name,
                'host'        => $url['host'],
                'port'        => (int) ($url['port'] ?? 443),
                'server_port' => (int) ($url['port'] ?? 443),
                'rate'        => 1,
                'protocol_settings' => $ps,
            ];
        } catch (\Exception) {
            return null;
        }
    }

    private function parseTrojan(string $link): ?array
    {
        try {
            $url = parse_url($link);
            if (!$url || empty($url['host'])) return null;

            parse_str($url['query'] ?? '', $params);
            $name     = isset($url['fragment']) ? urldecode($url['fragment']) : $url['host'];
            $net      = $params['type'] ?? 'tcp';
            $insecure = (($params['allowInsecure'] ?? $params['insecure'] ?? '0') === '1');
            $sni      = $params['sni'] ?? $params['peer'] ?? null;
            $code     = $this->extractUuid(isset($url['user']) ? rawurldecode($url['user']) : null);

            return [
                'type'        => 'trojan',
                'code'        => $code,
                'name'        => $name,
                'host'        => $url['host'],
                'port'        => (int) ($url['port'] ?? 443),
                'server_port' => (int) ($url['port'] ?? 443),
                'rate'        => 1,
                'protocol_settings' => [
                    'tls'              => 1,
                    'network'          => $net,
                    'network_settings' => $this->buildNetworkSettings($net, $params),
                    'server_name'      => $sni,
                    'allow_insecure'   => $insecure,
                    'tls_settings'     => [
                        'server_name'    => $sni,
                        'allow_insecure' => $insecure,
                    ],
                ],
            ];
        } catch (\Exception) {
            return null;
        }
    }

    private function parseSs(string $link): ?array
    {
        try {
            $raw      = substr($link, strlen('ss://'));
            $fragment = '';

            if (($pos = strpos($raw, '#')) !== false) {
                $fragment = urldecode(substr($raw, $pos + 1));
                $raw      = substr($raw, 0, $pos);
            }

            // SIP002: userinfo@host:port  (userinfo may be plain or base64)
            if (str_contains($raw, '@')) {
                $atPos    = strrpos($raw, '@');
                $userinfo = substr($raw, 0, $atPos);
                $hostport = substr($raw, $atPos + 1);

                // Attempt base64 decode of userinfo
                $decoded = base64_decode($userinfo, strict: false);
                if ($decoded !== false && str_contains($decoded, ':')) {
                    $userinfo = $decoded;
                }

                [$method, $password] = explode(':', $userinfo, 2);
            } else {
                // Legacy: base64(method:password@host:port)
                $decoded = base64_decode($raw, strict: false);
                if (!$decoded || !str_contains($decoded, '@')) return null;

                $atPos    = strrpos($decoded, '@');
                $userinfo = substr($decoded, 0, $atPos);
                $hostport = substr($decoded, $atPos + 1);
                [$method, $password] = explode(':', $userinfo, 2);
            }

            [$host, $port] = $this->splitHostPort($hostport);
            $name = $fragment ?: $host;

            return [
                'type'        => 'shadowsocks',
                'code'        => $this->extractUuid($password ?? null),
                'name'        => $name,
                'host'        => $host,
                'port'        => (int) $port,
                'server_port' => (int) $port,
                'rate'        => 1,
                'protocol_settings' => [
                    'cipher' => $method,
                ],
            ];
        } catch (\Exception) {
            return null;
        }
    }

    private function parseHysteria2(string $link): ?array
    {
        try {
            // Strip hy2:// alias
            if (str_starts_with($link, 'hy2://')) {
                $link = 'hysteria2://' . substr($link, strlen('hy2://'));
            }

            $url = parse_url($link);
            if (!$url || empty($url['host'])) return null;

            parse_str($url['query'] ?? '', $params);
            $name = isset($url['fragment']) ? urldecode($url['fragment']) : $url['host'];
            $port = $url['port'] ?? 443;

            // Port hopping: ?mport=4000-5000
            if (!empty($params['mport'])) {
                $port = $params['mport'];
            }

            $obfsOpen = isset($params['obfs']) && $params['obfs'] !== '';

            return [
                'type'        => 'hysteria',
                'code'        => $this->extractUuid(isset($url['user']) ? rawurldecode($url['user']) : null),
                'name'        => $name,
                'host'        => $url['host'],
                'port'        => $port,
                'server_port' => is_numeric($port) ? (int) $port : $port,
                'rate'        => 1,
                'protocol_settings' => [
                    'version' => 2,
                    'obfs'    => [
                        'open'     => $obfsOpen,
                        'type'     => $params['obfs'] ?? 'salamander',
                        'password' => $params['obfs-password'] ?? null,
                    ],
                    'tls' => [
                        'server_name'    => $params['sni'] ?? null,
                        'allow_insecure' => (($params['insecure'] ?? '0') === '1'),
                    ],
                ],
            ];
        } catch (\Exception) {
            return null;
        }
    }

    private function parseHysteria(string $link): ?array
    {
        try {
            $url = parse_url($link);
            if (!$url || empty($url['host'])) return null;

            parse_str($url['query'] ?? '', $params);
            $name = isset($url['fragment']) ? urldecode($url['fragment']) : $url['host'];

            return [
                'type'        => 'hysteria',
                'name'        => $name,
                'host'        => $url['host'],
                'port'        => (int) ($url['port'] ?? 443),
                'server_port' => (int) ($url['port'] ?? 443),
                'rate'        => 1,
                'protocol_settings' => [
                    'version'  => 1,
                    'bandwidth' => [
                        'up'   => isset($params['upmbps'])   ? (int) $params['upmbps']   : null,
                        'down' => isset($params['downmbps']) ? (int) $params['downmbps'] : null,
                    ],
                    'obfs' => [
                        'open'     => !empty($params['obfs']),
                        'type'     => 'salamander',
                        'password' => $params['obfs'] ?? null,
                    ],
                    'alpn' => $params['alpn'] ?? null,
                    'tls'  => [
                        'server_name'    => $params['peer'] ?? null,
                        'allow_insecure' => (($params['insecure'] ?? '0') === '1'),
                    ],
                ],
            ];
        } catch (\Exception) {
            return null;
        }
    }

    private function parseTuic(string $link): ?array
    {
        try {
            // tuic://uuid:password@host:port?params#name
            $url = parse_url($link);
            if (!$url || empty($url['host'])) return null;

            parse_str($url['query'] ?? '', $params);
            $name = isset($url['fragment']) ? urldecode($url['fragment']) : $url['host'];

            $alpn = isset($params['alpn']) ? explode(',', $params['alpn']) : ['h3'];

            return [
                'type'        => 'tuic',
                'code'        => $this->extractUuid(isset($url['user']) ? rawurldecode($url['user']) : null),
                'name'        => $name,
                'host'        => $url['host'],
                'port'        => (int) ($url['port'] ?? 443),
                'server_port' => (int) ($url['port'] ?? 443),
                'rate'        => 1,
                'protocol_settings' => [
                    'version'            => 5,
                    'congestion_control' => $params['congestion_control'] ?? 'cubic',
                    'alpn'               => $alpn,
                    'udp_relay_mode'     => $params['udp_relay_mode'] ?? 'native',
                    'tls' => [
                        'server_name'    => $params['sni'] ?? null,
                        'allow_insecure' => (($params['allow_insecure'] ?? $params['insecure'] ?? '0') === '1'),
                    ],
                ],
            ];
        } catch (\Exception) {
            return null;
        }
    }

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    private function buildNetworkSettings(string $net, array $d): ?array
    {
        return match ($net) {
            'ws' => [
                'path'    => $d['path'] ?? '/',
                'headers' => ['Host' => $d['host'] ?? ''],
            ],
            'h2', 'http' => [
                'path' => $d['path'] ?? '/',
                'host' => $d['host'] ?? '',
            ],
            'grpc' => [
                'serviceName' => $d['serviceName'] ?? $d['path'] ?? '',
            ],
            'tcp' => (($d['headerType'] ?? $d['type'] ?? 'none') === 'http') ? [
                'header' => [
                    'type'    => 'http',
                    'request' => [
                        'path'    => [$d['path'] ?? '/'],
                        'headers' => ['Host' => [$d['host'] ?? '']],
                    ],
                ],
            ] : null,
            default => null,
        };
    }

    /**
     * Return $value if it looks like a UUID, otherwise null.
     * Used to pick up a custom node ID (code) carried inside a proxy link's
     * user-id/password field, e.g. vmess `id`, vless/tuic user-info, etc.
     */
    private function extractUuid(?string $value): ?string
    {
        if (!$value) {
            return null;
        }

        return preg_match('/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i', $value) ? $value : null;
    }

    private function splitHostPort(string $hostport): array
    {
        // IPv6: [::1]:port
        if (str_starts_with($hostport, '[')) {
            $close = strpos($hostport, ']');
            $host  = substr($hostport, 1, $close - 1);
            $port  = ltrim(substr($hostport, $close + 1), ':');
        } else {
            $parts = explode(':', $hostport);
            $port  = array_pop($parts);
            $host  = implode(':', $parts);
        }
        return [$host, $port];
    }
}
