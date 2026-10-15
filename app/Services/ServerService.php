<?php

namespace App\Services;

use App\Models\Server;
use App\Models\ServerMachine;
use App\Models\ServerRoute;
use App\Models\User;
use App\Services\Plugin\HookManager;
use App\Utils\CacheKey;
use App\Utils\Helper;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Collection;

class ServerService
{

    /**
     * 获取所有服务器列表
     * @return Collection
     */
    public static function getAllServers(): Collection
    {
        $query = Server::orderBy('sort', 'ASC');

        return $query->get()->append([
            'last_check_at',
            'last_push_at',
            'online',
            'is_online',
            'available_status',
            'cache_key',
            'load_status',
            'metrics',
            'online_conn'
        ]);
    }

    /**
     * 获取机器下所有已启用节点
     */
    public static function getMachineNodes(ServerMachine $machine): Collection
    {
        return Server::where('machine_id', $machine->id)
            ->where('enabled', true)
            ->orderBy('sort', 'ASC')
            ->get();
    }

    /**
     * 获取指定用户可用的服务器列表
     * @param User $user
     * @return array
     */
    public static function getAvailableServers(User $user): array
    {
        $servers = Server::whereJsonContains('group_ids', (int) $user->group_id)
            ->where('show', true)
            ->where(function ($query) {
                $query->whereNull('transfer_enable')
                    ->orWhere('transfer_enable', 0)
                    ->orWhereRaw('u + d < transfer_enable');
            })
            ->orderBy('sort', 'ASC')
            ->get()
            ->append(['last_check_at', 'last_push_at', 'online', 'is_online', 'available_status', 'cache_key', 'server_key']);

        $servers = collect($servers)->map(function ($server) use ($user) {
            // 判断动态端口
            if (str_contains($server->port, '-')) {
                $port = $server->port;
                $server->port = (int) Helper::randomPort($port);
                $server->ports = $port;
            } else {
                $server->port = (int) $server->port;
            }
            $server->password = $server->generateServerPassword($user);
            $server->rate = $server->getCurrentRate();
            return $server;
        })->toArray();

        return $servers;
    }

    /**
     * 判断节点是否「可用/可下发」。
     *
     * 以**面板实测连通**为主：最近一次测速成功（latency_ms 非空且未过期）即视为可用；
     * 同时兼容 agent 心跳（is_online）。两者满足其一即可——很多部署节点 agent 不上报
     * 心跳（is_online=0）但节点其实能用，这时实测结果才是真相。
     *
     * @param array $s 节点数组（含 is_online / latency_ms / last_latency_at）
     */
    public static function isNodeAvailable(array $s, ?int $now = null, ?int $stale = null): bool
    {
        if (!empty($s['is_online'])) {
            return true;
        }
        $now ??= time();
        $stale ??= self::effectiveStaleSeconds();
        return ($s['latency_ms'] ?? null) !== null
            && !empty($s['last_latency_at'])
            && ($now - (int) $s['last_latency_at']) <= $stale;
    }

    /**
     * 实际使用的「测速结果新鲜度窗口」（秒）。
     *
     * 为什么不直接用 server_latency_stale_seconds：开启自动测速时，这个窗口必须
     * 至少覆盖一个测速周期，否则两次测速之间节点必然集体过期 → 全部被判不可用 →
     * 客户端全不下发（现象：UI 显示「测速于 18m」却整列不下发，因为间隔 30m 而
     * stale 只有 10m）。所以这里取 `max(配置值, 测速间隔×1.5)`：1.5 倍留出单轮
     * 测速耗时、网络抖动、偶发失败一轮的余量，保证节点在下一轮测速补上之前都算新鲜。
     *
     * 关闭自动测速时维持配置原值（此时新鲜度完全由管理员/外部来源掌控）。
     */
    public static function effectiveStaleSeconds(): int
    {
        $stale = (int) admin_setting('server_latency_stale_seconds', 600);
        if ((bool) admin_setting('server_latency_probe_enable', 1)) {
            $intervalSec = max(1, (int) admin_setting('server_latency_probe_interval', 3)) * 60;
            $stale = max($stale, (int) ($intervalSec * 1.5));
        }
        return $stale;
    }

