<?php

namespace App\Http\Controllers\V2\Admin\Server;

use App\Exceptions\ApiException;
use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\ServerSave;
use App\Models\Server;
use App\Models\ServerGroup;
use App\Models\User;
use App\Services\ProxyParserService;
use App\Services\ServerService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

class ManageController extends Controller
{
    public function getNodes(Request $request)
    {
        $servers = ServerService::getAllServers()->map(function ($item) {
            $item['groups'] = ServerGroup::whereIn('id', $item['group_ids'] ?? [])->get(['name', 'id']);
            $item['parent'] = $item->parent;
            return $item;
        });
        return $this->success($servers);
    }

    public function sort(Request $request)
    {
        ini_set('post_max_size', '1m');
        $params = $request->validate([
            '*.id' => 'numeric',
            '*.order' => 'numeric'
        ]);

        try {
            DB::beginTransaction();
            collect($params)->each(function ($item) {
                if (isset($item['id']) && isset($item['order'])) {
                    Server::where('id', $item['id'])->update(['sort' => $item['order']]);
                }
            });
            DB::commit();
        } catch (\Exception $e) {
            DB::rollBack();
            Log::error($e);
            return $this->fail([500, '保存失败']);

        }
        return $this->success(true);
    }

    public function save(ServerSave $request)
    {
        $params = $request->validated();
        if ($request->input('id')) {
            $server = Server::find($request->input('id'));
            if (!$server) {
                return $this->fail([400202, '服务器不存在']);
            }
            try {
                $server->update($params);
                return $this->success(true);
            } catch (\Exception $e) {
                Log::error($e);
                return $this->fail([500, '保存失败']);
            }
        }

        try {
            Server::create($params);
            return $this->success(true);
        } catch (\Exception $e) {
            Log::error($e);
            return $this->fail([500, '创建失败']);
        }
    }

    /**
     * 解析代理链接或订阅地址，返回预览结果（不写库）。
     */
    public function parseNodes(Request $request)
    {
        $params = $request->validate([
            'type'    => 'required|in:links,subscription',
            'content' => 'required_if:type,links|nullable|string|max:500000',
            'url'     => 'required_if:type,subscription|nullable|url|max:2048',
        ]);

        $parser = new ProxyParserService();

        $parsed = $params['type'] === 'links'
            ? $parser->parseFromText($params['content'] ?? '')
            : $parser->parseFromSubscription($params['url'] ?? '');

        $servers = array_map(function ($s, $i) {
            return [
                'index'   => $i,
                'success' => true,
                'type'    => $s['type'] ?? null,
                'code'    => $s['code'] ?? null,
                'name'    => $s['name'] ?? null,
                'host'    => $s['host'] ?? null,
                'port'    => $s['port'] ?? null,
            ];
        }, $parsed['servers'], array_keys($parsed['servers']));

        return $this->success([
            'servers'      => $servers,
            'parse_errors' => $parsed['errors'],
        ]);
    }

    /**
     * 批量添加节点
     * 接受 servers 数组，每个元素结构与 save 接口一致（不含 id 字段）。
     * 返回每条记录的处理结果，事务回滚只影响单条失败记录，其余继续插入。
     */
    public function batchSave(Request $request)
    {
        $request->validate([
            'servers'   => 'required|array|min:1|max:100',
            'servers.*' => 'array',
        ]);

        $list    = $request->input('servers');
        $results = [];
        $created = 0;
        $failed  = 0;

        foreach ($list as $index => $serverData) {
            // 基础字段校验
            $type = $serverData['type'] ?? null;
            if (!Server::isValidType($type)) {
                $results[] = ['index' => $index, 'success' => false, 'error' => "type 不合法: {$type}"];
                $failed++;
                continue;
            }

            $required = ['name', 'host', 'port', 'server_port', 'rate'];
            foreach ($required as $field) {
                if (!isset($serverData[$field]) || $serverData[$field] === '') {
                    $results[] = ['index' => $index, 'success' => false, 'error' => "{$field} 不能为空"];
                    $failed++;
                    continue 2;
                }
            }

            // 不允许通过批量接口更新已有节点
            unset($serverData['id']);

            try {
                $server    = Server::create($serverData);
                $results[] = ['index' => $index, 'success' => true, 'id' => $server->id, 'name' => $server->name];
                $created++;
            } catch (\Exception $e) {
                Log::error('batchSave server error', ['index' => $index, 'error' => $e->getMessage()]);
                $results[] = ['index' => $index, 'success' => false, 'error' => '创建失败: ' . $e->getMessage()];
                $failed++;
            }
        }

        return $this->success([
            'created' => $created,
            'failed'  => $failed,
            'results' => $results,
        ]);
    }

