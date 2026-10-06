<?php

namespace App\Http\Controllers\V1\Guest;

use App\Http\Controllers\Controller;
use App\Services\Plugin\HookManager;

class WebsiteController extends Controller
{
    /**
     * 对外官网（独立部署的 Next.js 落地页）的内容配置。
     *
     * 官网的品牌名、标语、首页大标题、公告条、底部 CTA、联系邮箱等文案，原本
     * 都硬编码在前端仓库里，改一个字就得重新部署。这里把这些「经常要改、但不
     * 涉及结构」的文案收口到后台，由运营随时改、官网拉取即生效，无需发版。
     *
     * 跟 comm/anti_loss 一样是纯展示配置，官网在用户未登录时也要渲染，所以放在
     * guest 接口、不做鉴权、不加密——这里没有任何账号级敏感凭据。
     *
     * 所有字段缺省即返回空字符串 / 空数组，官网侧用内置兜底文案补齐，保证后台
     * 一个字都没填时官网仍是一套完整默认文案。
     */
    public function config()
    {
        $data = [
            'brand_name' => (string) admin_setting('website_brand_name', ''),
            'brand_tagline' => (string) admin_setting('website_brand_tagline', ''),
            'brand_description' => (string) admin_setting('website_brand_description', ''),
            'announcement_enabled' => (int) admin_setting('website_announcement_enabled', 0) ? 1 : 0,
            'announcement_text' => (string) admin_setting('website_announcement_text', ''),
            'announcement_link_text' => (string) admin_setting('website_announcement_link_text', ''),
            'announcement_link_href' => (string) admin_setting('website_announcement_link_href', ''),
            'hero_title' => (string) admin_setting('website_hero_title', ''),
            'hero_subtitle' => (string) admin_setting('website_hero_subtitle', ''),
            'hero_stats' => $this->normalizeStats(admin_setting('website_hero_stats', [])),
            'cta_title' => (string) admin_setting('website_cta_title', ''),
            'cta_subtitle' => (string) admin_setting('website_cta_subtitle', ''),
            'footer_contact' => (string) admin_setting('website_footer_contact', ''),
            // 各平台下载地址直接复用「APP 更新」里配置的下载链接，官网不再单独维护
            // 一份。某个平台没配就返回空串，官网据此把按钮显示成「敬请期待」不可点。
            'downloads' => [
                'windows' => (string) admin_setting('windows_download_url', ''),
                'macos' => (string) admin_setting('macos_download_url', ''),
                'android' => (string) admin_setting('android_download_url', ''),
                'ios' => (string) admin_setting('ios_download_url', ''),
            ],
        ];

        $data = HookManager::filter('guest_website_config', $data);

        return $this->success($data);
    }

    /**
     * 后台存的是一整段 JSON，可能被手改或残留历史脏数据；统一清洗成官网预期的
     * `[{value, label}]`，把两个字段都为空的条目丢掉——首页三个数据圈里出现一个
     * 空圈比少一个更难看。
     *
     * @param  mixed  $stats
     * @return array<int, array{value: string, label: string}>
     */
    private function normalizeStats($stats): array
    {
        if (!is_array($stats)) {
            return [];
        }

        $result = [];
        foreach ($stats as $item) {
            if (!is_array($item)) {
                continue;
            }
            $value = trim((string) ($item['value'] ?? ''));
            $label = trim((string) ($item['label'] ?? ''));
            if ($value === '' && $label === '') {
                continue;
            }
            $result[] = [
                'value' => $value,
                'label' => $label,
            ];
        }

        return $result;
    }
}
