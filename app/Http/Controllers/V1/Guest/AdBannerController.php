<?php

namespace App\Http\Controllers\V1\Guest;

use App\Http\Controllers\Controller;
use App\Services\Plugin\HookManager;

class AdBannerController extends Controller
{
    /**
     * 客户端启动页广告位配置。运营可在后台"站点设置-广告位"里随时替换图片/
     * 跳转链接，客户端下次拉取时即生效，无需发版。
     */
    public function config()
    {
        $data = [
            'enabled' => (bool) admin_setting('ad_banner_enabled', false),
            'image_url' => admin_setting('ad_banner_image_url', ''),
            'click_url' => admin_setting('ad_banner_click_url', ''),
            'countdown_seconds' => (int) admin_setting('ad_banner_countdown_seconds', 5),
        ];

        $data = HookManager::filter('guest_ad_banner_config', $data);

        return $this->success($data);
    }
}
