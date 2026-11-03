<?php

namespace App\Console\Commands;

use App\Models\Server;
use App\Services\ProxyParserService;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Log;

/**
 * 定时导入已保存的订阅预设（node_import_subscriptions）。
 *
 * 规则：
 *  - 仅处理 schedule_enabled 的预设；按各自 interval_hours 判断是否到期（--force 忽略）
 *  - refetch_balala 的预设：先跑 balala-sub:fetch 取最新订阅链接，再导入
 *  - 去重：按「订阅标识(tag)」整批替换——先删带该 tag 的旧节点，再导入最新节点
 */
class ImportSubscriptions extends Command
{
    protected $signature = 'server:import-subscriptions
        {--id= : 只处理指定预设 id}
        {--force : 忽略时间间隔，立即导入}';

    protected $description = '定时导入已保存的订阅预设，按订阅标识整批替换节点';

    public function handle(): int
    {
        $presets = admin_setting('node_import_subscriptions', []);
        if (!is_array($presets) || empty($presets)) {
            $this->info('没有已保存的订阅预设');
            return self::SUCCESS;
        }

        $onlyId = $this->option('id');
        $force  = (bool) $this->option('force');
        $parser = new ProxyParserService();
        $changed = false;

        foreach ($presets as $i => $preset) {
            if ($onlyId && ($preset['id'] ?? '') !== $onlyId) {
                continue;
            }
            if (!$onlyId && empty($preset['schedule_enabled'])) {
                continue;
            }
            if (!$force && !$this->isDue($preset)) {
                continue;
            }

            $name = $preset['name'] ?? ($preset['id'] ?? '?');
            $this->line("▶ 导入预设：{$name}");

            try {
                $url = $this->resolveUrl($preset);
                if (!$url) {
                    $this->error("  ✗ 无可用订阅链接，跳过");
                    continue;
                }

                $parsed = $parser->parseFromSubscription($url);
                $servers = $parsed['servers'] ?? [];
                if (empty($servers)) {
                    $this->error("  ✗ 订阅解析出 0 个节点，跳过（不动已有节点）");
                    continue;
                }

                $tag = trim((string) ($preset['tag'] ?? ''));
                $deleted = 0;
                if ($tag !== '') {
                    // 按标识整批替换：先删旧
                    $deleted = Server::whereJsonContains('tags', $tag)->delete();
                }

                $created = $this->importServers($servers, $preset, $tag);
                $this->info("  ✓ 删除旧节点 {$deleted} 个，导入新节点 {$created} 个");

                $presets[$i]['last_run_at'] = now()->toIso8601String();
                $changed = true;
            } catch (\Throwable $e) {
                Log::error('[import-subscriptions] 预设导入失败', ['name' => $name, 'error' => $e->getMessage()]);
                $this->error("  ✗ 失败：" . $e->getMessage());
            }
        }

        if ($changed) {
            admin_setting(['node_import_subscriptions' => array_values($presets)]);
        }

        return self::SUCCESS;
    }

    /**
     * 是否到期：now >= last_run_at + interval_hours。
     */
    protected function isDue(array $preset): bool
    {
        $interval = (int) ($preset['interval_hours'] ?? 24);
        if ($interval <= 0) {
            $interval = 24;
        }
        $last = $preset['last_run_at'] ?? null;
        if (!$last) {
            return true;
        }
        try {
            return now()->greaterThanOrEqualTo(\Illuminate\Support\Carbon::parse($last)->addHours($interval));
        } catch (\Throwable $e) {
            return true;
        }
    }

    /**
     * 解析本次导入使用的订阅链接。Balala 源先重抓最新。
     */
    protected function resolveUrl(array $preset): ?string
    {
        if (!empty($preset['refetch_balala'])) {
            try {
                Artisan::call('balala-sub:fetch', ['--json' => true]);
                $out = json_decode(trim(Artisan::output()), true);
                if (is_array($out) && !empty($out['subscribe_url'])) {
                    return $out['subscribe_url'];
                }
                $this->warn('  balala 重抓未返回链接，回退使用预设里存的链接');
            } catch (\Throwable $e) {
                $this->warn('  balala 重抓失败（' . $e->getMessage() . '），回退使用预设里存的链接');
            }
        }
        return $preset['url'] ?? null;
    }

    /**
     * 按预设默认参数导入节点，返回成功数量。
     */
    protected function importServers(array $servers, array $preset, string $tag): int
    {
        $created = 0;
        foreach ($servers as $serverData) {
            if (!empty($preset['group_ids'])) {
                $serverData['group_ids'] = array_values(array_map('intval', $preset['group_ids']));
            }
            if (array_key_exists('rate', $preset) && $preset['rate'] !== null) {
                $serverData['rate'] = $preset['rate'];
            }
            $serverData['show'] = (int) ($preset['show'] ?? 0);
            if ($tag !== '') {
                $serverData['tags'] = [$tag];
            }
            unset($serverData['id']);

            try {
                Server::create($serverData);
                $created++;
            } catch (\Throwable $e) {
                Log::warning('[import-subscriptions] 单节点创建失败', [
                    'name'  => $serverData['name'] ?? '',
                    'error' => $e->getMessage(),
                ]);
            }
        }
        return $created;
    }
}