    /**
     * 「每个国家只下发一个节点」：按国家去重，每组选最优的一个。
     *
     * 仅在订阅下发路径调用（ClientController::doSubscribe），不影响后台节点列表等
     * 其它共用 getAvailableServers 的地方。入参/出参都是 getAvailableServers 产出的
     * 数组（含 is_online / online / country_code / latency_ms / last_latency_at）。
     *
     * 规则：
     *  1. 可用性以 agent 上报的 is_online 为准（协议无关），先剔除离线节点；
     *  2. 有 country_code 的按国家分组，每组优先取「延迟新鲜且最低」者，
     *     若全组无有效延迟（如 UDP 节点 ping 不到）则按在线人数/ sort 兜底取一个；
     *  3. 没有 country_code 的节点原样保留（不参与去重，避免误删）；
     *  4. server_country_naming 开启时，把入选节点名规范成「🇺🇸 国家名」，
     *     便于客户端按国家分组与显示旗帜。
     *
     * @param  array  $servers
     * @return array
     */
    public static function pickBestPerCountry(array $servers): array
    {
        $now = time();
        $stale = self::effectiveStaleSeconds();
        $naming = (bool) admin_setting('server_country_naming', true);

        $noCountry = [];
        $byCountry = [];
        foreach ($servers as $server) {
            if (!self::isNodeAvailable($server, $now, $stale)) {
                continue; // 不可用节点不下发
            }
            $cc = $server['country_code'] ?? null;
            if (blank($cc)) {
                $noCountry[] = $server;
            } else {
                $byCountry[strtoupper($cc)][] = $server;
            }
        }

        $picked = [];
        foreach ($byCountry as $cc => $group) {
            $best = self::chooseBestInCountry($group, $now, $stale);
            if ($naming) {
                $best['name'] = self::countryDisplayName($cc);
            }
            $picked[] = $best;
        }

        $result = array_merge($picked, $noCountry);
        // 稳定排序，避免客户端里节点顺序乱跳
        usort($result, fn($a, $b) => ($a['sort'] ?? 0) <=> ($b['sort'] ?? 0));
        return array_values($result);
    }

    /**
     * 在同一国家的候选里选最优：优先延迟新鲜且最低者；否则按在线人数、sort 兜底。
     */
    private static function chooseBestInCountry(array $group, int $now, int $stale): array
    {
        $fresh = array_values(array_filter($group, function ($s) use ($now, $stale) {
            return ($s['latency_ms'] ?? null) !== null
                && !empty($s['last_latency_at'])
                && ($now - (int) $s['last_latency_at']) <= $stale;
        }));

        if (!empty($fresh)) {
            usort($fresh, fn($a, $b) => (int) $a['latency_ms'] <=> (int) $b['latency_ms']);
            return $fresh[0];
        }

        usort($group, function ($a, $b) {
            $oa = (int) ($a['online'] ?? 0);
            $ob = (int) ($b['online'] ?? 0);
            if ($oa !== $ob) {
                return $oa <=> $ob; // 在线人数少的优先
            }
            return ($a['sort'] ?? 0) <=> ($b['sort'] ?? 0);
        });
        return $group[0];
    }

    /**
     * 国家码 -> 「🇺🇸 国家名」。旗帜由两位国家码转 regional-indicator 计算；
     * 国家名用常见国家中文映射，未收录时回退到国家码本身。
     */
    public static function countryDisplayName(string $cc): string
    {
        $cc = strtoupper($cc);
        $name = self::COUNTRY_NAMES_ZH[$cc] ?? $cc;
        return self::countryFlag($cc) . ' ' . $name;
    }

    /**
     * 两位 ISO 国家码 -> 旗帜 emoji（regional indicator symbols）。
     */
    public static function countryFlag(string $cc): string
    {
        $cc = strtoupper($cc);
        if (strlen($cc) !== 2 || !ctype_alpha($cc)) {
            return '🏳️';
        }
        $flag = '';
        foreach (str_split($cc) as $ch) {
            $flag .= mb_chr(0x1F1E6 + (ord($ch) - ord('A')), 'UTF-8');
        }
        return $flag;
    }

