<?php

namespace Plugin\BalalaSub\Commands;

use App\Models\Plugin;
use App\Models\Server;
use App\Services\ProxyParserService;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;

/**
 * 定时/手动：抓取 Balala 最新订阅 → 按订阅标识整批替换节点。
 *
 * 开关与参数来自插件配置：
 *   auto_import, auto_import_interval_hours, auto_import_tag,
 *   auto_import_group_ids, auto_import_rate, auto_import_show, auto_import_client
 */
class ImportCommand extends Command
{
    protected $signature = 'balala-sub:import {--force : 忽略开关与间隔，立即导入}';

    protected $description = '抓取 Balala 最新订阅并按标识整批替换节点';

    public const LAST_RUN_KEY = 'plugin:balala_sub:last_import';
    public const STATUS_KEY   = 'plugin:balala_sub:import_status';
    public const REQUEST_KEY  = 'plugin:balala_sub:import_requested';

    public function handle(): int
    {
        $cfg = $this->pluginConfig();
        $force = (bool) $this->option('force');

        // 后台「立即导入」按钮会写一个请求标志；定时任务(cron CLI)每分钟执行时看到它就强制导入一次。
        // 把重活放在可靠的 CLI 环境，避免 web/Octane 进程 spawn 浏览器的各种坑。
        try {
            if (Cache::get(self::REQUEST_KEY)) {
                Cache::forget(self::REQUEST_KEY);
                $force = true;
                $this->info('检测到手动导入请求，立即执行一次');
            }
        } catch (\Throwable $e) {
            // 缓存不可用则忽略（仅影响手动请求识别）
        }

        if (!$force && empty($cfg['auto_import'])) {
            $this->info('定时导入未开启（auto_import=false），跳过');
            return self::SUCCESS;
        }

        $interval = max(1, (int) ($cfg['auto_import_interval_hours'] ?? 24));
        if (!$force && !$this->isDue($interval)) {
            return self::SUCCESS;
        }

        $tag = trim((string) ($cfg['auto_import_tag'] ?? ''));
        $client = trim((string) ($cfg['auto_import_client'] ?? 'v2rayn')) ?: 'v2rayn';

        // 1) 抓最新订阅（balala-sub:fetch 同时会更新缓存）
        $this->line('▶ 抓取最新订阅…');
        try {
            Artisan::call('balala-sub:fetch', ['--json' => true]);
            $out = json_decode(trim(Artisan::output()), true);
        } catch (\Throwable $e) {
            return $this->done(false, '抓取异常：' . $e->getMessage());
        }
        if (!is_array($out) || empty($out['success'])) {
            return $this->done(false, '抓取失败：' . (is_array($out) ? ($out['error'] ?? '未知') : '无有效输出'));
        }

        $url = $this->pickUrl($out, $client);
        if (!$url) {
            return $this->done(false, '未找到可用订阅链接');
        }

        // 2) 解析订阅
        $parsed = (new ProxyParserService())->parseFromSubscription($url);
        $servers = $parsed['servers'] ?? [];
        if (empty($servers)) {
            return $this->done(false, '订阅解析出 0 个节点，保留现有节点不动');
        }

        // 2.5) 过滤机场塞进订阅的「信息条目」(到期/流量/官网等)——它们不是真节点，
        //      sing-box 测速碰到会崩("闪退")。按名称关键词剔除。
        $before = count($servers);
        $servers = array_values(array_filter($servers, fn ($s) => !$this->isInfoNode((string) ($s['name'] ?? ''), $cfg)));
        $skipped = $before - count($servers);
        if ($skipped > 0) {
            $this->line("  已过滤 {$skipped} 个信息条目(非真实节点)");
        }
        if (empty($servers)) {
            return $this->done(false, '过滤信息条目后无真实节点，保留现有节点不动');
        }

        // 3) 按标识整批替换：先删旧
        $deleted = 0;
        if ($tag !== '') {
            $deleted = Server::whereJsonContains('tags', $tag)->delete();
        } else {
            $this->warn('未配置订阅标识，无法"替换"，只做新增（建议填写标识）');
        }

        // 4) 导入
        $groupIds = $this->parseGroupIds($cfg['auto_import_group_ids'] ?? '');
        $rate = isset($cfg['auto_import_rate']) ? (float) $cfg['auto_import_rate'] : 1;
        $show = (int) ($cfg['auto_import_show'] ?? 0);

        if (empty($groupIds)) {
            $this->warn('⚠ 未配置权限组(auto_import_group_ids)，导入的节点无法测速和订阅下发');
        }

        $created = 0;
        $nameCount = [];
        foreach ($servers as $sd) {
            if ($groupIds) {
                $sd['group_ids'] = $groupIds;
            }
            $sd['rate'] = $rate;
            $sd['show'] = $show;
            if ($tag !== '') {
                $sd['tags'] = [$tag];
            }
            unset($sd['id']);
            try {
                Server::create($sd);
                $created++;
            } catch (\Throwable $e) {
                Log::warning('[balala_sub:import] 单节点创建失败', ['name' => $sd['name'] ?? '', 'error' => $e->getMessage()]);
            }
        }

        Cache::put(self::LAST_RUN_KEY, now()->toIso8601String(), 86400 * 30);
        return $this->done(true, "删除旧节点 {$deleted} 个，导入 {$created} 个", ['deleted' => $deleted, 'created' => $created]);
    }

