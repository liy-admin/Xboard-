<?php

namespace App\WebSocket;

use App\Models\Server;
use App\Models\ServerMachine;
use App\Services\DeviceStateService;
use App\Services\NodeRegistry;
use App\Services\ServerService;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Redis;
use Workerman\Connection\TcpConnection;
use Workerman\Timer;
use Workerman\Worker;

class NodeWorker
{
    private const AUTH_TIMEOUT = 10;
    private const PING_INTERVAL = 55;

    public const HEARTBEAT_CACHE_KEY = 'ws_server:heartbeat';
    private const HEARTBEAT_INTERVAL = 10;
    private const HEARTBEAT_TTL = 30;

    private Worker $worker;

    private array $handlers = [
        'pong' => [NodeEventHandlers::class, 'handlePong'],
        'node.status' => [NodeEventHandlers::class, 'handleNodeStatus'],
        'report.devices' => [NodeEventHandlers::class, 'handleDeviceReport'],
        'request.devices' => [NodeEventHandlers::class, 'handleDeviceRequest'],
    ];

    public function __construct(string $host, int $port)
    {
        $this->worker = new Worker("websocket://{$host}:{$port}");
        $this->worker->count = 1;
        $this->worker->name = 'xboard-ws-server';
    }

    public function run(): void
    {
        $this->setupLogging();
        $this->setupCallbacks();
        Worker::runAll();
    }

    private function setupLogging(): void
    {
        $logPath = storage_path('logs');
        if (!is_dir($logPath)) {
            mkdir($logPath, 0777, true);
        }
        Worker::$logFile = $logPath . '/xboard-ws-server.log';
        Worker::$pidFile = $logPath . '/xboard-ws-server.pid';
    }

    private function setupCallbacks(): void
    {
        $this->worker->onWorkerStart = [$this, 'onWorkerStart'];
        $this->worker->onConnect = [$this, 'onConnect'];
        $this->worker->onWebSocketConnect = [$this, 'onWebSocketConnect'];
        $this->worker->onMessage = [$this, 'onMessage'];
        $this->worker->onClose = [$this, 'onClose'];
    }

    public function onWorkerStart(Worker $worker): void
    {
        Log::info("[WS] Worker started, pid={$worker->id}");
        $this->subscribeRedis();
        $this->setupTimers();
    }

    private function setupTimers(): void
    {
        Cache::put(self::HEARTBEAT_CACHE_KEY, time(), self::HEARTBEAT_TTL);
        Timer::add(self::HEARTBEAT_INTERVAL, function () {
            Cache::put(self::HEARTBEAT_CACHE_KEY, time(), self::HEARTBEAT_TTL);
        });

        Timer::add(self::PING_INTERVAL, function () {
            $seen = [];

            foreach (NodeRegistry::getConnectedNodeIds() as $nodeId) {
                $conn = NodeRegistry::get($nodeId);
                if ($conn) {
                    $oid = spl_object_id($conn);
                    if (!isset($seen[$oid])) {
                        $seen[$oid] = true;
                        $conn->send(json_encode(['event' => 'ping']));
                    }
                }
            }

            foreach (NodeRegistry::getConnectedMachineIds() as $machineId) {
                $conn = NodeRegistry::getMachine($machineId);
                if ($conn) {
                    $oid = spl_object_id($conn);
                    if (!isset($seen[$oid])) {
                        $seen[$oid] = true;
                        $conn->send(json_encode(['event' => 'ping']));
                    }
                }
            }

            // 保活客户端连接（Hiddify 等 App）
            ClientRegistry::broadcast('ping');
        });

        Timer::add(10, function () {
            $pendingNodeIds = Redis::spop('device:push_pending_nodes', 100);
            if (empty($pendingNodeIds)) {
                return;
            }

            $service = app(DeviceStateService::class);
            foreach ($pendingNodeIds as $nodeId) {
                $nodeId = (int) $nodeId;
                if (NodeRegistry::get($nodeId) !== null) {
                    NodeEventHandlers::pushDeviceStateToNode($nodeId, $service);
                }
            }
        });
    }