    /**
     * 从节点名解析国家码（ISO alpha-2 大写），解析不到返回 null。
     *
     * 对「中转/relay」节点，host 的 GeoIP 是入口国（常是国内），而落地国通常写在
     * 节点名里（🇯🇵 / 日本 / JP）。所以 tcp 探测模式下优先用节点名判国家。
     * 优先级：旗帜 emoji > 中文名关键词 > 英文名/国家码关键词。
     */
    public static function parseCountryFromName(?string $name): ?string
    {
        if (blank($name)) {
            return null;
        }

        // 1) 旗帜 emoji：两个连续的 regional indicator symbol（U+1F1E6..U+1F1FF）
        if (preg_match_all('/[\x{1F1E6}-\x{1F1FF}]/u', $name, $m) && count($m[0]) >= 2) {
            $a = mb_ord($m[0][0], 'UTF-8') - 0x1F1E6;
            $b = mb_ord($m[0][1], 'UTF-8') - 0x1F1E6;
            if ($a >= 0 && $a < 26 && $b >= 0 && $b < 26) {
                return chr(ord('A') + $a) . chr(ord('A') + $b);
            }
        }

        // 2) 中文名关键词（含常见别名）
        foreach (self::COUNTRY_NAME_ALIASES as $cc => $aliases) {
            foreach ($aliases as $kw) {
                if (mb_strpos($name, $kw) !== false) {
                    return $cc;
                }
            }
        }

        // 3) 英文名 / 国家码（按单词边界，避免 "US" 命中 "Russia" 之类的误伤）
        $upper = strtoupper($name);
        foreach (self::COUNTRY_EN_KEYWORDS as $cc => $kws) {
            foreach ($kws as $kw) {
                if (preg_match('/\b' . preg_quote($kw, '/') . '\b/', $upper)) {
                    return $cc;
                }
            }
        }

        return null;
    }

    /** 中文名/别名关键词 -> 国家码（用于 parseCountryFromName） */
    private const COUNTRY_NAME_ALIASES = [
        'HK' => ['香港', '港'], 'TW' => ['台湾', '臺灣', '台'], 'MO' => ['澳门', '澳門'],
        'JP' => ['日本', '东京', '大阪'], 'KR' => ['韩国', '韓國', '首尔'],
        'SG' => ['新加坡', '狮城'], 'MY' => ['马来', '馬來'], 'TH' => ['泰国', '泰國'],
        'VN' => ['越南'], 'PH' => ['菲律宾', '菲律賓'], 'ID' => ['印尼', '印度尼西亚'],
        'IN' => ['印度'], 'TR' => ['土耳其'], 'AE' => ['阿联酋', '迪拜'], 'IL' => ['以色列'],
        'US' => ['美国', '美國', '洛杉矶', '硅谷', '圣何塞', '纽约'], 'CA' => ['加拿大'],
        'MX' => ['墨西哥'], 'BR' => ['巴西'], 'AR' => ['阿根廷'], 'GB' => ['英国', '英國', '伦敦'],
        'DE' => ['德国', '德國', '法兰克福'], 'FR' => ['法国', '法國', '巴黎'],
        'NL' => ['荷兰', '荷蘭', '阿姆斯特丹'], 'RU' => ['俄罗斯', '俄羅斯', '莫斯科'],
        'UA' => ['乌克兰', '烏克蘭'], 'IT' => ['意大利'], 'ES' => ['西班牙'], 'PT' => ['葡萄牙'],
        'SE' => ['瑞典'], 'NO' => ['挪威'], 'FI' => ['芬兰', '芬蘭'], 'DK' => ['丹麦', '丹麥'],
        'PL' => ['波兰', '波蘭'], 'CH' => ['瑞士'], 'AT' => ['奥地利', '奧地利'],
        'IE' => ['爱尔兰', '愛爾蘭'], 'BE' => ['比利时', '比利時'], 'CZ' => ['捷克'],
        'RO' => ['罗马尼亚', '羅馬尼亞'], 'AU' => ['澳大利亚', '澳洲', '悉尼', '墨尔本'],
        'NZ' => ['新西兰', '紐西蘭'], 'ZA' => ['南非'], 'CN' => ['中国', '中國', '大陆', '回国'],
    ];

