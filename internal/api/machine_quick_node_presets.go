package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// 一键创建节点可选的协议。TCP 被拦时 UDP 协议可能仍然可达（反之亦然），所以同时提供两类。
// 需要证书的协议用节点端生成的自签名证书（cert_mode=self），客户端链接带 insecure。

type quickNodePreset struct {
	key         string
	label       string
	nodeType    string
	transport   string // tcp、udp、tcp+udp，仅用于展示
	defaultPort int64
	build       func(sni, host string) (settings, cert map[string]any, err error)
}

var quickNodePresets = []quickNodePreset{
	{"vless_reality", "VLESS-Reality", "vless", "tcp", 443, buildVlessReality},
	{"trojan_reality", "Trojan-Reality", "trojan", "tcp", 8443, buildTrojanReality},
	{"anytls", "AnyTLS", "anytls", "tcp", 8446, buildSelfSigned},
	{"vmess_ws", "VMess-WS", "vmess", "tcp", 8080, buildVmessWS},
	{"shadowsocks", "Shadowsocks", "shadowsocks", "tcp+udp", 8388, buildShadowsocks},
	{"hysteria2", "Hysteria2", "hysteria", "udp", 8444, buildHysteria2},
	{"tuic", "TUIC", "tuic", "udp", 8445, buildTuic},
}

func quickNodePresetByKey(key string) (quickNodePreset, bool) {
	for _, p := range quickNodePresets {
		if p.key == key {
			return p, true
		}
	}
	return quickNodePreset{}, false
}

type quickNodeItem struct {
	preset quickNodePreset
	port   int64 // 0 表示自动分配
}

// parseQuickNodeItems 读取 protocols 参数；没有时按旧接口创建一个 VLESS + Reality（port 缺省 443）。
func parseQuickNodeItems(m map[string]any) ([]quickNodeItem, error) {
	raw, ok := m["protocols"].([]any)
	if !ok || len(raw) == 0 {
		port := int64(443)
		if v, ok := mInt(m, "port"); ok {
			port = v
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("端口必须在 1-65535 之间")
		}
		preset, _ := quickNodePresetByKey("vless_reality")
		return []quickNodeItem{{preset: preset, port: port}}, nil
	}
	seen := map[string]bool{}
	items := make([]quickNodeItem, 0, len(raw))
	for _, v := range raw {
		obj, _ := v.(map[string]any)
		key, _ := mStr(obj, "protocol")
		preset, ok := quickNodePresetByKey(key)
		if !ok {
			return nil, fmt.Errorf("不支持的协议：%s", key)
		}
		if seen[key] {
			return nil, fmt.Errorf("协议重复：%s", preset.label)
		}
		seen[key] = true
		item := quickNodeItem{preset: preset}
		if port, ok := mInt(obj, "port"); ok && port != 0 {
			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("端口必须在 1-65535 之间")
			}
			item.port = port
		}
		items = append(items, item)
	}
	return items, nil
}

// assignQuickNodePorts 先占住明确指定的端口（冲突则报错），再为其余协议从缺省端口起找空闲端口。
// used 为这台宿主机上已占用的端口及占用者描述。
func assignQuickNodePorts(items []quickNodeItem, used map[int64]string) error {
	for _, it := range items {
		if it.port == 0 {
			continue
		}
		if owner, taken := used[it.port]; taken {
			return fmt.Errorf("端口 %d 已被这台服务器上的%s使用", it.port, owner)
		}
		used[it.port] = it.preset.label
	}
	for i := range items {
		if items[i].port != 0 {
			continue
		}
		port := items[i].preset.defaultPort
		for used[port] != "" {
			port++
		}
		if port > 65535 {
			return fmt.Errorf("%s 找不到空闲端口", items[i].preset.label)
		}
		items[i].port = port
		used[port] = items[i].preset.label
	}
	return nil
}

func realitySettings(sni string) (map[string]any, error) {
	priv, pub, err := realityKeyPair()
	if err != nil {
		return nil, err
	}
	shortID := make([]byte, 8)
	if _, err := rand.Read(shortID); err != nil {
		return nil, err
	}
	return map[string]any{
		"server_name":    sni,
		"server_port":    quickNodeRealityHandshake,
		"public_key":     pub,
		"private_key":    priv,
		"short_id":       hex.EncodeToString(shortID),
		"allow_insecure": false,
	}, nil
}

func buildVlessReality(sni, _ string) (map[string]any, map[string]any, error) {
	rs, err := realitySettings(sni)
	if err != nil {
		return nil, nil, err
	}
	return map[string]any{
		"tls": 2, "network": "tcp", "flow": "xtls-rprx-vision", "reality_settings": rs,
		"utls": map[string]any{"enabled": true, "fingerprint": "chrome"},
	}, nil, nil
}

func buildTrojanReality(sni, _ string) (map[string]any, map[string]any, error) {
	rs, err := realitySettings(sni)
	if err != nil {
		return nil, nil, err
	}
	return map[string]any{
		"tls": 2, "network": "tcp", "reality_settings": rs,
		"utls": map[string]any{"enabled": true, "fingerprint": "chrome"},
	}, nil, nil
}

func selfSignedTLS(sni string) (tls, cert map[string]any) {
	return map[string]any{"server_name": sni, "allow_insecure": true},
		map[string]any{"cert_mode": "self", "domain": sni}
}

func buildSelfSigned(sni, _ string) (map[string]any, map[string]any, error) {
	tls, cert := selfSignedTLS(sni)
	return map[string]any{"tls": tls}, cert, nil
}

func buildHysteria2(sni, _ string) (map[string]any, map[string]any, error) {
	tls, cert := selfSignedTLS(sni)
	return map[string]any{"version": 2, "tls": tls}, cert, nil
}

func buildTuic(sni, _ string) (map[string]any, map[string]any, error) {
	tls, cert := selfSignedTLS(sni)
	return map[string]any{
		"version": 5, "congestion_control": "bbr", "udp_relay_mode": "native", "alpn": []any{"h3"}, "tls": tls,
	}, cert, nil
}

func buildShadowsocks(_, _ string) (map[string]any, map[string]any, error) {
	return map[string]any{"cipher": "aes-128-gcm"}, nil, nil
}

func buildVmessWS(_, host string) (map[string]any, map[string]any, error) {
	path := make([]byte, 8)
	if _, err := rand.Read(path); err != nil {
		return nil, nil, err
	}
	return map[string]any{
		"tls": 0, "network": "ws",
		"network_settings": map[string]any{
			"path":    "/" + hex.EncodeToString(path),
			"headers": map[string]any{"Host": strings.TrimSpace(host)},
		},
	}, nil, nil
}