    /**
     * 从剪切板内容或订阅链接批量导入节点。
     *
     * type=links   : content 为多行代理 URI 文本（vmess/vless/trojan/ss/hysteria2/tuic）
     * type=subscription : url 为订阅地址，自动抓取并解析
     *
     * defaults 可覆盖所有导入节点的公共字段（group_ids / route_ids / rate / show / tags）。
     */
    public function batchImport(Request $request)
    {
        $params = $request->validate([
            'type'              => 'required|in:links,subscription',
            'content'           => 'required_if:type,links|nullable|string|max:500000',
            'url'               => 'required_if:type,subscription|nullable|url|max:2048',
            'defaults'          => 'nullable|array',
            'defaults.group_ids'=> 'nullable|array',
            'defaults.group_ids.*' => 'integer',
            'defaults.route_ids'=> 'nullable|array',
            'defaults.route_ids.*' => 'integer',
            'defaults.rate'     => 'nullable|numeric|min:0',
            'defaults.show'     => 'nullable|integer|in:0,1',
            'defaults.tags'     => 'nullable|array',
        ]);

        $parser = new ProxyParserService();

        if ($params['type'] === 'links') {
            $parsed = $parser->parseFromText($params['content'] ?? '');
        } else {
            $parsed = $parser->parseFromSubscription($params['url'] ?? '');
        }

        $defaults    = $params['defaults'] ?? [];
        $parseErrors = $parsed['errors'];
        $results     = [];
        $created     = 0;
        $failed      = 0;

        foreach ($parsed['servers'] as $index => $serverData) {
            // `code` is intentionally not unique: links from the same subscription
            // commonly share one auth UUID across many nodes (it identifies the
            // account, not the node), and the caller wants that UUID preserved as
            // the custom node ID on every node that carries it.

            // Merge caller-supplied defaults; caller wins on group_ids/route_ids/tags/rate/show
            if (!empty($defaults['group_ids'])) {
                $serverData['group_ids'] = $defaults['group_ids'];
            }
            if (!empty($defaults['route_ids'])) {
                $serverData['route_ids'] = $defaults['route_ids'];
            }
            if (!empty($defaults['tags'])) {
                $serverData['tags'] = $defaults['tags'];
            }
            if (array_key_exists('rate', $defaults) && $defaults['rate'] !== null) {
                $serverData['rate'] = $defaults['rate'];
            }
            $serverData['show'] = (int) ($defaults['show'] ?? 0);

            // Never carry over an id from parsed data
            unset($serverData['id']);

            try {
                $server = Server::create($serverData);
                $results[] = [
                    'index'   => $index,
                    'success' => true,
                    'id'      => $server->id,
                    'name'    => $server->name,
                    'type'    => $server->type,
                ];
                $created++;
            } catch (\Exception $e) {
                Log::error('batchImport server error', [
                    'index' => $index,
                    'name'  => $serverData['name'] ?? '',
                    'error' => $e->getMessage(),
                ]);
                $results[] = [
                    'index'   => $index,
                    'success' => false,
                    'name'    => $serverData['name'] ?? '',
                    'error'   => '创建失败: ' . $e->getMessage(),
                ];
                $failed++;
            }
        }

        return $this->success([
            'created'      => $created,
            'failed'       => $failed,
            'parse_errors' => $parseErrors,
            'results'      => $results,
        ]);
    }

    public function update(Request $request)
    {
        $params = $request->validate([
            'id' => 'required|integer',
            'show' => 'nullable|integer',
            'machine_id' => 'nullable|integer',
            'enabled' => 'nullable|boolean',
        ]);

        $server = Server::find($request->id);
        if (!$server) {
            return $this->fail([400202, '服务器不存在']);
        }

        if (array_key_exists('show', $params)) {
            $server->show = (int) $params['show'];
        }
        if (array_key_exists('machine_id', $params)) {
            $server->machine_id = $params['machine_id'] ?: null;
        }
        if (array_key_exists('enabled', $params)) {
            $server->enabled = (bool) $params['enabled'];
        }

        if (!$server->save()) {
            return $this->fail([500, '保存失败']);
        }

        return $this->success(true);
    }

    /**
     * 删除
     * @param \Illuminate\Http\Request $request
     * @return \Illuminate\Http\JsonResponse
     */
    public function drop(Request $request)
    {
        $request->validate([
            'id' => 'required|integer',
        ]);
        $server = Server::find($request->id);
        if (!$server) {
            return $this->fail([400202, '服务器不存在']);
        }
        if ($server->delete() === false) {
            return $this->fail([500, '删除失败']);
        }

        return $this->success(true);
    }