    /** 英文名/国家码关键词（大写，按单词边界匹配） */
    private const COUNTRY_EN_KEYWORDS = [
        'HK' => ['HK', 'HONG KONG'], 'TW' => ['TW', 'TAIWAN'], 'JP' => ['JP', 'JAPAN', 'TOKYO'],
        'KR' => ['KR', 'KOREA', 'SEOUL'], 'SG' => ['SG', 'SINGAPORE'], 'US' => ['US', 'USA', 'UNITED STATES', 'LOS ANGELES'],
        'GB' => ['GB', 'UK', 'UNITED KINGDOM', 'LONDON'], 'DE' => ['DE', 'GERMANY'], 'FR' => ['FR', 'FRANCE', 'PARIS'],
        'NL' => ['NL', 'NETHERLANDS'], 'RU' => ['RU', 'RUSSIA'], 'CA' => ['CA', 'CANADA'], 'AU' => ['AU', 'AUSTRALIA'],
        'MY' => ['MY', 'MALAYSIA'], 'TH' => ['TH', 'THAILAND'], 'VN' => ['VN', 'VIETNAM'], 'IN' => ['IN', 'INDIA'],
        'TR' => ['TR', 'TURKEY'], 'AE' => ['AE', 'DUBAI'], 'CN' => ['CN', 'CHINA'],
    ];

    /**
     * 常见国家/地区中文名映射（非穷举，未收录回退到国家码）。
     */
    private const COUNTRY_NAMES_ZH = [
        'HK' => '香港', 'TW' => '台湾', 'MO' => '澳门', 'CN' => '中国大陆',
        'JP' => '日本', 'KR' => '韩国', 'SG' => '新加坡', 'MY' => '马来西亚',
        'TH' => '泰国', 'VN' => '越南', 'PH' => '菲律宾', 'ID' => '印尼',
        'IN' => '印度', 'TR' => '土耳其', 'AE' => '阿联酋', 'IL' => '以色列',
        'US' => '美国', 'CA' => '加拿大', 'MX' => '墨西哥', 'BR' => '巴西',
        'AR' => '阿根廷', 'GB' => '英国', 'DE' => '德国', 'FR' => '法国',
        'NL' => '荷兰', 'RU' => '俄罗斯', 'UA' => '乌克兰', 'IT' => '意大利',
        'ES' => '西班牙', 'PT' => '葡萄牙', 'SE' => '瑞典', 'NO' => '挪威',
        'FI' => '芬兰', 'DK' => '丹麦', 'PL' => '波兰', 'CH' => '瑞士',
        'AT' => '奥地利', 'IE' => '爱尔兰', 'BE' => '比利时', 'CZ' => '捷克',
        'RO' => '罗马尼亚', 'AU' => '澳大利亚', 'NZ' => '新西兰', 'ZA' => '南非',
    ];

    /**
     * 计算一组节点的「版本号」。
     *
     * 基于每个节点的 cache_key（见 Server::cacheKey，内含 updated_at 与
     * is_online），所以新增/删除节点、修改节点配置、节点上下线都会让它变化。
     *
     * 客户端用它判断"要不要重新下载订阅"（ServerController::revision）、
     * fetch 接口用它做 ETag、管理端用它统计更新覆盖率——三处必须共用同一个
     * 实现，任何一处算法漂移都会造成漏更新或统计对不上。
     */
    public static function calcNodeRevision(array $servers): string
    {
        return sha1(json_encode(array_column($servers, 'cache_key')));
    }

