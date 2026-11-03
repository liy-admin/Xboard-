package sub

import (
	"encoding/json"
	"strconv"
	"strings"

	"xboard/internal/model"
)

// ShareLink 生成单个节点给某个用户的分享链接（vless://、vmess:// 等，与通用订阅逐行内容一致），
// 供管理端导出后直接导入 v2rayN 等客户端。parent 为 r 自身或其父节点（SS 密钥材料）。
// 协议没有标准链接格式时返回空串。
func ShareLink(r, parent model.Server, u *model.User) string {
	ps := Map{}
	if len(r.ProtocolSettings) > 0 {
		_ = json.Unmarshal(r.ProtocolSettings, &ps)
	}
	ps = NormalizeProtocolSettings(r.Type, ps)
	var port any
	if strings.Contains(r.Port, "-") {
		port = randomPort(r.Port)
	} else {
		n, _ := strconv.Atoi(r.Port)
		port = n
	}
	s := Map{
		"id": r.ID, "type": r.Type, "name": r.Name, "host": r.Host,
		"port": port, "server_port": r.ServerPort, "protocol_settings": ps,
		"password": serverPassword(r, parent, u, ps),
	}
	pw := Str(s["password"])
	var link string
	switch r.Type {
	case "vmess":
		link = genVmess(pw, s)
	case "vless":
		link = genVless(pw, s)
	case "shadowsocks":
		link = genSS(pw, s)
	case "trojan":
		link = genTrojan(pw, s)
	case "hysteria":
		link = genHysteria(pw, s)
	case "anytls":
		link = genAnyTLS(pw, s)
	case "socks":
		link = genSocks(pw, s)
	case "tuic":
		link = genTuic(pw, s)
	case "http":
		link = genHTTP(pw, s)
	}
	return strings.TrimSpace(link)
}
