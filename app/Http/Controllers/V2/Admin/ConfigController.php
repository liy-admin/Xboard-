<?php

namespace App\Http\Controllers\V2\Admin;

use App\Exceptions\ApiException;
use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\ConfigSave;
use App\Models\SubscribeTemplate;
use App\Services\MailService;
use App\Services\TelegramService;
use App\Services\ThemeService;
use App\Utils\Dict;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Str;

class ConfigController extends Controller
{


    public function getEmailTemplate()
    {
        $path = resource_path('views/mail/');
        $files = array_map(function ($item) use ($path) {
            return str_replace($path, '', $item);
        }, glob($path . '*'));
        return $this->success($files);
    }

    public function getThemeTemplate()
    {
        $path = public_path('theme/');
        $files = array_map(function ($item) use ($path) {
            return str_replace($path, '', $item);
        }, glob($path . '*'));
        return $this->success($files);
    }

    public function testSendMail(Request $request)
    {
        $params = $request->validate([
            'email' => 'nullable|email',
        ], [
            'email.email' => '收件邮箱格式不正确',
        ]);

        $mailLog = MailService::sendEmail([
            'email' => $params['email'] ?? $request->user()->email,
            'subject' => 'This is xboard test email',
            'template_name' => 'notify',
            'template_value' => [
                'name' => admin_setting('app_name', 'XBoard'),
                'content' => 'This is xboard test email',
                'url' => admin_setting('app_url')
            ]
        ]);
        $mailLog['hint'] = $this->mailErrorHint($mailLog['error'] ?? null);
        return response([
            'data' => $mailLog,
        ]);
    }

    /**
     * 把 SMTP 报错归类成前端可翻译的提示码，方便管理员判断是网络还是账号问题。
     */
    private function mailErrorHint(?string $error): ?string
    {
        if ($error === null || $error === '') {
            return null;
        }
        return match (true) {
            (bool) preg_match('/getaddrinfo|php_network_getaddresses|Name or service not known|resolve/i', $error) => 'dns',
            (bool) preg_match('/timed out|timeout|Connection refused|could not be established|Network is unreachable/i', $error) => 'connect',
            (bool) preg_match('/535|authenticat|Username and Password|credentials/i', $error) => 'auth',
            (bool) preg_match('/SSL|TLS|certificate|crypto|wrong version/i', $error) => 'ssl',
            default => 'unknown',
        };
    }
    public function setTelegramWebhook(Request $request)
    {
        $hookUrl = $this->resolveTelegramWebhookUrl();
        if (blank($hookUrl)) {
            return $this->fail([422, 'Telegram Webhook地址未配置']);
        }
        $hookUrl .= '?' . http_build_query([
            'access_token' => md5(admin_setting('telegram_bot_token', $request->input('telegram_bot_token')))
        ]);
        $telegramService = new TelegramService($request->input('telegram_bot_token'));
        $telegramService->getMe();
        $telegramService->setWebhook(url: $hookUrl);
        $telegramService->registerBotCommands();
        return $this->success([
            'success' => true,
            'webhook_url' => $hookUrl,
            'webhook_base_url' => $this->getTelegramWebhookBaseUrl(),
        ]);
    }

    public function fetch(Request $request)
    {
        $key = $request->input('key');
        $configMappings = $this->getConfigMappings();
        if ($key && isset($configMappings[$key])) {
            return $this->success([$key => $configMappings[$key]]);
        }

        return $this->success($configMappings);
    }