    /**
     * 根据权限组获取可用的用户列表
     * @param array $groupIds
     * @return Collection
     */
    public static function getAvailableUsers(Server $node)
    {
        $groupIds = $node->group_ids ?? [];
        if (empty($groupIds)) {
            return collect();
        }
        $users = User::toBase()
            ->whereIn('group_id', $groupIds)
            ->whereRaw('u + d < transfer_enable')
            ->where(function ($query) {
                $query->where('expired_at', '>=', time())
                    ->orWhere('expired_at', NULL);
            })
            ->where('banned', 0)
            ->select([
                'id',
                'uuid',
                'speed_limit',
                'device_limit'
            ])
            ->get();
        return HookManager::filter('server.users.get', $users, $node);
    }

    // 获取路由规则
    public static function getRoutes(array $routeIds)
    {
        $routes = ServerRoute::select(['id', 'match', 'action', 'action_value'])->whereIn('id', $routeIds)->get();
        return $routes;
    }

    /**
     * 处理节点流量数据汇报
     */
    public static function processTraffic(Server $node, array $traffic): void
    {
        $data = array_filter($traffic, fn($item) =>
            is_array($item) && count($item) === 2
            && is_numeric($item[0]) && is_numeric($item[1])
        );

        if (empty($data)) {
            return;
        }

        $nodeType = strtoupper($node->type);
        $nodeId = $node->id;

        Cache::put(CacheKey::get("SERVER_{$nodeType}_ONLINE_USER", $nodeId), count($data), 3600);
        Cache::put(CacheKey::get("SERVER_{$nodeType}_LAST_PUSH_AT", $nodeId), time(), 3600);

        (new UserService())->trafficFetch($node, $node->type, $data);
    }

    /**
     * 处理节点在线设备汇报
     */
    public static function processAlive(int $nodeId, array $alive): void
    {
        $service = app(DeviceStateService::class);
        foreach ($alive as $uid => $ips) {
            $service->setDevices((int) $uid, $nodeId, (array) $ips);
        }
    }

    /**
     * 处理节点连接数汇报
     */
    public static function processOnline(Server $node, array $online): void
    {
        $cacheTime = max(300, (int) admin_setting('server_push_interval', 60) * 3);
        $nodeType = $node->type;
        $nodeId = $node->id;

        foreach ($online as $uid => $conn) {
            $cacheKey = CacheKey::get("USER_ONLINE_CONN_{$nodeType}_{$nodeId}", $uid);
            Cache::put($cacheKey, (int) $conn, $cacheTime);
        }
    }

    /**
     * 处理节点负载状态汇报
     */
    public static function processStatus(Server $node, array $status): void
    {
        $nodeType = strtoupper($node->type);
        $nodeId = $node->id;

        $statusData = [
            'cpu' => (float) ($status['cpu'] ?? 0),
            'mem' => [
                'total' => (int) ($status['mem']['total'] ?? 0),
                'used' => (int) ($status['mem']['used'] ?? 0),
            ],
            'swap' => [
                'total' => (int) ($status['swap']['total'] ?? 0),
                'used' => (int) ($status['swap']['used'] ?? 0),
            ],
            'disk' => [
                'total' => (int) ($status['disk']['total'] ?? 0),
                'used' => (int) ($status['disk']['used'] ?? 0),
            ],
            'updated_at' => now()->timestamp,
            'kernel_status' => $status['kernel_status'] ?? null,
        ];

        $cacheTime = max(300, (int) admin_setting('server_push_interval', 60) * 3);
        cache([
            CacheKey::get("SERVER_{$nodeType}_LOAD_STATUS", $nodeId) => $statusData,
            CacheKey::get("SERVER_{$nodeType}_LAST_LOAD_AT", $nodeId) => now()->timestamp,
        ], $cacheTime);
    }

    /**
     * 标记节点心跳
     */
    public static function touchNode(Server $node): void
    {
        Cache::put(
            CacheKey::get('SERVER_' . strtoupper($node->type) . '_LAST_CHECK_AT', $node->id),
            time(),
            3600
        );
    }

