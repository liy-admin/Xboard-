<?php

namespace App\Http\Requests\Admin;

use Illuminate\Foundation\Http\FormRequest;

class ConfigSave extends FormRequest
{
    const RULES = [
        // invite & commission
        'invite_force' => '',
        'invite_commission' => 'integer|nullable',
        'invite_gen_limit' => 'integer|nullable',
        'invite_never_expire' => '',
        'commission_first_time_enable' => '',
        'commission_auto_check_enable' => '',
        'commission_withdraw_limit' => 'nullable|numeric',
        'commission_withdraw_method' => 'nullable|array',
        'withdraw_close_enable' => '',
        'commission_distribution_enable' => '',
        'commission_distribution_l1' => 'nullable|numeric',
        'commission_distribution_l2' => 'nullable|numeric',
        'commission_distribution_l3' => 'nullable|numeric',
        'referral_headline' => 'nullable',
        'referral_subtext' => 'nullable',
        'referral_duration_percent' => 'nullable|numeric|min:0|max:100',
        // 自定义分享链接（可含 :code 占位符），跟 discover_* 一样用宽松的
        // 'nullable'，不加 url 校验：逐字符自动保存时用户还没打完整个网址
        // 加 url 规则会导致每次按键都 422，整批保存失败。
        'referral_share_url' => 'nullable',
        // discover
        // 两个地址字段用宽松的 'nullable'（不加 url 校验）：设置页是逐字符
        // 自动保存的，若用 'url' 规则，用户还没打完整个网址时每次按键都会
        // 触发 422 校验失败，导致这一批（同一次请求里的全部 discover_*
        // 字段）整体保存不了，且前端没有把校验错误提示出来，看起来就是
        // "无法保存"。
        'discover_connected_page_url' => 'nullable',
        'discover_prompt_image_url' => 'nullable',
        'discover_prompt_badge_text' => 'nullable',
        'discover_prompt_body' => 'nullable',
        'discover_prompt_button_text' => 'nullable',
        'discover_prompt_cancel_text' => 'nullable',
        // anti-loss（软件防丢失）
        // 地址类字段同样用宽松的 'nullable'（不加 url 校验）：设置页逐字符
        // 自动保存，加 url 规则会在没打完时每次按键都 422，整批存不下去。
        'anti_loss_qr_data' => 'nullable',
        'anti_loss_qr_image_url' => 'nullable',
        'anti_loss_qr_logo_url' => 'nullable',
        'anti_loss_title' => 'nullable',
        'anti_loss_body' => 'nullable',
        'anti_loss_save_button_text' => 'nullable',
        'anti_loss_saved_toast' => 'nullable',
        'anti_loss_follow_text' => 'nullable',
        'anti_loss_socials' => 'nullable|array',
        'anti_loss_socials.*.name' => 'nullable|string|max:64',
        'anti_loss_socials.*.url' => 'nullable|string|max:512',
        'anti_loss_socials.*.icon_url' => 'nullable|string|max:512',
        // ad banner
        // 客服（Chatwoot）接入配置，客户端通过加密接口读取
        'support_chatwoot_enabled' => 'boolean',
        'support_chatwoot_base_url' => 'nullable|url',
        'support_chatwoot_website_token' => 'nullable|string|max:191',
        'support_chatwoot_locale' => 'nullable|string|max:16',
        'ad_banner_enabled' => 'boolean',
        'ad_banner_image_url' => 'nullable|url',
        'ad_banner_click_url' => 'nullable|url',
        'ad_banner_countdown_seconds' => 'integer|nullable|min:1|max:60',
        // 官网（对外展示的 Next.js 落地页）内容配置
        // 地址 / 文本类字段同样用宽松的 'nullable'（不加 url 校验）：设置页逐
        // 字符自动保存，加 url 规则会在没打完时每次按键都 422，整批存不下去。
        'website_brand_name' => 'nullable|string|max:128',
        'website_brand_tagline' => 'nullable|string|max:255',
        'website_brand_description' => 'nullable|string|max:1000',
        'website_announcement_enabled' => 'boolean',
        'website_announcement_text' => 'nullable|string|max:500',
        'website_announcement_link_text' => 'nullable|string|max:64',
        'website_announcement_link_href' => 'nullable|string|max:512',
        'website_hero_title' => 'nullable|string|max:255',
        'website_hero_subtitle' => 'nullable|string|max:500',
        'website_hero_stats' => 'nullable|array',
        'website_hero_stats.*.value' => 'nullable|string|max:32',
        'website_hero_stats.*.label' => 'nullable|string|max:32',
        'website_cta_title' => 'nullable|string|max:255',
        'website_cta_subtitle' => 'nullable|string|max:500',
        'website_footer_contact' => 'nullable|string|max:255',
        // site
        'logo' => 'nullable|url',
        'force_https' => '',
        'stop_register' => '',
        'app_name' => '',
        'app_description' => '',
        'app_url' => 'nullable|url',
        'subscribe_url' => 'nullable',
        'try_out_enable' => '',
        'try_out_plan_id' => 'integer',
        'try_out_hour' => 'numeric',
        'default_device_limit' => 'integer|nullable|min:0',
        'tos_url' => 'nullable|url',
        'currency' => '',
        'currency_symbol' => '',
        'ticket_must_wait_reply' => '',
        // subscribe
        'plan_change_enable' => '',
        'reset_traffic_method' => 'in:0,1,2,3,4',
        'surplus_enable' => '',
        'new_order_event_id' => '',
        'renew_order_event_id' => '',
        'change_order_event_id' => '',
        'show_info_to_server_enable' => '',
        'show_protocol_to_server_enable' => '',
        'subscribe_path' => '',
        // server
        'server_token' => 'nullable|min:16',
        'server_pull_interval' => 'integer',
        'server_push_interval' => 'integer',
        'device_limit_mode' => 'integer',
        'server_ws_enable' => 'boolean',
        'server_ws_url' => 'nullable|url',
        // 节点测速 + 按国家去重下发
        'server_latency_probe_enable' => 'boolean',
        'server_latency_timeout_ms' => 'integer|min:200|max:10000',
        'server_latency_stale_seconds' => 'integer|min:60',
        'server_dedup_by_country' => 'boolean',
        'server_country_naming' => 'boolean',
        'server_auto_manage_visibility' => 'boolean',
        'geoip_method' => 'nullable|in:api,maxmind',
        'geoip_maxmind_db_path' => 'nullable|string',
        'node_probe_mode' => 'nullable|in:tcp,singbox',
        'node_probe_singbox_path' => 'nullable|string',
        'node_probe_singbox_auto_download' => 'boolean',
        'node_probe_singbox_version' => 'nullable|string',
        'node_probe_ip_echo_url' => 'nullable|url',
        'node_probe_dns' => 'nullable|string',
        // frontend
        'frontend_theme' => '',
        'frontend_theme_sidebar' => 'nullable|in:dark,light',
        'frontend_theme_header' => 'nullable|in:dark,light',
        'frontend_theme_color' => 'nullable|in:default,darkblue,black,green',
        'frontend_background_url' => 'nullable|url',
        // email
        'email_host' => '',
        'email_port' => '',
        'email_username' => '',
        'email_password' => '',
        'email_encryption' => '',
        'email_from_address' => '',
        'remind_mail_enable' => '',
        // telegram
        'telegram_bot_enable' => '',
        'telegram_bot_token' => '',
        'telegram_webhook_url' => 'nullable|url',
        'telegram_discuss_id' => '',
        'telegram_channel_id' => '',
        'telegram_discuss_link' => 'nullable|url',
        // app
        'windows_version' => '',
        'windows_download_url' => '',
        'windows_force_update' => 'boolean',
        'macos_version' => '',
        'macos_download_url' => '',
        'macos_force_update' => 'boolean',
        'android_version' => '',
        'android_download_url' => '',
        'android_force_update' => 'boolean',
        'ios_version' => '',
        'ios_download_url' => '',
        'ios_force_update' => 'boolean',
        // app update copy (后端可配的更新文案：通用 + 各平台覆盖；客户端优先取平台专属，回退通用)
        'update_title' => 'nullable|string',
        'update_message' => 'nullable|string',
        'update_button' => 'nullable|string',
        'update_changelog' => 'nullable|string',
        'windows_update_title' => 'nullable|string',
        'windows_update_message' => 'nullable|string',
        'windows_update_button' => 'nullable|string',
        'windows_update_changelog' => 'nullable|string',
        'macos_update_title' => 'nullable|string',
        'macos_update_message' => 'nullable|string',
        'macos_update_button' => 'nullable|string',
        'macos_update_changelog' => 'nullable|string',
        'android_update_title' => 'nullable|string',
        'android_update_message' => 'nullable|string',
        'android_update_button' => 'nullable|string',
        'android_update_changelog' => 'nullable|string',
        'ios_update_title' => 'nullable|string',
        'ios_update_message' => 'nullable|string',
        'ios_update_button' => 'nullable|string',
        'ios_update_changelog' => 'nullable|string',
        // safe
        'user_web_login_enable' => 'boolean',
        'email_whitelist_enable' => 'boolean',
        'email_whitelist_suffix' => 'nullable|array',
        'email_gmail_limit_enable' => 'boolean',
        'captcha_enable' => 'boolean',
        'captcha_type' => 'in:recaptcha,turnstile,recaptcha-v3',
        'recaptcha_enable' => 'boolean',
        'recaptcha_key' => '',
        'recaptcha_site_key' => '',
        'recaptcha_v3_secret_key' => '',
        'recaptcha_v3_site_key' => '',
        'recaptcha_v3_score_threshold' => 'numeric|min:0|max:1',
        'turnstile_secret_key' => '',
        'turnstile_site_key' => '',
        'email_verify' => 'bool',
        'safe_mode_enable' => 'boolean',
        'register_limit_by_ip_enable' => 'boolean',
        'register_limit_count' => 'integer',
        'register_limit_expire' => 'integer',
        'secure_path' => 'min:8|regex:/^[\w-]*$/',
        'password_limit_enable' => 'boolean',
        'password_limit_count' => 'integer',
        'password_limit_expire' => 'integer',
        'default_remind_expire' => 'boolean',
        'default_remind_traffic' => 'boolean',
        'subscribe_template_singbox' => 'nullable',
        'subscribe_template_clash' => 'nullable',
        'subscribe_template_clashmeta' => 'nullable',
        'subscribe_template_stash' => 'nullable',
        'subscribe_template_surge' => 'nullable',
        'subscribe_template_surfboard' => 'nullable',
        // lucky draw
        'lucky_draw_enabled' => 'boolean',
        'lucky_draw_cooldown_seconds' => 'integer|nullable|min:0',
        'lucky_draw_reward_hours' => 'integer|nullable|min:0',
        'lucky_draw_daily_target_winners' => 'integer|nullable|min:0',
        'lucky_draw_min_probability_percent' => 'numeric|nullable|min:0|max:100',
        'lucky_draw_max_probability_percent' => 'numeric|nullable|min:0|max:100',
    ];
    /**
     * Get the validation rules that apply to the request.
     *
     * @return array
     */
    public function rules()
    {
        return self::RULES;
    }