    /**
     * 获取配置映射数据
     * 
     * @return array 配置映射数组
     */
    private function getConfigMappings(): array
    {
        return [
            'invite' => [
                'invite_force' => (bool) admin_setting('invite_force', 0),
                'invite_commission' => admin_setting('invite_commission', 10),
                'invite_gen_limit' => admin_setting('invite_gen_limit', 5),
                'invite_never_expire' => (bool) admin_setting('invite_never_expire', 0),
                'commission_first_time_enable' => (bool) admin_setting('commission_first_time_enable', 1),
                'commission_auto_check_enable' => (bool) admin_setting('commission_auto_check_enable', 1),
                'commission_withdraw_limit' => admin_setting('commission_withdraw_limit', 100),
                'commission_withdraw_method' => admin_setting('commission_withdraw_method', Dict::WITHDRAW_METHOD_WHITELIST_DEFAULT),
                'withdraw_close_enable' => (bool) admin_setting('withdraw_close_enable', 0),
                'commission_distribution_enable' => (bool) admin_setting('commission_distribution_enable', 0),
                'commission_distribution_l1' => admin_setting('commission_distribution_l1'),
                'commission_distribution_l2' => admin_setting('commission_distribution_l2'),
                'commission_distribution_l3' => admin_setting('commission_distribution_l3'),
                'referral_headline' => admin_setting('referral_headline', '推荐朋友，领永久会员'),
                'referral_subtext' => admin_setting(
                    'referral_subtext',
                    '好友安装快连后，填写您的推荐码(:code)即算推荐成功；当其每次获得会员，您均可薅取 :percent% 的时长！永久有效！'
                ),
                'referral_duration_percent' => admin_setting('referral_duration_percent', 20),
                'referral_share_url' => admin_setting('referral_share_url', ''),
            ],
            'lucky_draw' => [
                'lucky_draw_enabled' => (bool) admin_setting('lucky_draw_enabled', 1),
                'lucky_draw_cooldown_seconds' => (int) admin_setting('lucky_draw_cooldown_seconds', 3600),
                'lucky_draw_reward_hours' => (int) admin_setting('lucky_draw_reward_hours', 1),
                'lucky_draw_daily_target_winners' => (int) admin_setting('lucky_draw_daily_target_winners', 500),
                'lucky_draw_min_probability_percent' => (float) admin_setting('lucky_draw_min_probability_percent', 1),
                'lucky_draw_max_probability_percent' => (float) admin_setting('lucky_draw_max_probability_percent', 50),
            ],
            'discover' => [
                'discover_connected_page_url' => admin_setting('discover_connected_page_url'),
                'discover_prompt_image_url' => admin_setting('discover_prompt_image_url'),
                'discover_prompt_badge_text' => admin_setting('discover_prompt_badge_text'),
                'discover_prompt_body' => admin_setting('discover_prompt_body'),
                'discover_prompt_button_text' => admin_setting('discover_prompt_button_text'),
                'discover_prompt_cancel_text' => admin_setting('discover_prompt_cancel_text'),
            ],
            'anti_loss' => [
                'anti_loss_qr_data' => admin_setting('anti_loss_qr_data', ''),
                'anti_loss_qr_image_url' => admin_setting('anti_loss_qr_image_url', ''),
                'anti_loss_qr_logo_url' => admin_setting('anti_loss_qr_logo_url', ''),
                'anti_loss_title' => admin_setting('anti_loss_title', ''),
                'anti_loss_body' => admin_setting('anti_loss_body', ''),
                'anti_loss_save_button_text' => admin_setting('anti_loss_save_button_text', ''),
                'anti_loss_saved_toast' => admin_setting('anti_loss_saved_toast', ''),
                'anti_loss_follow_text' => admin_setting('anti_loss_follow_text', ''),
                'anti_loss_socials' => admin_setting('anti_loss_socials', []),
            ],
            'support' => [
                'support_chatwoot_enabled' => (bool) admin_setting('support_chatwoot_enabled', 0),
                'support_chatwoot_base_url' => admin_setting('support_chatwoot_base_url', ''),
                'support_chatwoot_website_token' => admin_setting('support_chatwoot_website_token', ''),
                'support_chatwoot_locale' => admin_setting('support_chatwoot_locale', 'zh_CN'),
            ],
            'ad_banner' => [
                'ad_banner_enabled' => (bool) admin_setting('ad_banner_enabled', 0),
                'ad_banner_image_url' => admin_setting('ad_banner_image_url', ''),
                'ad_banner_click_url' => admin_setting('ad_banner_click_url', ''),
                'ad_banner_countdown_seconds' => (int) admin_setting('ad_banner_countdown_seconds', 5),
            ],
            'site' => [
                'logo' => admin_setting('logo'),
                'force_https' => (int) admin_setting('force_https', 0),
                'stop_register' => (int) admin_setting('stop_register', 0),
                'app_name' => admin_setting('app_name', 'XBoard'),
                'app_description' => admin_setting('app_description', 'XBoard is best!'),
                'app_url' => admin_setting('app_url'),
                'subscribe_url' => admin_setting('subscribe_url'),
                'try_out_plan_id' => (int) admin_setting('try_out_plan_id', 0),
                'try_out_hour' => (int) admin_setting('try_out_hour', 1),
                'default_device_limit' => (int) admin_setting('default_device_limit', 4),
                'tos_url' => admin_setting('tos_url'),
                'currency' => admin_setting('currency', 'CNY'),
                'currency_symbol' => admin_setting('currency_symbol', '¥'),
                'ticket_must_wait_reply' => (bool) admin_setting('ticket_must_wait_reply', 0),
            ],
            'website' => [
                'website_brand_name' => admin_setting('website_brand_name', ''),
                'website_brand_tagline' => admin_setting('website_brand_tagline', ''),
                'website_brand_description' => admin_setting('website_brand_description', ''),
                'website_announcement_enabled' => (bool) admin_setting('website_announcement_enabled', 0),
                'website_announcement_text' => admin_setting('website_announcement_text', ''),
                'website_announcement_link_text' => admin_setting('website_announcement_link_text', ''),
                'website_announcement_link_href' => admin_setting('website_announcement_link_href', ''),
                'website_hero_title' => admin_setting('website_hero_title', ''),
                'website_hero_subtitle' => admin_setting('website_hero_subtitle', ''),
                'website_hero_stats' => admin_setting('website_hero_stats', []),
                'website_cta_title' => admin_setting('website_cta_title', ''),
                'website_cta_subtitle' => admin_setting('website_cta_subtitle', ''),
                'website_footer_contact' => admin_setting('website_footer_contact', ''),
            ],
            'subscribe' => [
                'plan_change_enable' => (bool) admin_setting('plan_change_enable', 1),
                'reset_traffic_method' => (int) admin_setting('reset_traffic_method', 0),
                'surplus_enable' => (bool) admin_setting('surplus_enable', 1),
                'new_order_event_id' => (int) admin_setting('new_order_event_id', 0),
                'renew_order_event_id' => (int) admin_setting('renew_order_event_id', 0),
                'change_order_event_id' => (int) admin_setting('change_order_event_id', 0),
                'show_info_to_server_enable' => (bool) admin_setting('show_info_to_server_enable', 0),
                'show_protocol_to_server_enable' => (bool) admin_setting('show_protocol_to_server_enable', 0),
                'default_remind_expire' => (bool) admin_setting('default_remind_expire', 1),
                'default_remind_traffic' => (bool) admin_setting('default_remind_traffic', 1),
                'subscribe_path' => admin_setting('subscribe_path', 's'),
            ],
            'frontend' => [
                'frontend_theme' => admin_setting('frontend_theme', 'Xboard'),
                'frontend_theme_sidebar' => admin_setting('frontend_theme_sidebar', 'light'),
                'frontend_theme_header' => admin_setting('frontend_theme_header', 'dark'),
                'frontend_theme_color' => admin_setting('frontend_theme_color', 'default'),
                'frontend_background_url' => admin_setting('frontend_background_url'),
            ],
            'server' => [
                'server_token' => admin_setting('server_token'),
                'server_pull_interval' => admin_setting('server_pull_interval', 60),
                'server_push_interval' => admin_setting('server_push_interval', 60),
                'device_limit_mode' => (int) admin_setting('device_limit_mode', 0),
                'server_ws_enable' => (bool) admin_setting('server_ws_enable', 1),
                'server_ws_url' => admin_setting('server_ws_url', ''),
                'server_latency_probe_enable' => (bool) admin_setting('server_latency_probe_enable', 1),
                'server_latency_timeout_ms' => (int) admin_setting('server_latency_timeout_ms', 2000),
                'server_latency_stale_seconds' => (int) admin_setting('server_latency_stale_seconds', 600),
                'server_dedup_by_country' => (bool) admin_setting('server_dedup_by_country', 0),
                'server_country_naming' => (bool) admin_setting('server_country_naming', 1),
                'server_auto_manage_visibility' => (bool) admin_setting('server_auto_manage_visibility', 0),
                'geoip_method' => admin_setting('geoip_method', 'api'),
                'geoip_maxmind_db_path' => admin_setting('geoip_maxmind_db_path', ''),
                'node_probe_mode' => admin_setting('node_probe_mode', 'tcp'),
                'node_probe_singbox_path' => admin_setting('node_probe_singbox_path', ''),
                'node_probe_singbox_auto_download' => (bool) admin_setting('node_probe_singbox_auto_download', 1),
                'node_probe_singbox_version' => admin_setting('node_probe_singbox_version', '1.14.2'),
                'node_probe_ip_echo_url' => admin_setting('node_probe_ip_echo_url', 'http://ip-api.com/json/?fields=status,countryCode,query'),
                'node_probe_dns' => admin_setting('node_probe_dns', '1.1.1.1'),
            ],
            'email' => [
                'email_host' => admin_setting('email_host'),
                'email_port' => admin_setting('email_port'),
                'email_username' => admin_setting('email_username'),
                'email_password' => admin_setting('email_password'),
                'email_encryption' => admin_setting('email_encryption'),
                'email_from_address' => admin_setting('email_from_address'),
                'remind_mail_enable' => (bool) admin_setting('remind_mail_enable', false),
            ],
            'telegram' => [
                'telegram_bot_enable' => (bool) admin_setting('telegram_bot_enable', 0),
                'telegram_bot_token' => admin_setting('telegram_bot_token'),
                'telegram_webhook_url' => admin_setting('telegram_webhook_url'),
                'telegram_discuss_link' => admin_setting('telegram_discuss_link')
            ],
            'app' => [
                'windows_version' => admin_setting('windows_version', ''),
                'windows_download_url' => admin_setting('windows_download_url', ''),
                'windows_force_update' => (bool) admin_setting('windows_force_update', 0),
                'macos_version' => admin_setting('macos_version', ''),
                'macos_download_url' => admin_setting('macos_download_url', ''),
                'macos_force_update' => (bool) admin_setting('macos_force_update', 0),
                'android_version' => admin_setting('android_version', ''),
                'android_download_url' => admin_setting('android_download_url', ''),
                'android_force_update' => (bool) admin_setting('android_force_update', 0),
                'ios_version' => admin_setting('ios_version', ''),
                'ios_download_url' => admin_setting('ios_download_url', ''),
                'ios_force_update' => (bool) admin_setting('ios_force_update', 0),
                // 更新文案（通用 + 各平台覆盖）：客户端在（强制）更新界面优先用平台专属，
                // 其次用通用，均为空则回退 App 内置文案。
                'update_title' => admin_setting('update_title', ''),
                'update_message' => admin_setting('update_message', ''),
                'update_button' => admin_setting('update_button', ''),
                'update_changelog' => admin_setting('update_changelog', ''),
                'windows_update_title' => admin_setting('windows_update_title', ''),
                'windows_update_message' => admin_setting('windows_update_message', ''),
                'windows_update_button' => admin_setting('windows_update_button', ''),
                'windows_update_changelog' => admin_setting('windows_update_changelog', ''),
                'macos_update_title' => admin_setting('macos_update_title', ''),
                'macos_update_message' => admin_setting('macos_update_message', ''),
                'macos_update_button' => admin_setting('macos_update_button', ''),
                'macos_update_changelog' => admin_setting('macos_update_changelog', ''),
                'android_update_title' => admin_setting('android_update_title', ''),
                'android_update_message' => admin_setting('android_update_message', ''),
                'android_update_button' => admin_setting('android_update_button', ''),
                'android_update_changelog' => admin_setting('android_update_changelog', ''),
                'ios_update_title' => admin_setting('ios_update_title', ''),
                'ios_update_message' => admin_setting('ios_update_message', ''),
                'ios_update_button' => admin_setting('ios_update_button', ''),
                'ios_update_changelog' => admin_setting('ios_update_changelog', ''),
            ],
            'safe' => [
                'user_web_login_enable' => (bool) admin_setting('user_web_login_enable', 1),
                'email_verify' => (bool) admin_setting('email_verify', 0),
                'safe_mode_enable' => (bool) admin_setting('safe_mode_enable', 0),
                'secure_path' => admin_setting('secure_path', admin_setting('frontend_admin_path', hash('crc32b', config('app.key')))),
                'email_whitelist_enable' => (bool) admin_setting('email_whitelist_enable', 0),
                'email_whitelist_suffix' => admin_setting('email_whitelist_suffix', Dict::EMAIL_WHITELIST_SUFFIX_DEFAULT),
                'email_gmail_limit_enable' => (bool) admin_setting('email_gmail_limit_enable', 0),
                'captcha_enable' => (bool) admin_setting('captcha_enable', 0),
                'captcha_type' => admin_setting('captcha_type', 'recaptcha'),
                'recaptcha_key' => admin_setting('recaptcha_key', ''),
                'recaptcha_site_key' => admin_setting('recaptcha_site_key', ''),
                'recaptcha_v3_secret_key' => admin_setting('recaptcha_v3_secret_key', ''),
                'recaptcha_v3_site_key' => admin_setting('recaptcha_v3_site_key', ''),
                'recaptcha_v3_score_threshold' => admin_setting('recaptcha_v3_score_threshold', 0.5),
                'turnstile_secret_key' => admin_setting('turnstile_secret_key', ''),
                'turnstile_site_key' => admin_setting('turnstile_site_key', ''),
                'register_limit_by_ip_enable' => (bool) admin_setting('register_limit_by_ip_enable', 0),
                'register_limit_count' => admin_setting('register_limit_count', 3),
                'register_limit_expire' => admin_setting('register_limit_expire', 60),
                'password_limit_enable' => (bool) admin_setting('password_limit_enable', 1),
                'password_limit_count' => admin_setting('password_limit_count', 5),
                'password_limit_expire' => admin_setting('password_limit_expire', 60),
                // 保持向后兼容
                'recaptcha_enable' => (bool) admin_setting('captcha_enable', 0)
            ],
            'subscribe_template' => [
                'subscribe_template_singbox' => $this->formatTemplateContent(
                    subscribe_template('singbox') ?? '',
                    'json'
                ),
                'subscribe_template_clash' => subscribe_template('clash') ?? '',
                'subscribe_template_clashmeta' => subscribe_template('clashmeta') ?? '',
                'subscribe_template_stash' => subscribe_template('stash') ?? '',
                'subscribe_template_surge' => subscribe_template('surge') ?? '',
                'subscribe_template_surfboard' => subscribe_template('surfboard') ?? ''
            ]
        ];
    }

