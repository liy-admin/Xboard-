<?php

namespace App\Services;

use App\Models\User;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Redis;

class DeviceStateService
{
    private const PREFIX = 'user_devices:';
    private const TTL = 300;                     // device state ttl
    private const DB_THROTTLE = 10;             // update db throttle

    /**
     * 移除 Redis key 的前缀
     */
    private function removeRedisPrefix(string $key): string
    {
        $prefix = config('database.redis.options.prefix', '');
        return $prefix ? substr($key, strlen($prefix)) : $key;
    }

    /**
     * 批量设置设备
     * 用于 HTTP /alive 和 WebSocket report.devices
     */
    public function setDevices(int $userId, int $nodeId, array $ips): void
    {
        $key = self::PREFIX . $userId;
        $timestamp = time();

        $this->removeNodeDevices($nodeId, $userId);

        // Normalize: strip port suffix and deduplicate
        $ips = array_values(array_unique(array_map([self::class, 'normalizeIP'], $ips)));

        if (!empty($ips)) {
            $fields = [];
            foreach ($ips as $ip) {
                $fields["{$nodeId}:{$ip}"] = $timestamp;
            }
            Redis::hMset($key, $fields);
            Redis::expire($key, self::TTL);
        }

        $this->notifyUpdate($userId);
    }

    /**
     * 获取某节点的所有设备数据
     * 返回: {userId: [ip1, ip2, ...], ...}
     */
    public function getNodeDevices(int $nodeId): array
    {
        $keys = Redis::keys(self::PREFIX . '*');
        $prefix = "{$nodeId}:";
        $result = [];
        foreach ($keys as $key) {
            $actualKey = $this->removeRedisPrefix($key);
            $uid = (int) substr($actualKey, strlen(self::PREFIX));
            $data = Redis::hgetall($actualKey);
            foreach ($data as $field => $timestamp) {
                if (str_starts_with($field, $prefix)) {
                    $ip = substr($field, strlen($prefix));
                    $result[$uid][] = $ip;
                }
            }
        }

        return $result;
    }

    /**
     * 删除某节点某用户的设备
     */
    public function removeNodeDevices(int $nodeId, int $userId): void
    {
        $key = self::PREFIX . $userId;
        $prefix = "{$nodeId}:";

        foreach (Redis::hkeys($key) as $field) {
            if (str_starts_with($field, $prefix)) {
                Redis::hdel($key, $field);
            }
        }
    }

    /**
     * 清除节点所有设备数据（用于节点断开连接）
     */
    public function clearAllNodeDevices(int $nodeId): array
    {
        $oldDevices = $this->getNodeDevices($nodeId);
        $prefix = "{$nodeId}:";

        foreach ($oldDevices as $userId => $ips) {
            $key = self::PREFIX . $userId;
            foreach (Redis::hkeys($key) as $field) {
                if (str_starts_with($field, $prefix)) {
                    Redis::hdel($key, $field);
                }
            }
            $this->notifyUpdate($userId);
        }

        return array_keys($oldDevices);
    }

    /**
     * get user device count (deduplicated by IP, filter expired data)
     */
    public function getDeviceCount(int $userId): int
    {
        $data = Redis::hgetall(self::PREFIX . $userId);
        $now = time();
        $ips = [];

        foreach ($data as $field => $timestamp) {
            if ($now - $timestamp <= self::TTL) {
                $ips[] = substr($field, strpos($field, ':') + 1);
            }
        }

        return count(array_unique($ips));
    }

    /**
     * get user device count (for alivelist interface)
     */
    public function getAliveList(Collection $users): array
    {
        if ($users->isEmpty()) {
            return [];
        }

        $result = [];
        foreach ($users as $user) {
            $count = $this->getDeviceCount($user->id);
            if ($count > 0) {
                $result[$user->id] = $count;
            }
        }

        return $result;
    }

    /**
     * 获取某用户当前所有未过期的设备条目（供用户自助查看/管理使用）。
     * 返回 [{node_id, ip, last_seen}, ...]。
     */
    public function getUserDevices(int $userId): array
    {
        $data = Redis::hgetall(self::PREFIX . $userId);
        $now = time();
        $result = [];

        foreach ($data as $field => $timestamp) {
            if ($now - $timestamp > self::TTL) {
                continue;
            }
            [$nodeId, $ip] = explode(':', $field, 2);
            $result[] = [
                'node_id' => (int) $nodeId,
                'ip' => $ip,
                'last_seen' => (int) $timestamp,
            ];
        }

        return $result;
    }

    /**
     * 移除某用户在某节点上的单个设备 IP（用户自助"踢出设备"）。
     *
     * 注意：这只是把该 IP 从 Redis 追踪表里删除，让节点在下一次校验设备数
     * 时不再把它算进占用——如果该节点确实实现了 device_limit 校验，效果是
     * 腾出一个名额给新设备登录，但并不会主动断开这个 IP 当前已建立的连接
     * （面板和代理节点之间没有可以强制断线的接口）。而且如果这个 IP 此刻
     * 仍在该节点上活跃，节点下一次 /alive 上报会重新把它写回来
     * （见 [setDevices] 里 report 会整节点覆盖）。
     */
    public function removeDevice(int $userId, int $nodeId, string $ip): void
    {
        $key = self::PREFIX . $userId;
        Redis::hdel($key, "{$nodeId}:{$ip}");
        $this->notifyUpdate($userId);
    }

    /**
     * get devices of multiple users (for sync.devices, filter expired data)
     */
    public function getUsersDevices(array $userIds): array
    {
        $result = [];
        $now = time();
        foreach ($userIds as $userId) {
            $data = Redis::hgetall(self::PREFIX . $userId);
            if (!empty($data)) {
                $ips = [];
                foreach ($data as $field => $timestamp) {
                    if ($now - $timestamp <= self::TTL) {
                        $ips[] = substr($field, strpos($field, ':') + 1);
                    }
                }
                if (!empty($ips)) {
                    $result[$userId] = array_unique($ips);
                }
            }
        }

        return $result;
    }

    /**
     * Strip port from IP address: "1.2.3.4:12345" → "1.2.3.4", "[::1]:443" → "::1"
     */
    private static function normalizeIP(string $ip): string
    {
        // [IPv6]:port
        if (preg_match('/^\[(.+)\]:\d+$/', $ip, $m)) {
            return $m[1];
        }
        // IPv4:port
        if (preg_match('/^(\d+\.\d+\.\d+\.\d+):\d+$/', $ip, $m)) {
            return $m[1];
        }
        return $ip;
    }

    /**
     * 获取所有当前在线用户（online_count > 0 且 last_online_at 在 TTL 内）
     */
    public function getOnlineUsers(): array
    {
        $threshold = now()->subSeconds(self::TTL);

        return User::query()
            ->where('online_count', '>', 0)
            ->where('last_online_at', '>=', $threshold)
            ->get(['id', 'online_count'])
            ->map(fn($u) => ['user_id' => $u->id, 'device_count' => $u->online_count])
            ->values()
            ->toArray();
    }

    /**
     * notify update (throttle control)
     */
    public function notifyUpdate(int $userId): void
    {
        $dbThrottleKey = "device:db_throttle:{$userId}";

        // if (Redis::setnx($dbThrottleKey, 1)) {
        //     Redis::expire($dbThrottleKey, self::DB_THROTTLE);

            User::query()
                ->whereKey($userId)
                ->update([
                    'online_count' => $this->getDeviceCount($userId),
                    'last_online_at' => now(),
                ]);
        // }
    }
}
