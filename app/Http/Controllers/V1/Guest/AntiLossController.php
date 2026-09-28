<?php

namespace App\Http\Controllers\V1\Guest;

use App\Http\Controllers\Controller;
use App\Services\Plugin\HookManager;

class AntiLossController extends Controller
{
    /**
     * 客户端侧边栏「软件防丢失」页面的内容配置。
     *
     * 这一页的意义就是「官网被墙 / App 被下架之后，用户还能找到备用入口」，
     * 所以二维码指向哪里、页面上写什么字、跟哪些社交账号，全部由运营在后台
     * 随时改，客户端无需发版——一旦备用入口挂了还得等发版，这页就白做了。
     *
     * 跟 discover/ad_banner 一样是纯展示配置，且要在「用户还没登录、甚至账号
     * 已失效」时也能打开（那正是最需要找回备用入口的时候），所以放在 guest
     * 接口、不做任何鉴权，也不加密——这里没有 support 那种账号级敏感凭据。
     *
     * 所有文本字段缺省即返回空/null，由客户端用内置兜底文案补齐；二维码内容
     * 缺省时客户端会退回站点地址，保证最差也能扫出官网。
     */
    public function config()
    {
        $data = [
            'qr_data' => admin_setting('anti_loss_qr_data', ''),
            'qr_image_url' => admin_setting('anti_loss_qr_image_url', ''),
            'qr_logo_url' => admin_setting('anti_loss_qr_logo_url', ''),
            'title' => admin_setting('anti_loss_title', ''),
            'body' => admin_setting('anti_loss_body', ''),
            'save_button_text' => admin_setting('anti_loss_save_button_text', ''),
            'saved_toast' => admin_setting('anti_loss_saved_toast', ''),
            'follow_text' => admin_setting('anti_loss_follow_text', ''),
            'socials' => $this->normalizeSocials(admin_setting('anti_loss_socials', [])),
        ];

        $data = HookManager::filter('guest_anti_loss_config', $data);

        return $this->success($data);
    }

    /**
     * 后台存的是一整段 JSON，可能被手改、或残留历史脏数据；这里统一清洗成
     * 客户端预期的 `[{name, url, icon_url}]`，把没有链接的条目直接丢掉——一个
     * 点了没反应的社媒图标比没有更糟。
     *
     * @param  mixed  $socials
     * @return array<int, array{name: string, url: string, icon_url: string}>
     */
    private function normalizeSocials($socials): array
    {
        if (!is_array($socials)) {
            return [];
        }

        $result = [];
        foreach ($socials as $item) {
            if (!is_array($item)) {
                continue;
            }
            $url = trim((string) ($item['url'] ?? ''));
            if ($url === '') {
                continue;
            }
            $scheme = strtolower(parse_url($url, PHP_URL_SCHEME) ?? '');
            if (!in_array($scheme, ['http', 'https', 'tg', ''], true)) {
                continue;
            }
            $result[] = [
                'name' => trim((string) ($item['name'] ?? '')),
                'url' => $url,
                'icon_url' => trim((string) ($item['icon_url'] ?? '')),
            ];
        }

        return $result;
    }
}
