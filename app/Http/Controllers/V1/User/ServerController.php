<?php

namespace App\Http\Controllers\V1\User;

use App\Http\Controllers\Controller;
use App\Http\Resources\NodeResource;
use App\Models\User;
use App\Services\ServerService;
use App\Services\UserService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Log;

class ServerController extends Controller
{
    public function fetch(Request $request)
    {
        $user = User::find($request->user()->id);
        $servers = [];
        $userService = new UserService();
        if ($userService->isAvailable($user)) {
            $servers = ServerService::getAvailableServers($user);
        }
        $eTag = ServerService::calcNodeRevision($servers);
        if (strpos($request->header('If-None-Match', ''), $eTag) !== false ) {
            return response(null,304);
        }
        $data = NodeResource::collection($servers);
        return response([
            'data' => $data
        ])->header('ETag', "\"{$eTag}\"");
    }

    /**
     * 节点集版本号。
     *
     * 只返回一个哈希，让客户端能以极低成本判断"我这份订阅是不是已经过期了"，
     * 而不必每次都下载并解密完整订阅。哈希算法与 fetch() 的 ETag 完全一致：
     * 基于每个节点的 cache_key（见 Server::cacheKey，内含 updated_at 与
     * is_online），所以新增/删除节点、修改节点配置、节点上下线都会让它变化。
     *
     * 两处必须保持一致，否则会出现"版本号说没变、实际订阅内容已经变了"的漏更新。
     */
    public function revision(Request $request)
    {
        $user = User::find($request->user()->id);
        $servers = [];
        $userService = new UserService();
        if ($userService->isAvailable($user)) {
            $servers = ServerService::getAvailableServers($user);
        }
        return response([
            'data' => [
                'revision' => ServerService::calcNodeRevision($servers),
            ],
        ]);
    }

    /**
     * 客户端上报「我已经同步到这个版本的节点」。
     *
     * 用来统计一次节点变更实际覆盖到了多少用户：改完节点后，用
     *   SELECT COUNT(*) FROM v2_user WHERE node_revision = '<新版本号>'
     * 就能看到已经更新的人数（新版本号即 revision() 的返回值）。
     *
     * 记录的是客户端**自称**的版本号，不强制等于服务端当前值——这样还能看出
     * 还有多少人停留在旧版本、分别停在哪个版本上。
     */
    public function synced(Request $request)
    {
        $request->validate([
            'revision' => ['required', 'string', 'size:40'],
        ]);

        $user = $request->user();
        try {
            $user->node_revision = $request->input('revision');
            $user->node_revision_at = time();
            $user->save();
        } catch (\Throwable $e) {
            // 纯遥测，写不进去（典型情况：代码已部署但 migrate 还没跑）不该让
            // 客户端拿到 500——它本来就是 fire-and-forget，失败只影响统计数字。
            Log::warning('failed to record node_revision for user ' . $user->id . ': ' . $e->getMessage());
        }

        return $this->success(true);
    }
}
