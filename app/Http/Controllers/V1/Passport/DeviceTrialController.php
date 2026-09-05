<?php

namespace App\Http\Controllers\V1\Passport;

use App\Http\Controllers\Controller;
use App\Models\DeviceBinding;
use App\Models\Plan;
use App\Models\User;
use App\Services\AuthService;
use App\Services\UserService;
use Illuminate\Database\QueryException;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;

/**
 * 设备码体验时长领取。
 *
 * - 首次 claim：创建用户（以设备码为标识）+ 发放试用套餐 + 设 claimed_trial = true
 * - 重复 claim（已领取过）：找到绑定的用户，签发新 token，不再发套餐
 * - 退出后再 claim：找回原试用用户，更新绑定，签发 token
 *
 * claimed_trial 一旦为 true 永不重置，是试用发放的唯一判据。
 */
class DeviceTrialController extends Controller
{
    public function claim(Request $request)
    {
        $deviceId = trim((string) $request->input('device_id', ''));
        if ($deviceId === '') {
            return $this->fail([400, __('Invalid device')]);
        }

        $binding = DeviceBinding::where('device_id', $deviceId)->first();

        if ($binding && $binding->user_id) {
            $boundUser = User::find($binding->user_id);
            if ($boundUser) {
                if ($this->isTrialUser($boundUser)) {
                    return $this->reauthenticateTrialUser($request, $boundUser, $deviceId);
                }
                // 绑定指向真实用户（用户退出后再次 claim）——找回原试用用户
                $trialUser = $this->findTrialUser($deviceId);
                if ($trialUser) {
                    $binding->user_id = $trialUser->id;
                    $binding->save();
                    return $this->reauthenticateTrialUser($request, $trialUser, $deviceId);
                }
                // 原试用用户被删除，清空 user_id，走新建流程
                $binding->user_id = null;
                $binding->save();
            } else {
                $binding->user_id = null;
                $binding->save();
            }
        }

        // 到这里：要么无 binding，要么有 binding 但 user_id 为 null
        // 检查是否有可用的试用计划（首次领取需要）
        if (!$binding || !$binding->claimed_trial) {
            $tryOutPlanId = (int) admin_setting('try_out_plan_id', 0);
            if (!$tryOutPlanId || !Plan::find($tryOutPlanId)) {
                return $this->fail([403, __('Trial is not available')]);
            }
        }

        return $this->createTrialUser($request, $deviceId, $binding);
    }

    private function isTrialUser(User $user): bool
    {
        return str_ends_with($user->email, '@device.local');
    }

    private function findTrialUser(string $deviceId): ?User
    {
        return User::where('device_id', $deviceId)
            ->where('email', 'like', '%@device.local')
            ->first();
    }

    private function reauthenticateTrialUser(Request $request, User $user, string $deviceId)
    {
        if ($user->banned) {
            return $this->fail([400, __('Your account has been suspended')]);
        }

        $authService = new AuthService($user);
        if ($limitError = $authService->checkDeviceLimit($deviceId)) {
            return $this->fail([403, $limitError]);
        }

        $data = $authService->generateAuthData($request->input('device_name'), $deviceId);
        $data['is_device_trial'] = true;
        return $this->success($data);
    }

    /**
     * 创建试用用户。
     *
     * setTryOutPlan 内部会检查 claimed_trial tag：
     * - binding 不存在或 claimed_trial = false → 发放试用套餐
     * - claimed_trial = true → 跳过发放（试用用户被删后重建的情况）
     */
    private function createTrialUser(Request $request, string $deviceId, ?DeviceBinding $binding)
    {
        try {
            return DB::transaction(function () use ($request, $deviceId, $binding) {
                $userService = app(UserService::class);
                $user = $userService->createUser([
                    'email' => $this->deviceEmail($deviceId),
                    'password' => Str::random(32),
                    'device_id' => $deviceId,
                ]);
                $user->remarks = '[设备体验] ' . $this->deviceDisplayCode($deviceId);

                if (!$user->save()) {
                    throw new \RuntimeException('Failed to create trial account');
                }

                if ($binding) {
                    $binding->user_id = $user->id;
                    $binding->claimed_trial = true;
                    $binding->save();
                } else {
                    DeviceBinding::create([
                        'device_id' => $deviceId,
                        'user_id' => $user->id,
                        'claimed_trial' => true,
                    ]);
                }

                $authService = new AuthService($user);
                $data = $authService->generateAuthData($request->input('device_name'), $deviceId);
                $data['is_device_trial'] = true;
                return $this->success($data);
            });
        } catch (QueryException $e) {
            $binding = DeviceBinding::where('device_id', $deviceId)->first();
            $user = $binding && $binding->user_id ? User::find($binding->user_id) : null;
            if ($user) {
                return $this->reauthenticateTrialUser($request, $user, $deviceId);
            }
            return $this->fail([500, __('Failed to create trial account')]);
        }
    }

    private function deviceEmail(string $deviceId): string
    {
        return $this->deviceDisplayCode($deviceId) . '@device.local';
    }

    /**
     * FNV-1a 哈希映射到 9 位数字，与客户端 deviceIdDisplayCode() 算法一致。
     */
    private function deviceDisplayCode(string $deviceId): string
    {
        $hash = 0x811c9dc5;
        $prime = 0x01000193;
        for ($i = 0, $len = strlen($deviceId); $i < $len; $i++) {
            $hash ^= ord($deviceId[$i]);
            $hash = ($hash * $prime) & 0xFFFFFFFF;
        }
        return (string) (100000000 + ($hash % 900000000));
    }
}
