<?php

namespace App\Http\Controllers\V1\Guest;

use App\Http\Controllers\Controller;
use App\Services\Plugin\HookManager;

class DiscoverController extends Controller
{
    /**
     * 客户端“大家都在玩”页面的内容配置。
     *
     * connected_page_url：已连接状态下整页由该 H5 地址渲染。
     * prompt_*：未连接状态下客户端原生提示卡片使用的文案/图片，运营可在
     * 后台“站点设置”里随时修改，客户端无需发版。
     */
    public function config()
    {
        $data = [
            'connected_page_url' => admin_setting('discover_connected_page_url'),
            'prompt_image_url' => admin_setting('discover_prompt_image_url'),
            'prompt_badge_text' => admin_setting('discover_prompt_badge_text'),
            'prompt_body' => admin_setting('discover_prompt_body'),
            'prompt_button_text' => admin_setting('discover_prompt_button_text'),
            'prompt_cancel_text' => admin_setting('discover_prompt_cancel_text'),
        ];

        $data = HookManager::filter('guest_discover_config', $data);

        return $this->success($data);
    }
}
