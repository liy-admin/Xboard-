<?php

namespace App\Console\Commands;

use App\Models\Server;
use App\Services\NodeProbeService;
use App\Services\ServerService;
use Illuminate\Console\Command;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Cache;

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

    /** 进度状态缓存键（供管理端进度条轮询，刷新后仍可见） */
    public const STATUS_CACHE_KEY = 'server_latency_probe_status';

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

        $mode = admin_setting('node_probe_mode', 'singbox');
        $total = $servers->count();
        $this->info("探测模式：{$mode}，自动显隐：" . ($autoVisibility ? '开' : '关') . "，共 {$total} 个节点。");

        $startedAt = time();
        $this->writeStatus([
            'running' => true,
            'total' => $total,
            'done' => 0,
            'ok' => 0,
            'mode' => $mode,
            'current' => '',
            'started_at' => $startedAt,
            'finished_at' => null,
        ]);

        $now = time();
        $done = 0;
        $okCount = 0;
        // id => ['latency' => ?int, 'cc' => ?string, 'working' => bool]
        $probe = [];
        try {
            foreach ($servers as $server) {
                $this->writeStatus(['current' => $server->name], true);
                $r = $prober->probe($server);

                // 「没法测」（协议不支持 / 无可用用户 / sing-box 起不来）不是节点不通，
                // 保留上一次的测速结果，不要把能用的节点误判成不可达。
                $untestable = !empty($r['untestable']);
                if (!$untestable) {
                    $server->latency_ms = $r['latency_ms'];
                    $server->exit_ip = $r['exit_ip'] ?? null;
                    $server->last_latency_at = $now;
                }
                if (!empty($r['country_code'])) {
                    $server->country_code = $r['country_code'];
                }
                $server->saveQuietly(); // 测速字段频繁变化，不动 updated_at（避免每次都触发订阅更新）

                // 可用性统一走 isNodeAvailable：实测连通（新鲜）或 agent 在线。
                // 「没法测」的节点因此沿用上次结果，而不是立刻被判不可用。
                $working = ServerService::isNodeAvailable([
                    'is_online' => $server->is_online,
                    'latency_ms' => $server->latency_ms,
                    'last_latency_at' => $server->last_latency_at,
                ]);
                $probe[$server->id] = [
                    // 用节点当前值（「没法测」时即上一次的结果），供每国选最快排序
                    'latency' => $server->latency_ms,
                    'cc' => $server->country_code,
                    'working' => $working,
                ];

                $done++;
                if ($working) {
                    $okCount++;
                }
                $this->writeStatus(['done' => $done, 'ok' => $okCount], true);

                $latencyText = $untestable
                    ? '未测试(沿用上次)'
                    : ($r['latency_ms'] === null
                    ? '不可达'
                    : sprintf(
                        '%dms (avg %s · %d/%d)',
                        $r['latency_ms'],
                        $r['avg_latency'] ?? $r['latency_ms'],
                        $r['success_tests'] ?? 1,
                        $r['tests'] ?? 1
                    ));
                $this->line(sprintf(
                    '%-28s %-18s -> %s%s%s',
                    mb_strimwidth($server->name, 0, 26, '…'),
                    $server->host,
                    $latencyText,
                    !empty($server->country_code) ? "  [{$server->country_code}]" : '',
                    !empty($r['exit_ip']) ? "  出口 {$r['exit_ip']}" : ''
                ));
                if (!empty($r['error'])) {
                    $this->warn('  ! ' . $r['error']);
                }
            }

            // 自动显隐依赖「探测了全部节点」来做每国选优；只探测了部分节点（--id）时不执行，
            // 否则会把没探测到的节点误隐藏。
            if ($autoVisibility && !$this->option('id')) {
                $this->applyVisibility($servers, $probe);
            }
        } finally {
            $this->writeStatus([
                'running' => false,
                'current' => '',
                'finished_at' => time(),
            ], true);
        }

        return self::SUCCESS;
    }

    /**
     * 写进度到缓存。$merge=true 时合并进已有状态（否则整体覆盖）。
     * 额外写 updated_at，供端点判断进度是否「卡死」（进程被杀时兜底）。
     */
    private function writeStatus(array $data, bool $merge = false): void
    {
        if ($merge) {
            $data = array_merge((array) Cache::get(self::STATUS_CACHE_KEY, []), $data);
        }
        $data['updated_at'] = time();
        Cache::put(self::STATUS_CACHE_KEY, $data, 3600);
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

        // 先算出每个节点「应不应该显示」
        $targets = []; // id => bool
        foreach ($servers as $server) {
            $p = $probe[$server->id] ?? null;
            $working = $p['working'] ?? false;
            $cc = $p['cc'] ?? null;
            // 显示：每国最优 或 无国家但可达；其余隐藏
            $targets[$server->id] = in_array($server->id, $showIds, true) || ($working && blank($cc));
        }

        // 安全网：如果这一轮算出来「一个都不显示」，极可能是本次探测整体异常
        // （sing-box 没就绪 / 网络抖动 / 测试地址不可达），不要把全部节点隐藏，
        // 否则订阅会变空、客户端没节点。此时保持现有 show 不动。
        if (!in_array(true, $targets, true)) {
            $this->warn('自动显隐已跳过：本轮没有任何可达节点（疑似探测异常），保持现有显隐不变以免下发为空。');
            return;
        }

        $shown = 0;
        $hidden = 0;
        foreach ($servers as $server) {
            $shouldShow = $targets[$server->id] ?? false;
            if ((bool) $server->show !== $shouldShow) {
                $server->show = $shouldShow;
                $server->save(); // 变更才 bump updated_at
            }
            $shouldShow ? $shown++ : $hidden++;
        }

        $this->info("自动显隐完成：显示 {$shown} 个，隐藏 {$hidden} 个。");
    }
}
