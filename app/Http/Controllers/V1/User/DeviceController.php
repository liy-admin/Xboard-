<?php

namespace App\Http\Controllers\V1\User;

use App\Http\Controllers\Controller;
use App\Models\Server;
use App\Services\DeviceStateService;
use Illuminate\Http\Request;

class DeviceController extends Controller
{
    /**
     * 当前用户所有在线设备（未过期，按节点+IP 区分）。
     */
    public function fetch(Request $request)
    {
        $user = $request->user();
        $deviceService = new DeviceStateService();
        $devices = $deviceService->getUserDevices($user->id);

        $nodeIds = array_values(array_unique(array_column($devices, 'node_id')));
        $nodeNames = Server::query()->whereIn('id', $nodeIds)->pluck('name', 'id');

        $data = array_map(function ($device) use ($nodeNames) {
            return [
                'node_id' => $device['node_id'],
                'node_name' => $nodeNames[$device['node_id']] ?? null,
                'ip' => $device['ip'],
                'last_seen_at' => $device['last_seen'],
            ];
        }, $devices);

        return $this->success($data);
    }

    /**
     * 踢出当前用户在指定节点上的一个设备 IP。
     *
     * 语义见 [DeviceStateService::removeDevice]：只是清掉追踪记录腾出名额，
     * 不保证强制断开该设备当前的连接。
     */
    public function kick(Request $request)
    {
        $user = $request->user();
        $nodeId = (int) $request->input('node_id');
        $ip = (string) $request->input('ip');

        if ($nodeId <= 0 || $ip === '') {
            return $this->fail([400, '参数错误']);
        }

        $deviceService = new DeviceStateService();
        $deviceService->removeDevice($user->id, $nodeId, $ip);

        return $this->success(true);
    }
}
