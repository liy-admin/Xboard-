<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

/**
 * 把节点 host 解析成 ISO alpha-2 国家码（大写），用于「每国只下发一个节点」。
 *
 * 默认走 ip-api.com 的免费批量接口（免 key，单请求 ≤ 100 个 IP，15 req/min）。
 * 结果由调用方（CheckServerLatency）缓存到 v2_server.country_code 列，只在国家码
 * 缺失或 host 变化时才会再次调用，避免频繁外呼触发限流。
 *
 * 预留 maxmind 本地库分支（geoip_method=maxmind），当前未实现，回退到 api。
 */
class GeoIpService
{
    /**
     * 批量解析 host 列表 => ['host' => 'US' | null]。
     *
     * @param  string[]  $hosts
     * @return array<string, string|null>
     */
    public function lookupCountries(array $hosts): array
    {
        $hosts = array_values(array_unique(array_filter($hosts)));
        $result = array_fill_keys($hosts, null);
        if (empty($hosts)) {
            return $result;
        }

        // host -> IP（域名解析失败时 gethostbyname 原样返回 host）
        $hostToIp = [];
        foreach ($hosts as $host) {
            $ip = filter_var($host, FILTER_VALIDATE_IP) ? $host : @gethostbyname($host);
            if ($ip && filter_var($ip, FILTER_VALIDATE_IP)) {
                $hostToIp[$host] = $ip;
            }
        }
        if (empty($hostToIp)) {
            return $result;
        }

        $method = admin_setting('geoip_method', 'api');
        $ipToCountry = $method === 'maxmind'
            ? $this->lookupViaMaxmind(array_values($hostToIp))
            : $this->lookupViaApi(array_values($hostToIp));

        foreach ($hostToIp as $host => $ip) {
            $cc = $ipToCountry[$ip] ?? null;
            $result[$host] = $cc ? strtoupper($cc) : null;
        }
        return $result;
    }

    /**
     * ip-api.com 批量查询，返回 ['1.2.3.4' => 'US']。失败返回空数组。
     *
     * @param  string[]  $ips
     * @return array<string, string>
     */
    private function lookupViaApi(array $ips): array
    {
        $out = [];
        // 批量上限 100，分片请求。
        foreach (array_chunk($ips, 100) as $chunk) {
            try {
                $payload = array_map(
                    fn($ip) => ['query' => $ip, 'fields' => 'status,countryCode,query'],
                    $chunk
                );
                $response = Http::timeout(15)->post('http://ip-api.com/batch', $payload);
                if (!$response->ok()) {
                    Log::warning('GeoIpService: ip-api batch 返回非 200', ['status' => $response->status()]);
                    continue;
                }
                foreach ($response->json() ?? [] as $row) {
                    if (($row['status'] ?? null) === 'success' && !empty($row['countryCode']) && !empty($row['query'])) {
                        $out[$row['query']] = $row['countryCode'];
                    }
                }
            } catch (\Throwable $e) {
                Log::warning('GeoIpService: ip-api 查询异常', ['message' => $e->getMessage()]);
            }
        }
        return $out;
    }

    /**
     * MaxMind GeoLite2 本地库查询（占位，未实现）。
     * 需引入 geoip2/geoip2 并提供 geoip_maxmind_db_path 指向 .mmdb 文件。
     *
     * @param  string[]  $ips
     * @return array<string, string>
     */
    private function lookupViaMaxmind(array $ips): array
    {
        Log::warning('GeoIpService: geoip_method=maxmind 尚未实现，回退到 api');
        return $this->lookupViaApi($ips);
    }
}