    /**
     * Update node metrics and load status
     */
    public static function updateMetrics(Server $node, array $metrics): void
    {
        $nodeType = strtoupper($node->type);
        $nodeId = $node->id;
        $cacheTime = max(300, (int) admin_setting('server_push_interval', 60) * 3);

        $metricsData = [
            'uptime' => (int) ($metrics['uptime'] ?? 0),
            'goroutines' => (int) ($metrics['goroutines'] ?? 0),
            'active_connections' => (int) ($metrics['active_connections'] ?? 0),
            'total_connections' => (int) ($metrics['total_connections'] ?? 0),
            'total_users' => (int) ($metrics['total_users'] ?? 0),
            'active_users' => (int) ($metrics['active_users'] ?? 0),
            'inbound_speed' => (int) ($metrics['inbound_speed'] ?? 0),
            'outbound_speed' => (int) ($metrics['outbound_speed'] ?? 0),
            'cpu_per_core' => $metrics['cpu_per_core'] ?? [],
            'load' => $metrics['load'] ?? [],
            'speed_limiter' => $metrics['speed_limiter'] ?? [],
            'gc' => $metrics['gc'] ?? [],
            'api' => $metrics['api'] ?? [],
            'ws' => $metrics['ws'] ?? [],
            'limits' => $metrics['limits'] ?? [],
            'updated_at' => now()->timestamp,
            'kernel_status' => (bool) ($metrics['kernel_status'] ?? false),
        ];

        Cache::put(
            CacheKey::get('SERVER_' . $nodeType . '_METRICS', $nodeId),
            $metricsData,
            $cacheTime
        );
    }