    protected function pickUrl(array $out, string $client): ?string
    {
        foreach (($out['subscriptions'] ?? []) as $s) {
            if (stripos((string) ($s['client'] ?? ''), $client) !== false && !empty($s['url'])) {
                return $s['url'];
            }
        }
        return $out['subscribe_url'] ?? null;
    }

    /** 内置信息条目关键词（机场塞进订阅的非节点展示项） */
    protected const INFO_KEYWORDS = [
        'expire', 'traffic', 'sync', 'reset', '剩余', '流量', '到期', '过期', '有效期',
        '重置', '套餐', '距离', '官网', '网址', '最新', '群组', '频道', '客服', '邀请',
        '教程', '公告', '续费', '购买', '订阅', '节点异常', '更新',
    ];

    /** 判断是否为「信息条目」而非真实节点 */
    protected function isInfoNode(string $name, array $cfg): bool
    {
        $name = trim($name);
        if ($name === '') {
            return false;
        }
        $lower = mb_strtolower($name);
        // URL 样式的名字(含 http 或 像域名)基本是官网/网址条目
        if (str_contains($lower, 'http') || preg_match('/[a-z0-9-]+\.(com|net|org|io|xyz|cc|me|top|shop|vip)\b/i', $name)) {
            return true;
        }
        $keywords = self::INFO_KEYWORDS;
        $extra = trim((string) ($cfg['info_node_keywords'] ?? ''));
        if ($extra !== '') {
            foreach (preg_split('/[,\s]+/', $extra) as $kw) {
                $kw = trim($kw);
                if ($kw !== '') {
                    $keywords[] = mb_strtolower($kw);
                }
            }
        }
        foreach ($keywords as $kw) {
            if ($kw !== '' && str_contains($lower, $kw)) {
                return true;
            }
        }
        return false;
    }

    protected function parseGroupIds($raw): array
    {
        if (is_array($raw)) {
            return array_values(array_filter(array_map('intval', $raw)));
        }
        $raw = trim((string) $raw);
        if ($raw === '') {
            return [];
        }
        return array_values(array_filter(array_map('intval', preg_split('/[,\s]+/', $raw))));
    }

    protected function isDue(int $intervalHours): bool
    {
        $last = Cache::get(self::LAST_RUN_KEY);
        if (!$last) {
            return true;
        }
        try {
            return now()->greaterThanOrEqualTo(\Illuminate\Support\Carbon::parse($last)->addHours($intervalHours));
        } catch (\Throwable $e) {
            return true;
        }
    }

    protected function done(bool $ok, string $msg, array $extra = []): int
    {
        try {
            Cache::put(self::STATUS_KEY, ['ok' => $ok, 'message' => $msg, 'at' => now()->toIso8601String()] + $extra, 86400 * 7);
        } catch (\Throwable $e) {
            // 缓存不可用不影响结果
        }
        if ($ok) {
            $this->info('✅ ' . $msg);
            return self::SUCCESS;
        }
        $this->error('❌ ' . $msg);
        return self::FAILURE;
    }

    protected function pluginConfig(): array
    {
        $plugin = Plugin::where('code', 'balala_sub')->first();
        return ($plugin && $plugin->config) ? (json_decode($plugin->config, true) ?: []) : [];
    }
}