    /**
     * 批量删除节点
     * @param \Illuminate\Http\Request $request
     * @return \Illuminate\Http\JsonResponse
     */
    public function batchDelete(Request $request)
    {
        $request->validate([
            'ids' => 'required|array',
            'ids.*' => 'integer',
        ]);

        $ids = $request->input('ids');
        if (empty($ids)) {
            return $this->fail([400, '请选择要删除的节点']);
        }

        try {
            $deleted = Server::whereIn('id', $ids)->delete();
            if ($deleted === false) {
                return $this->fail([500, '批量删除失败']);
            }
            return $this->success(true);
        } catch (\Exception $e) {
            Log::error($e);
            return $this->fail([500, '批量删除失败']);
        }
    }

    /**
     * 重置节点流量
     * @param \Illuminate\Http\Request $request
     * @return \Illuminate\Http\JsonResponse
     */
    public function resetTraffic(Request $request)
    {
        $request->validate([
            'id' => 'required|integer',
        ]);

        $server = Server::find($request->id);
        if (!$server) {
            return $this->fail([400202, '服务器不存在']);
        }

        try {
            $server->u = 0;
            $server->d = 0;
            $server->save();
            
            Log::info("Server {$server->id} ({$server->name}) traffic reset by admin");
            return $this->success(true);
        } catch (\Exception $e) {
            Log::error($e);
            return $this->fail([500, '重置失败']);
        }
    }

    /**
     * 批量重置节点流量
     * @param \Illuminate\Http\Request $request
     * @return \Illuminate\Http\JsonResponse
     */
    public function batchResetTraffic(Request $request)
    {
        $request->validate([
            'ids' => 'required|array',
            'ids.*' => 'integer',
        ]);

        $ids = $request->input('ids');
        if (empty($ids)) {
            return $this->fail([400, '请选择要重置的节点']);
        }

        try {
            Server::whereIn('id', $ids)->update([
                'u' => 0,
                'd' => 0,
            ]);
            
            Log::info("Servers " . implode(',', $ids) . " traffic reset by admin");
            return $this->success(true);
        } catch (\Exception $e) {
            Log::error($e);
            return $this->fail([500, '批量重置失败']);
        }
    }

    /**
     * 批量更新节点属性（show等）
     */
    public function batchUpdate(Request $request)
    {
        $params = $request->validate([
            'ids' => 'required|array',
            'ids.*' => 'integer',
            'show' => 'nullable|integer|in:0,1',
            'enabled' => 'nullable|boolean',
            'machine_id' => 'nullable|integer',
        ]);

        $ids = $params['ids'];
        if (empty($ids)) {
            return $this->fail([400, '请选择要更新的节点']);
        }

        $update = [];
        if (array_key_exists('show', $params) && $params['show'] !== null) {
            $update['show'] = (int) $params['show'];
        }
        if (array_key_exists('enabled', $params) && $params['enabled'] !== null) {
            $update['enabled'] = (bool) $params['enabled'];
        }
        if (array_key_exists('machine_id', $params)) {
            $update['machine_id'] = $params['machine_id'] ?: null;
        }

        if (empty($update)) {
            return $this->fail([400, '没有可更新的字段']);
        }

        try {
            $servers = Server::whereIn('id', $ids)->get();
            DB::transaction(function () use ($servers, $update) {
                /** @var Server $server */
                foreach ($servers as $server) {
                    $server->update($update);
                }
            });
            return $this->success(true);
        } catch (\Exception $e) {
            Log::error($e);
            return $this->fail([500, '批量更新失败']);
        }
    }

    /**
     * 复制节点
     * @param \Illuminate\Http\Request $request
     * @return \Illuminate\Http\JsonResponse
     */
    public function copy(Request $request)
    {
        $server = Server::find($request->input('id'));
        if (!$server) {
            return $this->fail([400202, '服务器不存在']);
        }

        $copiedServer = $server->replicate();
        $copiedServer->show = 0;
        $copiedServer->code = null;
        $copiedServer->u = 0;
        $copiedServer->d = 0;
        $copiedServer->save();

        return $this->success(true);
    }

