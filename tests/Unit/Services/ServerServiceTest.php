<?php

namespace Tests\Unit\Services;

use App\Services\ServerService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class ServerServiceTest extends TestCase
{
    use RefreshDatabase;

    private function server(array $overrides = []): array
    {
        return array_merge([
            'id' => random_int(1, 1_000_000),
            'name' => 'node',
            'type' => 'vmess',
            'host' => 'example.com',
            'sort' => 0,
            'is_online' => 1,
            'online' => 0,
            'country_code' => null,
            'latency_ms' => null,
            'last_latency_at' => null,
        ], $overrides);
    }

    public function test_picks_lowest_latency_available_node_per_country(): void
    {
        admin_setting(['server_country_naming' => false, 'server_latency_stale_seconds' => 600]);
        $now = time();

        $servers = [
            $this->server(['id' => 1, 'name' => 'US-A', 'country_code' => 'US', 'latency_ms' => 120, 'last_latency_at' => $now]),
            $this->server(['id' => 2, 'name' => 'US-B', 'country_code' => 'US', 'latency_ms' => 50, 'last_latency_at' => $now]),
            $this->server(['id' => 3, 'name' => 'JP-A', 'country_code' => 'JP', 'latency_ms' => 80, 'last_latency_at' => $now]),
        ];

        $result = ServerService::pickBestPerCountry($servers);

        $this->assertCount(2, $result, '每个国家只保留一个节点');
        $byCode = collect($result)->keyBy('country_code');
        $this->assertSame(2, $byCode['US']['id'], 'US 应选延迟最低的 US-B');
        $this->assertSame(3, $byCode['JP']['id']);
    }

    public function test_excludes_unavailable_nodes(): void
    {
        admin_setting(['server_country_naming' => false, 'server_latency_stale_seconds' => 600]);
        $now = time();

        $servers = [
            // 既离线又没有新鲜测速结果 -> 不可用
            $this->server(['id' => 1, 'country_code' => 'US', 'is_online' => 0, 'latency_ms' => null, 'last_latency_at' => null]),
            $this->server(['id' => 2, 'country_code' => 'US', 'is_online' => 1, 'latency_ms' => 200, 'last_latency_at' => $now]),
        ];

        $result = ServerService::pickBestPerCountry($servers);

        $this->assertCount(1, $result);
        $this->assertSame(2, $result[0]['id'], '不可用（离线且无测速）节点不应入选');
    }

    public function test_probe_success_counts_even_when_agent_offline(): void
    {
        // agent 不上报心跳（is_online=0）但面板实测连通 -> 应视为可用并下发
        admin_setting(['server_country_naming' => false, 'server_latency_stale_seconds' => 600]);
        $now = time();

        $servers = [
            $this->server(['id' => 1, 'country_code' => 'HK', 'is_online' => 0, 'latency_ms' => 120, 'last_latency_at' => $now]),
        ];

        $result = ServerService::pickBestPerCountry($servers);

        $this->assertCount(1, $result);
        $this->assertSame(1, $result[0]['id'], '实测连通的节点即使 agent 离线也应下发');
    }

    public function test_stale_latency_offline_is_excluded(): void
    {
        admin_setting(['server_country_naming' => false, 'server_latency_stale_seconds' => 600]);

        $servers = [
            // 离线 + 测速结果过期 -> 不可用
            $this->server(['id' => 1, 'country_code' => 'HK', 'is_online' => 0, 'latency_ms' => 50, 'last_latency_at' => time() - 10000]),
        ];

        $result = ServerService::pickBestPerCountry($servers);

        $this->assertCount(0, $result, '离线且延迟已过期的节点不应下发');
    }

    public function test_falls_back_when_no_fresh_latency(): void
    {
        admin_setting(['server_country_naming' => false, 'server_latency_stale_seconds' => 600]);
        $stale = time() - 10_000; // 远超过阈值

        $servers = [
            // 两个 UDP 节点都没有新鲜延迟：按在线人数少者兜底
            $this->server(['id' => 1, 'type' => 'hysteria', 'country_code' => 'HK', 'online' => 30, 'latency_ms' => 5, 'last_latency_at' => $stale]),
            $this->server(['id' => 2, 'type' => 'hysteria', 'country_code' => 'HK', 'online' => 3, 'latency_ms' => null, 'last_latency_at' => null]),
        ];

        $result = ServerService::pickBestPerCountry($servers);

        $this->assertCount(1, $result);
        $this->assertSame(2, $result[0]['id'], '无新鲜延迟时按在线人数少者兜底');
    }

    public function test_keeps_nodes_without_country(): void
    {
        admin_setting(['server_country_naming' => false, 'server_latency_stale_seconds' => 600]);
        $now = time();

        $servers = [
            $this->server(['id' => 1, 'country_code' => 'US', 'latency_ms' => 50, 'last_latency_at' => $now]),
            $this->server(['id' => 2, 'country_code' => null, 'name' => 'custom-1']),
            $this->server(['id' => 3, 'country_code' => '', 'name' => 'custom-2']),
        ];

        $result = ServerService::pickBestPerCountry($servers);

        $ids = collect($result)->pluck('id')->sort()->values()->all();
        $this->assertSame([1, 2, 3], $ids, '无国家码的节点原样保留');
    }

    public function test_applies_country_naming_when_enabled(): void
    {
        admin_setting(['server_country_naming' => true, 'server_latency_stale_seconds' => 600]);
        $now = time();

        $servers = [
            $this->server(['id' => 1, 'name' => 'raw-name', 'country_code' => 'US', 'latency_ms' => 50, 'last_latency_at' => $now]),
        ];

        $result = ServerService::pickBestPerCountry($servers);

        $this->assertSame('🇺🇸 美国', $result[0]['name']);
    }

    public function test_country_flag_falls_back_for_invalid_code(): void
    {
        $this->assertSame('🏳️', ServerService::countryFlag('XYZ'));
        $this->assertSame('🇯🇵', ServerService::countryFlag('jp'));
    }
}
