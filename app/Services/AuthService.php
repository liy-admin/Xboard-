<?php

namespace App\Services;

use App\Models\User;
use Illuminate\Support\Str;
use Laravel\Sanctum\PersonalAccessToken;

class AuthService
{
    private User $user;

    public function __construct(User $user)
    {
        $this->user = $user;
    }

    /**
     * @param string|null $deviceName 客户端传入的设备描述（如 "Android · Pixel 6"），
     *                                用于在"设备登录管理"列表里让用户认出是哪台设备。
     * @param string|null $deviceId   客户端设备的稳定硬件/系统标识（Android ID /
     *                                identifierForVendor / Windows deviceId 等），
     *                                用于按"不同设备数"而不是"登录次数"去执行
     *                                device_limit，见 [checkDeviceLimit]。
     *                                两者都不传时退回随机字符串，兼容旧客户端/网页登录。
     */
    public function generateAuthData(?string $deviceName = null, ?string $deviceId = null): array
    {
        $tokenName = self::encodeTokenName($deviceName, $deviceId);

        // Create a new Sanctum token with device info
        $token = $this->user->createToken(
            $tokenName, // token name (device identifier)
            ['*'], // abilities
            now()->addYear() // expiration
        );

        // Format token: remove ID prefix and add Bearer
        $tokenParts = explode('|', $token->plainTextToken);
        $formattedToken = 'Bearer ' . ($tokenParts[1] ?? $tokenParts[0]);

        return [
            'token' => $this->user->token,
            'auth_data' => $formattedToken,
            'is_admin' => $this->user->is_admin,
            'enc_key' => $this->user->getOrCreateEncKey(),
        ];
    }

    /**
     * 登录/注册即将签发新 token 前的设备名额检查。
     *
     * 按 token.name 里编码的 device_id 去重统计"不同设备数"——没有 device_id
     * 的旧数据/网页登录/第三方客户端不计入，也不参与下面的顶替逻辑，避免
     * 把"同一台设备重复登录"或"网页查看一下账户"误判成占用了一个设备名额。
     *
     * 同一设备号重新登录时，直接撤销它名下的旧 token 再放行——不占用新名额，
     * 也避免同一设备积攒一堆过期前都算数的僵尸 token。
     *
     * @return string|null 超出限制时返回给用户看的错误文案；未超限（含不限制/无
     *                      device_id）时返回 null，调用方据此决定是否放行登录。
     */
    public function checkDeviceLimit(?string $deviceId): ?string
    {
        $limit = (int) ($this->user->device_limit ?? 0);
        if ($limit <= 0 || $deviceId === null || trim($deviceId) === '') {
            return null;
        }
        $deviceId = trim($deviceId);

        $deviceTokenIds = [];
        foreach ($this->user->tokens()->get(['id', 'name']) as $token) {
            $decoded = json_decode((string) $token->name, true);
            $tokenDeviceId = is_array($decoded) ? ($decoded['device_id'] ?? null) : null;
            if ($tokenDeviceId === null || $tokenDeviceId === '') {
                continue;
            }
            $deviceTokenIds[$tokenDeviceId][] = $token->id;
        }

        if (isset($deviceTokenIds[$deviceId])) {
            $this->user->tokens()->whereIn('id', $deviceTokenIds[$deviceId])->delete();
            return null;
        }

        if (count($deviceTokenIds) >= $limit) {
            return __('This account has reached the device limit (:limit). Please sign out an old device in device management first.', ['limit' => $limit]);
        }

        return null;
    }

    private static function encodeTokenName(?string $deviceName, ?string $deviceId): string
    {
        $deviceName = $deviceName !== null && trim($deviceName) !== '' ? trim($deviceName) : null;
        $deviceId = $deviceId !== null && trim($deviceId) !== '' ? trim($deviceId) : null;
        if ($deviceName === null && $deviceId === null) {
            return Str::random(20);
        }
        return json_encode(['device_name' => $deviceName, 'device_id' => $deviceId]);
    }

    public function getSessions(): array
    {
        return $this->user->tokens()->get()->toArray();
    }

    public function removeSession(string $sessionId): bool
    {
        $this->user->tokens()->where('id', $sessionId)->delete();
        return true;
    }

    public function removeAllSessions(): bool
    {
        $this->user->tokens()->delete();
        return true;
    }

    public static function findUserByBearerToken(string $bearerToken): ?User
    {
        $token = str_replace('Bearer ', '', $bearerToken);
        
        $accessToken = PersonalAccessToken::findToken($token);
        
        $tokenable = $accessToken?->tokenable;
        
        return $tokenable instanceof User ? $tokenable : null;
    }

    /**
     * 解密认证数据
     *
     * @param string $authorization
     * @return array|null 用户数据或null
     */
    public static function decryptAuthData(string $authorization): ?array
    {
        $user = self::findUserByBearerToken($authorization);
        
        if (!$user) {
            return null;
        }
        
        return [
            'id' => $user->id,
            'email' => $user->email,
            'is_admin' => (bool)$user->is_admin,
            'is_staff' => (bool)$user->is_staff
        ];
    }
}
