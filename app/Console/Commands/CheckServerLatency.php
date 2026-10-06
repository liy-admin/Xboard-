<?php

namespace App\Console\Commands;

use App\Models\Server;
use App\Services\NodeProbeService;
use Illuminate\Console\Command;
use Illuminate\Support\Collection;

/**
 * 节点测速 + 国家解析，并（可选）按测速结果自动设置节点显隐。两种探测模式（node_probe_mode）：
 *   - singbox：面板用 sing-box 核心实跑节点，测真实出口 IP/国家/延迟（最准确，可自动下载 sing-box）。
 *   - tcp：TCP 握手测延迟 + 节点名/GeoIP 判国家（轻量兜底，无需额外依赖）。
 *
 * 结果写入 v2_server.latency_ms / exit_ip / last_latency_at / country_code。
 *
 * 当 server_auto_manage_visibility 开启时：探测「全部」父节点（含已隐藏的，便于主节点
 * 掉线后自动启用备用节点），并把 show 设为「每个国家延迟最低的可达节点 + 无国家的可达
 * 节点」，其余自动隐藏——让节点管理里的显隐直接反映实际下发情况。
 */
class CheckServerLatency extends Command
{
    protected $signature = 'check:server-latency {--id=* : 只探测指定节点ID（调试用）}';

    protected $description = '节点测速（sing-box 实跑 / TCP 握手）、国家解析与自动显隐';

    public function handle(NodeProbeService $prober): int
    {
        if (!(bool) admin_setting('server_latency_probe_enable', true)) {
            $this->info('server_latency_probe_enable 关闭，跳过测速。');
            return self::SUCCESS;
        }

        $autoVisibility = (bool) admin_setting('server_auto_manage_visibility', false);

        $query = Server::whereNull('parent_id');
        // 自动显隐时要探测全部节点（含隐藏的），否则只探测当前展示的
        if (!$autoVisibility) {
            $query->where('show', true);
        }
        if ($ids = $this->option('id')) {
            $query->whereIn('id', $ids);
        }
        $servers = $query->get();

        if ($servers->isEmpty()) {
            $this->info('没有需要测速的节点。');
            return self::SUCCESS;
        }

        $mode = admin_setting('node_probe_mode', 'tcp');
        $this->info("探测模式：{$mode}，自动显隐：" . ($autoVisibility ? '开' : '关') . "，共 {$servers->count()} 个节点。");

        $now = time();
        // id => ['latency' => ?int, 'cc' => ?string, 'working' => bool]
        $probe = [];
        foreach ($servers as $server) {
            $r = $prober->probe($server);

            $server->latency_ms = $r['latency_ms'];
            $server->exit_ip = $r['exit_ip'] ?? null;
            $server->last_latency_at = $now;
            if (!empty($r['country_code'])) {
                $server->country_code = $r['country_code'];
            }
            $server->saveQuietly(); // 测速字段频繁变化，不动 updated_at（避免每次都触发订阅更新）

            $probe[$server->id] = [
                'latency' => $r['latency_ms'],
                'cc' => $server->country_code,
                'working' => $r['latency_ms'] !== null,
            ];

            $this->line(sprintf(
                '%-28s %-18s -> %s%s%s',
                mb_strimwidth($server->name, 0, 26, '…'),
                $server->host,
                $r['latency_ms'] === null ? '不可达' : "{$r['latency_ms']}ms",
                !empty($server->country_code) ? "  [{$server->country_code}]" : '',
                !empty($r['exit_ip']) ? "  出口 {$r['exit_ip']}" : ''
            ));
            if (!empty($r['error'])) {
                $this->warn('  ! ' . $r['error']);
            }
        }

        if ($autoVisibility) {
            $this->applyVisibility($servers, $probe);
        }

        return self::SUCCESS;
    }

    /**
     * 按测速结果设置 show：每国选延迟最低的可达节点 + 无国家的可达节点为显示，其余隐藏。
     * 只有 show 真正变化时才 save（bump updated_at → 触发客户端订阅更新）。
     *
     * @param  Collection<int, Server>  $servers
     * @param  array<int, array{latency: ?int, cc: ?string, working: bool}>  $probe
     */
    private function applyVisibility(Collection $servers, array $probe): void
    {
        // 每个国家选延迟最低的可达节点
        $bestByCountry = []; // cc => [id, latency]
        foreach ($servers as $server) {
            $p = $probe[$server->id] ?? null;
            if (!$p || !$p['working'] || blank($p['cc'])) {
                continue;
            }
            $cc = strtoupper($p['cc']);
            $lat = $p['latency'] ?? PHP_INT_MAX;
            if (!isset($bestByCountry[$cc]) || $lat < $bestByCountry[$cc]['latency']) {
                $bestByCountry[$cc] = ['id' => $server->id, 'latency' => $lat];
            }
        }
        $showIds = array_column($bestByCountry, 'id');

        $shown = 0;
        $hidden = 0;
        foreach ($servers as $server) {
            $p = $probe[$server->id] ?? null;
            $working = $p['working'] ?? false;
            $cc = $p['cc'] ?? null;

            // 显示：每国最优 或 无国家但可达；其余隐藏
            $shouldShow = in_array($server->id, $showIds, true) || ($working && blank($cc));

            if ((bool) $server->show !== $shouldShow) {
                $server->show = $shouldShow;
                $server->save(); // 变更才 bump updated_at
            }
            $shouldShow ? $shown++ : $hidden++;
        }

        $this->info("自动显隐完成：显示 {$shown} 个，隐藏 {$hidden} 个。");
    }
}
