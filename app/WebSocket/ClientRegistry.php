<?php

namespace App\WebSocket;

use Workerman\Connection\TcpConnection;

/**
 * 客户端（Hiddify 等终端 App）WebSocket 连接注册表，运行在 Workerman 进程内。
 *
 * 与 {@see \App\Services\NodeRegistry}（节点↔面板）不同，这里维护的是
 * 「用户 ↔ 其所有在线客户端连接」。同一个用户可能多设备同时在线，所以
 * 一个 userId 对应一组连接。用途：当某用户的可下发节点集发生变化时，
 * 面板向该用户（或全体）在线客户端推一条「订阅已更新，请立即重拉」信号，
 * 客户端收到后触发自身既有的订阅强制更新流程。
 */
class ClientRegistry
{
    /** @var array<int, array<int, TcpConnection>> userId → [objectId → connection] */
    private static array $connections = [];

    public static function add(int $userId, TcpConnection $conn): void
    {
        self::$connections[$userId][spl_object_id($conn)] = $conn;
    }

    /**
     * 移除某连接。仅当该连接确实属于此 userId 时才移除。
     */
    public static function remove(int $userId, TcpConnection $conn): void
    {
        unset(self::$connections[$userId][spl_object_id($conn)]);
        if (empty(self::$connections[$userId])) {
            unset(self::$connections[$userId]);
        }
    }

    /**
     * 给指定用户的所有在线客户端发送事件。返回成功发送的连接数。
     */
    public static function sendToUser(int $userId, string $event, array $data = []): int
    {
        $conns = self::$connections[$userId] ?? [];
        if (empty($conns)) {
            return 0;
        }

        $payload = json_encode([
            'event' => $event,
            'data' => $data,
            'timestamp' => time(),
        ]);

        $sent = 0;
        foreach ($conns as $conn) {
            if ($conn->getStatus() === TcpConnection::STATUS_ESTABLISHED) {
                $conn->send($payload);
                $sent++;
            }
        }
        return $sent;
    }

    /**
     * 广播给所有在线客户端。返回成功发送的连接数。
     */
    public static function broadcast(string $event, array $data = []): int
    {
        $payload = json_encode([
            'event' => $event,
            'data' => $data,
            'timestamp' => time(),
        ]);

        $sent = 0;
        foreach (self::$connections as $conns) {
            foreach ($conns as $conn) {
                if ($conn->getStatus() === TcpConnection::STATUS_ESTABLISHED) {
                    $conn->send($payload);
                    $sent++;
                }
            }
        }
        return $sent;
    }

    /**
     * 当前在线客户端连接总数（跨所有用户）。
     */
    public static function count(): int
    {
        $total = 0;
        foreach (self::$connections as $conns) {
            $total += count($conns);
        }
        return $total;
    }

    /**
     * 当前有客户端在线的用户数。
     */
    public static function userCount(): int
    {
        return count(self::$connections);
    }
}
