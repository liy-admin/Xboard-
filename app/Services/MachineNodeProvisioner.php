<?php

namespace App\Services;

use App\Models\Server;
use App\Models\ServerMachine;

/**
 * 为服务器（机器）一键创建预配置好的节点。
 *
 * 目前只提供 VLESS + Reality 模板：不需要证书，sing-box / xray 内核都支持，
 * Reality 密钥对由面板生成。创建出的节点是普通节点，之后可以在节点管理里随意修改。
 */
class MachineNodeProvisioner
{
    public const DEFAULT_PORT = 443;
    public const DEFAULT_REALITY_SERVER_NAME = 'www.microsoft.com';

    /**
     * @param array<int|string> $groupIds 权限组 ID
     */
    public function createRealityNode(
        ServerMachine $machine,
        array $groupIds,
        string $host,
        int $port = self::DEFAULT_PORT,
        ?string $name = null,
        ?string $serverName = null,
        bool $show = true,
    ): Server {
        $keys = self::generateRealityKeys();
        $serverName = trim((string) $serverName) ?: self::DEFAULT_REALITY_SERVER_NAME;
        $name = trim((string) $name) ?: $machine->name;

        return Server::create([
            // type 必须在 protocol_settings 之前：后者的 mutator 依赖 type
            'type' => Server::TYPE_VLESS,
            'name' => mb_substr($name, 0, 255),
            // 用户匹配使用 whereJsonContains((string) group_id)，因此保存为字符串
            'group_ids' => array_values(array_unique(array_map(fn ($id) => (string) (int) $id, $groupIds))),
            'route_ids' => [],
            'rate' => '1',
            'host' => $host,
            'port' => (string) $port,
            'server_port' => $port,
            'show' => $show,
            'enabled' => true,
            'machine_id' => $machine->id,
            'protocol_settings' => [
                'tls' => 2,
                'network' => 'tcp',
                'flow' => 'xtls-rprx-vision',
                'reality_settings' => [
                    'server_name' => $serverName,
                    'server_port' => '443',
                    'public_key' => $keys['public_key'],
                    'private_key' => $keys['private_key'],
                    'short_id' => $keys['short_id'],
                    'allow_insecure' => false,
                ],
                'utls' => [
                    'enabled' => true,
                    'fingerprint' => 'chrome',
                ],
            ],
        ]);
    }

    /**
     * 生成 Reality 使用的 X25519 密钥对：32 字节，base64 URL 安全编码、无填充，
     * 与 `xray x25519` 的输出格式一致。
     *
     * @return array{private_key: string, public_key: string, short_id: string}
     */
    public static function generateRealityKeys(): array
    {
        $private = random_bytes(SODIUM_CRYPTO_SCALARMULT_SCALARBYTES);
        // 与 `xray x25519` 一样做 clamp，保存规范化的私钥
        $private[0] = chr(ord($private[0]) & 248);
        $private[31] = chr((ord($private[31]) & 127) | 64);
        $public = sodium_crypto_scalarmult_base($private);

        return [
            'private_key' => self::base64Url($private),
            'public_key' => self::base64Url($public),
            'short_id' => bin2hex(random_bytes(8)),
        ];
    }

    private static function base64Url(string $bytes): string
    {
        return rtrim(strtr(base64_encode($bytes), '+/', '-_'), '=');
    }
}
