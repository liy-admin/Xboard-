package sub

import (
	"encoding/json"
	"strconv"
	"strings"

	"xboard/internal/model"
)

// ProbeOutbound 为测速生成单节点的 sing-box outbound，对应 SingBoxProbe::buildSingleOutbound。
// 不做 show / 流量过滤（隐藏节点也要能测速）；parent 为 r 自身或其父节点（用于 SS 密钥材料）。
// 协议不受支持时返回 nil。
func ProbeOutbound(r, parent model.Server, u *model.User) Map {
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
	code := ""
	if r.Code != nil {
		code = *r.Code
	}
	s := Map{
		"id": r.ID, "type": r.Type, "code": code, "name": r.Name, "host": r.Host,
		"port": port, "server_port": r.ServerPort, "protocol_settings": ps,
		"rate": currentRate(r), "password": serverPassword(r, parent, u, ps),
	}
	b := &sbBuilder{ctx: &Ctx{}, cfg: Map{}}
	return b.outboundFor(s)
}