    public function onConnect(TcpConnection $conn): void
    {
        $conn->authTimer = Timer::add(self::AUTH_TIMEOUT, function () use ($conn) {
            if (empty($conn->nodeId) && empty($conn->machineNodeIds) && empty($conn->clientUserId)) {
                $conn->close(json_encode([
                    'event' => 'error',
                    'data' => ['message' => 'auth timeout'],
                ]));
            }
        }, [], false);
    }

    public function onWebSocketConnect(TcpConnection $conn, $httpMessage): void
    {
        $queryString = '';
        if (is_string($httpMessage)) {
            $queryString = parse_url($httpMessage, PHP_URL_QUERY) ?? '';
        } elseif ($httpMessage instanceof \Workerman\Protocols\Http\Request) {
            $queryString = $httpMessage->queryString();
        }

        parse_str($queryString, $params);

        if (isset($conn->authTimer)) {
            Timer::del($conn->authTimer);
        }

        // 判断认证模式
        if (!empty($params['client_token'])) {
            $this->authenticateClient($conn, $params);
        } elseif (!empty($params['machine_id'])) {
            $this->authenticateMachine($conn, $params);
        } else {
            $this->authenticateNode($conn, $params);
        }
    }

    /**
     * 客户端（Hiddify 等 App）认证：用订阅 token（= users.token）识别用户，
     * 注册进 ClientRegistry。连接只用于接收「订阅已更新」信号，不处理节点业务。
     */
    private function authenticateClient(TcpConnection $conn, array $params): void
    {
        $token = (string) ($params['client_token'] ?? '');
        if ($token === '') {
            $conn->close(json_encode([
                'event' => 'error',
                'data' => ['message' => 'invalid client token'],
            ]));
            return;
        }

        $user = \App\Models\User::where('token', $token)->first(['id']);
        if (!$user) {
            $conn->close(json_encode([
                'event' => 'error',
                'data' => ['message' => 'client not found'],
            ]));
            return;
        }

        $conn->clientUserId = (int) $user->id;
        ClientRegistry::add((int) $user->id, $conn);

        Log::debug("[WS] Client user#{$user->id} connected", [
            'remote' => $conn->getRemoteIp(),
            'clients' => ClientRegistry::count(),
        ]);

        $conn->send(json_encode([
            'event' => 'auth.success',
            'data' => ['user_id' => (int) $user->id],
        ]));
    }

    /**
     * 旧模式：单节点认证
     */
    private function authenticateNode(TcpConnection $conn, array $params): void
    {
        $token = $params['token'] ?? '';
        $nodeId = (int) ($params['node_id'] ?? 0);

        $serverToken = admin_setting('server_token', '');
        if ($token === '' || $serverToken === '' || !hash_equals($serverToken, $token)) {
            $conn->close(json_encode([
                'event' => 'error',
                'data' => ['message' => 'invalid token'],
            ]));
            return;
        }

        $node = ServerService::getServer($nodeId, null);
        if (!$node) {
            $conn->close(json_encode([
                'event' => 'error',
                'data' => ['message' => 'node not found'],
            ]));
            return;
        }

        $conn->nodeId = $nodeId;
        NodeRegistry::add($nodeId, $conn);
        Cache::put("node_ws_alive:{$nodeId}", true, 86400);

        app(DeviceStateService::class)->clearAllNodeDevices($nodeId);

        Log::debug("[WS] Node#{$nodeId} connected", [
            'remote' => $conn->getRemoteIp(),
            'total' => NodeRegistry::count(),
        ]);

        $conn->send(json_encode([
            'event' => 'auth.success',
            'data' => ['node_id' => $nodeId],
        ]));

        NodeEventHandlers::pushFullSync($conn, $node);

        // 节点上线→可用集可能变化，通知客户端重拉（开关/节流在 Service 内处理）
        \App\Services\ClientPushService::broadcastSubscriptionUpdated('node.online');
    }

