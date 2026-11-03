package sub

import "encoding/json"

// Shadowsocks SIP008 订阅（对应 app/Protocols/Shadowsocks.php）。
// 字段顺序与 PHP 数组插入顺序一致：servers, bytes_used, bytes_remaining, version。

var sip008Allowed = map[string]bool{"shadowsocks": true}

var sip008Ciphers = []string{"aes-128-gcm", "aes-256-gcm", "aes-192-gcm", "chacha20-ietf-poly1305"}

type sip008Server struct {
	ID         int64  `json:"id"`
	Remarks    string `json:"remarks"`
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
}

type sip008Out struct {
	Servers        []sip008Server `json:"servers"`
	BytesUsed      int64          `json:"bytes_used"`
	BytesRemaining int64          `json:"bytes_remaining"`
	Version        int            `json:"version"`
}

// BuildSIP008 生成 SIP008 JSON 订阅。
func BuildSIP008(ctx *Ctx) (string, string) {
	used := ctx.User.U + ctx.User.D
	out := sip008Out{
		Servers:        []sip008Server{},
		BytesUsed:      used,
		BytesRemaining: ctx.User.TransferEnab - used,
		Version:        1,
	}
	for _, s := range filterServers(ctx, sip008Allowed, nil) {
		if !cipherIn(Str(Get(s, "protocol_settings.cipher")), sip008Ciphers...) {
			continue
		}
		out.Servers = append(out.Servers, sip008Server{
			ID:         Int64Of(s["id"]),
			Remarks:    Str(s["name"]),
			Server:     Str(s["host"]),
			ServerPort: Int(s["port"]),
			Password:   Str(s["password"]),
			Method:     Str(Get(s, "protocol_settings.cipher")),
		})
	}
	b, _ := json.Marshal(out)
	return string(b), "application/json"
}

// Int64Of 把 JSON 解析出的数值转为 int64。
func Int64Of(v any) int64 { return int64(Int(v)) }
