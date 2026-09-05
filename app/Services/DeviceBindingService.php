<?php

namespace App\Services;

use App\Models\DeviceBinding;
use App\Models\User;

/**
 * 登录/注册时将设备绑定到真实用户。
 *
 * - 无 binding → 创建新记录（claimed_trial 保持 false，登录不算领取试用）
 * - 有 binding → 只更新 user_id 为真实用户，不改 claimed_trial
 */
class DeviceBindingService
{
    public static function bind(?string $deviceId, User $user): void
    {
        $deviceId = $deviceId !== null ? trim($deviceId) : '';
        if ($deviceId === '') {
            return;
        }

        $binding = DeviceBinding::where('device_id', $deviceId)->first();
        if (!$binding) {
            DeviceBinding::create([
                'device_id' => $deviceId,
                'user_id' => $user->id,
                'claimed_trial' => false,
            ]);
            return;
        }

        if ((int) $binding->user_id !== (int) $user->id) {
            $binding->user_id = $user->id;
            $binding->save();
        }
    }
}
