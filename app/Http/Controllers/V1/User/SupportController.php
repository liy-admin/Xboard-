<?php

namespace App\Http\Controllers\V1\User;

use App\Http\Controllers\Controller;
use App\Services\Plugin\HookManager;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Log;

class SupportController extends Controller
{
    /**
     * 客户端「在线客服」的接入配置（自建 Chatwoot 的地址 + 网站令牌）。
     *
     * 内容跟 discover/ad_banner 一样由后台随时改、客户端无需发版；区别是这里
     * 的 website_token 相当于「谁都能往你客服台发消息」的凭据，放在 guest
     * 接口上等于公开挂在网上，所以：
     *  1. 走 user 中间件，必须带 Authorization；
     *  2. 响应体用该账号的 enc_key 做 AES-256-GCM 加密，信封格式跟加密订阅
     *     端点 (`/{subscribe_path}-enc/{token}`) 完全一致，客户端复用同一个
     *     解密器。
     *
     * 密钥是账号级的，不写死在客户端里，所以就算 apk 被反编译，也拿不到一把
     * 能解开所有人响应的万能钥匙。
     */
    public function config(Request $request)
    {
        $data = [
            'enabled' => (bool) admin_setting('support_chatwoot_enabled', false),
            // 结尾斜杠在客户端拼 URL 时会变成 `//widget`，统一在这里去掉，
            // 省得每个客户端各写一遍。
            'base_url' => rtrim((string) admin_setting('support_chatwoot_base_url', ''), '/'),
            'website_token' => (string) admin_setting('support_chatwoot_website_token', ''),
            'locale' => (string) admin_setting('support_chatwoot_locale', 'zh_CN'),
        ];

        $data = HookManager::filter('user_support_config', $data);

        $encKey = $request->user()->getOrCreateEncKey();
        if ($encKey === '') {
            // 拿不到密钥就只剩「明文下发」和「报错」两条路，这里选报错：
            // 客服入口临时不可用，总好过把 website_token 裸奔出去。
            Log::error('support config: enc_key unavailable for user ' . $request->user()->id);
            return $this->fail([500, '客服配置暂不可用，请稍后重试']);
        }

        $plaintext = json_encode($data, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
        $key = hash('sha256', $encKey, true);
        $nonce = random_bytes(12);
        $tag = '';
        $ciphertext = openssl_encrypt($plaintext, 'aes-256-gcm', $key, OPENSSL_RAW_DATA, $nonce, $tag);
        if ($ciphertext === false) {
            Log::error('support config: openssl_encrypt failed');
            return $this->fail([500, '客服配置暂不可用，请稍后重试']);
        }

        return $this->success([
            'v' => 1,
            'alg' => 'aes-256-gcm',
            'n' => base64_encode($nonce),
            'c' => base64_encode($ciphertext),
            't' => base64_encode($tag),
        ]);
    }
}