    /**
     * 新模式：机器认证，自动注册该机器下所有已启用节点
     */
    private function authenticateMachine(TcpConnection $conn, array $params): void
    {
        $machineId = (int) ($params['machine_id'] ?? 0);
        $token = $params['token'] ?? '';

        $machine = ServerMachine::where('id', $machineId)
            ->where('token', $token)
            ->first();

        if (!$machine || !$machine->is_active) {
            $conn->close(json_encode([
                'event' => 'error',
                'data' => ['message' => 'invalid machine credentials'],
            ]));
            return;
        }

        $nodes = ServerService::getMachineNodes($machine);

        $machine->forceFill(['last_seen_at' => now()->timestamp])->saveQuietly();
        NodeRegistry::addMachine($machineId, $conn);

        // 把同一个连接注册到该机器下所有节点
        $nodeIds = [];
        $deviceService = app(DeviceStateService::class);
        foreach ($nodes as $node) {
            NodeRegistry::add($node->id, $conn);
            Cache::put("node_ws_alive:{$node->id}", true, 86400);
            $deviceService->clearAllNodeDevices($node->id);
            $nodeIds[] = $node->id;
        }

        // 连接上记录所属机器和节点列表
        $conn->machineId = $machineId;
        $conn->machineNodeIds = $nodeIds;

        Log::debug("[WS] Machine#{$machineId} connected, nodes: " . implode(',', $nodeIds), [
            'remote' => $conn->getRemoteIp(),
            'total' => NodeRegistry::count(),
            'machines' => NodeRegistry::machineCount(),
        ]);

        $conn->send(json_encode([
            'event' => 'auth.success',
            'data' => [
                'machine_id' => $machineId,
                'node_ids' => $nodeIds,
            ],
        ]));

        // 为每个节点推送完整同步
        foreach ($nodes as $node) {
            NodeEventHandlers::pushFullSync($conn, $node);
        }

        // 节点（机器）上线→可用集可能变化，通知客户端重拉
        \App\Services\ClientPushService::broadcastSubscriptionUpdated('node.online');
    }

    public function onMessage(TcpConnection $conn, $data): void
    {
        $msg = json_decode($data, true);
        if (!is_array($msg)) {
            return;
        }

        $event = $msg['event'] ?? '';

        // 客户端连接：只做保活（pong），不处理节点业务
        if (!empty($conn->clientUserId)) {
            return;
        }

        // 机器连接：从消息中读取 node_id 来分派到具体节点
        if (!empty($conn->machineNodeIds)) {
            if ($event === 'pong') {
                foreach ($conn->machineNodeIds as $nid) {
                    Cache::put("node_ws_alive:{$nid}", true, 86400);
                }
                return;
            }

            $nodeId = (int) ($msg['data']['node_id'] ?? 0);
            if ($nodeId <= 0 || !in_array($nodeId, $conn->machineNodeIds, true)) {
                return;
            }
            if (isset($this->handlers[$event])) {
                $handler = $this->handlers[$event];
                $handler($conn, $nodeId, $msg['data'] ?? []);
            }
            return;
        }

        // 旧模式：单节点
        $nodeId = $conn->nodeId ?? null;
        if (isset($this->handlers[$event]) && $nodeId) {
            $handler = $this->handlers[$event];
            $handler($conn, $nodeId, $msg['data'] ?? []);
        }
    }

