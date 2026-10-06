<?php

namespace App\Protocols;

use App\Models\Server;
use Illuminate\Support\Arr;

/**
 * 复用 SingBox 协议里成熟的「单节点 outbound 生成」逻辑，用于面板侧实际跑一遍
 * 节点、测真实出口 IP/延迟（见 NodeProbeService）。
 *
 * 用法：
 *   $outbound = (new SingBoxProbe($user, [$serverArray]))->buildSingleOutbound();
 *
 * 构造时把 clientName/clientVersion 置空，父类 filterServersByVersion 会直接放行
 * （protocolRequirements 只有 'sing-box' 分支、无 '*' 全局分支），不会把节点过滤掉；
 * 版本相关的可选特性（ECH 等）在生成时自然省略，不影响连通性探测。
 */
class SingBoxProbe extends SingBox
{
    /**
     * 构造单个节点的 outbound（含 tag/transport/tls 等），失败返回 null。
     */
    public function buildSingleOutbound(): ?array
    {
        $item = Arr::first($this->servers);
        if (!$item) {
            return null;
        }

        $protocol_settings = $item['protocol_settings'];
        $password = $item['password'];

        return match ($item['type']) {
            Server::TYPE_SHADOWSOCKS => $this->buildShadowsocks($password, $item),
            Server::TYPE_TROJAN => $this->buildTrojan($password, $item),
            Server::TYPE_VMESS => $this->buildVmess($password, $item),
            Server::TYPE_VLESS => in_array(data_get($protocol_settings, 'network'), ['tcp', 'ws', 'grpc', 'http', 'quic', 'httpupgrade'])
                ? $this->buildVless($password, $item)
                : null,
            Server::TYPE_HYSTERIA => $this->buildHysteria($password, $item),
            Server::TYPE_TUIC => $this->buildTuic($password, $item),
            Server::TYPE_ANYTLS => $this->buildAnyTLS($password, $item),
            Server::TYPE_SOCKS => $this->buildSocks($password, $item),
            Server::TYPE_HTTP => $this->buildHttp($password, $item),
            default => null,
        };
    }
}