    public function save(ConfigSave $request)
    {
        $data = $request->validated();

        $templateKeys = [
            'subscribe_template_singbox' => 'singbox',
            'subscribe_template_clash' => 'clash',
            'subscribe_template_clashmeta' => 'clashmeta',
            'subscribe_template_stash' => 'stash',
            'subscribe_template_surge' => 'surge',
            'subscribe_template_surfboard' => 'surfboard',
        ];

        foreach ($data as $k => $v) {
            if (isset($templateKeys[$k])) {
                SubscribeTemplate::setContent($templateKeys[$k], $v);
                continue;
            }
            if ($k == 'frontend_theme') {
                $themeService = app(ThemeService::class);
                $themeService->switch($v);
            }
            admin_setting([$k => $v]);
        }

        return $this->success(true);
    }

    /**
     * 上传"大家都在玩"已连接状态展示的静态 HTML 文件。
     *
     * 客户端不再用任何平台 WebView 渲染这个页面，而是把 HTML 文本拉下来
     * 用纯 Flutter 控件解析展示，所以这里存的必须是不依赖 JS 的静态页面。
     * 上传成功后直接把生成的公网 URL 写入 discover_connected_page_url，
     * 管理员不需要再手动复制粘贴。
     *
     * @throws ApiException
     */
    public function uploadDiscoverHtml(Request $request)
    {
        $request->validate([
            'file' => [
                'required',
                'file',
                'mimes:html,htm,txt',
                'max:2048', // 最大2MB
            ]
        ], [
            'file.required' => '请选择HTML文件',
            'file.file' => '无效的文件类型',
            'file.mimes' => '文件必须是 .html 格式',
            'file.max' => '文件大小不能超过2MB',
        ]);

        $file = $request->file('file');
        $uploadDir = public_path('uploads/discover');
        if (!File::exists($uploadDir)) {
            File::makeDirectory($uploadDir, 0755, true);
        }
        if (!is_writable($uploadDir)) {
            throw new ApiException('上传目录无写入权限');
        }

        // Store as .txt so browsers won't execute inline scripts when the URL
        // is opened directly. Clients fetch the file as plain text anyway.
        $filename = Str::random(32) . '.txt';
        $file->move($uploadDir, $filename);

        $publicUrl = url('/uploads/discover/' . $filename);
        admin_setting(['discover_connected_page_url' => $publicUrl]);

        return $this->success(['url' => $publicUrl]);
    }