    public function onClose(TcpConnection $conn): void
    {
        // 客户端连接：仅从客户端注册表移除
        if (!empty($conn->clientUserId)) {
            ClientRegistry::remove((int) $conn->clientUserId, $conn);
            Log::debug("[WS] Client user#{$conn->clientUserId} disconnected", [
                'clients' => ClientRegistry::count(),
            ]);
            return;
        }

        $service = app(DeviceStateService::class);

        // 机器模式：清理所有关联节点
        if (!empty($conn->machineNodeIds)) {
            $machineId = $conn->machineId ?? 'unknown';
            foreach ($conn->machineNodeIds as $nodeId) {
                NodeRegistry::remove($nodeId, $conn);
                Cache::forget("node_ws_alive:{$nodeId}");

                $affectedUserIds = $service->clearAllNodeDevices($nodeId);
                foreach ($affectedUserIds as $userId) {
                    $service->notifyUpdate($userId);
                }
            }

            if (!empty($conn->machineId)) {
                NodeRegistry::removeMachine((int) $conn->machineId, $conn);
            }

            Log::debug("[WS] Machine#{$machineId} disconnected", [
                'nodes' => $conn->machineNodeIds,
                'total' => NodeRegistry::count(),
                'machines' => NodeRegistry::machineCount(),
            ]);

            // 节点（机器）下线→可用集可能变化，通知客户端重拉
            \App\Services\ClientPushService::broadcastSubscriptionUpdated('node.offline');
            return;
        }

        // 旧模式：单节点
        if (!empty($conn->nodeId)) {
            $nodeId = $conn->nodeId;
            NodeRegistry::remove($nodeId, $conn);
            Cache::forget("node_ws_alive:{$nodeId}");

            $affectedUserIds = $service->clearAllNodeDevices($nodeId);
            foreach ($affectedUserIds as $userId) {
                $service->notifyUpdate($userId);
            }

            Log::debug("[WS] Node#{$nodeId} disconnected", [
                'total' => NodeRegistry::count(),
                'affected_users' => count($affectedUserIds),
            ]);

            // 节点下线→可用集可能变化，通知客户端重拉
            \App\Services\ClientPushService::broadcastSubscriptionUpdated('node.offline');
        }
    }

    private function subscribeRedis(): void
    {
        $host = config('database.redis.default.host', '127.0.0.1');
        $port = config('database.redis.default.port', 6379);

        if (str_starts_with($host, '/')) {
            $redisUri = "unix://{$host}";
        } else {
            $redisUri = "redis://{$host}:{$port}";
        }

        $redis = new \Workerman\Redis\Client($redisUri);

        $password = config('database.redis.default.password');
        if ($password) {
            $redis->auth($password);
        }

        $prefix = config('database.redis.options.prefix', '');
        $channel = $prefix . 'node:push';
        $clientChannel = $prefix . \App\Services\ClientPushService::CHANNEL;

        $redis->subscribe([$channel, $clientChannel], function ($chan, $message) use ($clientChannel) {
            $payload = json_decode($message, true);
            if (!is_array($payload)) {
                return;
            }

            // 客户端推送频道：分派给 ClientRegistry（定向用户或全体广播）
            if ($chan === $clientChannel) {
                $event = $payload['event'] ?? '';
                if ($event === '') {
                    return;
                }
                $data = $payload['data'] ?? [];
                $scope = $payload['scope'] ?? 'all';
                if ($scope === 'user' && !empty($payload['user_id'])) {
                    $sent = ClientRegistry::sendToUser((int) $payload['user_id'], $event, $data);
                } else {
                    $sent = ClientRegistry::broadcast($event, $data);
                }
                if ($sent) {
                    Log::debug("[WS] Pushed {$event} to {$sent} client(s)");
                }
                return;
            }

            $event = $payload['event'] ?? '';
            $data = $payload['data'] ?? [];

            // Machine-level events (e.g., sync.nodes)
            $machineId = $payload['machine_id'] ?? null;
            if ($machineId && $event) {
                // Update server-side registry when node membership changes
                if ($event === 'sync.nodes') {
                    $nodeIds = array_map('intval', array_column($data['nodes'] ?? [], 'id'));
                    NodeRegistry::refreshMachineNodes((int) $machineId, $nodeIds);
                }

                $sent = NodeRegistry::sendMachine((int) $machineId, $event, $data);
                if ($sent) {
                    Log::debug("[WS] Pushed {$event} to machine#{$machineId}");
                }
                return;
            }

            // Per-node events
            $nodeId = $payload['node_id'] ?? null;
            if (!$nodeId || !$event) {
                return;
            }

            $sent = NodeRegistry::send((int) $nodeId, $event, $data);
            if ($sent) {
                Log::debug("[WS] Pushed {$event} to node#{$nodeId}");
            }
        });

        Log::info("[WS] Subscribed to Redis channels: {$channel}, {$clientChannel}");
    }
}
