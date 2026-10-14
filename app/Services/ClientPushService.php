<?php

namespace App\Services;

use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Redis;

/**
 * 面板 → 客户端（Hiddify 等 App）WebSocket 信号推送的发布端。
 *
 * 走 Redis pub/sub 的 `client:push` 频道，由 Workerman 的
 * {@see \App\WebSocket\NodeWorker} 订阅后分派给 {@see \App\WebSocket\ClientRegistry}。
 *
 * 设计为「只推信号、不推数据」：客户端收到 `subscription.updated` 后，
 * 调用自身既有的订阅强制更新流程重新拉取节点列表（数据仍走 HTTP 订阅接口），
 * 从而做到节点变化秒级下发，而无需等待客户端的定时刷新间隔。
 */
class ClientPushService
{
    public const CHANNEL = 'client:push';
    public const EVENT_SUBSCRIPTION_UPDATED = 'subscription.updated';

    /** 广播节流：相同 reason 的全局广播在该秒数内只发一次，避免风暴 */
    private const BROADCAST_THROTTLE = 3;

    /**
     * 通知「可下发节点集发生变化」。当前为全局广播（配置选定），
     * 所有在线客户端收到后各自按自身权限重拉订阅。
     *
     * @param string $reason 变化来源，便于日志与节流（如 visibility / node.online / node.offline）
     */
    public static function broadcastSubscriptionUpdated(string $reason = ''): void
    {
        if (!(bool) admin_setting('client_ws_enable', false)) {
            return;
        }

        // 节流：短时间内的多次变化合并为一次广播
        $throttleKey = 'client_push:throttle:' . ($reason !== '' ? $reason : 'all');
        try {
            if (Cache::get($throttleKey)) {
                return;
            }
            Cache::put($throttleKey, 1, self::BROADCAST_THROTTLE);
        } catch (\Throwable $e) {
            // 缓存不可用时不因节流而阻断推送
        }

        self::publish([
            'scope' => 'all',
            'event' => self::EVENT_SUBSCRIPTION_UPDATED,
            'data' => ['reason' => $reason],
        ]);
    }

    /**
     * 通知单个用户的客户端重拉订阅（预留定向推送能力）。
     */
    public static function notifyUser(int $userId, string $reason = ''): void
    {
        if (!(bool) admin_setting('client_ws_enable', false)) {
            return;
        }

        self::publish([
            'scope' => 'user',
            'user_id' => $userId,
            'event' => self::EVENT_SUBSCRIPTION_UPDATED,
            'data' => ['reason' => $reason],
        ]);
    }

    private static function publish(array $payload): void
    {
        try {
            Redis::publish(self::CHANNEL, json_encode($payload));
        } catch (\Throwable $e) {
            Log::warning("[ClientPush] Redis publish failed: {$e->getMessage()}", [
                'scope' => $payload['scope'] ?? null,
                'event' => $payload['event'] ?? null,
            ]);
        }
    }
}
