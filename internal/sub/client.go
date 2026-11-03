package sub

import (
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// protocolOrder 与 PHP ProtocolManager 的匹配顺序一致（类名字母序的逆序，后定义者优先）。
// Ported=false 的协议尚未移植，暂以 General 输出兜底。
var protocolOrder = []struct {
	Name   string
	Flags  []string
	Ported bool
}{
	{"surge", []string{"surge"}, true},
	{"surfboard", []string{"surfboard"}, true},
	{"stash", []string{"stash"}, true},
	{"singbox", []string{"sing-box", "hiddify", "sfm"}, true},
	{"shadowsocks", []string{"shadowsocks"}, true},
	{"shadowrocket", []string{"shadowrocket"}, true},
	{"quantumultx", []string{"quantumult%20x", "quantumult-x"}, true},
	{"loon", []string{"loon"}, true},
	{"general", []string{"general", "v2rayn", "v2rayng", "passwall", "ssrplus", "sagernet"}, true},
	{"clashmeta", []string{"meta", "verge", "flclash", "nekobox", "clashmetaforandroid"}, true},
	{"clash", []string{"clash"}, true},
}

func allFlags() []string {
	var out []string
	for _, p := range protocolOrder {
		out = append(out, p.Flags...)
	}
	return out
}

// MatchProtocol 返回命中的协议名；无命中返回 "general"。
func MatchProtocol(flag string) (name string, ported bool) {
	for _, p := range protocolOrder {
		for _, f := range p.Flags {
			if strings.Contains(flag, f) {
				return p.Name, p.Ported
			}
		}
	}
	return "general", true
}

var (
	reNameVer = regexp.MustCompile(`([a-zA-Z0-9\-_]+)[/\s]+(v?[0-9]+(?:\.[0-9]+){0,2})`)
	reAnyVer  = regexp.MustCompile(`/v?(\d+(?:\.\d+){0,2})`)
)

type ClientInfo struct{ Flag, Name, Version string }

// ParseClient 对应 ClientController::getClientInfo。
func ParseClient(flag string) ClientInfo {
	flag = strings.ToLower(flag)
	ci := ClientInfo{Flag: flag}
	flags := allFlags()
	if m := reNameVer.FindStringSubmatch(flag); m != nil {
		n := strings.ToLower(m[1])
		ci.Version = strings.TrimPrefix(m[2], "v")
		for _, f := range flags {
			if f == n {
				ci.Name = n
				break
			}
		}
	}
	if ci.Name == "" {
		sorted := append([]string(nil), flags...)
		sort.SliceStable(sorted, func(i, j int) bool { return len(sorted[i]) > len(sorted[j]) })
		for _, n := range sorted {
			if strings.Contains(flag, n) {
				ci.Name = n
				if ci.Version == "" {
					re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(n) + `[/\s]+(v?[0-9]+(?:\.[0-9]+){0,2})`)
					if m := re.FindStringSubmatch(flag); m != nil {
						ci.Version = strings.TrimPrefix(m[1], "v")
					}
				}
				break
			}
		}
	}
	if ci.Version == "" {
		if m := reAnyVer.FindStringSubmatch(flag); m != nil {
			ci.Version = m[1]
		}
	}
	return ci
}

var validTypes = []string{"hysteria", "vless", "trojan", "vmess", "tuic", "shadowsocks", "anytls", "socks", "naive", "http", "mieru"}

func splitList(s string) []string {
	var out []string
	for _, p := range regexp.MustCompile(`[|,｜]+`).Split(s, -1) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func ParseTypes(s string) []string {
	if strings.TrimSpace(s) == "" || s == "all" {
		return validTypes
	}
	var out []string
	for _, t := range splitList(s) {
		for _, v := range validTypes {
			if t == v {
				out = append(out, t)
			}
		}
	}
	return out
}

func ParseFilter(s string) []string {
	if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 20 {
		return nil
	}
	return splitList(s)
}

// FilterServers 对应 ClientController::filterServers。
func FilterServers(servers []Map, types, keywords []string) []Map {
	var out []Map
	for _, s := range servers {
		if len(types) > 0 {
			ok := false
			for _, t := range types {
				if t == Str(s["type"]) {
					ok = true
				}
			}
			if !ok {
				continue
			}
		}
		if len(keywords) > 0 {
			match := false
			tags, _ := s["tags"].([]any)
			for _, k := range keywords {
				if strings.Contains(strings.ToLower(Str(s["name"])), strings.ToLower(k)) {
					match = true
				}
				for _, t := range tags {
					if Str(t) == k {
						match = true
					}
				}
			}
			if !match {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

func TrafficConvert(b float64) string {
	const kb, mb, gb = 1024.0, 1048576.0, 1073741824.0
	r := func(v float64) float64 { return math.Round(v*100) / 100 }
	switch {
	case b > gb:
		return fmt.Sprintf("%v GB", r(b/gb))
	case b > mb:
		return fmt.Sprintf("%v MB", r(b/mb))
	case b > kb:
		return fmt.Sprintf("%v KB", r(b/kb))
	case b < 0:
		return "0"
	}
	return fmt.Sprintf("%v B", r(b))
}

// AddInfoServers 对应 setSubscribeInfoToServers：在列表头部插入过滤数/剩余流量/到期提示。
// 「距离下次重置」需要 TrafficResetService，尚未移植，故暂不插入该行。
func AddInfoServers(servers []Map, ctx *Ctx, rejected int, showInfo bool) []Map {
	if len(servers) == 0 {
		return servers
	}
	mk := func(name string) Map { m := cloneMap(servers[0]); m["name"] = name; return m }
	if rejected > 0 {
		servers = append([]Map{mk(fmt.Sprintf("过滤掉%d条线路", rejected))}, servers...)
	}
	if !showInfo {
		return servers
	}
	u := ctx.User
	exp := "长期有效"
	if u.ExpiredAt != nil && *u.ExpiredAt != 0 {
		exp = fmtDate(*u.ExpiredAt)
	}
	rest := "无限"
	if u.TransferEnab != 0 { // 流量上限为 0 视为无限流量
		rest = TrafficConvert(float64(u.TransferEnab - (u.U + u.D)))
	}
	servers = append([]Map{mk("套餐到期：" + exp)}, servers...)
	servers = append([]Map{mk("剩余流量：" + rest)}, servers...)
	return servers
}

var protoPrefix = map[string]string{
	"vless": "[vless]", "shadowsocks": "[ss]", "vmess": "[vmess]", "trojan": "[trojan]",
	"tuic": "[tuic]", "socks": "[socks]", "anytls": "[anytls]",
}

func AddPrefix(servers []Map) []Map {
	out := make([]Map, len(servers))
	for i, s := range servers {
		s = cloneMap(s)
		t := Str(s["type"])
		prefix := protoPrefix[t]
		if t == "hysteria" {
			prefix = "[Hy]"
			if Int(Get(s, "protocol_settings.version")) == 2 {
				prefix = "[Hy2]"
			}
		}
		s["name"] = prefix + Str(s["name"])
		out[i] = s
	}
	return out
}
