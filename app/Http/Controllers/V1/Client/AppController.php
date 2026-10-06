<?php

namespace App\Http\Controllers\V1\Client;

use App\Http\Controllers\Controller;
use App\Services\ServerService;
use App\Services\UserService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\File;
use Symfony\Component\Yaml\Yaml;

class AppController extends Controller
{
    public function getConfig(Request $request)
    {
        $servers = [];
        $user = $request->user();
        $userService = new UserService();
        if ($userService->isAvailable($user)) {
            $servers = ServerService::getAvailableServers($user);
        }
        $defaultConfig = base_path() . '/resources/rules/app.clash.yaml';
        $customConfig = base_path() . '/resources/rules/custom.app.clash.yaml';
        if (File::exists($customConfig)) {
            $config = Yaml::parseFile($customConfig);
        } else {
            $config = Yaml::parseFile($defaultConfig);
        }
        $proxy = [];
        $proxies = [];

        foreach ($servers as $item) {
            $protocol_settings = $item['protocol_settings'];
            if ($item['type'] === 'shadowsocks'
                && in_array(data_get($protocol_settings, 'cipher'), [
                    'aes-128-gcm',
                    'aes-192-gcm',
                    'aes-256-gcm',
                    'chacha20-ietf-poly1305'
                ])
            ) {
                array_push($proxy, \App\Protocols\Clash::buildShadowsocks($user['uuid'], $item));
                array_push($proxies, $item['name']);
            }
            if ($item['type'] === 'vmess') {
                array_push($proxy, \App\Protocols\Clash::buildVmess($user['uuid'], $item));
                array_push($proxies, $item['name']);
            }
            if ($item['type'] === 'trojan') {
                array_push($proxy, \App\Protocols\Clash::buildTrojan($user['uuid'], $item));
                array_push($proxies, $item['name']);
            }
        }

        $config['proxies'] = array_merge($config['proxies'] ? $config['proxies'] : [], $proxy);
        foreach ($config['proxy-groups'] as $k => $v) {
            $config['proxy-groups'][$k]['proxies'] = array_merge($config['proxy-groups'][$k]['proxies'], $proxies);
        }
        return(Yaml::dump($config));
    }

    public function getVersion(Request $request)
    {
        if (strpos($request->header('user-agent'), 'tidalab/4.0.0') !== false
            || strpos($request->header('user-agent'), 'tunnelab/4.0.0') !== false
        ) {
            if (strpos($request->header('user-agent'), 'Win64') !== false) {
                $data = [
                        'version' => admin_setting('windows_version'),
                        'download_url' => admin_setting('windows_download_url')
                ];
            } else {
                $data = [
                        'version' => admin_setting('macos_version'),
                        'download_url' => admin_setting('macos_download_url')
                ];
            }
        }else{
            $data = [
                'windows_version' => admin_setting('windows_version'),
                'windows_download_url' => admin_setting('windows_download_url'),
                'windows_force_update' => (bool) admin_setting('windows_force_update', 0),
                'macos_version' => admin_setting('macos_version'),
                'macos_download_url' => admin_setting('macos_download_url'),
                'macos_force_update' => (bool) admin_setting('macos_force_update', 0),
                'android_version' => admin_setting('android_version'),
                'android_download_url' => admin_setting('android_download_url'),
                'android_force_update' => (bool) admin_setting('android_force_update', 0),
            ];
            // 更新文案：通用键 + 各平台覆盖键。客户端在（强制）更新界面优先取平台专属
            // （如 android_update_title），没有再取通用（update_title），均为空则用 App
            // 内置文案。只在管理员配置了内容时才下发对应键，避免空串污染。
            foreach (['update_title', 'update_message', 'update_button', 'update_changelog'] as $key) {
                $generic = admin_setting($key, '');
                if ($generic !== '' && $generic !== null) {
                    $data[$key] = $generic;
                }
                foreach (['windows', 'macos', 'android', 'ios'] as $platform) {
                    $platformKey = "{$platform}_{$key}";
                    $value = admin_setting($platformKey, '');
                    if ($value !== '' && $value !== null) {
                        $data[$platformKey] = $value;
                    }
                }
            }
        }
        return $this->success($data);
    }
}
