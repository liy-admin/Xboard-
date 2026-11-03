package sub

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// 对应 Server::PROTOCOL_CONFIGURATIONS 与 castSettingsWithConfig：
// 读取节点的 protocol_settings 时，只保留各协议声明过的字段，缺失补默认值，并转换类型。
// 管理端节点列表、订阅生成、节点测速都经过这一步，三处看到的配置才一致。

type specKind int

const (
	kInt specKind = iota
	kBool
	kString
	kArray
	kObject
)

type fieldSpec struct {
	kind   specKind
	def    any
	fields map[string]fieldSpec // 仅 kObject
}

func fInt(def any) fieldSpec    { return fieldSpec{kind: kInt, def: def} }
func fBool(def any) fieldSpec   { return fieldSpec{kind: kBool, def: def} }
func fString(def any) fieldSpec { return fieldSpec{kind: kString, def: def} }
func fArray(def any) fieldSpec  { return fieldSpec{kind: kArray, def: def} }
func fObject(def any, fields map[string]fieldSpec) fieldSpec {
	return fieldSpec{kind: kObject, def: def, fields: fields}
}

func merge(parts ...map[string]fieldSpec) map[string]fieldSpec {
	out := map[string]fieldSpec{}
	for _, p := range parts {
		for k, v := range p {
			out[k] = v
		}
	}
	return out
}

var (
	echFields = map[string]fieldSpec{
		"enabled": fBool(false), "config": fString(nil), "query_server_name": fString(nil),
		"key": fString(nil), "key_path": fString(nil), "config_path": fString(nil),
	}
	tlsObject = fObject(nil, map[string]fieldSpec{
		"server_name": fString(nil), "allow_insecure": fBool(false), "ech": fObject(nil, echFields),
	})

	multiplexCfg = map[string]fieldSpec{"multiplex": fObject(nil, map[string]fieldSpec{
		"enabled": fBool(false), "protocol": fString("yamux"), "max_connections": fInt(nil), "padding": fBool(false),
		"brutal": fObject(nil, map[string]fieldSpec{
			"enabled": fBool(false), "up_mbps": fInt(nil), "down_mbps": fInt(nil),
		}),
	})}
	realityCfg = map[string]fieldSpec{"reality_settings": fObject(nil, map[string]fieldSpec{
		"server_name": fString(nil), "server_port": fString(nil), "public_key": fString(nil),
		"private_key": fString(nil), "short_id": fString(nil), "allow_insecure": fBool(false),
	})}
	utlsCfg = map[string]fieldSpec{"utls": fObject(nil, map[string]fieldSpec{
		"enabled": fBool(false), "fingerprint": fString("chrome"),
	})}
)

var defaultPaddingScheme = []any{
	"stop=8", "0=30-30", "1=100-400", "2=400-500,c,500-1000,c,500-1000,c,500-1000,c,500-1000",
	"3=9-9,500-1000", "4=500-1000", "5=500-1000", "6=500-1000", "7=500-1000",
}

var protocolConfigurations = map[string]map[string]fieldSpec{
	"trojan": merge(map[string]fieldSpec{
		"tls": fInt(1), "network": fString(nil), "network_settings": fArray(nil), "server_name": fString(nil),
		"allow_insecure": fBool(false), "tls_settings": tlsObject,
	}, realityCfg, multiplexCfg, utlsCfg),
	"vmess": merge(map[string]fieldSpec{
		"tls": fInt(0), "network": fString(nil), "rules": fArray(nil), "network_settings": fArray(nil),
		"tls_settings": tlsObject,
	}, multiplexCfg, utlsCfg),
	"vless": merge(map[string]fieldSpec{
		"tls": fInt(0), "tls_settings": tlsObject, "flow": fString(nil),
		"encryption": fObject(nil, map[string]fieldSpec{
			"enabled": fBool(false), "encryption": fString(nil), "decryption": fString(nil),
		}),
		"network": fString(nil), "network_settings": fArray(nil),
	}, realityCfg, multiplexCfg, utlsCfg),
	"shadowsocks": {
		"cipher": fString(nil), "obfs": fString(nil), "obfs_settings": fArray(nil),
		"plugin": fString(nil), "plugin_opts": fString(nil),
	},
	"hysteria": {
		"version":   fInt(2),
		"bandwidth": fObject(nil, map[string]fieldSpec{"up": fInt(nil), "down": fInt(nil)}),
		"obfs": fObject(nil, map[string]fieldSpec{
			"open": fBool(false), "type": fString("salamander"), "password": fString(nil),
		}),
		"tls": tlsObject, "hop_interval": fInt(nil),
	},
	"tuic": {
		"version": fInt(5), "congestion_control": fString("cubic"), "alpn": fArray([]any{"h3"}),
		"udp_relay_mode": fString("native"), "tls": tlsObject,
	},
	"anytls": {"padding_scheme": fArray(defaultPaddingScheme), "tls": tlsObject},
	"socks":  {"tls": fInt(0), "tls_settings": tlsObject},
	"naive":  {"tls": fInt(0), "tls_settings": tlsObject},
	"http":   {"tls": fInt(0), "tls_settings": tlsObject},
	"mieru": merge(map[string]fieldSpec{
		"transport": fString("TCP"), "traffic_pattern": fString(""),
	}, multiplexCfg),
}

// NormalizeProtocolSettings 对应 Server::getProtocolSettingsAttribute。类型未知时原样返回。
func NormalizeProtocolSettings(typ string, raw map[string]any) map[string]any {
	cfg, ok := protocolConfigurations[typ]
	if !ok {
		if raw == nil {
			return map[string]any{}
		}
		return raw
	}
	return castSettings(raw, cfg)
}

func castSettings(raw map[string]any, cfg map[string]fieldSpec) map[string]any {
	out := make(map[string]any, len(cfg))
	for k, spec := range cfg {
		out[k] = castValue(raw[k], spec)
	}
	return out
}

func castValue(v any, spec fieldSpec) any {
	if v == nil && spec.kind != kObject {
		return cloneDefault(spec.def)
	}
	switch spec.kind {
	case kInt:
		return phpInt(v)
	case kBool:
		return Truthy(v)
	case kString:
		return phpString(v)
	case kArray:
		switch t := v.(type) {
		case []any, map[string]any:
			return t
		}
		return []any{v}
	case kObject:
		if m, ok := v.(map[string]any); ok {
			return castSettings(m, spec.fields)
		}
		return cloneDefault(spec.def)
	}
	return v
}

// cloneDefault 复制数组型默认值，避免多个节点共享同一份切片。
func cloneDefault(def any) any {
	if a, ok := def.([]any); ok {
		return append([]any(nil), a...)
	}
	return def
}

// phpInt 对应 PHP (int) 强转。
func phpInt(v any) int64 {
	switch t := v.(type) {
	case bool:
		if t {
			return 1
		}
		return 0
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case json.Number:
		// 管理端请求体用 UseNumber 解析，数字是 json.Number；按字符串规则转换。
		return phpInt(string(t))
	case string:
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return int64(f)
		}
		return 0
	case []any:
		if len(t) > 0 {
			return 1
		}
		return 0
	case map[string]any:
		if len(t) > 0 {
			return 1
		}
		return 0
	}
	return 0
}

// phpString 对应 PHP (string) 强转（数字不带多余的小数点，布尔 true 为 "1"、false 为 ""）。
func phpString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "1"
		}
		return ""
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}