    public function messages()
    {
        // illiteracy prompt
        return [
            'app_url.url' => '站点URL格式不正确，必须携带http(s)://',
            'subscribe_url.url' => '订阅URL格式不正确，必须携带http(s)://',
            'server_token.min' => '通讯密钥长度必须大于16位',
            'tos_url.url' => '服务条款URL格式不正确，必须携带http(s)://',
            'telegram_webhook_url.url' => 'Telegram Webhook地址格式不正确，必须携带http(s)://',
            'telegram_discuss_link.url' => 'Telegram群组地址必须为URL格式，必须携带http(s)://',
            'logo.url' => 'LOGO URL格式不正确，必须携带https(s)://',
            'secure_path.min' => '后台路径长度最小为8位',
            'secure_path.regex' => '后台路径只能为字母或数字',
            'captcha_type.in' => '人机验证类型只能选择 recaptcha、turnstile 或 recaptcha-v3',
            'recaptcha_v3_score_threshold.numeric' => 'reCAPTCHA v3 分数阈值必须为数字',
            'recaptcha_v3_score_threshold.min' => 'reCAPTCHA v3 分数阈值不能小于0',
            'recaptcha_v3_score_threshold.max' => 'reCAPTCHA v3 分数阈值不能大于1'
        ];
    }
}