    public static function buildNodeConfig(Server $node): array
    {
        $nodeType = $node->type;
        $protocolSettings = $node->protocol_settings;
        $serverPort = $node->server_port;
        $host = $node->host;

        $baseConfig = [
            'protocol' => $nodeType,
            'listen_ip' => '0.0.0.0',
            'server_port' => (int) $serverPort,
            'network' => data_get($protocolSettings, 'network'),
            'networkSettings' => data_get($protocolSettings, 'network_settings') ?: null,
        ];

        $response = match ($nodeType) {
            'shadowsocks' => [
                ...$baseConfig,
                'cipher' => $protocolSettings['cipher'],
                'plugin' => $protocolSettings['plugin'],
                'plugin_opts' => $protocolSettings['plugin_opts'],
                'server_key' => match ($protocolSettings['cipher']) {
                        '2022-blake3-aes-128-gcm' => Helper::getServerKey($node->created_at, 16),
                        '2022-blake3-aes-256-gcm' => Helper::getServerKey($node->created_at, 32),
                        default => null,
                    },
            ],
            'vmess' => [
                ...$baseConfig,
                'tls' => (int) $protocolSettings['tls'],
                'tls_settings' => $protocolSettings['tls_settings'],
                'multiplex' => data_get($protocolSettings, 'multiplex'),
            ],
            'trojan' => [
                ...$baseConfig,
                'host' => $host,
                'server_name' => data_get($protocolSettings, 'tls_settings.server_name'),
                'multiplex' => data_get($protocolSettings, 'multiplex'),
                'tls' => (int) $protocolSettings['tls'],
                'tls_settings' => match ((int) $protocolSettings['tls']) {
                        2 => $protocolSettings['reality_settings'],
                        default => $protocolSettings['tls_settings'],
                    },
            ],
            'vless' => [
                ...$baseConfig,
                'tls' => (int) $protocolSettings['tls'],
                'flow' => $protocolSettings['flow'],
                'decryption' => match (data_get($protocolSettings, 'encryption.enabled')) {
                    true => data_get($protocolSettings, 'encryption.decryption'),
                    default => null,
                },
                'tls_settings' => match ((int) $protocolSettings['tls']) {
                        2 => $protocolSettings['reality_settings'],
                        default => $protocolSettings['tls_settings'],
                    },
                'multiplex' => data_get($protocolSettings, 'multiplex'),
            ],
            'hysteria' => [
                ...$baseConfig,
                'server_port' => (int) $serverPort,
                'version' => (int) $protocolSettings['version'],
                'host' => $host,
                'server_name' => $protocolSettings['tls']['server_name'],
                'tls_settings' => $protocolSettings['tls'],
                'up_mbps' => (int) $protocolSettings['bandwidth']['up'],
                'down_mbps' => (int) $protocolSettings['bandwidth']['down'],
                ...match ((int) $protocolSettings['version']) {
                        1 => ['obfs' => $protocolSettings['obfs']['password'] ?? null],
                        2 => [
                            'obfs' => $protocolSettings['obfs']['open'] ? $protocolSettings['obfs']['type'] : null,
                            'obfs-password' => $protocolSettings['obfs']['password'] ?? null,
                        ],
                        default => [],
                    },
            ],
            'tuic' => [
                ...$baseConfig,
                'version' => (int) $protocolSettings['version'],
                'server_port' => (int) $serverPort,
                'server_name' => $protocolSettings['tls']['server_name'],
                'congestion_control' => $protocolSettings['congestion_control'],
                'tls_settings' => $protocolSettings['tls'],
                'auth_timeout' => '3s',
                'zero_rtt_handshake' => false,
                'heartbeat' => '3s',
            ],
            'anytls' => [
                ...$baseConfig,
                'server_port' => (int) $serverPort,
                'server_name' => $protocolSettings['tls']['server_name'],
                'tls_settings' => $protocolSettings['tls'],
                'padding_scheme' => $protocolSettings['padding_scheme'],
            ],
            'socks' => [
                ...$baseConfig,
                'server_port' => (int) $serverPort,
                'tls' => (int) data_get($protocolSettings, 'tls', 0),
                'tls_settings' => data_get($protocolSettings, 'tls_settings'),
            ],
            'naive' => [
                ...$baseConfig,
                'server_port' => (int) $serverPort,
                'tls' => (int) $protocolSettings['tls'],
                'tls_settings' => $protocolSettings['tls_settings'],
            ],
            'http' => [
                ...$baseConfig,
                'server_port' => (int) $serverPort,
                'tls' => (int) $protocolSettings['tls'],
                'tls_settings' => $protocolSettings['tls_settings'],
            ],
            'mieru' => [
                ...$baseConfig,
                'server_port' => (int) $serverPort,
                'transport' => data_get($protocolSettings, 'transport', 'TCP'),
                'traffic_pattern' => $protocolSettings['traffic_pattern'],
            ],
            default => [],
        };

        if (!empty($node['route_ids'])) {
            $response['routes'] = self::getRoutes($node['route_ids']);
        }

        if (!empty($node['custom_outbounds'])) {
            $response['custom_outbounds'] = $node['custom_outbounds'];
        }

        if (!empty($node['custom_routes'])) {
            $response['custom_routes'] = $node['custom_routes'];
        }

        if (!empty($node['cert_config'])) {
            $certConfig = $node['cert_config'];
            // Normalize: accept both "mode" and "cert_mode" from the database
            if (isset($certConfig['mode']) && !isset($certConfig['cert_mode'])) {
                $certConfig['cert_mode'] = $certConfig['mode'];
                unset($certConfig['mode']);
            }
            if (data_get($certConfig, 'cert_mode') !== 'none') {
                $response['cert_config'] = $certConfig;
            }
        }

        return $response;
    }

    /**
     * 根据协议类型和标识获取服务器
     * @param int $serverId
     * @param string $serverType
     * @return Server|null
     */
    public static function getServer($serverId, ?string $serverType = null): Server | null
    {
        // `code` is not guaranteed unique per type (e.g. imported nodes may share
        // one proxy account's UUID across several servers), so when several rows
        // match this returns an arbitrary one of them.
        return Server::query()
            ->when($serverType, function ($query) use ($serverType) {
                $query->where('type', Server::normalizeType($serverType));
            })
            ->where(function ($query) use ($serverId) {
                $query->where('code', $serverId)
                    ->orWhere('id', $serverId);
            })
            ->orderByRaw('CASE WHEN code = ? THEN 0 ELSE 1 END', [$serverId])
            ->first();
    }
}