    /**
     * Generate ECH (Encrypted Client Hello) key pair.
     * Returns PEM-encoded ECH key (server-side) and ECH config (client-side).
     */
    public function generateEchKey(Request $request)
    {
        $publicName = $request->input('public_name', 'ech.example.com');
        if (strlen($publicName) < 1 || strlen($publicName) > 253) {
            throw new ApiException('public_name must be a valid domain (1-253 bytes)');
        }

        // Generate X25519 key pair
        $privateKey = random_bytes(32);
        $publicKey = sodium_crypto_scalarmult_base($privateKey);

        $configId = random_int(0, 255);

        // Build ECHConfigContents (draft-ietf-tls-esni-18)
        $contents = '';
        $contents .= pack('C', $configId);                // config_id
        $contents .= pack('n', 0x0020);                   // kem_id: DHKEM(X25519)
        $contents .= pack('n', 32) . $publicKey;          // public_key (length-prefixed)
        // cipher_suites: 2 suites × 4 bytes = 8 bytes
        $contents .= pack('n', 8);                        // cipher_suites byte length
        $contents .= pack('nn', 0x0001, 0x0001);          // HKDF-SHA256 + AES-128-GCM
        $contents .= pack('nn', 0x0001, 0x0003);          // HKDF-SHA256 + ChaCha20Poly1305
        $contents .= pack('C', 0);                        // max_name_length
        $contents .= pack('C', strlen($publicName)) . $publicName;
        $contents .= pack('n', 0);                        // extensions: empty

        // ECHConfig = version(2) + length(2) + contents
        $echConfig = pack('n', 0xfe0d) . pack('n', strlen($contents)) . $contents;

        // ECHConfigList = total_length(2) + configs
        $echConfigList = pack('n', strlen($echConfig)) . $echConfig;

        // ECH Keys = private_key_len(2) + key(32) + config_len(2) + config
        $echKeysPayload = pack('n', 32) . $privateKey . pack('n', strlen($echConfig)) . $echConfig;

        $keyPem = "-----BEGIN ECH KEYS-----\n"
            . chunk_split(base64_encode($echKeysPayload), 64, "\n")
            . "-----END ECH KEYS-----";

        $configPem = "-----BEGIN ECH CONFIGS-----\n"
            . chunk_split(base64_encode($echConfigList), 64, "\n")
            . "-----END ECH CONFIGS-----";

        return $this->success([
            'key' => $keyPem,
            'config' => $configPem,
        ]);
    }

    /**
     * 节点变更的客户端覆盖情况统计。
     *
     * 每个权限组能看到的节点集不同（getAvailableServers 按 group_id 过滤），
     * 版本号也就不同，所以不能拿一个全局版本号去比——必须按组分别算出当前
     * 版本号，再统计该组里有多少人已经上报同步到了这个版本。
     */
    public function syncStats(Request $request)
    {
        // 注意：max() 是查询构造器聚合，绕过模型的 timestamp cast，拿到的是
        // 数据库里的原始值。v2_server.updated_at 存的是 'Y-m-d H:i:s' 字符串，
        // 直接 (int) 强转会得到年份（"2026-08-30 ..." → 2026），前端就会显示成
        // 1970 年。这里显式按实际格式转换，并兼容存成 unix 时间戳的部署。
        $rawLastChange = Server::max('updated_at');
        $lastChangeAt = null;
        if ($rawLastChange !== null && $rawLastChange !== '') {
            $lastChangeAt = is_numeric($rawLastChange)
                ? (int) $rawLastChange
                : strtotime((string) $rawLastChange);
            if ($lastChangeAt === false) {
                $lastChangeAt = null;
            }
        }

        $groupIds = User::whereNotNull('group_id')->distinct()->pluck('group_id');
        $synced = 0;
        foreach ($groupIds as $groupId) {
            // 只需要一个能代表"该组可见哪些节点"的探针用户即可；cache_key 不含
            // 用户相关字段，所以这里不需要真实用户（uuid 仅用于满足
            // generateServerPassword 的调用，不参与版本号计算）。
            $probe = new User(['group_id' => $groupId, 'uuid' => '00000000-0000-0000-0000-000000000000']);
            $revision = ServerService::calcNodeRevision(ServerService::getAvailableServers($probe));
            $synced += User::where('group_id', $groupId)
                ->where('node_revision', $revision)
                ->count();
        }

        return $this->success([
            // 已经同步到各自权限组当前最新节点集的用户数
            'synced' => $synced,
            // 曾经上报过的用户数：与 total 的差值就是"还在用不支持上报的旧客户端"，
            // 不区分开的话会把老客户端误算成"没更新"
            'reported' => User::whereNotNull('node_revision')->count(),
            'total' => User::count(),
            'last_node_change_at' => $lastChangeAt,
        ]);
    }
}