    /**
     * 格式化模板内容
     * 
     * @param mixed $content 模板内容
     * @param string $format 输出格式 (json|string)
     * @return string 格式化后的内容
     */
    private function formatTemplateContent(mixed $content, string $format = 'string'): string
    {
        return match ($format) {
            'json' => match (true) {
                    is_array($content) => json_encode(
                        value: $content,
                        flags: JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES
                    ),

                    is_string($content) && str($content)->isJson() => rescue(
                        callback: fn() => json_encode(
                            value: json_decode($content, associative: true, flags: JSON_THROW_ON_ERROR),
                            flags: JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES
                        ),
                        rescue: $content,
                        report: false
                    ),

                    default => str($content)->toString()
                },

            default => str($content)->toString()
        };
    }

    private function getTelegramWebhookBaseUrl(): ?string
    {
        $customUrl = trim((string) admin_setting('telegram_webhook_url', ''));
        if ($customUrl !== '') {
            return rtrim($customUrl, '/');
        }

        $appUrl = trim((string) admin_setting('app_url', ''));
        if ($appUrl !== '') {
            return rtrim($appUrl, '/');
        }

        return null;
    }

    private function resolveTelegramWebhookUrl(): ?string
    {
        $baseUrl = $this->getTelegramWebhookBaseUrl();
        if (!$baseUrl) {
            return null;
        }

        if (str_contains($baseUrl, '/api/v1/guest/telegram/webhook')) {
            return $baseUrl;
        }

        return $baseUrl . '/api/v1/guest/telegram/webhook';
    }
}
